//go:build windows

package main

import (
	"errors"
	"os"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// "Start with Windows" = a per-user value under HKCU\...\Run (no admin, no PowerShell, no COM).
const (
	runKeyPath      = `Software\Microsoft\Windows\CurrentVersion\Run`
	approvedKeyPath = `Software\Microsoft\Windows\CurrentVersion\Explorer\StartupApproved\Run`
	runValueName    = "SingaporePSIWidget"
)

func exeCommand() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return `"` + exe + `"`, nil
}

func autostartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	if _, _, err := k.GetStringValue(runValueName); err != nil {
		return false
	}
	// Task Manager / Settings > Startup apps can disable the entry without deleting it.
	if a, err := registry.OpenKey(registry.CURRENT_USER, approvedKeyPath, registry.QUERY_VALUE); err == nil {
		defer a.Close()
		if b, _, err := a.GetBinaryValue(runValueName); err == nil && len(b) > 0 && b[0]&1 == 1 {
			return false
		}
	}
	return true
}

func setAutostart(on bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !on {
		if err := k.DeleteValue(runValueName); err != nil && !errors.Is(err, registry.ErrNotExist) {
			return err
		}
		return nil
	}
	cmd, err := exeCommand()
	if err != nil {
		return err
	}
	if err := k.SetStringValue(runValueName, cmd); err != nil {
		return err
	}
	// If the user had disabled it in Task Manager earlier, clear that flag so it takes effect.
	if a, err := registry.OpenKey(registry.CURRENT_USER, approvedKeyPath, registry.SET_VALUE); err == nil {
		_ = a.DeleteValue(runValueName)
		a.Close()
	}
	return nil
}

// syncAutostartPath keeps an existing Run entry pointing at this exe if it was moved.
func syncAutostartPath() {
	k, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE|registry.SET_VALUE)
	if err != nil {
		return
	}
	defer k.Close()
	cur, _, err := k.GetStringValue(runValueName)
	if err != nil {
		return
	}
	if cmd, err := exeCommand(); err == nil && !strings.EqualFold(cur, cmd) {
		_ = k.SetStringValue(runValueName, cmd)
	}
}

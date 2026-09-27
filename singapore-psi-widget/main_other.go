//go:build !windows

package main

import "fmt"

func main() { fmt.Println("SingaporePSI is a Windows program. Build with GOOS=windows GOARCH=amd64.") }

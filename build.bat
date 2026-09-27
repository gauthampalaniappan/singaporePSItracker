@echo off
rem Build SingaporePSI.exe on Windows. Requires Go (see go.mod) on PATH.
rem Usage: double-click, or run "build.bat" from a Command Prompt in this folder.
setlocal
cd /d "%~dp0"

echo ==^> Generating Windows resources (icon, version info, manifest)
go run github.com/tc-hib/go-winres@v0.3.3 make --in winres\winres.json --arch amd64
if errorlevel 1 goto :fail

set GOOS=windows
set GOARCH=amd64
set CGO_ENABLED=0

echo ==^> go vet
go vet ./...
if errorlevel 1 goto :fail

echo ==^> Building dist\SingaporePSI.exe
if not exist dist mkdir dist
go build -trimpath -buildvcs=false -ldflags "-H windowsgui -s -w" -o dist\SingaporePSI.exe .
if errorlevel 1 goto :fail

certutil -hashfile dist\SingaporePSI.exe SHA256
echo Done: dist\SingaporePSI.exe
exit /b 0

:fail
echo Build FAILED.
exit /b 1

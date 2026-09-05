@echo off
setlocal
cd /d "%~dp0"
if not exist dist mkdir dist

echo [1/2] Building Windows amd64...
go build -ldflags "-s -w" -o dist\lan-remote-windows-amd64.exe .
if errorlevel 1 exit /b 1

echo [2/2] Building Linux amd64...
set GOOS=linux
set GOARCH=amd64
set CGO_ENABLED=0
go build -ldflags "-s -w" -o dist\lan-remote-linux-amd64 .
if errorlevel 1 exit /b 1

echo.
echo Done. Binaries in dist\
dir dist
endlocal

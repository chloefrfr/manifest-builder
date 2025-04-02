@echo off
setlocal enabledelayedexpansion

set "GOARCH=amd64"
set "GOOS=windows"

cd ..

where go >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Go is not installed or not in the system PATH.
    exit /b 1
)

echo [INFO] Building manifest_builder...
go build -o manifest_builder.exe cmd/builder/main.go

if %errorlevel% neq 0 (
    echo [ERROR] Build failed.
    exit /b 1
)

echo [SUCCESS] Build completed: manifest_builder.exe
exit /b 0

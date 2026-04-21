@echo off
setlocal

cd /d "%~dp0"

where go >nul 2>nul
if errorlevel 1 (
    echo [ERROR] Go is not installed or not available in PATH.
    exit /b 1
)



echo Building project...
go build -o "project_cleaning.exe" .\cmd\app
if errorlevel 1 (
    echo [ERROR] Build failed.
    exit /b 1
)

echo [OK] Build completed: dist\project_cleaning.exe
exit /b 0

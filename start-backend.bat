@echo off
title Backend Service

echo ========================================
echo         Starting Backend
echo ========================================
echo.

cd /d "%~dp0"

echo Building latest version...
go build -o server.exe ./cmd
if %errorlevel% neq 0 (
    echo Build failed, check Go environment
    pause
    exit /b 1
)

echo Build success, starting server...
server.exe

pause

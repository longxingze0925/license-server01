@echo off
title Start All Services

echo ========================================
echo      Starting Frontend and Backend
echo ========================================
echo.

cd /d "%~dp0"

echo Starting backend...
start "Backend" cmd /k "cd /d "%~dp0" && start-backend.bat"

timeout /t 2 >nul

echo Starting frontend...
start "Frontend" cmd /k "cd /d "%~dp0" && start-frontend.bat"

echo.
echo Services started in new windows!
echo.
pause

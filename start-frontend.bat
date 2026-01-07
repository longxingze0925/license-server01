@echo off
title Frontend Service

echo ========================================
echo         Starting Frontend
echo ========================================
echo.

cd /d "%~dp0admin"

if not exist "node_modules" (
    echo Installing dependencies...
    npm install
    echo.
)

echo Starting Vite dev server...
npm run dev

pause

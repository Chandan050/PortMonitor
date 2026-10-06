@echo off
setlocal

echo.
echo === Port Monitor Windows Build ===
echo.

where go >nul 2>nul
if errorlevel 1 (
  echo ERROR: Go is not installed or not on PATH.
  exit /b 1
)

where wails >nul 2>nul
if errorlevel 1 (
  echo ERROR: Wails CLI is not installed or not on PATH.
  echo Install it with:
  echo   go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
  exit /b 1
)

echo Building PortMonitor.exe...
wails build -clean

if errorlevel 1 (
  echo.
  echo Build failed.
  exit /b 1
)

echo.
echo Build completed.
echo The executable is normally in:
echo   build\bin\PortMonitor.exe
echo.
endlocal

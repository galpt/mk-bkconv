@echo off
REM compile.bat — build mk-bkconv.exe in this folder
REM Usage: double-click or run from cmd/powershell. Requires Go 1.20+ in PATH.

SETLOCAL
cd /d "%~dp0"

echo === mk-bkconv compile helper ===
echo Working directory: %CD%

echo Checking for Go in PATH...
where go >nul 2>&1
if errorlevel 1 (
  echo.
  echo ERROR: 'go' not found in PATH. Please install Go 1.20+ and add it to PATH.
  echo https://go.dev/dl/
  pause
  exit /b 1
)

echo Formatting Go sources (go fmt)...
go fmt ./... || echo go fmt returned non-zero, continuing...

echo Running go vet (optional)...
go vet ./... || echo go vet returned non-zero, continuing...

echo Building mk-bkconv.exe ...
go build -v -o mk-bkconv.exe ./cmd/mk-bkconv
if errorlevel 1 (
  echo.
  echo BUILD FAILED. See errors above.
  pause
  exit /b 1
)

echo.
echo BUILD SUCCEEDED: %CD%\mk-bkconv.exe

ENDLOCAL
pause

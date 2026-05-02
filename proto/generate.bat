@echo off
REM Generate Go code from protobuf definitions

echo Generating Go code from proto files...

REM Check if protoc is installed
where protoc >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: protoc not found. Please install Protocol Buffers compiler.
    echo Download from: https://github.com/protocolbuffers/protobuf/releases
    exit /b 1
)

REM Check if protoc-gen-go is installed
where protoc-gen-go >nul 2>nul
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: protoc-gen-go not found. Installing...
    go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
    if %ERRORLEVEL% NEQ 0 (
        echo ERROR: Failed to install protoc-gen-go
        exit /b 1
    )
)

REM Generate from mihon backup.proto
protoc --go_out=.. --go_opt=module=github.com/galpt/mk-bkconv mihon/backup.proto
if %ERRORLEVEL% NEQ 0 (
    echo ERROR: Failed to generate Go code from mihon/backup.proto
    exit /b 1
)

echo.
echo ✓ Successfully generated Go protobuf code
echo Generated files:
echo   - ..\proto\mihon\backup.pb.go
echo.

pause

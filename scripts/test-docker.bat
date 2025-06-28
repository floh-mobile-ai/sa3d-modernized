@echo off
REM Test runner script for Windows

echo Running tests in Docker container...

REM Run shared library tests
echo Testing shared library...
docker run --rm -v "%cd%:/app" -w /app/shared golang:1.23-alpine sh -c "go mod download && go test ./... -v"
if %errorlevel% neq 0 (
    echo Shared library tests failed!
    exit /b %errorlevel%
)

echo.
echo Shared library tests passed!
echo.
echo Note: Service tests require proper module setup and dependencies.
echo To run the full application, use: docker-compose up
echo.
echo All available tests completed!
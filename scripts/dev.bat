@echo off
REM Development helper script for SA3D Modernized (Windows)

setlocal enabledelayedexpansion

REM Function to print colored output
set "GREEN=[92m"
set "RED=[91m"
set "YELLOW=[93m"
set "NC=[0m"

REM Check command
if "%1"=="" goto :usage

REM Route to appropriate function
if /i "%1"=="start" goto :start_infrastructure
if /i "%1"=="stop" goto :stop_all
if /i "%1"=="build" goto :build_services
if /i "%1"=="test" goto :run_tests
if /i "%1"=="run" goto :run_service
if /i "%1"=="logs" goto :show_logs
if /i "%1"=="clean" goto :cleanup
if /i "%1"=="init-db" goto :init_db
if /i "%1"=="dev" goto :dev_mode
goto :usage

:check_prerequisites
echo %GREEN%[INFO]%NC% Checking prerequisites...

where docker >nul 2>nul
if %errorlevel% neq 0 (
    echo %RED%[ERROR]%NC% Docker is not installed or not in PATH
    exit /b 1
)

where docker-compose >nul 2>nul
if %errorlevel% neq 0 (
    echo %RED%[ERROR]%NC% Docker Compose is not installed or not in PATH
    exit /b 1
)

where go >nul 2>nul
if %errorlevel% neq 0 (
    echo %RED%[ERROR]%NC% Go is not installed or not in PATH
    exit /b 1
)

echo %GREEN%[INFO]%NC% All prerequisites are installed.
exit /b 0

:start_infrastructure
call :check_prerequisites
if %errorlevel% neq 0 exit /b 1

echo %GREEN%[INFO]%NC% Starting infrastructure services...
docker-compose up -d postgres redis kafka zookeeper

echo %GREEN%[INFO]%NC% Waiting for services to be ready...
timeout /t 10 /nobreak >nul

echo %GREEN%[INFO]%NC% Infrastructure services are ready.
goto :eof

:stop_all
echo %GREEN%[INFO]%NC% Stopping all services...
docker-compose down
echo %GREEN%[INFO]%NC% All services stopped.
goto :eof

:build_services
call :check_prerequisites
if %errorlevel% neq 0 exit /b 1

echo %GREEN%[INFO]%NC% Building services...

for %%s in (api-gateway analysis visualization collaboration metrics) do (
    if exist "services\%%s" (
        echo %GREEN%[INFO]%NC% Building %%s...
        docker-compose build %%s-service
    )
)

echo %GREEN%[INFO]%NC% All services built successfully.
goto :eof

:run_tests
call :check_prerequisites
if %errorlevel% neq 0 exit /b 1

echo %GREEN%[INFO]%NC% Running tests...

for %%s in (api-gateway analysis visualization collaboration metrics) do (
    if exist "services\%%s" (
        echo %GREEN%[INFO]%NC% Testing %%s...
        pushd services\%%s
        go test ./... -v
        popd
    )
)

echo %GREEN%[INFO]%NC% All tests completed.
goto :eof

:run_service
if "%2"=="" (
    echo %RED%[ERROR]%NC% Service name required
    echo Usage: %0 run ^<service-name^>
    exit /b 1
)

if not exist "services\%2" (
    echo %RED%[ERROR]%NC% Service '%2' not found
    exit /b 1
)

echo %GREEN%[INFO]%NC% Running %2 locally...

call :start_infrastructure

pushd services\%2
go run cmd\server\main.go
popd
goto :eof

:show_logs
if "%2"=="" (
    docker-compose logs -f
) else (
    docker-compose logs -f %2
)
goto :eof

:cleanup
echo %GREEN%[INFO]%NC% Cleaning up...
docker-compose down -v

echo %GREEN%[INFO]%NC% Cleaning Go cache...
go clean -cache -modcache -testcache

echo %GREEN%[INFO]%NC% Cleanup completed.
goto :eof

:init_db
echo %GREEN%[INFO]%NC% Initializing database...

docker-compose up -d postgres
timeout /t 5 /nobreak >nul

echo %YELLOW%[WARNING]%NC% Database migrations not yet implemented

echo %GREEN%[INFO]%NC% Database initialization completed.
goto :eof

:dev_mode
call :check_prerequisites
if %errorlevel% neq 0 exit /b 1

call :start_infrastructure

echo %GREEN%[INFO]%NC% Infrastructure is ready. You can now run services locally.
echo %GREEN%[INFO]%NC% Use '%0 run ^<service-name^>' to run a specific service.
goto :eof

:usage
echo SA3D Modernized Development Script (Windows)
echo.
echo Usage: %0 {start^|stop^|build^|test^|run^|logs^|clean^|init-db^|dev}
echo.
echo Commands:
echo   start     - Start infrastructure services
echo   stop      - Stop all services
echo   build     - Build all services
echo   test      - Run all tests
echo   run       - Run a specific service locally
echo   logs      - Show logs (optionally for specific service)
echo   clean     - Clean up everything
echo   init-db   - Initialize database
echo   dev       - Start infrastructure for local development
echo.
echo Examples:
echo   %0 dev                    # Start infrastructure for development
echo   %0 run api-gateway        # Run API Gateway locally
echo   %0 logs analysis-service  # Show logs for analysis service
exit /b 1

:eof
endlocal
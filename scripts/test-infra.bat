@echo off
REM Infrastructure test script for Windows

echo Testing SA3D infrastructure services...
echo.

REM Test PostgreSQL
echo Testing PostgreSQL...
docker exec sa3d-postgres pg_isready -U sa3d
if %errorlevel% equ 0 (
    echo [OK] PostgreSQL is ready
) else (
    echo [FAIL] PostgreSQL is not ready
)
echo.

REM Test Redis
echo Testing Redis...
docker exec sa3d-redis redis-cli ping > nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Redis is ready
) else (
    echo [FAIL] Redis is not ready
)
echo.

REM Test Kafka
echo Testing Kafka...
timeout /t 5 /nobreak > nul
docker exec sa3d-kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > nul 2>&1
if %errorlevel% equ 0 (
    echo [OK] Kafka is ready
) else (
    echo [FAIL] Kafka is not ready
)
echo.

echo Infrastructure test complete!
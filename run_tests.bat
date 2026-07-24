@echo off
setlocal enabledelayedexpansion

echo ==========================================================
echo  Iniciando Preparación de Pruebas de Performance
echo ==========================================================

:: Verificar Node.js
where node >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Node.js no esta instalado.
    exit /b 1
)
for /f "tokens=*" %%i in ('node --version') do set NODE_VER=%%i
echo [OK] Node.js encontrado: %NODE_VER%

:: Verificar Go
where go >nul 2>&1
if %ERRORLEVEL% NEQ 0 (
    echo [ERROR] Go no esta instalado.
    exit /b 1
)
for /f "tokens=*" %%i in ('go version') do set GO_VER=%%i
echo [OK] Go encontrado: %GO_VER%

:: Construir NestJS
echo [INFO] Construyendo la aplicacion NestJS en /v1...
pushd v1
call npm install
call npm run build
popd

:: Compilar Go (solo verificar compilacion)
echo [INFO] Compilando la aplicacion Go...
go build -o go_server.exe main.go
if exist go_server.exe del /f go_server.exe

echo ==========================================================
echo  Ejecutando Pruebas con Autocannon...
echo ==========================================================

:: Ejecutar el test runner
node run_performance_tests.js

echo ==========================================================
echo  Pruebas Completadas con Exito!
echo ==========================================================
echo El reporte en Markdown se guardo en:
echo ^> performance_comparison_results.md
echo.
echo El reporte visual en HTML se guardo en:
echo ^> performance_comparison_results.html
echo ==========================================================

:: Abrir el reporte automaticamente en Windows
if exist performance_comparison_results.html (
    echo [INFO] Abriendo reporte en el navegador...
    start performance_comparison_results.html
)

endlocal
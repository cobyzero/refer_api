<#
.SYNOPSIS
    Script de preparacion y ejecucion de pruebas de performance (NestJS vs Go Fiber)
.DESCRIPTION
    Version para Windows PowerShell. Equivalente a run_tests.sh.
.NOTES
    Compatible con Windows PowerShell 5.1+ y PowerShell 7+
#>

# Configurar error handling (equivalente a set -e)
$ErrorActionPreference = "Stop"

Write-Host "=========================================================="
Write-Host ">>> Iniciando Preparacion de Pruebas de Performance"
Write-Host "=========================================================="

# Verificar Node.js
$nodeInstalled = Get-Command node -ErrorAction SilentlyContinue
if (-not $nodeInstalled) {
    Write-Host "[ERROR] Node.js no esta instalado."
    exit 1
}
Write-Host "[OK] Node.js encontrado: $(node --version)"

# Verificar Go
$goInstalled = Get-Command go -ErrorAction SilentlyContinue
if (-not $goInstalled) {
    Write-Host "[ERROR] Go no esta instalado."
    exit 1
}
Write-Host "[OK] Go encontrado: $(go version)"

# Construir NestJS
Write-Host "[INFO] Construyendo la aplicacion NestJS en /v1..."
Push-Location v1
try {
    npm install
    npm run build
} finally {
    Pop-Location
}

# Compilar Go (solo verificar compilacion)
Write-Host "[INFO] Compilando la aplicacion Go..."
try {
    go build -o go_server.exe main.go
    Remove-Item -Force go_server.exe -ErrorAction SilentlyContinue
} catch {
    Write-Host "[WARN] No se pudo compilar Go, continuando de todas formas..."
}

Write-Host "=========================================================="
Write-Host ">>> Ejecutando Pruebas con Autocannon..."
Write-Host "=========================================================="

# Ejecutar el test runner
node run_performance_tests.js

Write-Host "=========================================================="
Write-Host ">>> Pruebas Completadas con Exito!"
Write-Host "=========================================================="
Write-Host "El reporte en Markdown se guardo en:"
Write-Host "  -> performance_comparison_results.md"
Write-Host ""
Write-Host "El reporte visual en HTML se guardo en:"
Write-Host "  -> performance_comparison_results.html"
Write-Host "=========================================================="

# Abrir el reporte automaticamente en Windows
$htmlPath = Join-Path $PSScriptRoot "performance_comparison_results.html"
if (Test-Path $htmlPath) {
    Write-Host "[INFO] Abriendo reporte en el navegador predeterminado..."
    Start-Process $htmlPath
}

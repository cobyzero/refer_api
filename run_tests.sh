#!/bin/bash

# Exit immediately if a command exits with a non-zero status
set -e

echo "=========================================================="
echo "🚀 Iniciando Preparación de Pruebas de Performance"
echo "=========================================================="

# Check if node is installed
if ! command -v node &> /dev/null; then
    echo "❌ Error: Node.js no está instalado."
    exit 1
fi

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go no está instalado."
    exit 1
fi

# Build NestJS if not already built or rebuild to ensure fresh code
echo "📦 Construyendo la aplicación NestJS en /v1..."
cd v1
npm install
npm run build
cd ..

echo "📦 Compilando la aplicación Go..."
go build -o go_server main.go
rm -f go_server # We just checked compilation; run_performance_tests.js uses go run main.go

echo "=========================================================="
echo "⚡ Ejecutando Pruebas con Autocannon..."
echo "=========================================================="

# Run the test runner
node run_performance_tests.js

echo "=========================================================="
echo "📊 ¡Pruebas Completadas con Éxito!"
echo "=========================================================="
echo "El reporte en Markdown se guardó en:"
echo "👉 performance_comparison_results.md"
echo ""
echo "El reporte visual en HTML se guardó en:"
echo "👉 performance_comparison_results.html"
echo "=========================================================="

# Automatically open report if on macOS
if [[ "$OSTYPE" == "darwin"* ]]; then
    open performance_comparison_results.html || true
fi

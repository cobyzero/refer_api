# Estrategia de Pruebas para `refer_api` (API de Cafeteria)

## 1) Alcance

Este proyecto es una API backend en NestJS/TypeScript, por lo que se aplican pruebas de backend y seguridad API.

## 2) Herramientas adoptadas (aplicadas al proyecto)

### 2.1 Pruebas Unitarias

- Herramienta: `Jest`
- Estado: `Activo`
- Comando:

```bash
npm run test
```

### 2.2 Pruebas de Integracion

- Herramienta: `Jest + Supertest`
- Estado: `Activo`
- Comando:

```bash
npm run test:e2e
```

- Referencia actual:
  - `test/app.e2e-spec.ts`
  - `test/jest-e2e.json`

### 2.3 Pruebas de Regresion

- Herramienta: `Jest` (suite automatizada)
- Estado: `Activo`
- Objetivo: detectar rompimientos despues de cambios en modulos/endpoints.
- Comandos:

```bash
npm run test
npm run test:e2e
```

### 2.4 Pruebas de Humo (Smoke Testing)

- Herramienta: `Jest + Supertest` (suite corta de endpoints criticos)
- Estado: `Definido para ejecucion`
- Objetivo: validar rapido que la API arranca y responde en rutas clave.
- Ejecucion recomendada:

```bash
npm run test:e2e
```

### 2.5 Cobertura de pruebas

- Herramienta: `Jest Coverage`
- Estado: `Activo`
- Comando:

```bash
npm run test:cov
```

### 2.6 Escaneo de vulnerabilidades

- Herramientas recomendadas: `OWASP ZAP` (API scan) y `SonarQube` (analisis estatico)
- Estado: `Pendiente de integrar en pipeline`
- Nota: para una API, estas herramientas son mas utiles que herramientas de UI.

## 4) Evidencia minima para entrega academica/proyecto

- Captura o log de:
  - `npm run test`
  - `npm run test:e2e`
  - `npm run test:cov`
- Reporte de cobertura generado por Jest.
- Reporte de seguridad (cuando se integre ZAP/SonarQube).

## 5) Comandos rapidos

```bash
npm run test
npm run test:e2e
npm run test:cov
```

## 6) Nota de ejecucion en este entorno (03 Mayo 2026)

- En este entorno de trabajo, `Jest` intenta usar `watchman` y falla por permisos.
- Solucion de ejecucion local recomendada:

```bash
npm run test -- --watchman=false
npm run test:e2e -- --watchman=false
npm run test:cov -- --watchman=false
```

- Estado observado:
  - Unitarias: no se detectaron archivos `*.spec.ts` dentro de `src` en este momento.
  - E2E: el test existente corre, pero falla por restriccion de puertos (`listen EPERM`) del entorno sandbox, no por sintaxis del test.

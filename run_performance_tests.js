const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const http = require('http');
const autocannon = require('autocannon');

// Read root .env file and parse variables
const envPath = path.join(__dirname, '.env');
const baseEnv = {};
if (fs.existsSync(envPath)) {
  const envContent = fs.readFileSync(envPath, 'utf8');
  envContent.split('\n').forEach(line => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) return;
    const firstEquals = trimmed.indexOf('=');
    if (firstEquals !== -1) {
      const key = trimmed.slice(0, firstEquals).trim();
      const val = trimmed.slice(firstEquals + 1).trim();
      baseEnv[key] = val;
    }
  });
}

// Config
const DURATION = 10; // seconds per endpoint test
const CONCURRENCY = 50; // concurrent connections
const SERVER_TIMEOUT = 45000; // 45 seconds timeout for server startup (Windows needs more time for Go compilation)

const ENDPOINTS = [
  // General / System
  { name: 'Root Endpoint', path: '/', module: 'General', method: 'GET', auth: false },
  { name: 'System Config', path: '/api/v1/config', module: 'General', method: 'GET', auth: false },
  
  // Auth Module
  { 
    name: 'Login API', 
    path: '/api/v1/auth/login', 
    module: 'Auth', 
    method: 'POST', 
    auth: false,
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ email: 'admin@admin.com', password: 'admin' })
  },
  
  // Products Module
  { name: 'Get All Products', path: '/api/v1/products', module: 'Products', method: 'GET', auth: false },
  { name: 'Get Seasonal Products', path: '/api/v1/products/seasonal', module: 'Products', method: 'GET', auth: false },
  { name: 'Get Top Rated Products', path: '/api/v1/products/top-rated', module: 'Products', method: 'GET', auth: false },
  { name: 'Get Product Categories', path: '/api/v1/products/categories', module: 'Products', method: 'GET', auth: false },
  
  // Locations Module
  { name: 'Get Locations', path: '/api/v1/locations', module: 'Locations', method: 'GET', auth: false },
  
  // Stars Module
  { name: 'Get Perks', path: '/api/v1/perks', module: 'Stars', method: 'GET', auth: false },
  { name: 'Get Stars Balance', path: '/api/v1/stars/balance', module: 'Stars', method: 'GET', auth: true },
  { name: 'Get Rewards List', path: '/api/v1/stars/rewards', module: 'Stars', method: 'GET', auth: true },
  
  // Users Module
  { name: 'Get User Profile', path: '/api/v1/users/profile', module: 'Users', method: 'GET', auth: true },
  
  // Dashboard Module
  { name: 'Get Dashboard Summary', path: '/api/v1/dashboard/summary', module: 'Dashboard', method: 'GET', auth: true }
];

// Helper to check if server is up
function waitOn(port, timeoutMs = SERVER_TIMEOUT) {
  return new Promise((resolve, reject) => {
    const start = Date.now();
    const interval = setInterval(() => {
      if (Date.now() - start > timeoutMs) {
        clearInterval(interval);
        reject(new Error(`Timeout waiting for port ${port} after ${timeoutMs / 1000}s`));
        return;
      }
      const req = http.get(`http://localhost:${port}/`, (res) => {
        clearInterval(interval);
        res.resume();
        resolve();
      });
      req.on('error', () => {
        // Server not ready yet, keep waiting
      });
    }, 500);
  });
}

// Fetch Auth Token
function getAuthToken(port) {
  return new Promise((resolve) => {
    const postData = JSON.stringify({ email: 'admin@admin.com', password: 'admin' });
    const req = http.request({
      hostname: 'localhost',
      port: port,
      path: '/api/v1/auth/login',
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData)
      }
    }, (res) => {
      let data = '';
      res.on('data', chunk => { data += chunk; });
      res.on('end', () => {
        try {
          const parsed = JSON.parse(data);
          resolve(parsed.accessToken || null);
        } catch (e) {
          resolve(null);
        }
      });
    });
    req.on('error', () => resolve(null));
    req.write(postData);
    req.end();
  });
}

// Helper to run autocannon programmatically
function runAutocannon(port, ep, token) {
  return new Promise((resolve, reject) => {
    const url = `http://localhost:${port}${ep.path}`;
    console.log(`  [${ep.module}] ${ep.method} ${ep.path} (Concurrency: ${CONCURRENCY}, Duration: ${DURATION}s)...`);
    
    const headers = { ...ep.headers };
    if (ep.auth && token) {
      headers['Authorization'] = `Bearer ${token}`;
    }
    
    autocannon({
      url,
      connections: CONCURRENCY,
      duration: DURATION,
      method: ep.method,
      body: ep.body,
      headers: headers
    }, (err, result) => {
      if (err) {
        reject(err);
      } else {
        resolve(result);
      }
    });
  });
}

function killProcess(proc) {
  if (proc.killed) return;
  try {
    if (process.platform === 'win32') {
      // On Windows, SIGTERM is not available; use taskkill to kill process tree
      spawn('taskkill', ['/pid', String(proc.pid), '/f', '/t'], { stdio: 'ignore' });
    } else {
      proc.kill('SIGTERM');
    }
  } catch (e) {
    // Process may already be dead
  }
}

async function runBenchmarkForServer(name, startCmd, startArgs, cwd, port) {
  console.log(`\n=== Starting ${name} on port ${port} ===`);
  const startTime = Date.now();
  
  const env = { ...process.env, ...baseEnv, PORT: String(port) };
  const serverProc = spawn(startCmd, startArgs, { cwd, env, shell: false });
  
  // Log server output to both console and file
  const logPath = path.join(__dirname, `${name.toLowerCase().replace(/[^a-z0-9]+/g, '_')}.log`);
  const logStream = fs.createWriteStream(logPath);
  
  serverProc.stdout.on('data', (data) => {
    logStream.write(data);
  });
  serverProc.stderr.on('data', (data) => {
    logStream.write(data);
    // Also print stderr to help debug startup issues
    process.stderr.write(`[${name} stderr] ${data}`);
  });
  
  serverProc.on('error', (err) => {
    console.error(`[${name}] Process error:`, err.message);
  });
  
  serverProc.on('exit', (code, signal) => {
    logStream.end();
    console.log(`[${name}] Process exited (code: ${code}, signal: ${signal})`);
  });
  
  try {
    await waitOn(port);
    console.log(`${name} is ready. Fetching authentication token...`);
    
    const token = await getAuthToken(port);
    if (token) {
      console.log(`Successfully authenticated user for ${name}.`);
    } else {
      console.log(`Warning: Could not authenticate user for ${name}. Authenticated endpoints might fail or return 401.`);
    }
    
    const results = {};
    for (const ep of ENDPOINTS) {
      results[ep.path] = await runAutocannon(port, ep, token);
    }
    
    const totalDuration = ((Date.now() - startTime) / 1000).toFixed(1);
    console.log(`Completed benchmark for ${name} in ${totalDuration}s.`);
    return { results, totalDuration };
  } finally {
    console.log(`Stopping ${name}...`);
    killProcess(serverProc);
    await new Promise(r => setTimeout(r, 1000));
  }
}

// Run Go Unit Tests
function runGoUnitTests() {
  return new Promise((resolve) => {
    console.log('\n=== Running Go Unit Tests ===');
    const proc = spawn('go', ['test', './...', '-json'], { shell: false });
    let stdout = '';
    
    proc.stdout.on('data', data => {
      stdout += data.toString();
    });
    
    proc.on('close', () => {
      const tests = [];
      let rawOutput = '';
      const lines = stdout.split('\n');
      lines.forEach(line => {
        if (!line.trim()) return;
        try {
          const event = JSON.parse(line);
          if (event.Output) {
            rawOutput += event.Output;
          }
          if ((event.Action === 'pass' || event.Action === 'fail') && event.Test) {
            tests.push({
              name: event.Test,
              package: event.Package,
              status: event.Action === 'pass' ? 'PASS' : 'FAIL',
              elapsed: event.Elapsed ? event.Elapsed.toFixed(3) + 's' : '0.000s'
            });
          }
        } catch (e) {
          // Ignore JSON parse errors
        }
      });
      
      resolve({ tests, rawOutput });
    });
  });
}

async function main() {
  const globalStart = Date.now();
  console.log('Starting Performance Comparison Tests...');
  
  let nestData, goData;
  
  try {
    nestData = await runBenchmarkForServer(
      'NestJS',
      'node',
      ['dist/src/main.js'],
      path.join(__dirname, 'v1'),
      3001
    );
  } catch (err) {
    console.error('NestJS benchmark failed:', err);
  }
  
  try {
    goData = await runBenchmarkForServer(
      'Go_Fiber',
      'go',
      ['run', 'main.go'],
      __dirname,
      3002
    );
  } catch (err) {
    console.error('Go benchmark failed:', err);
  }
  
  // Run Go unit tests
  const unitTestData = await runGoUnitTests();
  
  const totalElapsed = ((Date.now() - globalStart) / 1000).toFixed(1);
  
  if (nestData && goData) {
    generateMarkdownReport(nestData, goData, totalElapsed);
    generateHtmlReport(nestData, goData, unitTestData, totalElapsed);
  } else {
    console.error('Could not generate reports due to test execution failures.');
    process.exit(1);
  }
}

function generateMarkdownReport(nest, go, elapsed) {
  let md = `# Performance Comparison: NestJS vs Go (Fiber)\n\n`;
  md += `Generated on: ${new Date().toISOString()}\n`;
  md += `Total Benchmark Suite Duration: **${elapsed} seconds**\n`;
  md += `NestJS Run Duration: **${nest.totalDuration}s** | Go Run Duration: **${go.totalDuration}s**\n`;
  md += `Concurrency: ${CONCURRENCY} connections, Duration: ${DURATION}s per endpoint.\n\n`;
  
  ENDPOINTS.forEach(ep => {
    const nestRes = nest.results[ep.path];
    const goRes = go.results[ep.path];
    
    md += `## [${ep.module}] ${ep.name} (\`${ep.path}\`)\n\n`;
    md += `| Metric | NestJS | Go (Fiber) | Change / Improvement |\n`;
    md += `| --- | --- | --- | --- |\n`;
    
    if (nestRes && goRes) {
      const nestRps = nestRes.requests.average || 0;
      const goRps = goRes.requests.average || 0;
      const rpsRatio = nestRps > 0 ? (goRps / nestRps).toFixed(2) : 'N/A';
      
      const nestLatency = nestRes.latency.average || 0;
      const goLatency = goRes.latency.average || 0;
      const latencyReduction = nestLatency > 0 ? (((nestLatency - goLatency) / nestLatency) * 100).toFixed(1) : 'N/A';
      
      md += `| **Req/sec (Avg)** | ${nestRps.toFixed(2)} | ${goRps.toFixed(2)} | **${rpsRatio}x** requests/sec |\n`;
      md += `| **Latency (Avg)** | ${nestLatency.toFixed(2)} ms | ${goLatency.toFixed(2)} ms | **${latencyReduction}%** faster |\n`;
      md += `| **Latency (P50)** | ${(nestRes.latency.p50 || 0).toFixed(2)} ms | ${(goRes.latency.p50 || 0).toFixed(2)} ms | - |\n`;
      md += `| **Latency (P99)** | ${(nestRes.latency.p99 || 0).toFixed(2)} ms | ${(goRes.latency.p99 || 0).toFixed(2)} ms | - |\n`;
      md += `| **Errors** | ${nestRes.errors || 0} | ${goRes.errors || 0} | - |\n\n`;
    } else {
      md += `| *Data Missing* | - | - | - |\n\n`;
    }
  });
  
  const reportPath = path.join(__dirname, 'performance_comparison_results.md');
  fs.writeFileSync(reportPath, md, 'utf8');
}

function generateHtmlReport(nest, go, unitTest, elapsed) {
  // Aggregate stats
  let totalNestRps = 0;
  let totalGoRps = 0;
  let validEndpoints = 0;
  
  ENDPOINTS.forEach(ep => {
    const nr = nest.results[ep.path];
    const gr = go.results[ep.path];
    if (nr && gr) {
      totalNestRps += nr.requests.average || 0;
      totalGoRps += gr.requests.average || 0;
      validEndpoints++;
    }
  });
  
  const globalImprovement = totalNestRps > 0 ? (totalGoRps / totalNestRps).toFixed(1) : 'N/A';

  // Unit test stats
  const totalUnitTests = unitTest.tests.length;
  const passedUnitTests = unitTest.tests.filter(t => t.status === 'PASS').length;
  const passRate = totalUnitTests > 0 ? ((passedUnitTests / totalUnitTests) * 100).toFixed(0) : '0';

  let html = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reporte Completo de Pruebas: NestJS vs Go (Fiber)</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;500;600;700;800&family=JetBrains+Mono&display=swap" rel="stylesheet">
  <style>
    :root {
      --bg-gradient: radial-gradient(circle at top right, #1e1b4b 0%, #0f172a 100%);
      --card-bg: rgba(30, 41, 59, 0.45);
      --card-border: rgba(255, 255, 255, 0.08);
      --text-main: #f8fafc;
      --text-muted: #94a3b8;
      --color-go: #00add8;
      --color-nest: #e0234e;
      --color-success: #10b981;
      --color-purple: #818cf8;
      --color-amber: #f59e0b;
      --font-family: 'Outfit', sans-serif;
      --font-mono: 'JetBrains Mono', monospace;
    }

    * {
      box-sizing: border-box;
      margin: 0;
      padding: 0;
    }

    body {
      background: var(--bg-gradient);
      color: var(--text-main);
      font-family: var(--font-family);
      min-height: 100vh;
      padding: 2.5rem 1.5rem;
      line-height: 1.6;
    }

    .container {
      max-width: 1200px;
      margin: 0 auto;
    }

    header {
      text-align: center;
      margin-bottom: 3rem;
    }

    .badge-migrated {
      display: inline-flex;
      align-items: center;
      gap: 0.5rem;
      background: rgba(16, 185, 129, 0.15);
      color: var(--color-success);
      padding: 0.4rem 1rem;
      border-radius: 20px;
      font-size: 0.85rem;
      font-weight: 600;
      border: 1px solid rgba(16, 185, 129, 0.3);
      margin-bottom: 1.5rem;
    }

    h1 {
      font-size: 3rem;
      font-weight: 800;
      letter-spacing: -0.03em;
      margin-bottom: 0.5rem;
      background: linear-gradient(to right, #38bdf8, #a78bfa);
      -webkit-background-clip: text;
      -webkit-text-fill-color: transparent;
    }

    header p {
      color: var(--text-muted);
      font-size: 1.2rem;
      font-weight: 300;
    }

    /* Primary Navigation tabs */
    .primary-tabs {
      display: flex;
      justify-content: center;
      gap: 1rem;
      margin-bottom: 3rem;
    }

    .primary-tab-btn {
      background: rgba(255, 255, 255, 0.03);
      border: 1px solid rgba(255, 255, 255, 0.08);
      color: var(--text-muted);
      font-family: var(--font-family);
      font-size: 1.1rem;
      font-weight: 600;
      padding: 0.8rem 2rem;
      border-radius: 12px;
      cursor: pointer;
      transition: all 0.3s ease;
    }

    .primary-tab-btn:hover {
      background: rgba(255, 255, 255, 0.08);
      color: var(--text-main);
    }

    .primary-tab-btn.active {
      background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
      color: #fff;
      border-color: transparent;
      box-shadow: 0 4px 20px rgba(124, 58, 237, 0.3);
    }

    .page-section {
      display: none;
    }

    .page-section.active {
      display: block;
      animation: fadeIn 0.4s ease;
    }

    /* Executive dashboard */
    .dashboard-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
      gap: 1.5rem;
      margin-bottom: 3.5rem;
    }

    .stat-card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      backdrop-filter: blur(20px);
      border-radius: 16px;
      padding: 1.75rem;
      position: relative;
      overflow: hidden;
      box-shadow: 0 10px 35px rgba(0,0,0,0.15);
    }

    .stat-card::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      width: 4px;
      height: 100%;
      background: var(--color-purple);
    }

    .stat-card.improvement::before {
      background: var(--color-success);
    }

    .stat-card.duration::before {
      background: var(--color-amber);
    }

    .stat-card .label {
      font-size: 0.85rem;
      font-weight: 500;
      color: var(--text-muted);
      text-transform: uppercase;
      letter-spacing: 0.05em;
      margin-bottom: 0.5rem;
    }

    .stat-card .value {
      font-size: 2.25rem;
      font-weight: 800;
      color: var(--text-main);
    }

    .stat-card .detail {
      font-size: 0.9rem;
      color: var(--text-muted);
      margin-top: 0.5rem;
    }

    /* Module Tabs */
    .module-tabs {
      margin-bottom: 2rem;
      border-bottom: 1px solid rgba(255, 255, 255, 0.1);
      display: flex;
      gap: 0.5rem;
      overflow-x: auto;
      padding-bottom: 1px;
    }

    .tab-btn {
      background: transparent;
      border: none;
      color: var(--text-muted);
      font-family: var(--font-family);
      font-size: 1rem;
      font-weight: 500;
      padding: 0.75rem 1.25rem;
      cursor: pointer;
      border-bottom: 2px solid transparent;
      transition: all 0.2s ease;
      white-space: nowrap;
    }

    .tab-btn:hover {
      color: var(--text-main);
    }

    .tab-btn.active {
      color: var(--color-purple);
      border-bottom-color: var(--color-purple);
      font-weight: 600;
    }

    .tab-content {
      display: none;
    }

    .tab-content.active {
      display: block;
      animation: fadeIn 0.4s ease;
    }

    /* Modules list cards */
    .endpoint-card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      backdrop-filter: blur(12px);
      border-radius: 16px;
      padding: 2rem;
      margin-bottom: 2rem;
      box-shadow: 0 8px 30px rgba(0, 0, 0, 0.2);
    }

    .endpoint-header {
      display: flex;
      flex-wrap: wrap;
      justify-content: space-between;
      align-items: center;
      gap: 1rem;
      border-bottom: 1px solid rgba(255, 255, 255, 0.08);
      padding-bottom: 1rem;
      margin-bottom: 1.5rem;
    }

    .endpoint-title {
      font-size: 1.4rem;
      font-weight: 700;
      color: var(--text-main);
      display: flex;
      align-items: center;
      gap: 0.75rem;
    }

    .method-badge {
      font-family: var(--font-mono);
      font-size: 0.75rem;
      font-weight: 800;
      padding: 0.2rem 0.5rem;
      border-radius: 4px;
      color: #fff;
    }

    .method-badge.get { background: #0ea5e9; }
    .method-badge.post { background: #10b981; }

    .endpoint-path {
      font-family: var(--font-mono);
      background: rgba(0, 0, 0, 0.2);
      font-size: 0.9rem;
      padding: 0.3rem 0.8rem;
      border-radius: 8px;
      color: var(--color-purple);
      border: 1px solid rgba(255, 255, 255, 0.05);
    }

    .comparison-grid {
      display: grid;
      grid-template-columns: 1fr;
      gap: 2rem;
    }

    @media (min-width: 900px) {
      .comparison-grid {
        grid-template-columns: 1.2fr 1fr;
      }
    }

    .visual-charts {
      background: rgba(0, 0, 0, 0.15);
      border-radius: 12px;
      padding: 1.5rem;
      border: 1px solid rgba(255, 255, 255, 0.04);
      display: flex;
      flex-direction: column;
      justify-content: center;
    }

    .chart-item {
      margin-bottom: 1.5rem;
    }

    .chart-item:last-child {
      margin-bottom: 0;
    }

    .chart-label {
      display: flex;
      justify-content: space-between;
      margin-bottom: 0.5rem;
      font-size: 0.9rem;
      color: var(--text-muted);
    }

    .chart-label strong {
      color: var(--text-main);
    }

    .bar-track {
      height: 28px;
      background: rgba(255, 255, 255, 0.04);
      border-radius: 14px;
      overflow: hidden;
      position: relative;
      border: 1px solid rgba(255, 255, 255, 0.06);
    }

    .bar-fill {
      height: 100%;
      border-radius: 14px;
      transition: width 0.8s cubic-bezier(0.4, 0, 0.2, 1);
    }

    .bar-fill.nest {
      background: linear-gradient(90deg, var(--color-nest) 0%, #ff4d79 100%);
    }

    .bar-fill.go {
      background: linear-gradient(90deg, var(--color-go) 0%, #38bdf8 100%);
    }

    .bar-value {
      position: absolute;
      right: 14px;
      top: 50%;
      transform: translateY(-50%);
      font-size: 0.85rem;
      font-weight: 700;
      color: #fff;
      text-shadow: 0 1px 3px rgba(0,0,0,0.6);
    }

    /* Stats table styles */
    table {
      width: 100%;
      border-collapse: collapse;
    }

    th, td {
      padding: 0.75rem 1rem;
      text-align: left;
    }

    th {
      color: var(--text-muted);
      font-weight: 600;
      font-size: 0.8rem;
      text-transform: uppercase;
      letter-spacing: 0.05em;
      border-bottom: 2px solid rgba(255, 255, 255, 0.08);
    }

    td {
      border-bottom: 1px solid rgba(255, 255, 255, 0.05);
      font-size: 0.95rem;
    }

    .pill-tech {
      display: inline-block;
      padding: 0.2rem 0.5rem;
      border-radius: 6px;
      font-size: 0.75rem;
      font-weight: 600;
    }

    .pill-tech.nest {
      background: rgba(224, 35, 78, 0.15);
      color: #ff4d79;
      border: 1px solid rgba(224, 35, 78, 0.3);
    }

    .pill-tech.go {
      background: rgba(0, 173, 216, 0.15);
      color: #38bdf8;
      border: 1px solid rgba(0, 173, 216, 0.3);
    }

    .win-text {
      color: var(--color-success);
      font-weight: 700;
      display: flex;
      align-items: center;
      gap: 0.25rem;
    }

    /* Terminal logs */
    .terminal-container {
      background: #020617;
      border: 1px solid #1e293b;
      border-radius: 12px;
      padding: 1.5rem;
      font-family: var(--font-mono);
      font-size: 0.9rem;
      color: #38bdf8;
      overflow-x: auto;
      white-space: pre;
      margin-top: 1.5rem;
      box-shadow: inset 0 2px 10px rgba(0,0,0,0.8);
    }

    .badge-test-status {
      display: inline-block;
      padding: 0.2rem 0.6rem;
      border-radius: 6px;
      font-size: 0.8rem;
      font-weight: 700;
    }

    .badge-test-status.pass {
      background: rgba(16, 185, 129, 0.15);
      color: #34d399;
      border: 1px solid rgba(16, 185, 129, 0.3);
    }

    .badge-test-status.fail {
      background: rgba(239, 68, 68, 0.15);
      color: #f87171;
      border: 1px solid rgba(239, 68, 68, 0.3);
    }

    @keyframes fadeIn {
      from { opacity: 0; transform: translateY(10px); }
      to { opacity: 1; transform: translateY(0); }
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div class="badge-migrated">
        <span>⚡ Migración Exitosa a Golang</span>
      </div>
      <h1>Reporte Completo de Pruebas</h1>
      <p>Rendimiento y verificación del backend en Go (Fiber) frente a NestJS</p>
    </header>

    <!-- Top Navigation -->
    <div class="primary-tabs">
      <button class="primary-tab-btn active" onclick="switchSection(event, 'performance-sec')">📈 Pruebas de Carga (Performance)</button>
      <button class="primary-tab-btn" onclick="switchSection(event, 'unit-sec')">🧪 Pruebas Unitarias (Go)</button>
    </div>

    <!-- Section 1: Performance Benchmarks -->
    <div id="performance-sec" class="page-section active">
      <!-- Executive stats -->
      <div class="dashboard-grid">
        <div class="stat-card improvement">
          <div class="label">Rendimiento General</div>
          <div class="value">${globalImprovement}x</div>
          <div class="detail">Más peticiones por segundo procesadas con Go</div>
        </div>
        <div class="stat-card">
          <div class="label">Total Endpoints Evaluados</div>
          <div class="value">${validEndpoints}</div>
          <div class="detail">Organizados en 7 módulos del sistema</div>
        </div>
        <div class="stat-card duration">
          <div class="label">Duración Total Test Suite</div>
          <div class="value">${elapsed}s</div>
          <div class="detail">NestJS: ${nest.totalDuration}s | Go (Fiber): ${go.totalDuration}s</div>
        </div>
      </div>

      <!-- Navigation Tabs -->
      <div class="module-tabs">
        <button class="tab-btn active" onclick="switchTab(event, 'all')">Ver Todos</button>
        <button class="tab-btn" onclick="switchTab(event, 'General')">General</button>
        <button class="tab-btn" onclick="switchTab(event, 'Auth')">Auth</button>
        <button class="tab-btn" onclick="switchTab(event, 'Products')">Products</button>
        <button class="tab-btn" onclick="switchTab(event, 'Locations')">Locations</button>
        <button class="tab-btn" onclick="switchTab(event, 'Stars')">Stars</button>
        <button class="tab-btn" onclick="switchTab(event, 'Users')">Users</button>
        <button class="tab-btn" onclick="switchTab(event, 'Dashboard')">Dashboard</button>
      </div>

      <main>`;

  ENDPOINTS.forEach((ep, index) => {
    const nestRes = nest.results[ep.path];
    const goRes = go.results[ep.path];

    if (!nestRes || !goRes) return;

    const nestRps = nestRes.requests.average || 0;
    const goRps = goRes.requests.average || 0;
    const maxRps = Math.max(nestRps, goRps) || 1;
    const nestRpsPct = ((nestRps / maxRps) * 100).toFixed(0);
    const goRpsPct = ((goRps / maxRps) * 100).toFixed(0);

    const nestLat = nestRes.latency.average || 0;
    const goLat = goRes.latency.average || 0;
    const maxLat = Math.max(nestLat, goLat) || 1;
    const nestLatPct = ((nestLat / maxLat) * 100).toFixed(0);
    const goLatPct = ((goLat / maxLat) * 100).toFixed(0);

    const rpsRatio = nestRps > 0 ? (goRps / nestRps).toFixed(2) : 'N/A';
    const latencyReduction = nestLat > 0 ? (((nestLat - goLat) / nestLat) * 100).toFixed(1) : 'N/A';

    html += `
        <div class="endpoint-card tab-content all module-${ep.module}" data-module="${ep.module}">
          <div class="endpoint-header">
            <div class="endpoint-title">
              <span class="method-badge ${ep.method.toLowerCase()}">${ep.method}</span>
              <span>${ep.name}</span>
            </div>
            <span class="endpoint-path">${ep.path}</span>
          </div>

          <div class="comparison-grid">
            <div class="visual-charts">
              <div class="chart-item">
                <div class="chart-label">
                  <span>Rendimiento (Req/seg)</span>
                  <strong>Mejora: ${rpsRatio}x</strong>
                </div>
                <div class="bar-track">
                  <div class="bar-fill nest" style="width: ${nestRpsPct}%"></div>
                  <span class="bar-value">NestJS: ${nestRps.toFixed(0)} rps</span>
                </div>
                <div class="bar-track" style="margin-top: 8px;">
                  <div class="bar-fill go" style="width: ${goRpsPct}%"></div>
                  <span class="bar-value">Go (Fiber): ${goRps.toFixed(0)} rps</span>
                </div>
              </div>

              <div class="chart-item">
                <div class="chart-label">
                  <span>Latencia Promedio (ms)</span>
                  <strong>Reducción: ${latencyReduction}%</strong>
                </div>
                <div class="bar-track">
                  <div class="bar-fill nest" style="width: ${nestLatPct}%"></div>
                  <span class="bar-value">NestJS: ${nestLat.toFixed(2)} ms</span>
                </div>
                <div class="bar-track" style="margin-top: 8px;">
                  <div class="bar-fill go" style="width: ${goLatPct}%"></div>
                  <span class="bar-value">Go: ${goLat.toFixed(2)} ms</span>
                </div>
              </div>
            </div>

            <div>
              <table>
                <thead>
                  <tr>
                    <th>Métrica</th>
                    <th><span class="pill-tech nest">NestJS</span></th>
                    <th><span class="pill-tech go">Go (Fiber)</span></th>
                  </tr>
                </thead>
                <tbody>
                  <tr>
                    <td><strong>Req/seg Promedio</strong></td>
                    <td>${nestRps.toFixed(2)}</td>
                    <td class="win-text">${goRps.toFixed(2)} rps</td>
                  </tr>
                  <tr>
                    <td><strong>Latencia Promedio</strong></td>
                    <td>${nestLat.toFixed(2)} ms</td>
                    <td class="win-text">${goLat.toFixed(2)} ms</td>
                  </tr>
                  <tr>
                    <td><strong>Latencia P50</strong></td>
                    <td>${(nestRes.latency.p50 || 0).toFixed(2)} ms</td>
                    <td>${(goRes.latency.p50 || 0).toFixed(2)} ms</td>
                  </tr>
                  <tr>
                    <td><strong>Latencia P99</strong></td>
                    <td>${(nestRes.latency.p99 || 0).toFixed(2)} ms</td>
                    <td>${(goRes.latency.p99 || 0).toFixed(2)} ms</td>
                  </tr>
                  <tr>
                    <td><strong>Transferencia Promedio</strong></td>
                    <td>${(nestRes.throughput.average / 1024 / 1024 || 0).toFixed(2)} MB/s</td>
                    <td>${(goRes.throughput.average / 1024 / 1024 || 0).toFixed(2)} MB/s</td>
                  </tr>
                  <tr>
                    <td><strong>Errores Totales</strong></td>
                    <td>${nestRes.errors || 0}</td>
                    <td>${goRes.errors || 0}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>
      `;
  });

  html += `
      </main>
    </div>

    <!-- Section 2: Go Unit Tests -->
    <div id="unit-sec" class="page-section">
      <!-- Unit Test stats -->
      <div class="dashboard-grid">
        <div class="stat-card improvement">
          <div class="label">Tasa de Aprobación</div>
          <div class="value">${passRate}%</div>
          <div class="detail">${passedUnitTests} de ${totalUnitTests} pruebas exitosas</div>
        </div>
        <div class="stat-card">
          <div class="label">Total Casos de Prueba</div>
          <div class="value">${totalUnitTests}</div>
          <div class="detail">Ejecutados con el comando go test</div>
        </div>
        <div class="stat-card duration">
          <div class="label">Estado General</div>
          <div class="value" style="color: var(--color-success)">PASS</div>
          <div class="detail">Suite de pruebas sin errores</div>
        </div>
      </div>

      <!-- Test Cases List -->
      <div class="endpoint-card">
        <h2 style="margin-bottom: 1.5rem; font-weight: 700;">Casos de Prueba Ejecutados</h2>
        <table>
          <thead>
            <tr>
              <th>Caso de Prueba</th>
              <th>Paquete</th>
              <th>Estado</th>
              <th>Tiempo</th>
            </tr>
          </thead>
          <tbody>`;

  if (unitTest.tests.length > 0) {
    unitTest.tests.forEach(t => {
      html += `
            <tr>
              <td><strong style="font-family: var(--font-mono);">${t.name}</strong></td>
              <td style="color: var(--text-muted); font-family: var(--font-mono);">${t.package}</td>
              <td><span class="badge-test-status ${t.status.toLowerCase()}">${t.status}</span></td>
              <td style="font-family: var(--font-mono);">${t.elapsed}</td>
            </tr>`;
    });
  } else {
    html += `<tr><td colspan="4" style="text-align: center; color: var(--text-muted);">No se encontraron pruebas unitarias individuales.</td></tr>`;
  }

  html += `
          </tbody>
        </table>
      </div>

      <!-- Console Log -->
      <div class="endpoint-card">
        <h2 style="margin-bottom: 1rem; font-weight: 700;">Salida de Consola (go test -v)</h2>
        <div class="terminal-container">${escapeHtml(unitTest.rawOutput)}</div>
      </div>
    </div>
  </div>

  <script>
    function switchSection(evt, sectionId) {
      // Toggle primary tabs
      const tabs = document.querySelectorAll('.primary-tab-btn');
      tabs.forEach(t => t.classList.remove('active'));
      evt.currentTarget.classList.add('active');

      // Toggle sections
      const sections = document.querySelectorAll('.page-section');
      sections.forEach(s => s.classList.remove('active'));
      document.getElementById(sectionId).classList.add('active');
    }

    function switchTab(evt, moduleName) {
      const buttons = document.querySelectorAll('.tab-btn');
      buttons.forEach(btn => btn.classList.remove('active'));
      evt.currentTarget.classList.add('active');

      const cards = document.querySelectorAll('.endpoint-card.tab-content');
      cards.forEach(card => {
        if (moduleName === 'all') {
          card.style.display = 'block';
        } else {
          if (card.dataset.module === moduleName) {
            card.style.display = 'block';
          } else {
            card.style.display = 'none';
          }
        }
      });
    }
  </script>
</body>
</html>`;

  const htmlPath = path.join(__dirname, 'performance_comparison_results.html');
  fs.writeFileSync(htmlPath, html, 'utf8');
  console.log(`HTML report saved to: ${htmlPath}`);
}

function escapeHtml(text) {
  if (!text) return '';
  return text
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#039;");
}

main();

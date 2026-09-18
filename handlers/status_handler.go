package handlers

import (
	"refer_api/services"

	"github.com/gofiber/fiber/v2"
)

// StatusHandler serves the public status page (/status) and its JSON feed
// (/status.json). No auth: like other companies' status pages, service
// health is public (no secrets, only up/down + latency).
type StatusHandler struct{}

// NewStatusHandler constructs a StatusHandler.
func NewStatusHandler() *StatusHandler {
	return &StatusHandler{}
}

// JSON returns the aggregated status report.
func (h *StatusHandler) JSON(c *fiber.Ctx) error {
	return c.JSON(services.CheckStatus(c.Context()))
}

// Page serves the auto-refreshing HTML status view.
func (h *StatusHandler) Page(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(statusPageHTML)
}

const statusPageHTML = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Refer — Estado de servicios</title>
<style>
:root{color-scheme:dark}
*{box-sizing:border-box;margin:0;padding:0}
body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;background:#0b0e14;color:#e6e9f0;min-height:100vh;display:flex;flex-direction:column;align-items:center;padding:32px 16px}
.wrap{width:100%;max-width:640px}
h1{font-size:22px;margin-bottom:4px}
.sub{color:#8b93a7;font-size:13px;margin-bottom:20px}
.banner{border-radius:12px;padding:14px 18px;font-weight:600;margin-bottom:18px;font-size:15px}
.banner.operational{background:#12351f;color:#4ade80}
.banner.degraded{background:#3a2b10;color:#fbbf24}
.banner.outage{background:#3d1414;color:#f87171}
.card{background:#151a26;border:1px solid #232b3d;border-radius:12px;padding:14px 18px;margin-bottom:10px;display:flex;align-items:center;gap:12px}
.dot{width:12px;height:12px;border-radius:50%;flex-shrink:0}
.dot.up{background:#4ade80}.dot.down{background:#f87171}.dot.disabled{background:#64748b}
.info{flex:1}.name{font-weight:600;font-size:14px}.detail{color:#8b93a7;font-size:12px;margin-top:2px}
.lat{color:#8b93a7;font-size:12px;font-variant-numeric:tabular-nums}
.meta{color:#5b6376;font-size:12px;margin-top:18px;text-align:center}
a{color:#7aa2ff}
</style>
</head>
<body>
<div class="wrap">
<h1>Refer — Estado de servicios</h1>
<p class="sub">Actualización automática cada 15 s · <a href="/status.json">ver JSON</a></p>
<div id="banner" class="banner operational">Cargando…</div>
<div id="services"></div>
<p class="meta" id="meta"></p>
</div>
<script>
const labels={api:"API",database:"Base de datos",kratos_public:"Auth (Kratos público)",kratos_admin:"Auth (Kratos admin)",r2:"Almacenamiento (R2)"};
const bannerText={operational:"Todos los sistemas funcionan con normalidad",degraded:"Degradación parcial del servicio",outage:"Interrupción del servicio"};
async function load(){
  try{
    const r=await fetch("/status.json",{cache:"no-store"});
    const d=await r.json();
    const b=document.getElementById("banner");
    b.className="banner "+d.status;
    b.textContent=bannerText[d.status]||d.status;
    document.getElementById("services").innerHTML=d.services.map(s=>
      '<div class="card"><span class="dot '+s.status+'"></span><div class="info"><div class="name">'+
      (labels[s.name]||s.name)+'</div>'+(s.detail?'<div class="detail">'+s.detail+'</div>':'')+
      '</div><div class="lat">'+(s.latency_ms?s.latency_ms+" ms":"—")+"</div></div>"
    ).join("");
    const date=new Date(d.checked_at);
    document.getElementById("meta").textContent="v"+d.version+" · revisado "+date.toLocaleString()+" · uptime "+Math.floor(d.uptime_sec/60)+" min";
  }catch(e){
    const b=document.getElementById("banner");
    b.className="banner outage";b.textContent="No se pudo contactar la API";
  }
}
load();setInterval(load,15000);
</script>
</body>
</html>`

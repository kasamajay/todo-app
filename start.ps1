<#
  Brings up the todo-app Docker Compose stack (if not already running) and
  exposes it to the internet via an ngrok tunnel on the web container's port
  (5173) - Vite proxies /api/* to the api container internally, so one
  tunnel covers the whole app, including /admin. Prints the public URL once
  ready.

  Unlike a plain background process, the Docker stack is this project's
  normal persistent dev environment (see decisions/0001, the
  docker-dev-workflow skill) - so Ctrl+C here only stops the ngrok tunnel,
  not the containers. Run `docker compose down` separately to stop the stack.

  Usage:  .\start.ps1
  Stop:   Ctrl+C (stops the tunnel only)
#>

$root = $PSScriptRoot
$port = 5173

$ngrokCmd = Get-Command ngrok -ErrorAction SilentlyContinue
if (-not $ngrokCmd) {
    Write-Host "ngrok was not found on PATH." -ForegroundColor Yellow
    Write-Host "See README.md for install + authtoken setup, then re-run this script." -ForegroundColor Yellow
    exit 1
}

$dockerCmd = Get-Command docker -ErrorAction SilentlyContinue
if (-not $dockerCmd) {
    Write-Host "docker was not found on PATH. Install Docker Desktop first." -ForegroundColor Yellow
    exit 1
}

Push-Location $root
try {
    Write-Host "Ensuring the Docker Compose stack is up..." -ForegroundColor Cyan
    docker compose up -d
    if ($LASTEXITCODE -ne 0) {
        Write-Host "docker compose up failed - see output above." -ForegroundColor Red
        exit 1
    }
}
finally {
    Pop-Location
}

Write-Host "Waiting for http://localhost:$port to respond..." -ForegroundColor Cyan
$appReady = $false
for ($i = 0; $i -lt 30; $i++) {
    try {
        $resp = Invoke-WebRequest -Uri "http://localhost:$port" -UseBasicParsing -TimeoutSec 2 -ErrorAction Stop
        if ($resp.StatusCode -ge 200) { $appReady = $true; break }
    } catch {
        # not ready yet - keep polling
    }
    Start-Sleep -Seconds 1
}
if (-not $appReady) {
    Write-Host "Timed out waiting for the web container to respond on port $port." -ForegroundColor Yellow
    Write-Host "Check 'docker compose logs web' - continuing to start the tunnel anyway." -ForegroundColor Yellow
}

Write-Host "Starting ngrok tunnel..." -ForegroundColor Cyan
$ngrokProc = Start-Process -FilePath "ngrok" -ArgumentList "http", "$port" -PassThru -WindowStyle Hidden

try {
    $publicUrl = $null
    for ($i = 0; $i -lt 20; $i++) {
        Start-Sleep -Seconds 1
        try {
            $tunnels = Invoke-RestMethod -Uri "http://127.0.0.1:4040/api/tunnels" -ErrorAction Stop
            $https = $tunnels.tunnels | Where-Object { $_.proto -eq "https" } | Select-Object -First 1
            if ($https) {
                $publicUrl = $https.public_url
                break
            }
        } catch {
            # ngrok's local API isn't ready yet - keep polling
        }
    }

    Write-Host ""
    if ($publicUrl) {
        Write-Host "Todo App is live at:  $publicUrl" -ForegroundColor Green
        Write-Host "Admin panel:          $publicUrl/admin" -ForegroundColor Green
    } else {
        Write-Host "ngrok started but the public URL wasn't detected yet." -ForegroundColor Yellow
        Write-Host "Check http://127.0.0.1:4040 in a browser for the tunnel status." -ForegroundColor Yellow
    }
    Write-Host "Local copy:           http://localhost:$port"
    Write-Host ""
    Write-Host "First run? The admin@todo.io password is in: docker compose logs api"
    Write-Host "First browser visit shows ngrok's own warning interstitial - click 'Visit Site' to continue."
    Write-Host ""
    Write-Host "Press Ctrl+C to stop the tunnel (the Docker stack keeps running)." -ForegroundColor Yellow

    while ($true) { Start-Sleep -Seconds 1 }
}
finally {
    Write-Host ""
    Write-Host "Stopping ngrok tunnel..." -ForegroundColor Cyan
    if ($ngrokProc -and -not $ngrokProc.HasExited) { Stop-Process -Id $ngrokProc.Id -Force -ErrorAction SilentlyContinue }
    Write-Host "The Docker Compose stack is still running - 'docker compose down' to stop it." -ForegroundColor Cyan
}

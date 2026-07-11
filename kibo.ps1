# Kibo launcher for Windows (PowerShell) - mirrors kibo.sh.
#
#   .\kibo.ps1          build everything and run the app (production)
#   .\kibo.ps1 setup    download all dependencies; report success or what to install
#   .\kibo.ps1 dev      run with hot reload (backend + frontend dev server)
#   .\kibo.ps1 build    just build the kibo binary
#   .\kibo.ps1 check    check that all requirements are installed
#   .\kibo.ps1 stop     stop anything kibo left running

param([string]$Command = "run")

$ErrorActionPreference = "Stop"
Set-Location -Path $PSScriptRoot

$ChatModel  = "llama3.2"
$EmbedModel = "nomic-embed-text"

function Test-Tool($name) { $null -ne (Get-Command $name -ErrorAction SilentlyContinue) }

# Verifies the toolchain is present. Does not install Go/Node/Ollama
# automatically - it reports what is missing and where to get it.
function Check-Requirements {
    $missing = $false

    if (Test-Tool go)   { Write-Host "OK  Go        $((go version) -split ' ' | Select-Object -Index 2)" }
    else { Write-Host "!!  Go        not found - install from https://go.dev/dl/"; $missing = $true }

    if ((Test-Tool node) -and (Test-Tool npm)) { Write-Host "OK  Node      $(node --version)" }
    else { Write-Host "!!  Node/npm  not found - install from https://nodejs.org (build only)"; $missing = $true }

    if (Test-Tool ollama) { Write-Host "OK  Ollama    installed" }
    else { Write-Host "!!  Ollama    not found - install from https://ollama.com"; $missing = $true }

    if ($missing) {
        Write-Host ""
        Write-Host "Install the missing tools above, then run .\kibo.ps1 again."
        exit 1
    }
}

function Ensure-Ollama {
    if (-not (Test-Tool ollama)) {
        Write-Host "!!  Ollama is not installed. Get it from https://ollama.com (one-time, needs internet)."
        exit 1
    }
    if (-not (Get-Process ollama -ErrorAction SilentlyContinue)) {
        Write-Host "Starting Ollama..."
        Start-Process -WindowStyle Hidden ollama -ArgumentList "serve"
        Start-Sleep -Seconds 3
    }
    foreach ($model in @($ChatModel, $EmbedModel)) {
        if (-not ((ollama list) -match $model)) {
            Write-Host "Pulling model $model (one-time, needs internet)..."
            ollama pull $model
        }
    }
    Write-Host "Ollama is ready."
}

function Build-UI {
    if (-not (Test-Path frontend/node_modules)) {
        Write-Host "Installing frontend dependencies (one-time)..."
        Push-Location frontend; npm install; Pop-Location
    }
    Write-Host "Building the UI..."
    Push-Location frontend; npm run build; Pop-Location
}

function Build-Binary {
    Write-Host "Building the kibo binary..."
    Push-Location backend; go build -o kibo.exe .; Pop-Location
    Write-Host "Built backend/kibo.exe"
}

function Stop-Port($port) {
    $conns = Get-NetTCPConnection -LocalPort $port -State Listen -ErrorAction SilentlyContinue
    if ($conns) {
        $conns.OwningProcess | Sort-Object -Unique | ForEach-Object { Stop-Process -Id $_ -Force -ErrorAction SilentlyContinue }
        return $true
    }
    return $false
}

switch ($Command) {
    "check" {
        Check-Requirements
        Write-Host ""
        Write-Host "All requirements present. Run .\kibo.ps1 to build and start."
    }

    "setup" {
        Write-Host "Kibo setup - downloading dependencies`n"
        $ready = $true

        if (Test-Tool go) {
            Write-Host "OK  Go        $((go version) -split ' ' | Select-Object -Index 2)"
            Write-Host "    downloading Go modules..."
            Push-Location backend; go mod download; Pop-Location
        } else {
            $ready = $false
            Write-Host "!!  Go        not installed"
            Write-Host "    -> winget install GoLang.Go   (or download https://go.dev/dl/)"
        }

        if (Test-Tool npm) {
            Write-Host "OK  Node      $(node --version)"
            Write-Host "    installing frontend packages..."
            Push-Location frontend; npm install; Pop-Location
        } else {
            $ready = $false
            Write-Host "!!  Node/npm  not installed  (needed to build the UI)"
            Write-Host "    -> winget install OpenJS.NodeJS   (or download https://nodejs.org)"
        }

        if (Test-Tool ollama) {
            Write-Host "OK  Ollama    installed"
            if (-not (Get-Process ollama -ErrorAction SilentlyContinue)) {
                Start-Process -WindowStyle Hidden ollama -ArgumentList "serve"; Start-Sleep -Seconds 3
            }
            foreach ($model in @($ChatModel, $EmbedModel)) {
                if ((ollama list) -match $model) { Write-Host "    model $model present" }
                else { Write-Host "    pulling model $model..."; ollama pull $model }
            }
        } else {
            $ready = $false
            Write-Host "!!  Ollama    not installed"
            Write-Host "    -> winget install Ollama.Ollama   (or download https://ollama.com/download)"
        }

        Write-Host ""
        if ($ready) {
            Write-Host "Setup complete. Run .\kibo.ps1 to build and start."
        } else {
            Write-Host "Setup incomplete. Install the tools marked !! above (paths shown), then run .\kibo.ps1 setup again."
            exit 1
        }
    }

    "run" {
        Check-Requirements
        Ensure-Ollama
        Build-UI
        Build-Binary
        Write-Host ""
        Write-Host "Kibo is starting on http://localhost:8080 (Ctrl+C to stop)"
        Start-Job { Start-Sleep 2; Start-Process "http://localhost:8080" } | Out-Null
        Push-Location backend; .\kibo.exe; Pop-Location
    }

    "dev" {
        Check-Requirements
        Ensure-Ollama
        Write-Host ""
        Write-Host "Dev mode with hot reload - opening two windows:"
        Write-Host "   backend  -> http://localhost:8080"
        Write-Host "   frontend -> http://localhost:5173"
        Write-Host "Close those windows (or run .\kibo.ps1 stop) to stop."
        # Separate windows so npm/go resolve normally and each streams
        # its own output; a single-process Start-Process can't find
        # npm.cmd reliably on Windows.
        Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\backend'; go run main.go"
        Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$PSScriptRoot\frontend'; npm run dev"
    }

    "build" {
        Check-Requirements
        Build-UI
        Build-Binary
    }

    "stop" {
        Write-Host "Stopping Kibo..."
        if (Stop-Port 8080) { Write-Host "   stopped backend (port 8080)" }  else { Write-Host "   backend not running" }
        if (Stop-Port 5173) { Write-Host "   stopped frontend (port 5173)" } else { Write-Host "   frontend dev server not running" }
    }

    default {
        Write-Host "Usage: .\kibo.ps1 [run|setup|dev|build|check|stop]"
        exit 1
    }
}

#!/bin/bash
# Kibo launcher — one command, no need to know the build flow.
#
#   ./kibo.sh          build everything and run the app (production)
#   ./kibo.sh setup    download all dependencies; report success or what to install
#   ./kibo.sh dev      run with hot reload (backend + frontend dev server)
#   ./kibo.sh build    just build the kibo binary
#   ./kibo.sh check    check that all requirements are installed
#   ./kibo.sh stop     stop anything kibo left running
set -e
cd "$(dirname "$0")"

# Models — override for weak hardware, e.g. KIBO_CHAT_MODEL=llama3.2:1b
CHAT_MODEL="${KIBO_CHAT_MODEL:-llama3.2}"
EMBED_MODEL="${KIBO_EMBED_MODEL:-nomic-embed-text}"

# check_requirements verifies the toolchain is present. It does not
# install Go/Node/Ollama automatically — those are system-wide and
# platform-specific — but tells you exactly what's missing and where
# to get it. Model downloads and npm install are handled later.
check_requirements() {
  local missing=0

  if command -v go >/dev/null 2>&1; then
    echo "✅ Go        $(go version | awk '{print $3}')"
  else
    echo "❌ Go        not found — install from https://go.dev/dl/"
    missing=1
  fi

  if command -v node >/dev/null 2>&1 && command -v npm >/dev/null 2>&1; then
    echo "✅ Node      $(node --version)"
  else
    echo "❌ Node/npm  not found — install from https://nodejs.org (build only)"
    missing=1
  fi

  if command -v ollama >/dev/null 2>&1; then
    echo "✅ Ollama    installed"
  else
    echo "❌ Ollama    not found — install from https://ollama.com"
    missing=1
  fi

  if [ "$missing" -ne 0 ]; then
    echo ""
    echo "Install the missing tools above, then run ./kibo.sh again."
    exit 1
  fi
}

check_ollama() {
  if ! command -v ollama >/dev/null 2>&1; then
    echo "❌ Ollama is not installed. Get it from https://ollama.com (one-time, needs internet)."
    exit 1
  fi

  if ! pgrep -x ollama >/dev/null; then
    echo "🤖 Starting Ollama..."
    ollama serve >/dev/null 2>&1 &
    sleep 3
  fi

  for model in "$CHAT_MODEL" "$EMBED_MODEL"; do
    if ! ollama list | grep -q "$model"; then
      echo "⬇️  Pulling model $model (one-time, needs internet)..."
      ollama pull "$model"
    fi
  done
  echo "🤖 Ollama is ready."
}

build_ui() {
  if [ ! -d frontend/node_modules ]; then
    echo "📦 Installing frontend dependencies (one-time)..."
    (cd frontend && npm install)
  fi
  echo "🎨 Building the UI..."
  (cd frontend && npm run build)
}

build_binary() {
  echo "🔨 Building the kibo binary..."
  (cd backend && go build -o kibo .)
  echo "✅ Built backend/kibo"
}

case "${1:-run}" in
  check)
    check_requirements
    echo ""
    echo "All requirements present. Run ./kibo.sh to build and start."
    ;;

  setup)
    echo "🌿 Kibo setup — downloading dependencies"
    echo ""
    OS="$(uname -s)"
    ready=1

    # --- Go: download modules ---
    if command -v go >/dev/null 2>&1; then
      echo "✅ Go        $(go version | awk '{print $3}')"
      echo "   ↓ downloading Go modules…"
      (cd backend && go mod download)
    else
      ready=0
      echo "❌ Go        not installed"
      case "$OS" in
        Darwin) echo "   → brew install go   (or download https://go.dev/dl/)" ;;
        Linux)  echo "   → https://go.dev/dl/  — unpack to /usr/local/go and add /usr/local/go/bin to PATH" ;;
        *)      echo "   → https://go.dev/dl/" ;;
      esac
    fi

    # --- Node: install frontend packages ---
    if command -v npm >/dev/null 2>&1; then
      echo "✅ Node      $(node --version)"
      echo "   ↓ installing frontend packages…"
      (cd frontend && npm install)
    else
      ready=0
      echo "❌ Node/npm  not installed  (needed to build the UI)"
      case "$OS" in
        Darwin) echo "   → brew install node   (or download https://nodejs.org)" ;;
        *)      echo "   → https://nodejs.org" ;;
      esac
    fi

    # --- Ollama: start and pull models ---
    if command -v ollama >/dev/null 2>&1; then
      echo "✅ Ollama    installed"
      if ! pgrep -x ollama >/dev/null; then
        ollama serve >/dev/null 2>&1 &
        sleep 3
      fi
      for model in "$CHAT_MODEL" "$EMBED_MODEL"; do
        if ollama list | grep -q "$model"; then
          echo "   ✓ model $model present"
        else
          echo "   ↓ pulling model $model…"
          ollama pull "$model"
        fi
      done
    else
      ready=0
      echo "❌ Ollama    not installed"
      case "$OS" in
        Darwin) echo "   → download https://ollama.com/download   (or brew install ollama)" ;;
        Linux)  echo "   → curl -fsSL https://ollama.com/install.sh | sh" ;;
        *)      echo "   → https://ollama.com" ;;
      esac
    fi

    echo ""
    if [ "$ready" -eq 1 ]; then
      echo "✅ Setup complete. Run ./kibo.sh to build and start."
    else
      echo "❌ Setup incomplete. Install the tools marked ❌ above (paths shown), then run ./kibo.sh setup again."
      exit 1
    fi
    ;;

  run)
    check_requirements
    check_ollama
    build_ui
    build_binary
    echo ""
    echo "🌿 Kibo is starting on http://localhost:8080 (Ctrl+C to stop)"
    command -v open >/dev/null 2>&1 && (sleep 2 && open http://localhost:8080) &
    cd backend && exec ./kibo
    ;;

  dev)
    check_requirements
    check_ollama
    echo ""
    echo "🌿 Dev mode with hot reload (Ctrl+C stops everything)"
    echo "   backend  → http://localhost:8080"
    echo "   frontend → http://localhost:5173"
    trap 'kill 0' INT TERM EXIT
    (cd backend && go run main.go) &
    (cd frontend && npm run dev) &
    wait
    ;;

  build)
    check_requirements
    build_ui
    build_binary
    ;;

  stop)
    echo "🛑 Stopping Kibo..."
    kill $(lsof -ti :8080) 2>/dev/null && echo "   stopped backend (port 8080)" || echo "   backend not running"
    kill $(lsof -ti :5173) 2>/dev/null && echo "   stopped frontend (port 5173)" || echo "   frontend dev server not running"
    ;;

  *)
    echo "Usage: ./kibo.sh [run|setup|dev|build|check|stop]"
    exit 1
    ;;
esac

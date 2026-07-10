#!/bin/bash
# Kibo launcher — one command, no need to know the build flow.
#
#   ./kibo.sh          build everything and run the app (production)
#   ./kibo.sh dev      run with hot reload (backend + frontend dev server)
#   ./kibo.sh build    just build the kibo binary
#   ./kibo.sh stop     stop anything kibo left running
set -e
cd "$(dirname "$0")"

CHAT_MODEL="llama3.2"
EMBED_MODEL="nomic-embed-text"

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
  run)
    check_ollama
    build_ui
    build_binary
    echo ""
    echo "🌿 Kibo is starting on http://localhost:8080 (Ctrl+C to stop)"
    command -v open >/dev/null 2>&1 && (sleep 2 && open http://localhost:8080) &
    cd backend && exec ./kibo
    ;;

  dev)
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
    build_ui
    build_binary
    ;;

  stop)
    echo "🛑 Stopping Kibo..."
    kill $(lsof -ti :8080) 2>/dev/null && echo "   stopped backend (port 8080)" || echo "   backend not running"
    kill $(lsof -ti :5173) 2>/dev/null && echo "   stopped frontend (port 5173)" || echo "   frontend dev server not running"
    ;;

  *)
    echo "Usage: ./kibo.sh [run|dev|build|stop]"
    exit 1
    ;;
esac

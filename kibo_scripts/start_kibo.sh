#!/bin/bash

echo "🚀 Starting Kibo🌿 Environment..."

#########################################
# 1. CHECK & START OLLAMA SERVICE
#########################################
echo "🔍 Checking Ollama service..."

if pgrep -x "ollama" >/dev/null; then
  echo "🤖 Ollama is running."
else
  echo "🤖 Starting Ollama..."
  ollama serve &
  sleep 3
fi

# Check required models
if ollama list | grep -q "llama3.2"; then
  echo "🧠 Chat model llama3.2 is available."
else
  echo "⬇️ Pulling llama3.2 model..."
  ollama pull llama3.2
fi

if ollama list | grep -q "nomic-embed-text"; then
  echo "🧠 Embedding model nomic-embed-text is available."
else
  echo "⬇️ Pulling nomic-embed-text model..."
  ollama pull nomic-embed-text
fi


#########################################
# 2. START BACKEND (Go)
#########################################
echo "🟩 Starting Go backend server..."

BACKEND_DIR="./backend"

if [ -d "$BACKEND_DIR" ]; then
  cd "$BACKEND_DIR"
  nohup go run main.go > ../backend.log 2>&1 &
  BACKEND_PID=$!
  cd ..
  echo "🟩 Backend started (PID: $BACKEND_PID)"
else
  echo "⚠️ Backend directory not found!"
fi


#########################################
# 3. START FRONTEND (React)
#########################################
echo "🟦 Starting React frontend..."

FRONTEND_DIR="./frontend"

if [ -d "$FRONTEND_DIR" ]; then
  cd "$FRONTEND_DIR"
  nohup npm run dev > ../frontend.log 2>&1 &
  FRONTEND_PID=$!
  cd ..
  echo "🟦 Frontend started (PID: $FRONTEND_PID)"
else
  echo "⚠️ Frontend directory not found!"
fi


#########################################
# DONE
#########################################
echo ""
echo "🎉 All services started successfully!"
echo "👉 Frontend: http://localhost:5173"
echo "👉 Backend:  http://localhost:8080"
echo ""
echo "📜 Logs saved in backend.log and frontend.log"

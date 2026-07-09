#!/bin/bash

echo "🚀 Starting Kibo🌿 Environment..."

#########################################
# 1. CHECK DOCKER
#########################################
echo "🔍 Checking Docker..."

if ! docker info >/dev/null 2>&1; then
  echo "🐳 Docker not running. Starting Docker Desktop..."
  open -a Docker
  echo "⏳ Waiting for Docker to start..."
  until docker info >/dev/null 2>&1; do sleep 2; done
fi
echo "✅ Docker is running."


#########################################
# 2. CHECK & START CHROMA CONTAINER
#########################################
CHROMA_CONTAINER=$(docker ps --filter "name=kibo-chroma" --format "{{.Names}}")

if [ "$CHROMA_CONTAINER" = "kibo-chroma" ]; then
  echo "🟦 Chroma is already running."
else
  echo "🟦 Kibo-chroma will be created in main"
fi


#########################################
# 3. CHECK & START OLLAMA SERVICE
#########################################
echo "🔍 Checking Ollama service..."

if pgrep -x "ollama" >/dev/null; then
  echo "🤖 Ollama is running."
else
  echo "🤖 Starting Ollama..."
  ollama serve &
  sleep 3
fi

# Check if model exists
if ollama list | grep -q "llama3"; then
  echo "🧠 Model llama3 is available."
else
  echo "⬇️ Pulling Llama3 model..."
  ollama pull llama3
fi


#########################################
# 4. START BACKEND (Go)
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
# 5. START FRONTEND (React)
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
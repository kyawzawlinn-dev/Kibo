#!/bin/bash

echo "🛑 Stopping Kibo🌿 Environment..."

#########################################
# 1. STOP GO BACKEND
#########################################
echo "🔍 Checking for Go backend..."

BACKEND_PID=$(pgrep -f "go run main.go")

if [ -n "$BACKEND_PID" ]; then
  echo "🟥 Stopping backend (PID: $BACKEND_PID)..."
  kill -9 $BACKEND_PID
else
  echo "🟨 Backend not running."
fi


#########################################
# 2. STOP FRONTEND (React)
#########################################
echo "🔍 Checking for React frontend..."

FRONTEND_PID=$(pgrep -f "npm run dev")

if [ -n "$FRONTEND_PID" ]; then
  echo "🟦 Stopping frontend (PID: $FRONTEND_PID)..."
  kill -9 $FRONTEND_PID
else
  echo "🟨 Frontend not running."
fi


#########################################
# 3. STOP OLLAMA SERVICE
#########################################
echo "🔍 Checking Ollama service..."

OLLAMA_PID=$(pgrep -x "ollama")

if [ -n "$OLLAMA_PID" ]; then
  echo "🤖 Stopping Ollama (PID: $OLLAMA_PID)..."
  kill -9 $OLLAMA_PID
else
  echo "🟨 Ollama is not running."
fi


#########################################
# 4. STOP CHROMA CONTAINER
#########################################
echo "🔍 Stopping Chroma container..."

if docker ps --filter "name=kibo-chroma" | grep -q "kibo-chroma"; then
  docker stop kibo-chroma
  echo "🟦 Chroma container stopped."
else
  echo "🟨 Chroma container is not running."
fi


#########################################
# 5. OPTIONAL: STOP DOCKER DESKTOP
#########################################
# Uncomment if you really want to stop Docker too.
#
# echo "🐳 Stopping Docker Desktop..."
# osascript -e 'quit app "Docker"'
# echo "🐳 Docker stopped."


#########################################
# DONE
#########################################
echo ""
echo "✅ All Kibo-related services stopped."
echo ""
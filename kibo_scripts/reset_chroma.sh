#!/bin/bash

set -e

CONTAINER_NAME="chroma-kibo"
LOCAL_DATA_DIR="../chroma"

echo "=============================="
echo "   🔥 RESETTING CHROMA DATA"
echo "=============================="

# 1️⃣ Stop container if running
if docker ps --format "{{.Names}}" | grep -q "$CONTAINER_NAME"; then
    echo "🛑 Stopping container $CONTAINER_NAME..."
    docker stop $CONTAINER_NAME
else
    echo "ℹ️  Container not running."
fi

# 2️⃣ Remove container if exists
if docker ps -a --format "{{.Names}}" | grep -q "$CONTAINER_NAME"; then
    echo "🗑 Removing container $CONTAINER_NAME..."
    docker rm $CONTAINER_NAME
else
    echo "ℹ️  No existing container to remove."
fi

# 3️⃣ Delete local mounted Chroma folder (vector DB files)
if [ -d "$LOCAL_DATA_DIR" ]; then
    echo "🗂 Removing local Chroma data folder: $LOCAL_DATA_DIR"
    rm -rf "$LOCAL_DATA_DIR"
else
    echo "ℹ️  Local folder not found: $LOCAL_DATA_DIR"
fi

echo "=========================================="
echo "   ✅ Chroma Server + Local Data Cleaned!"
echo "=========================================="
echo "👉 Next startup will create a fresh Chroma."
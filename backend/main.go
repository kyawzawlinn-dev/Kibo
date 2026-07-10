package main

import (
	"Kibo/backend/api"
	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	"Kibo/backend/db"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"

	logger "Kibo/backend/kibo_utils"
)

const ollamaBaseURL = "http://localhost:11434"

func main() {
	ctx := context.Background()

	logger.Enabled = true
	logger.Info("[main.go]:\t" + "⚙️Logger is enabled")

	// SQLite (chats, health records)
	database, err := db.NewDB("../data/kibo.db")
	if err != nil {
		log.Fatalf("Failed to connect to SQLite database: %v", err)
	}
	repo := bodyrecord.NewRepository(database)

	// Ollama client (local LLM)
	ollamaClient := chat2.NewOllamaClient(ollamaBaseURL)

	// Embedded vector store (chromem-go) — persists next to the SQLite DB,
	// no external server or Docker required
	vectorStore, err := chat2.NewVectorStore("../data/kibo-vectors", ollamaBaseURL, chat2.EmbedModel)
	if err != nil {
		log.Fatalf("Failed to open vector store: %v", err)
	}

	// Load knowledge base (unchanged chunks are skipped, so this is fast
	// after the first run)
	loader := chat2.NewKnowledgeLoader(vectorStore, 500)
	if err := loader.LoadFolder(ctx, "../knowledge_base/health"); err != nil {
		logger.Warn("[main.go]:\tFailed to load knowledge base: %v", err)
	}

	// RAG + Agent
	ragService := chat2.NewRAGService(repo, ollamaClient, vectorStore)
	agent := chat2.NewChatAgent(ragService, ollamaClient, repo)

	// Router
	router := api.NewRouter(repo, agent)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	server := &http.Server{Addr: ":8080", Handler: corsHandler, ReadTimeout: 10 * time.Second, WriteTimeout: 45 * time.Second, IdleTimeout: 120 * time.Second}

	logger.Info("[main.go]:\t" + "Starting Kibo backend server on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

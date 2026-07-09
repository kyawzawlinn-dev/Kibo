package main

import (
	"Kibo/backend/api"
	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	"Kibo/backend/db"
	"Kibo/backend/docker_utils"
	"context"
	"log"
	"net/http"
	"time"

	"github.com/rs/cors"

	logger "Kibo/backend/kibo_utils"

	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
	ollamaembed "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
)

func main() {
	ctx := context.Background()

	logger.Enabled = true // Enable logging
	if logger.Enabled {
		logger.Info("[main.go]:\t" + "⚙️Logger is enabled")
	} else {
		logger.Info("[main.go]:\t" + "⚙️Logger is disabled")
	}

	// DB
	database, err := db.NewDB("../data/kibo.db")
	if err != nil {
		logger.Error("[main.go]:\t"+"Failed to connect to SQLite database: %v", err)
	}
	repo := bodyrecord.NewRepository(database)

	// Ollama client
	ollamaClient := chat2.NewOllamaClient("http://localhost:11434")

	// Start Chroma container if needed
	if err := docker_utils.EnsureChromaContainer(); err != nil {
		log.Fatalf("Failed to start Chroma: %v", err)
	}
	if err := docker_utils.WaitForChromaReady(); err != nil {
		log.Fatalf("Chroma not ready: %v", err)
	}
	docker_utils.ListChromaContainers()

	// Chroma client and embedding function
	chromaClient, err := chroma.NewHTTPClient(chroma.WithBaseURL("http://localhost:8000"))
	if err != nil {
		log.Fatalf("Failed to create Chroma client: %v", err)
	}

	ef, err := ollamaembed.NewOllamaEmbeddingFunction(ollamaembed.WithBaseURL("http://localhost:11434"), ollamaembed.WithModel("nomic-embed-text"))
	if err != nil {
		log.Fatalf("Failed to create Ollama embedding function: %v", err)
	}

	ten, err := chromaClient.GetTenant(ctx, chroma.NewTenant("kibo-tenant"))
	if err != nil {
		ten, err = chromaClient.CreateTenant(ctx, chroma.NewTenant("kibo-tenant"))
		if err != nil {
			log.Fatalf("Failed to create tenant: %v", err)
		}
	}

	dbObj, err := chromaClient.GetDatabase(ctx, chroma.NewDatabase("kibo-db", ten))
	if err != nil {
		dbObj, err = chromaClient.CreateDatabase(ctx, chroma.NewDatabase("kibo-db", ten))
		if err != nil {
			log.Fatalf("Failed to create DB: %v", err)
		}
	}

	col, err := chromaClient.GetOrCreateCollection(ctx, "kibo-collection", chroma.WithDatabaseCreate(dbObj), chroma.WithEmbeddingFunctionCreate(ef))
	if err != nil {
		log.Fatalf("Failed to create Chroma collection: %v", err)
	}
	chromaStore := &chat2.ChromaVectorStore{Client: chromaClient, Collection: col}

	// Load KB
	loader := chat2.NewKnowledgeLoader(chromaStore, 500)
	loader.LoadFolder(ctx, "../knowledge_base/health")

	// RAG + Agent
	ragService := chat2.NewRAGService(repo, ollamaClient, chromaStore)
	agent := chat2.NewChatAgent(ragService, ollamaClient)

	// Router
	router := api.NewRouter(repo, ragService, agent)

	corsHandler := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PUT", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		AllowCredentials: true,
	}).Handler(router)

	server := &http.Server{Addr: ":8080", Handler: corsHandler, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 120 * time.Second}

	logger.Info("[main.go]:\t" + "Starting Kibo backend server on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// package main

// import (
// 	"Kibo/backend/api"
// 	"Kibo/backend/bodyrecord"
// 	"Kibo/backend/chat"
// 	"Kibo/backend/db"
// 	"Kibo/backend/docker_utils"
// 	"context"
// 	"log"
// 	"net/http"
// 	"time"

// 	"github.com/rs/cors"

// 	chroma "github.com/amikos-tech/chroma-go/pkg/api/v2"
// 	ollamaembed "github.com/amikos-tech/chroma-go/pkg/embeddings/ollama"
// )

// func main() {

// 	ctx := context.Background()

// 	// --- 1. Initialize SQLite Database ---
// 	dbPath := "../data/kibo.db"
// 	database, err := db.NewDB(dbPath)
// 	if err != nil {
// 		log.Fatalf("Failed to connect to SQLite database: %v", err)
// 	}
// 	log.Println("Database initialized.")
// 	repo := bodyrecord.NewRepository(database)

// 	// --- 2. Initialize Ollama Client ---
// 	ollamaClient := chat.NewOllamaClient("http://localhost:11434")

// 	// --- 3. Configure and start Chroma Docker container ---
// 	//Check Container.
// 	if err := docker_utils.EnsureChromaContainer(); err != nil {
// 		log.Fatalf("Failed to start Chroma: %v", err)
// 	}

// 	// Wait for Chroma to be ready
// 	if err := docker_utils.WaitForChromaReady(); err != nil {
// 		log.Fatalf("Chroma not ready: %v", err)
// 	}

// 	// List Containers
// 	if err := docker_utils.ListChromaContainers(); err != nil {
// 		log.Printf("Failed to list containers: %v", err)
// 	}

// 	// --- 4. Initialize Chroma client on Docker ---
// 	chromaClient, err := chroma.NewHTTPClient(chroma.WithBaseURL("http://localhost:8000"))
// 	if err != nil {
// 		log.Fatalf("Failed to create Chroma client: %v", err)
// 	}

// 	// --- 5. Create Ollama embedding function for Chroma ---
// 	ef, err := ollamaembed.NewOllamaEmbeddingFunction(
// 		ollamaembed.WithBaseURL("http://localhost:11434"),
// 		ollamaembed.WithModel("nomic-embed-text"),
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to create Ollama embedding function: %v", err)
// 	}

// 	// --- 6. Check Chroma Tenant ---
// 	tenantName := "kibo-tenant"
// 	tenant, err := chromaClient.GetTenant(ctx, chroma.NewTenant(tenantName))
// 	if err != nil {
// 		// Not found → create it
// 		tenant, err = chromaClient.CreateTenant(ctx, chroma.NewTenant(tenantName))
// 		if err != nil {
// 			log.Fatalf("Failed to create tenant: %v", err)
// 		}
// 	}

// 	// --- 7. Create Database under Tenant ---
// 	dbObj, err := chromaClient.GetDatabase(context.Background(), chroma.NewDatabase("kibo-db", tenant))
// 	if err != nil {
// 		dbObj, err = chromaClient.CreateDatabase(context.Background(), chroma.NewDatabase("kibo-db", tenant))
// 		if err != nil {
// 			log.Fatalf("Failed to create Chroma database: %v", err)
// 		}
// 	}

// 	// --- 8. Create or get Collection with embedding function ---
// 	col, err := chromaClient.GetOrCreateCollection(
// 		ctx,
// 		"kibo-collection",
// 		chroma.WithDatabaseCreate(dbObj),
// 		chroma.WithEmbeddingFunctionCreate(ef), // REQUIRED for text embeddings
// 	)
// 	if err != nil {
// 		log.Fatalf("Failed to create Chroma collection: %v", err)
// 	}
// 	log.Println("Chroma vector store initialized:", col.Name())

// 	// --- 9. Initialize ChromaVectorStore ---
// 	chromaStore := &chat.ChromaVectorStore{
// 		Client:     chromaClient,
// 		Collection: col,
// 	}

// 	// --- 10. Delete duplicate documents in Chroma ---
// 	deletedCount, err := chat.DeleteDuplicateDocuments(chromaStore)
// 	if err != nil {
// 		log.Printf("Failed to delete duplicates: %v", err)
// 	} else {
// 		log.Printf("🗑 Deleted %d duplicate documents", deletedCount)
// 	}

// 	// inspect chromaClient
// 	chat.InspectChromaCollection(chromaStore)

// 	// --- 11 Load Knowledge Base ---
// 	loader := chat.NewKnowledgeLoader(chromaStore, 500) // chunk size 500 chars
// 	kbFolder := "../knowledge_base/health"
// 	err = loader.LoadFolder(ctx, kbFolder)
// 	if err != nil {
// 		log.Printf("Failed to load knowledge base: %v", err)
// 	}

// 	// starting communication with user

// 	// --- 12. Initialize RAG Service ---
// 	ragService := chat.NewRAGService(repo, ollamaClient, *chromaStore)

// 	// --- 13. Create ChatAgent (NEW IMPORTANT PART) ---
// 	chatAgent := chat.NewChatAgent(ragService, ollamaClient)

// 	// --- 14. Setup API router ---
// 	router := api.NewRouter(repo, ragService, chatAgent)
// 	log.Println("Router initialized.")

// 	// --- 15. CORS middleware ---
// 	corsHandler := cors.New(cors.Options{
// 		AllowedOrigins:   []string{"http://localhost:5173"},
// 		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
// 		AllowedHeaders:   []string{"Content-Type"},
// 		AllowCredentials: true,
// 	}).Handler(router)

// 	// --- 12. Start HTTP server ---
// 	server := &http.Server{
// 		Addr:         ":8080",
// 		Handler:      corsHandler,
// 		ReadTimeout:  10 * time.Second,
// 		WriteTimeout: 10 * time.Second,
// 		IdleTimeout:  120 * time.Second,
// 	}

// 	log.Println("Starting Kibo backend server on http://localhost:8080")
// 	if err := server.ListenAndServe(); err != nil {
// 		log.Fatalf("Server failed: %v", err)
// 	}
// }

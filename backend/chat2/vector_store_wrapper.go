package chat2

import (
	"context"
	"fmt"

	logger "Kibo/backend/kibo_utils"

	chromem "github.com/philippgille/chromem-go"
)

// -----------------------------
// VectorStore (embedded)
// -----------------------------
// VectorStore wraps chromem-go, a pure-Go embedded vector database.
// It persists to disk and uses Ollama to embed documents and queries,
// so no external vector DB (and no Docker) is needed.
type VectorStore struct {
	collection *chromem.Collection
}

// NewVectorStore opens (or creates) a persistent vector store at path.
// ollamaBaseURL is the Ollama server root, e.g. "http://localhost:11434".
func NewVectorStore(path, ollamaBaseURL, embedModel string) (*VectorStore, error) {
	db, err := chromem.NewPersistentDB(path, false)
	if err != nil {
		return nil, fmt.Errorf("failed to open vector store at %s: %w", path, err)
	}

	ef := chromem.NewEmbeddingFuncOllama(embedModel, ollamaBaseURL+"/api")
	col, err := db.GetOrCreateCollection("kibo-collection", nil, ef)
	if err != nil {
		return nil, fmt.Errorf("failed to open collection: %w", err)
	}

	return &VectorStore{collection: col}, nil
}

// Contains reports whether a document with this ID is already indexed.
// Used to skip re-embedding unchanged knowledge base chunks on startup.
func (s *VectorStore) Contains(ctx context.Context, id string) bool {
	_, err := s.collection.GetByID(ctx, id)
	return err == nil
}

// Index adds a document with metadata. The embedding is computed via Ollama.
func (s *VectorStore) Index(ctx context.Context, id, content string, metadata map[string]string) error {
	err := s.collection.AddDocument(ctx, chromem.Document{
		ID:       id,
		Content:  content,
		Metadata: metadata,
	})
	if err != nil {
		return fmt.Errorf("failed to index document %s: %w", id, err)
	}

	logger.Info("[vector_store_wrapper.go/Index]:\tindexed document ID: %s", id)
	return nil
}

// Search returns the content of the k most similar documents to the query.
func (s *VectorStore) Search(ctx context.Context, queryText string, k int) ([]string, error) {
	// chromem errors if k exceeds the number of stored documents
	if n := s.collection.Count(); k > n {
		k = n
	}
	if k <= 0 {
		return nil, nil
	}

	results, err := s.collection.Query(ctx, queryText, k, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %w", err)
	}

	out := make([]string, 0, len(results))
	for _, r := range results {
		out = append(out, r.Content)
	}
	return out, nil
}

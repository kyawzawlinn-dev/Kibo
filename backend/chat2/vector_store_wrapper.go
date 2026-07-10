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

// DeleteBySource removes every chunk that came from the given source
// file — used when a library article is edited or deleted so search
// never returns stale content.
func (s *VectorStore) DeleteBySource(ctx context.Context, source string) error {
	err := s.collection.Delete(ctx, map[string]string{"source": source}, nil)
	if err != nil {
		return fmt.Errorf("deleting chunks of %s: %w", source, err)
	}
	return nil
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

// SearchResult is one retrieved knowledge base passage.
type SearchResult struct {
	Content    string
	Source     string // knowledge base file the chunk came from
	Similarity float32
}

// minSimilarity filters out passages that merely happen to be the
// nearest neighbours of an unrelated query — they would pollute the
// prompt and produce bogus citations. Tuned for nomic-embed-text,
// where on-topic passages score ~0.6+ and off-topic ~0.5 and below
// (measured with single-message queries; revisit if the KB grows).
const minSimilarity = 0.6

// Search returns the k most similar documents to the query that clear
// the relevance threshold.
func (s *VectorStore) Search(ctx context.Context, queryText string, k int) ([]SearchResult, error) {
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

	out := make([]SearchResult, 0, len(results))
	for _, r := range results {
		logger.Debug("[vector_store_wrapper.go/Search]:\tsimilarity=%.3f source=%s", r.Similarity, r.Metadata["source"])
		if r.Similarity < minSimilarity {
			continue
		}
		out = append(out, SearchResult{
			Content:    r.Content,
			Source:     r.Metadata["source"],
			Similarity: r.Similarity,
		})
	}
	return out, nil
}

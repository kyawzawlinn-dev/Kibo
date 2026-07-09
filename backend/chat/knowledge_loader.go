package chat

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// -----------------------------
// KnowledgeLoader
// -----------------------------
// Reads files from a folder, optionally splits large files into chunks,
// generates stable IDs, attaches metadata, and indexes into Chroma.
type KnowledgeLoader struct {
	VectorStore *ChromaVectorStore
	ChunkSize   int // number of characters per chunk
}

// NewKnowledgeLoader creates a new loader
func NewKnowledgeLoader(store *ChromaVectorStore, chunkSize int) *KnowledgeLoader {
	return &KnowledgeLoader{
		VectorStore: store,
		ChunkSize:   chunkSize,
	}
}

// LoadFolder scans a folder for .txt/.md files and indexes them
func (kl *KnowledgeLoader) LoadFolder(ctx context.Context, folderPath string) error {
	files, err := os.ReadDir(folderPath)
	if err != nil {
		return fmt.Errorf("failed to read folder: %w", err)
	}

	for _, f := range files {
		if f.IsDir() {
			continue
		}

		ext := strings.ToLower(filepath.Ext(f.Name()))
		if ext != ".txt" && ext != ".md" {
			continue
		}

		fullPath := filepath.Join(folderPath, f.Name())
		err := kl.loadFile(ctx, fullPath)
		if err != nil {
			log.Printf("⚠️ Failed to load file %s: %v", fullPath, err)
		}
	}

	return nil
}

// loadFile reads a file, splits into chunks if needed, generates stable IDs, and indexes
func (kl *KnowledgeLoader) loadFile(ctx context.Context, filePath string) error {
	contentBytes, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}
	content := string(contentBytes)

	chunks := kl.chunkContent(content)

	for i, chunk := range chunks {
		stableID := generateStableID(filePath, i, chunk)
		metadata := map[string]string{
			"source":     filepath.Base(filePath),
			"chunkIndex": fmt.Sprintf("%d", i),
			"created_at": time.Now().Format(time.RFC3339),
		}

		err := kl.VectorStore.Index(ctx, stableID, chunk, metadata)
		if err != nil {
			log.Printf("⚠️ Failed to index chunk %d of %s: %v", i, filePath, err)
			continue
		}
		log.Printf("✅ Indexed chunk %d of file %s (ID=%s)", i, filePath, stableID)
	}

	return nil
}

// chunkContent splits content into fixed-size chunks
func (kl *KnowledgeLoader) chunkContent(content string) []string {
	if kl.ChunkSize <= 0 || len(content) <= kl.ChunkSize {
		return []string{content}
	}

	var chunks []string
	for start := 0; start < len(content); start += kl.ChunkSize {
		end := start + kl.ChunkSize
		if end > len(content) {
			end = len(content)
		}
		chunks = append(chunks, content[start:end])
	}
	return chunks
}

// -----------------------------
// generateStableID
// -----------------------------
// Create a deterministic ID from file path + chunk index + chunk content.
// This ensures that re-importing the same file/chunk does NOT create duplicates.
func generateStableID(filePath string, chunkIndex int, chunk string) string {
	fileName := filepath.Base(filePath) // stable across machines
	hash := md5.Sum([]byte(fmt.Sprintf("%s-%d-%s", fileName, chunkIndex, chunk)))
	return hex.EncodeToString(hash[:])
}

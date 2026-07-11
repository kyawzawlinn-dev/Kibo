package chat2

import "os"

// Model names used across the chat pipeline. Configurable via
// environment variables so a weak machine can drop to a smaller model
// without recompiling:
//
//	KIBO_CHAT_MODEL=llama3.2:1b   (lighter/faster than the 3B default)
//	KIBO_EMBED_MODEL=nomic-embed-text
var (
	// ChatModel generates replies, titles, and classifications.
	ChatModel = envOr("KIBO_CHAT_MODEL", "llama3.2")
	// EmbedModel embeds knowledge base chunks and search queries.
	EmbedModel = envOr("KIBO_EMBED_MODEL", "nomic-embed-text")
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

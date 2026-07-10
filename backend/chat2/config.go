package chat2

// Model names used across the chat pipeline. Kept in one place so
// swapping to a smaller/larger model is a single-line change.
const (
	// ChatModel generates replies, titles, and classifications.
	ChatModel = "llama3.2:latest"
	// EmbedModel embeds knowledge base chunks and search queries.
	EmbedModel = "nomic-embed-text"
)

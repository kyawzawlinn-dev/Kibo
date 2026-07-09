package chat2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// OllamaClient is a simple client for the Ollama API
type OllamaClient struct {
	baseURL string
	client  *http.Client
}

// NewOllamaClient creates a new client
func NewOllamaClient(baseURL string) *OllamaClient {
	return &OllamaClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// --- Ollama API Request/Response Structs ---
type OllamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func (c *OllamaClient) Generate(ctx context.Context, model, prompt string) (string, error) {
	reqBody := OllamaGenerateRequest{
		Model:  model,
		Prompt: prompt,
		Stream: false, // We want the full response, not a stream
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(reqBody); err != nil {
		return "", fmt.Errorf("failed to encode generate request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/api/generate", &buf)
	if err != nil {
		return "", fmt.Errorf("failed to create generate request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call generate API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned non-200 status: %s", resp.Status)
	}

	var ollamaResp OllamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("failed to decode generate response: %w", err)
	}

	return ollamaResp.Response, nil
}

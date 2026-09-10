// Package ollama is a minimal client for the local Ollama HTTP API.
// It supports vision inference (image + prompt) and model listing.
package ollama

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Client talks to a local Ollama server.
type Client struct {
	BaseURL string
	Model   string
	HTTP    *http.Client
}

// New returns a Client. BaseURL defaults to the GPU1 vision instance
// (http://localhost:11436) so image inference uses the big-VRAM GPU.
// Override with OLLAMA_HOST. Model defaults to $GET_IMG_MEANING_MODEL
// or "qwen2.5vl:7b".
func New() *Client {
	base := os.Getenv("OLLAMA_HOST")
	if base == "" {
		base = "http://localhost:11436"
	}
	model := os.Getenv("GET_IMG_MEANING_MODEL")
	if model == "" {
		model = "qwen2.5vl:7b"
	}
	return &Client{
		BaseURL: base,
		Model:   model,
		HTTP:    &http.Client{Timeout: 5 * time.Minute},
	}
}

// GenerateRequest mirrors the Ollama /api/generate payload.
type GenerateRequest struct {
	Model      string   `json:"model"`
	Prompt     string   `json:"prompt"`
	Images     []string `json:"images,omitempty"` // base64-encoded
	Stream     bool     `json:"stream"`
	NumPredict int      `json:"num_predict,omitempty"`
	Options    map[string]any `json:"options,omitempty"`
}

// GenerateResponse is the subset of the Ollama response we need.
type GenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	EvalCount int   `json:"eval_count"`
}

// Generate runs a single (optionally vision) inference and returns the text.
func (c *Client) Generate(ctx context.Context, req GenerateRequest) (string, error) {
	if req.Model == "" {
		req.Model = c.Model
	}
	body, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/generate", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama returned %d: %s", resp.StatusCode, string(raw))
	}

	var out GenerateResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", fmt.Errorf("bad ollama response: %w", err)
	}
	return out.Response, nil
}

// EncodeImage reads a file and returns its base64 data URI-free payload.
func EncodeImage(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// ListModels returns the names of models available on the server.
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	names := make([]string, 0, len(out.Models))
	for _, m := range out.Models {
		names = append(names, m.Name)
	}
	return names, nil
}

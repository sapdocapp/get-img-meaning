// Package mcp exposes get-img-meaning as a Model Context Protocol server.
// Tools: describe_image, read_text (OCR), health.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sapdocapp/get-img-meaning/internal/gpu"
	"github.com/sapdocapp/get-img-meaning/internal/ollama"
)

// Server wraps the MCP server and its dependencies.
type Server struct {
	ollama *ollama.Client
}

// New builds the MCP server with all tools registered.
func New() *mcp.Server {
	s := &Server{ollama: ollama.New()}
	server := mcp.NewServer(&mcp.Implementation{Name: "get-img-meaning", Version: "0.1.0"}, nil)

	server.AddTool(&mcp.Tool{
		Name:        "describe_image",
		Description: "Describe an image using the local vision model. Returns a natural-language description. Pass the image as base64 (or a local file path).",
		InputSchema: schema(map[string]any{
			"image":     strProp("The image as a base64 string, or a local file path."),
			"prompt":    strProp("Optional custom instruction for the description."),
			"max_words": intProp("Approximate word budget for the description."),
		}, []string{"image"}),
	}, s.describeImage)

	server.AddTool(&mcp.Tool{
		Name:        "read_text",
		Description: "Extract (OCR) all text from an image using the local vision model. Pass the image as base64 (or a local file path).",
		InputSchema: schema(map[string]any{
			"image":      strProp("The image as a base64 string, or a local file path."),
			"structured": boolProp("Return JSON lines with bounding boxes instead of plain text."),
		}, []string{"image"}),
	}, s.readText)

	server.AddTool(&mcp.Tool{
		Name:        "health",
		Description: "Report GPU availability, free VRAM, and whether the vision model is loaded.",
		InputSchema: schema(map[string]any{}, nil),
	}, s.health)

	return server
}

// describeArgs is the input to describe_image.
type describeArgs struct {
	Image    string `json:"image"`              // base64 or local file path
	Prompt   string `json:"prompt,omitempty"`   // optional custom instruction
	MaxWords int    `json:"max_words,omitempty"` // approximate word budget for the description
}

func (s *Server) describeImage(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a describeArgs
	if err := json.Unmarshal(req.Params.Arguments, &a); err != nil {
		return errResult("invalid arguments: %v", err), nil
	}
	img, err := resolveImage(a.Image)
	if err != nil {
		return errResult("%v", err), nil
	}
	prompt := a.Prompt
	if prompt == "" {
		prompt = "Describe this image in detail. Include objects, people, text, colors, and context."
	}
	numPredict := 0
	if a.MaxWords > 0 {
		numPredict = a.MaxWords * 13 / 10 // ~1.3 tokens per word
	}
	text, err := s.ollama.Generate(context.Background(), ollama.GenerateRequest{
		Prompt:     prompt,
		Images:     []string{img},
		Stream:     false,
		NumPredict: numPredict,
	})
	if err != nil {
		return errResult("inference failed: %v", err), nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
}

// readTextArgs is the input to read_text.
type readTextArgs struct {
	Image      string `json:"image"`               // base64 or local file path
	Structured bool   `json:"structured,omitempty"` // return JSON lines instead of plain text
}

func (s *Server) readText(_ context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	var a readTextArgs
	if err := json.Unmarshal(req.Params.Arguments, &a); err != nil {
		return errResult("invalid arguments: %v", err), nil
	}
	img, err := resolveImage(a.Image)
	if err != nil {
		return errResult("%v", err), nil
	}
	prompt := "Extract all text from this image. Return only the text exactly as it appears."
	if a.Structured {
		prompt = "Extract all text from this image. Return each distinct text element as a JSON object with fields {text, x, y, width, height}. Return a JSON array."
	}
	text, err := s.ollama.Generate(context.Background(), ollama.GenerateRequest{
		Prompt: prompt,
		Images: []string{img},
		Stream: false,
	})
	if err != nil {
		return errResult("inference failed: %v", err), nil
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, nil
}

func (s *Server) health(_ context.Context, _ *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	gpus, _ := gpu.Detect()
	best := gpu.BestForVision(gpus)
	models, _ := s.ollama.ListModels(context.Background())

	var b strings.Builder
	fmt.Fprintf(&b, "model: %s\n", s.ollama.Model)
	if best != nil {
		fmt.Fprintf(&b, "gpu: %s (index %d) free %d MiB / %d MiB\n", best.Name, best.Index, best.FreeMiB, best.TotalMiB)
	} else if len(gpus) == 0 {
		b.WriteString("gpu: none detected (falling back to CPU)\n")
	} else {
		b.WriteString("gpu: multiple detected\n")
	}
	fmt.Fprintf(&b, "models: %s\n", strings.Join(models, ", "))
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: b.String()}}}, nil
}

// resolveImage accepts either a base64 string or a local file path.
func resolveImage(v string) (string, error) {
	if v == "" {
		return "", fmt.Errorf("image is required (base64 or file path)")
	}
	// If it's a path to an existing file, read and encode it.
	if _, err := os.Stat(v); err == nil {
		return ollama.EncodeImage(v)
	}
	// Otherwise assume it's already base64.
	return v, nil
}

func errResult(format string, args ...any) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf(format, args...)}},
		IsError: true,
	}
}

// schema builds a JSON Schema object for a tool's input.
func schema(props map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": props,
		"required":   required,
	}
}

func strProp(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func intProp(desc string) map[string]any {
	return map[string]any{"type": "integer", "description": desc}
}

func boolProp(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

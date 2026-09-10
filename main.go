// get-img-meaning is a minimal, agent-first image-intelligence MCP + CLI.
// It runs local vision inference (Ollama) so any AI agent can understand images.
//
// Modes:
//
//	get-img-meaning serve                 # MCP over stdio (for Claude Code, Cursor, etc.)
//	get-img-meaning serve --http :8080    # MCP over streamable HTTP
//	get-img-meaning describe <image>      # CLI: describe an image
//	get-img-meaning ocr <image>           # CLI: extract text from an image
//	get-img-meaning status                # CLI: GPU + model status
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/sapdocapp/get-img-meaning/internal/gpu"
	"github.com/sapdocapp/get-img-meaning/internal/ollama"
	mcpserver "github.com/sapdocapp/get-img-meaning/internal/mcp"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "serve":
		cmdServe(os.Args[2:])
	case "describe":
		cmdDescribe(os.Args[2:])
	case "ocr":
		cmdOCR(os.Args[2:])
	case "status":
		cmdStatus()
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `get-img-meaning — local image intelligence for AI agents

Usage:
  get-img-meaning serve [--http :8080]   Run as an MCP server (stdio by default)
  get-img-meaning describe <image> [--max-words N] [--prompt "..."]
  get-img-meaning ocr <image> [--structured]
  get-img-meaning status

Environment:
  OLLAMA_HOST             Ollama base URL (default http://localhost:11434)
  GET_IMG_MEANING_MODEL   Vision model (default qwen2.5vl:7b)
`)
}

func cmdServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	httpAddr := fs.String("http", "", "if set, serve MCP over streamable HTTP at this address")
	fs.Parse(args)

	server := mcpserver.New()
	if *httpAddr != "" {
		handler := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil)
		log.Printf("MCP streamable HTTP listening on %s", *httpAddr)
		if err := http.ListenAndServe(*httpAddr, handler); err != nil {
			log.Fatalf("server failed: %v", err)
		}
		return
	}
	// Use the plain stdio transport. The LoggingTransport echoes every
	// request to stderr, which floods the pipe with base64 for large images
	// and can block the server.
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}

func cmdDescribe(args []string) {
	fs := flag.NewFlagSet("describe", flag.ExitOnError)
	maxWords := fs.Int("max-words", 0, "approximate word budget for the description")
	prompt := fs.String("prompt", "", "custom instruction")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: get-img-meaning describe <image> [--max-words N] [--prompt ...]")
		os.Exit(2)
	}
	img, err := ollama.EncodeImage(fs.Arg(0))
	if err != nil {
		log.Fatalf("read image: %v", err)
	}
	p := *prompt
	if p == "" {
		p = "Describe this image in detail. Include objects, people, text, colors, and context."
	}
	numPredict := 0
	if *maxWords > 0 {
		numPredict = *maxWords * 13 / 10
	}
	text, err := ollama.New().Generate(context.Background(), ollama.GenerateRequest{
		Prompt: p, Images: []string{img}, Stream: false, NumPredict: numPredict,
	})
	if err != nil {
		log.Fatalf("inference failed: %v", err)
	}
	fmt.Println(text)
}

func cmdOCR(args []string) {
	fs := flag.NewFlagSet("ocr", flag.ExitOnError)
	structured := fs.Bool("structured", false, "return JSON lines with bounding boxes")
	fs.Parse(args)
	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: get-img-meaning ocr <image> [--structured]")
		os.Exit(2)
	}
	img, err := ollama.EncodeImage(fs.Arg(0))
	if err != nil {
		log.Fatalf("read image: %v", err)
	}
	p := "Extract all text from this image. Return only the text exactly as it appears."
	if *structured {
		p = "Extract all text from this image. Return each distinct text element as a JSON object with fields {text, x, y, width, height}. Return a JSON array."
	}
	text, err := ollama.New().Generate(context.Background(), ollama.GenerateRequest{
		Prompt: p, Images: []string{img}, Stream: false,
	})
	if err != nil {
		log.Fatalf("inference failed: %v", err)
	}
	fmt.Println(text)
}

func cmdStatus() {
	gpus, _ := gpu.Detect()
	best := gpu.BestForVision(gpus)
	client := ollama.New()
	models, _ := client.ListModels(context.Background())

	fmt.Printf("model: %s\n", client.Model)
	if best != nil {
		fmt.Printf("gpu: %s (index %d) free %d MiB / %d MiB\n", best.Name, best.Index, best.FreeMiB, best.TotalMiB)
	} else if len(gpus) == 0 {
		fmt.Println("gpu: none detected (falling back to CPU)")
	} else {
		fmt.Println("gpu: multiple detected")
	}
	fmt.Printf("models: %v\n", models)
}

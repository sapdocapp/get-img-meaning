# What We Built — get-img-meaning v0.1 (minimal, local, agent-first)

**Date:** 2026-09-10
**Status:** WORKING — local MCP + CLI for image inference

## The decision

Per the founder's direction, v1 is deliberately **minimal and local-only**: a single Go binary
that lets any AI agent understand images using the local GPU. No Cloudflare, no find_similar,
no generation, no UI. Everything else is postponed (see `POSTPONED.md`).

## What exists now

### 1. Go binary (`get-img-meaning`)
- `main.go` — entry point; modes: `serve`, `describe`, `ocr`, `status`
- `internal/ollama/ollama.go` — minimal Ollama REST client (`/api/generate`, `/api/tags`)
- `internal/gpu/gpu.go` — NVIDIA GPU detection via `nvidia-smi` (no cgo, portable)
- `internal/mcp/server.go` — MCP server with 3 tools

### 2. MCP server (stdio + streamable HTTP)
Three tools, registered with the official `modelcontextprotocol/go-sdk`:
- **`describe_image(image, prompt?, max_words?)`** — natural-language description.
  `max_words` maps to Ollama `num_predict` (~1.3 tokens/word).
- **`read_text(image, structured?)`** — OCR / text extraction.
- **`health()`** — GPU + model status.

Image input accepts **base64** or a **local file path** (path is convenient for local stdio).

### 3. CLI (no TUI)
- `get-img-meaning describe <image> [--max-words N] [--prompt ...]`
- `get-img-meaning ocr <image> [--structured]`
- `get-img-meaning status`
- `get-img-meaning serve [--http :8080]`

## Verified working

- ✅ Builds with `go build` (Go 1.27, go-sdk v1.7.0)
- ✅ `status` detects GPU1 (RTX 3060, 12GB free) and lists models
- ✅ `describe` reads a real screenshot and returns an accurate description
- ✅ `ocr` extracts text from a real screenshot
- ✅ MCP handshake (initialize + tools/list) returns all 3 tools with schemas
- ✅ MCP `describe_image` works with both base64 and file-path input

## Key engineering decisions

1. **GPU1 vision instance by default.** `OLLAMA_HOST` defaults to `http://localhost:11436`
   (the GPU1-pinned Ollama), so image inference uses the big-VRAM GPU. The main instance
   (11434, GPU0) is for text. This matches the founder's "text on GPU0, image on GPU1" split.
2. **No cgo.** GPU detection parses `nvidia-smi` instead of NVML, so the binary stays
   portable and cross-compilable.
3. **Plain stdio transport.** The go-sdk's `LoggingTransport` echoes every request to stderr,
   which floods the pipe with base64 for large images and can block the server. We use the
   plain `StdioTransport`.
4. **Official go-sdk** (`modelcontextprotocol/go-sdk`) for the MCP server, with low-level
   `ToolHandler` + manual JSON schemas (avoids the typed-schema complexity).

## How to use it

```bash
# Build
go build -o get-img-meaning .

# Register for Claude Code
claude mcp add get-img-meaning -- /path/to/get-img-meaning serve

# CLI
./get-img-meaning describe photo.png --max-words 40
./get-img-meaning ocr screenshot.png
./get-img-meaning status
```

## Files

```
get-img-meaning/
├── main.go
├── go.mod / go.sum
├── README.md
├── doc/
│   ├── WHAT-WE-BUILT.md   (this file)
│   └── POSTPONED.md
└── internal/
    ├── ollama/ollama.go
    ├── gpu/gpu.go
    └── mcp/server.go
```

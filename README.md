# get-img-meaning

A minimal, **agent-first** image-intelligence MCP + CLI. It runs local vision inference
(Ollama) on your big-VRAM GPU so any AI agent — Claude Code, Cursor, OpenCode, custom agents —
can understand images without a cloud vision API.

> **Design goal:** the smallest thing that lets any agent read an image. No UI, no TUI,
> no Cloudflare (yet). One Go binary, two surfaces: **MCP** and **CLI**.

## Why

Most AI agents can't see images. This exposes your local GPU vision model as an MCP server,
so an agent can call `describe_image` / `read_text` and get a real understanding of a
screenshot, photo, or diagram — with your own hardware, your own data staying local.

## Features

- **MCP server** (stdio, or streamable HTTP with `--http`) with three tools:
  - `describe_image(image, prompt?, max_words?)` — natural-language description
  - `read_text(image, structured?)` — OCR / text extraction
  - `health()` — GPU + model status
- **CLI** (no TUI): `describe`, `ocr`, `status`, `serve`
- **GPU-aware**: detects NVIDIA GPUs via `nvidia-smi` (no cgo), reports free VRAM
- **Local-first**: talks to your own Ollama; images never leave your machine

## Install

```bash
# Build
go build -o get-img-meaning .

# Register as an MCP server for Claude Code
claude mcp add get-img-meaning -- /path/to/get-img-meaning serve
```

## Usage

```bash
# MCP server (stdio — for Claude Code, Cursor, etc.)
get-img-meaning serve

# MCP server over streamable HTTP
get-img-meaning serve --http :8080

# CLI
get-img-meaning describe photo.png --max-words 40
get-img-meaning ocr screenshot.png --structured
get-img-meaning status
```

## Configuration

| Env var | Default | Purpose |
|---------|---------|---------|
| `OLLAMA_HOST` | `http://localhost:11436` | Ollama base URL. Defaults to the **GPU1 vision instance** so image inference uses the big-VRAM GPU. |
| `GET_IMG_MEANING_MODEL` | `qwen2.5vl:7b` | Vision model to use. |

## Architecture

```
AI agent (Claude Code, Cursor, ...)
   │  MCP (stdio / streamable HTTP)
   ▼
get-img-meaning  ──►  Ollama (GPU1 vision instance)
   │                     │
   │  nvidia-smi          ▼
   └──► GPU detection   qwen2.5vl:7b (vision)
```

- **GPU0** (port 11434): text models
- **GPU1** (port 11436): vision — `get-img-meaning` defaults here

## Roadmap (postponed — see `POSTPONED.md`)

- `find_similar` (Pinterest-style similar-image search via Parallel Search MCP)
- Image generation (ComfyUI) + prompt-rewrite loop
- Cloudflare dual-app: expose the local GPU to the edge via Tunnel, R2/D1/Vectorize
- Affiliate-link automation for product search
- SQLite history + `search_history` tool

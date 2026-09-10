# Postponed — get-img-meaning future features

**Date:** 2026-09-10
**Status:** DEFERRED to v2+. These are the founder's answers to the council's questions,
captured as a handoff for future work. The current v0.1 is intentionally minimal and local.

---

## 1. find_similar — Pinterest-style similar-image search

The founder's vision: take a photo → OCR/describe it → search the web for similar products
or images → return links + metadata → **always persist** links + metadata to a local DB.

- **Semantics (open question):** description-based web search (v1 of this feature) vs true
  CLIP visual similarity (embeddings + vector store). The council leans description-based
  first, CLIP as an upgrade.
- **Parallel Search MCP integration:** aggregate `https://search.parallel.ai/mcp`
  (`web_search`, `web_fetch`) inside our server to guarantee persistence. Load-balance across
  multiple free search MCPs (DuckDuckGo, etc.) to dodge rate limits.
- **Caching:** every search result cached in D1 (cloud) / SQLite (local). Cache TTL 30 days
  during development, lower to 10 days as it matures, lower further as it gets popular.
- **Affiliate links:** auto-generate affiliate links via API/MCP so the founder earns
  commission when someone clicks and buys. (The founder believes this is a novel, profitable
  combination — "someone has probably made the pieces, but not this whole.")

## 2. Image generation (ComfyUI) + prompt-rewrite loop

- Generate an image with local ComfyUI → run the vision model on the result → get a better
  text description → rewrite the prompt → generate a new photo. Iterative.
- **Resolution:** up to 1500×1500 (highest quality) when ComfyUI is enabled.
- **Feature gating by app state:** if ComfyUI is running, the local MCP tells Cloudflare the
  feature is enabled, and the website shows the generation feature. If it's closed, the
  feature locks immediately and another feature enables.
- **Queue:** one request at a time, ~2s gap between requests so the GPU rests. Requests
  queue in Redis/bulky DB. If the user closes the site, queued requests are stored as cache
  but not processed.

## 3. Cloudflare dual-app — expose local GPU to the edge

Cloudflare has no GPU, so the local box is the GPU backend, exposed via **Cloudflare Tunnel**.

- **Local MCP** exposes the running AI app (Ollama / ComfyUI / Qdrant / Postgres).
- **Cloud MCP** (Worker) exposes front-end functionality. Cloud uses the founder's hardware
  via the tunnel, plus:
  - **R2** — store images
  - **D1** — store requests, JSON payloads, every agent message
  - **Vectorize** — help agents with information; cache every link + metadata of each website
    for agentic search
- **Web search from the Worker:** mix DuckDuckGo + Parallel Search MCPs; cache everything.
- **Auth:** Cloudflare Access Service Token (MCP clients can't do mTLS). mTLS via API Shield
  is a possible hardening. Only the founder's Worker calls the GPU; third parties not yet
  considered.
- **Rate limiting:** minimal — one request at a time, 2s gap. No parallel GPU usage.
- **Hosting:** the image MCP is not SAP-specific → likely `mcp.filipe.uk` (personal) in a
  separate personal Cloudflare account, vs `mcp.sapdoc.app` (corporate).

## 4. Persistence & history

- **Local:** SQLite on the computer (no parallel writes needed).
- **Cloud:** D1 in Cloudflare.
- **`search_history` tool:** deferred until login/users exist. The site is public/free for
  now; will close to registered users later.

## 5. GPU detection & models

- **Models to try:** `qwen2.5vl:7b` (current) and `minicpm-v:8b` (better OCR). Both can be
  downloaded and compared.
- **Dual-GPU logic:** the founder has two RTX 3060s. The "biggest free VRAM for image, text on
  the other" split is real but the exact approach is to be tested. NVIDIA-only (no AMD/Apple).
- **Concurrency:** NO parallel agents on the local box — a queue with ~2s gaps.

## 6. Documentation-generation projects (separate future thread)

The founder also pasted 50 documentation-generation projects (source → multi-agent → MCP →
docs). These are a **separate** future project, not part of get-img-meaning. Notable ones to
revisit: `divar-ir/ai-doc-gen`, `YannickTM/docu-mcp`, `jonverrier/AgentDoc`,
`modelcontextprotocol/go-sdk`, `upstash/context7`, `idosal/git-mcp`.

---

## Council's preliminary lean (for when v2 starts)

- **find_similar = description-based web search** for v1 of the feature; CLIP + vector store
  as a later upgrade. DB is a link log, not a similarity index.
- **Aggregate Parallel Search inside our server** to enforce persistence.
- **Auth = Access Service Token** (not mTLS) for the tunneled endpoint.
- **mcp.filipe.uk** for the personal image MCP, in a separate personal Cloudflare account.
- **GPU detection via `github.com/NVIDIA/go-nvml`** (maintained) when multi-GPU routing is real.

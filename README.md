# 🌿 Kibo — offline AI health companion

A fully offline health assistant for low-connectivity, low-power settings. It runs on a modest laptop with no internet: chat grounded in a local medical library, health tracking, instant first-aid cards, and per-person profiles — all from a single binary, with data that never leaves the device.

> ⚠️ Kibo is not a doctor. It's a companion and reference tool; always seek professional care for serious conditions.

## Screenshots

| Grounded answers with sources | Health log, daily records, and trends |
|---|---|
| ![Chat with cited answer](docs/screenshots/chat.png) | ![Health tab with log and records](docs/screenshots/health.png) |

| Instant first-aid cards (no AI) | One laptop serves every phone |
|---|---|
| ![Emergency first aid](docs/screenshots/emergency.png) | ![Share on Wi-Fi with QR code](docs/screenshots/share.png) |

| Browsable, editable health library | A profile for every family member |
|---|---|
| ![Health library](docs/screenshots/library.png) | ![Family profiles](docs/screenshots/profiles.png) |

## What it does

- **Ask health questions offline** — Kibo answers from a trusted local medical library and shows you the sources. It won't make up medical facts.
- **It knows your history** — answers use your own records and remember the conversation.
- **Emergency first aid** — cards for choking, bleeding, chest pain, snakebite, and more open instantly, even if the AI is off.
- **Track your health** — log illnesses and symptoms, record weight/sleep/activity/water, see trends, and print a one-page summary for the doctor.
- **Log by chatting** — say "yesterday I slept 5 hours" and it's saved.
- **Grow the library** — read, search, and add your own health articles.
- **Share over Wi-Fi** — any phone on the same network gets Kibo by scanning a QR code; no internet needed.
- **A profile per family member**, each with their own chats and records.
- **Your data stays yours** — one local file; export and import as CSV.

## Architecture

One Go binary serves the API and the UI; the only runtime dependency is [Ollama](https://ollama.com).

```
Browser ── React SPA (embedded via go:embed)
   │  same-origin /api
Go server (net/http + gorilla/mux)
   ├── SQLite ........ chats, records, profiles, health log
   ├── chromem-go .... in-process vector store (no external DB)
   └── Ollama ........ llama3.2 (chat) + nomic-embed-text (embeddings)
```

A chat message runs: emergency keyword match → first-aid card (deterministic, no LLM) · else a single classifier call · health questions go through RAG — retrieve the user's records and knowledge-base passages (vector search with a relevance threshold), generate a grounded answer, and append citations from the passages actually used. Conversation memory is read from SQLite, so it survives restarts.

Notable choices: the frontend is embedded with `go:embed` and the vector store runs in-process, so `go build` yields one ~14 MB binary with no Docker. First-aid cards and record confirmations never pass through the LLM — it can't reword first-aid steps or claim to have saved data it didn't. The SPA calls `/api` same-origin, so Wi-Fi sharing needs no per-device setup.

**Stack:** Go · SQLite · chromem-go · Ollama · React + TypeScript + Vite + Tailwind

## Quick start

Install [Go](https://go.dev/dl/), [Node.js](https://nodejs.org) (build only), and [Ollama](https://ollama.com). The script checks them, pulls the AI models if missing, builds, and runs:

```bash
./kibo.sh          # build and run → http://localhost:8080
./kibo.sh check    # verify all requirements are installed
./kibo.sh dev      # hot-reload dev mode → http://localhost:5173
./kibo.sh stop     # stop anything left running
```

The first run needs internet once (models, npm packages). After that Kibo is fully offline — all data stays in `data/`, and Node is never needed at runtime.

## Roadmap

- [ ] AI-suggested health-log entries from chat (one-tap confirm, never auto-saved)
- [ ] Import from Apple Health (`export.xml`) and Google Takeout
- [ ] Knowledge-base update packs distributable by USB stick
- [ ] Local-language support

## License

All rights reserved. See [LICENSE](LICENSE).

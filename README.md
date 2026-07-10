# 🌿 Kibo — Your Health Companion That Never Needs the Internet

> **One laptop. Zero internet. A personal health assistant for the places the cloud forgot.**

Millions of people live where the power cuts daily and the internet is a luxury — but their health questions don't wait for a connection. Kibo is a **fully offline AI health companion**: it tracks your health, answers your questions from a trusted medical library, and flags emergencies — all running locally on modest hardware, with your data never leaving your device.

> ⚠️ **Kibo is not a doctor.** It is a companion and reference tool. Always seek professional medical care for serious conditions.

## Why Kibo?

- **🔌 Works when nothing else does.** No cloud, no subscription, no signal required. After a one-time setup, Kibo runs entirely on your machine — through blackouts, outages, and dead zones.
- **🧠 An AI that actually knows *you*.** Kibo answers health questions using **your own tracked records** (weight, sleep, activity, diet), not generic advice.
- **📚 Answers you can trust.** Kibo doesn't let the AI freestyle medical facts. Every answer is grounded in a curated offline health library via RAG (retrieval-augmented generation).
- **🪶 Light enough for a 10-year-old laptop.** A small local model, no heavyweight infrastructure, built to sip battery.
- **🔒 Radically private.** Your health data lives in one SQLite file on your disk. Nothing is uploaded, ever — there's nowhere to upload it to.

## How it works

```
You: "I've had a headache for 3 days"
        │
        ▼
 ┌─── Kibo (all local) ────────────────────┐
 │ 1. Checks YOUR records (sleep ↓, BP ↑)  │
 │ 2. Searches the offline medical library │
 │ 3. Local AI writes a grounded answer    │
 │ 4. Red-flag symptoms → safety guidance  │
 └─────────────────────────────────────────┘
        │
        ▼
An answer based on your body, your history, and real sources.
```

**Stack:** Go backend · React + TypeScript UI · SQLite · vector search · [Ollama](https://ollama.com) (local LLM: `llama3.2` + `nomic-embed-text`)

### Kibo vs. plain Ollama

Ollama is an engine; Kibo is the vehicle. Raw Ollama has no memory of your health history, no grounding (small models hallucinate medical facts confidently), no safety rails, and no interface a non-technical family member can use. Kibo supplies the health records, the trusted medical library, the emergency protocols, and the product around the model.

## Current status

Kibo is under active development. What works today:

- ✅ Multi-chat conversations with a local LLM (auto-titled, stored in SQLite)
- ✅ RAG pipeline: markdown knowledge base → chunked → embedded → vector search → grounded answers
- ✅ Intent detection and answer routing
- ✅ Body record tracking with charts (weight, sleep, activity, water)

## Roadmap

The plan, in order:

**Phase 1 — Lightweight foundation (in progress on this branch)**
- [x] Replace Chroma + Docker with an embedded pure-Go vector store (no Docker at all)
- [x] Wire body/diet record API routes to the frontend
- [x] Per-chat conversation memory, read from the database so it survives restarts
- [x] Merge classifier calls to cut per-message LLM round-trips (faster replies on weak hardware)
- [x] Embed the built frontend into the Go binary (`go:embed`) → **one executable, no Node required**
- [x] Remove legacy `chat/` package and dead code

**Phase 2 — The health companion**
- [x] Log health data through chat ("I weighed 68kg today" → saved record)
- [ ] Emergency mode: instant, deterministic first-aid cards — no LLM in the loop
- [ ] Answers cite their knowledge-base source
- [ ] Expanded curated offline health library (first aid, common conditions, medications)

**Phase 3 — Built for the field**
- [ ] LAN sharing: one laptop serves the whole household or clinic over local Wi-Fi
- [ ] Printable health summary to bring to a doctor
- [ ] CSV export/import of health records (backup, device migration, spreadsheets)
- [ ] Import from Apple Health (export.xml) and Google Takeout
- [ ] Family profiles on one device
- [ ] Knowledge-base update packs distributable by USB stick (offline updates)
- [ ] Local-language support

## Running it

Requirements: [Ollama](https://ollama.com) with `llama3.2` and `nomic-embed-text` pulled. No Docker, no Node at runtime.

**Single binary (the product):**

```bash
cd frontend && npm run build   # bundle the UI into the Go module (needs Node once)
cd ../backend && go build -o kibo . && ./kibo
```

Then open http://localhost:8080 — the binary serves the app and the API, and stores all data in `data/`.

**Development (hot reload):**

```bash
# terminal 1 — backend API
cd backend && go run main.go

# terminal 2 — frontend with hot reload (proxies /api to the backend)
cd frontend && npm run dev
```

Then open http://localhost:5173.

## License

All rights reserved. See [LICENSE](LICENSE).

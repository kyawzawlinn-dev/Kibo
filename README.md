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

- ✅ **Single-binary app**: one Go executable serves the UI and API — no Docker, no Node at runtime
- ✅ Multi-chat conversations with a local LLM (auto-titled, per-chat memory that survives restarts)
- ✅ RAG pipeline with **source citations** and a relevance threshold — answers grounded in a 15-article offline health library
- ✅ **Library page**: read every article Kibo cites, search them, and add your own from the UI — new articles are indexed live, no restart
- ✅ **Log health data by chatting**: "yesterday I slept 5 hours and drank 2L" → saved records
- ✅ **Emergency mode**: 8 first-aid cards embedded in the binary; red-flag chat messages get the card in ~13 ms, no LLM involved
- ✅ Body record tracking: date-aware entries, combined trends chart, paginated history
- ✅ **Share on Wi-Fi**: one laptop serves every phone on the local network — scan a QR code, no internet involved

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
- [x] Emergency mode: instant, deterministic first-aid cards — no LLM in the loop
- [x] Answers cite their knowledge-base source
- [x] Expanded curated offline health library (diarrhea/ORS, mosquito-borne fevers, wound care, respiratory infections, pregnancy, child nutrition, water safety, mental wellbeing)

**Phase 3 — Built for the field**
- [x] LAN sharing: one laptop serves the whole household or clinic over local Wi-Fi
- [ ] Printable health summary to bring to a doctor
- [ ] CSV export/import of health records (backup, device migration, spreadsheets)
- [ ] Import from Apple Health (export.xml) and Google Takeout
- [ ] Family profiles on one device
- [ ] Knowledge-base update packs distributable by USB stick (offline updates)
- [ ] Local-language support

## Running it

One command does everything — checks [Ollama](https://ollama.com), pulls the models if missing, builds, and runs:

```bash
./kibo.sh          # build and run the app → http://localhost:8080
./kibo.sh dev      # development mode with hot reload → http://localhost:5173
./kibo.sh build    # just build the single binary (backend/kibo)
./kibo.sh stop     # stop anything kibo left running
```

The binary serves the app and the API together and stores all data in `data/`. No Docker; Node is only needed to build the UI, never at runtime.

## License

All rights reserved. See [LICENSE](LICENSE).

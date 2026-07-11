# 🌿 Kibo — Your Health Companion That Never Needs the Internet

> **One laptop. Zero internet. A personal health assistant for the places the cloud forgot.**

Millions of people live where the power cuts daily and the internet is a luxury — but their health questions don't wait for a connection. Kibo is a **fully offline AI health companion**: it tracks your family's health, answers questions from a trusted medical library, and puts first-aid steps one tap away — all running locally on modest hardware, with your data never leaving your device.

> ⚠️ **Kibo is not a doctor.** It is a companion and reference tool. Always seek professional medical care for serious conditions.

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

## Features

**Works when nothing else does.** After a one-time setup, Kibo runs entirely on your machine — through blackouts, outages, and dead zones. One ~14 MB Go binary serves the whole app; the only other piece is [Ollama](https://ollama.com) running the local AI models.

**An AI that actually knows you.** Tell Kibo *"yesterday I slept 5 hours and drank 2L of water"* — it saves the records and confirms exactly what it stored. Ask *"why do I keep getting headaches?"* and it answers using **your own health history**, not generic advice. Every family member gets their own profile with their own chats and records.

**Answers you can trust, with receipts.** Kibo doesn't let the AI freestyle medical facts. Answers are grounded in a curated offline health library (diarrhea & ORS, malaria & dengue, wound care, pregnancy danger signs, child nutrition, safe water, and more) and **cite their sources** — which you can open, read, edit, and extend right in the app. Off-topic questions get no fake citations: passages below a relevance threshold never reach the model.

**Emergency mode: instant, no AI required.** Choking, severe bleeding, chest pain, snakebite, child fever — curated first-aid cards are embedded in the binary and open in two taps. Red-flag chat messages ("someone is choking") get the matching card in ~13 milliseconds, before any AI is involved. When seconds matter, nothing generates — it just shows.

**One laptop serves the whole household or clinic.** Open *Share on Wi-Fi*, scan the QR code with any phone on the same network, and that phone has Kibo — no internet involved, a local hotspot is enough. Combined with profiles, one charged laptop becomes a family health station.

**One place for your health.** The *Health* tab holds a **health log** (the illnesses, symptoms, and visits a clinician actually asks about), a daily record sheet for vitals, trend charts, and a printable **doctor summary** — the one-page overview to bring to a short, rare appointment.

**Your data is a file, not a hostage.** Everything lives in a local SQLite database. Export your records as a plain CSV (USB-stick friendly), import them anywhere — re-importing a backup is deduplicated and always safe.

## How it works

```
You: "I've had a headache for 3 days"
        │
        ▼
 ┌─── Kibo (all local) ────────────────────┐
 │ 0. Red-flag check → first-aid card      │
 │ 1. Checks YOUR records (sleep ↓, BP ↑)  │
 │ 2. Searches the offline medical library │
 │ 3. Local AI writes a grounded answer    │
 │ 4. Sources appended from what was       │
 │    actually retrieved — never invented  │
 └─────────────────────────────────────────┘
```

**Stack:** a single Go binary (API + embedded React UI + embedded vector search via chromem-go) · SQLite · [Ollama](https://ollama.com) running `llama3.2` and `nomic-embed-text`.

### Kibo vs. plain Ollama

Ollama is an engine; Kibo is the vehicle. Raw Ollama has no memory of your health history, no grounding (small models hallucinate medical facts confidently), no emergency path that works without the model, and no interface a non-technical family member can use. Kibo supplies the records, the trusted library, the safety rails, and the product around the model.

## Quick start

Requirements: Go, Node.js (build only), and [Ollama](https://ollama.com). One command does everything — checks Ollama, pulls the models if missing, builds, and runs:

```bash
./kibo.sh          # build and run the app → http://localhost:8080
./kibo.sh dev      # development mode with hot reload → http://localhost:5173
./kibo.sh build    # just build the single binary (backend/kibo)
./kibo.sh stop     # stop anything kibo left running
```

The first run needs internet once (models, npm packages). After that, Kibo is fully offline: all data stays in `data/`, and Node is never needed at runtime.

## Roadmap

- [ ] Import from Apple Health (`export.xml`) and Google Takeout
- [ ] Knowledge-base update packs distributable by USB stick (offline updates)
- [ ] Local-language support
- [ ] Battery-saver behaviours for laptops running on inverters and power banks

## License

All rights reserved. See [LICENSE](LICENSE).

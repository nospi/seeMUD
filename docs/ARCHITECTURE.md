# SeeMUD Architecture

## Overview
SeeMUD connects to classic MUD games and provides a modern visual interface.
Because MUDs output unstructured text, we use AI-powered adapters to classify
and structure the data.

## Core Philosophy

**The Problem:** Classic MUDs are masterpieces from a golden era of gaming, but they output unstructured narrative text. We can't change the servers, and we don't want to - these games are perfect as they are.

**The Solution:** Build an intelligent proxy layer that sits between any MUD and SeeMUD. This proxy uses AI to classify and structure the unstructured text into a clean protocol that SeeMUD understands.

## Components

### 1. MUD Adapter (Per-MUD Parser + AI Classifier)
- Receives raw telnet output
- Maintains stateful context (running history)
- Uses AI to classify ambiguous text
- Outputs structured SeeMUD Protocol messages
- **One adapter per MUD** (WolfMUD, Astaria, etc.)

### 2. Proxy Layer (Message Router)
- Routes protocol messages to client
- Handles reconnection logic
- Manages adapter lifecycle
- Provides adapter selection/switching

### 3. SeeMUD Client (UI)
- Consumes SeeMUD Protocol (MUD-agnostic)
- Renders map, images, text
- Provides user commands and feedback
- **Knows nothing about specific MUDs**

### 4. Mapper (Graph Builder)
- Receives structured room data
- Builds spatial graph
- Handles multi-instance rooms (mazes)
- Uses stable room IDs from adapters

### 5. Image Generator (Visual Renderer)
- Uses room metadata (zone, atmosphere) for context
- Generates images with Stable Diffusion
- Leverages neighbor context from mapper
- Creates visual representation of player's imagination

## Data Flow

```
MUD Server (e.g., Astaria)
    ↓ (raw telnet - unstructured text)
Telnet Client
    ↓ (raw text lines)
MUD Adapter (Astaria-specific)
    ├─ Pattern Matching (fast path - known patterns)
    ├─ History Buffer (last 50 lines for context)
    ├─ AI Classifier (Claude/GPT - ambiguous cases)
    └─ Knowledge Base (learned patterns, improving over time)
    ↓ (SeeMUD Protocol - clean JSON messages)
Proxy Layer
    ↓ (WebSocket or local IPC)
SeeMUD Client
    ├─ Display (formatted text output)
    ├─ Mapper (builds room graph)
    └─ Image Generator (creates visuals)
```

## Why This Architecture?

### Preserves the Original Games
- No server modifications needed
- Original gameplay intact
- Respects the golden era of MUDs

### Enables Modern UX
- Structured data = reliable mapping
- Zone metadata = better image generation
- Clean separation = easier feature development

### Builds Institutional Knowledge
- Adapters learn patterns over time
- Knowledge base improves with use
- Community can share adapter improvements

### Extensible
- One adapter per MUD (isolated changes)
- Easy to add new MUDs
- Reusable AI classification logic

### Emotional Payoff
- See childhood games as you imagined them
- Visual representation of familiar places
- Modern interface for nostalgic gameplay

## Technical Stack

- **Language:** Go (adapters, proxy) + JavaScript/React (client UI)
- **Framework:** Wails (Go + Web UI in desktop app)
- **AI:** Claude API (primary) + Ollama (fallback for local)
- **Image Generation:** Stable Diffusion (local via API)
- **Protocol:** JSON messages over WebSocket/IPC

## Directory Structure

```
seeMUD/
├── internal/
│   ├── adapters/
│   │   ├── adapter.go           # Common interface
│   │   ├── wolfmud/
│   │   │   ├── wolfmud.go       # WolfMUD adapter
│   │   │   ├── patterns.go      # Pattern matching
│   │   │   └── knowledge.go     # Knowledge base
│   │   └── astaria/
│   │       ├── astaria.go       # Astaria adapter
│   │       ├── patterns.go
│   │       └── knowledge.go
│   ├── proxy/
│   │   ├── proxy.go             # Proxy server
│   │   └── router.go            # Message routing
│   ├── protocol/
│   │   ├── types.go             # Protocol message types
│   │   └── messages.go          # Message constructors
│   ├── ai/
│   │   ├── classifier.go        # AI classification interface
│   │   ├── claude.go            # Claude API client
│   │   └── ollama.go            # Ollama local client
│   ├── mapper/
│   │   ├── mapper.go            # Mapper (existing)
│   │   └── graph.go             # Room graph (existing)
│   ├── renderer/
│   │   └── stable_diffusion.go  # SD client (existing)
│   └── telnet/
│       └── client.go            # Telnet client (existing)
├── docs/
│   ├── ARCHITECTURE.md          # This file
│   ├── ADAPTER_DESIGN.md        # Adapter implementation guide
│   ├── PROTOCOL.md              # Protocol specification
│   ├── AI_INTEGRATION.md        # AI strategy and prompts
│   └── ROADMAP.md               # Implementation phases
├── frontend/                    # Wails frontend (existing)
└── main.go                      # Entry point (existing)
```

## Current State vs. Target State

### Current (Mixed Architecture)
```
MUD → Telnet → App → WolfMUD Parser → {Mapper, Display}
                              ↓
                    Half-baked parsing, MUD-specific logic in client
```

**Problems:**
- Parser tightly coupled to client
- MUD-specific logic scattered
- Unreliable room detection
- Hard to add new MUDs

### Target (Clean Separation)
```
MUD → Telnet → Adapter → Protocol → Client → {Mapper, Display}
                    ↓
              AI Classification
              Knowledge Base
```

**Benefits:**
- MUD-agnostic client
- Reliable room detection
- Easy to add new MUDs
- AI-powered intelligence

## Next Steps

See [ROADMAP.md](./ROADMAP.md) for implementation phases.

## Design Principles

1. **Separation of Concerns:** Client doesn't know about MUD specifics
2. **Intelligence at the Edge:** Adapters do the hard work, client is simple
3. **Learn Over Time:** Knowledge bases improve with use
4. **Graceful Degradation:** Works without AI, just less intelligent
5. **User Feedback:** Allow manual corrections to improve accuracy
6. **Performance:** Cache aggressively, minimize AI calls
7. **Respect the Source:** Don't modify MUD servers, adapt to them

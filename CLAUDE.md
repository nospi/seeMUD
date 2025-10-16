# SeeMUD Project Guidelines for AI Assistants

## Project Overview

SeeMUD is a visual MUD client built with Go and Wails (React frontend) that transforms classic text-based MUD games into immersive visual experiences using AI-powered image generation and intelligent text parsing.

**Core Philosophy:** Preserve the original MUD experience while enhancing it with modern visualisation—no server modifications required.

## Architecture Quick Reference

### Technology Stack
- **Backend:** Go 1.23 with Wails v2 framework
- **Frontend:** React with Vite
- **Image Generation:** Stable Diffusion (local API at http://127.0.0.1:7860)
- **AI Classification:** Planned (Claude API / Ollama local models)
- **Build System:** Wails CLI

### Key Components

```
internal/
├── telnet/          # MUD server connection via telnet
├── parser/          # Text parsing (currently WolfMUD-specific)
├── mapper/          # Spatial graph building with duplicate detection
└── renderer/        # Stable Diffusion image generation

frontend/src/
├── App.jsx          # Main application with three-column layout
├── Terminal.jsx     # MUD output display
└── Map.jsx          # Mini-map visualisation (2-level neighbours)
```

### Current State vs Target State

**Current:** Mixed architecture with MUD-specific parsing logic in client
**Target:** Clean separation with MUD-agnostic protocol and pluggable adapters

See `docs/ARCHITECTURE.md` for detailed migration plan.

## Development Guidelines

### Code Style
- **Go:** Follow standard Go conventions, use gofmt
- **React:** Functional components with hooks, clear prop types
- **Naming:** Australian English spelling (favour, colour, visualise)
- **Comments:** Explain *why*, not *what* (the code shows what)

### Key Patterns

#### 1. Parser Output (internal/parser/wolfmud.go)
The parser classifies MUD output into structured types:
```go
type OutputType int
const (
    TypeUnknown OutputType = iota
    TypeRoomTitle
    TypeRoomDescription
    TypeExits
    TypeInventory
    TypeMobs
)
```

#### 2. Mapper State (internal/mapper/mapper.go)
- Maintains spatial graph of rooms
- Handles duplicate room names with visit counts
- Provides neighbourhood context (2 levels) for image generation
- Persists maps per server

#### 3. Image Generation (app.go)
- Checks cache first (sanitised room name → PNG file)
- Generates with neighbour context for consistency
- Supports custom prompt additions for regeneration
- Caches in `cache/room_images/`

#### 4. Frontend-Backend Communication
Uses Wails bindings - Go methods are directly callable from React:
```javascript
// In React
import { ConnectToMUD, SendCommand, GetOutput } from '../wailsjs/go/main/App'
```

### Important Considerations

#### When Working on Parser
- **Problem:** Current parser is WolfMUD-specific but needs to be MUD-agnostic
- **Target:** Move MUD-specific logic to adapters (see `docs/ADAPTER_DESIGN.md`)
- **Pattern Matching:** Be cautious—MUD output varies significantly between servers
- **Context Matters:** Parser maintains state across lines (e.g., room title → description → exits)

#### When Working on Mapper
- **Duplicate Rooms:** Many MUDs have multiple instances of "A Dark Corridor"
- **Solution:** Use stable IDs (hash of name + description + exits)
- **Spatial Logic:** Only follow MUD-confirmed exits, not inferred reverse paths
- **Persistence:** Maps saved per server in `maps/` directory

#### When Working on Image Generation
- **Neighbour Context:** Include neighbouring room data for consistency
- **Prompt Engineering:** See `internal/renderer/prompts.go` for prompt construction
- **Performance:** Image generation takes 5-20s, cache aggressively
- **Custom Prompts:** Users can add style modifiers during regeneration

#### When Working on Frontend
- **Three-Column Layout:** Independent resize controls (see `frontend/src/App.jsx:27`)
- **Polling Pattern:** Frontend polls `GetOutput()` every 100ms for new MUD text
- **State Management:** React state, no Redux (keep it simple)
- **Map Rendering:** SVG-based mini-map in Map.jsx

### Common Tasks

#### Adding a New Feature
1. Check if it belongs in adapter layer (MUD-specific) or client (generic)
2. Update protocol if needed (see `docs/PROTOCOL.md`)
3. Implement Go backend first, then wire to frontend
4. Test with actual MUD connection, not mocked data

#### Debugging Parser Issues
1. Enable verbose logging in `app.go:195` (already logging parsed output)
2. Check raw MUD output vs parsed classification
3. Room detection is critical—verify TypeRoomTitle triggers mapper

#### Testing Image Generation
1. Ensure Stable Diffusion is running: `curl http://127.0.0.1:7860/sdapi/v1/sd-models`
2. Check logs for prompt construction
3. Verify cache directory exists: `cache/room_images/`
4. Image cache filenames are sanitised room names (lowercase, underscores)

#### Working with Maps
1. Map files stored in `maps/` as `{server_name}_{port}_map.json`
2. Graph structure: rooms (nodes) with exits (edges)
3. Use `GetMapData()` API for visualisation data
4. Don't modify graph directly—use mapper methods

## File Locations Reference

### Configuration
- `wails.json` - Wails project config (app name, build settings)
- `go.mod` - Go dependencies
- `frontend/package.json` - React dependencies

### Documentation
- `docs/PRD.md` - Product requirements and vision
- `docs/ARCHITECTURE.md` - System architecture and design philosophy
- `docs/PROTOCOL.md` - SeeMUD protocol specification
- `docs/ADAPTER_DESIGN.md` - How to implement MUD adapters
- `docs/AI_INTEGRATION.md` - AI classification strategy
- `docs/ROADMAP.md` - Development phases

### Entry Points
- `main.go` - Application entry point
- `app.go` - Main app struct with all backend logic
- `frontend/src/App.jsx` - Frontend entry component

### Key Internal Packages
- `internal/telnet/client.go` - Telnet connection handling
- `internal/parser/wolfmud.go` - Text parsing (needs refactoring)
- `internal/mapper/mapper.go` - Spatial mapping logic
- `internal/mapper/graph.go` - Room graph data structure
- `internal/renderer/stable_diffusion.go` - SD API client
- `internal/renderer/prompts.go` - Prompt engineering

## Development Workflow

### Running the App
```bash
# Development mode (hot reload)
wails dev

# Production build
wails build

# Quick launch scripts
./launch.sh           # Dev mode
./launch-binary.sh    # Run built binary
```

### Making Changes

1. **Backend changes (Go):**
   - Edit files in `internal/` or `app.go`
   - Wails will auto-rebuild in dev mode
   - Restart if adding new exported methods

2. **Frontend changes (React):**
   - Edit files in `frontend/src/`
   - Vite provides instant hot reload
   - Regenerate bindings if backend API changed: `wails generate module`

3. **Testing:**
   - Go tests: `go test ./...`
   - Manual testing: Connect to a real MUD server

### Commit Guidelines
- Use descriptive commit messages
- Separate refactoring from features
- Reference issue numbers if applicable
- Don't commit `cache/` or `maps/` directories (in .gitignore)

## Common Pitfalls

1. **Don't Assume MUD Format:** Every MUD formats text differently—test thoroughly
2. **Cache Invalidation:** Room images cached by sanitised name—rename detection is not implemented
3. **Concurrency:** Image generation is async—handle errors gracefully
4. **State Sync:** Frontend polls backend—don't assume instant updates
5. **Path Handling:** Use `filepath.Join()` for cross-platform compatibility
6. **Telnet Protocol:** Handle ANSI codes, IAC sequences (see telnet package)

## Quick Wins (Good First Tasks)

- Add support for additional exit directions (up, down, etc.)
- Improve prompt engineering for better image quality
- Add UI controls for image size/quality settings
- Implement search/filter for map rooms
- Add keyboard shortcuts for common commands
- Improve error messages for connection failures

## Advanced Tasks (Requires Deep Understanding)

- Implement MUD adapter system (see `docs/ADAPTER_DESIGN.md`)
- Add AI-powered text classification (see `docs/AI_INTEGRATION.md`)
- Implement multi-layer image compositing
- Add support for MXP protocol extensions
- Create plugin system for custom parsers

## When in Doubt

1. **Check Documentation:** Comprehensive docs in `docs/` directory
2. **Read the Code:** Code is well-commented, especially in mapper and parser
3. **Test with Real MUD:** Don't trust mocked data—connect to actual server
4. **Ask Questions:** Better to clarify than implement incorrectly

## Context for AI Assistants

This project is deeply personal to the developer—it's about bringing childhood MUD experiences to life with modern technology. The goal is not to replace the imagination-driven nature of MUDs, but to enhance it.

**Key emotional drivers:**
- Nostalgia for golden-era text games
- Desire to see familiar MUD locations as imagined
- Respect for original game servers (no modifications)
- Building something that fellow MUD enthusiasts will love

**When suggesting changes:**
- Prioritise preserving the MUD experience
- Consider performance (image generation is expensive)
- Think about extensibility (supporting multiple MUDs)
- Value simplicity over complexity

**Development philosophy:**
- Ship working features incrementally
- User experience over technical purity (for now)
- Clean architecture for long-term maintainability
- Open to significant refactoring when it makes sense

## Current Focus

The project is transitioning from prototype to production-ready:
- **Now:** Core features working, but architecture needs refactoring
- **Next:** Implement adapter system for MUD-agnostic client
- **Future:** AI-powered classification, multi-layer compositing, plugin system

See `docs/ROADMAP.md` for detailed implementation phases.

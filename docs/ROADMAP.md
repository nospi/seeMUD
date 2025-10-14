# Implementation Roadmap

## Overview

This document outlines the phased approach to implementing the MUD adapter architecture for SeeMUD.

**Philosophy:** Incremental delivery with working software at each phase.

## Phase 0: Documentation ✓

**Status:** Complete

### Deliverables
- [x] Architecture overview (ARCHITECTURE.md)
- [x] Adapter design guide (ADAPTER_DESIGN.md)
- [x] Protocol specification (PROTOCOL.md)
- [x] AI integration strategy (AI_INTEGRATION.md)
- [x] This roadmap (ROADMAP.md)

### Success Criteria
- Clear understanding of the target architecture
- Agreement on protocol design
- Confidence in AI strategy

**Time Estimate:** 2-3 hours
**Actual:** ~3 hours

---

## Phase 1: Foundation & Refactoring

**Goal:** Extract existing WolfMUD parser into new adapter architecture without breaking current functionality.

**Status:** Not started

### Tasks

#### 1.1: Create Package Structure
```
internal/
├── adapters/
│   ├── adapter.go           # Interface definition
│   ├── base.go              # Base adapter with common functionality
│   └── wolfmud/
│       ├── wolfmud.go       # Main adapter
│       ├── patterns.go      # Pattern matching
│       ├── knowledge.go     # Knowledge base
│       └── README.md        # WolfMUD-specific docs
├── protocol/
│   ├── types.go             # Message types
│   ├── messages.go          # Message constructors
│   └── validation.go        # Protocol validation
└── ai/
    ├── classifier.go        # AI interface
    ├── mock.go              # Mock for testing
    └── README.md
```

**Time Estimate:** 1 hour

#### 1.2: Define Protocol Types

Create protocol message types in `internal/protocol/types.go`:

```go
type Message struct {
    Type      string    `json:"type"`
    Timestamp time.Time `json:"timestamp"`
    Data      interface{} `json:"data"`
}

type RoomEntry struct {
    RoomID      string            `json:"room_id"`
    Name        string            `json:"name"`
    Description string            `json:"description"`
    Exits       []string          `json:"exits"`
    Metadata    map[string]string `json:"metadata"`
    Confidence  string            `json:"confidence"`
    IsNewRoom   bool              `json:"is_new_room"`
}

// ... other message types
```

**Time Estimate:** 1 hour

#### 1.3: Create Adapter Interface

Define `MUDAdapter` interface in `internal/adapters/adapter.go`:

```go
type MUDAdapter interface {
    Name() string
    ParseLine(line string) []protocol.Message
    GetContext() *ClassifiedContext
    // ... other methods
}
```

**Time Estimate:** 30 minutes

#### 1.4: Extract WolfMUD Parser

Move logic from `internal/parser/wolfmud.go` to `internal/adapters/wolfmud/`:

- Extract regex patterns
- Move parsing logic
- Add history buffer
- Add context tracking
- **Keep existing functionality intact**

**Time Estimate:** 3-4 hours

#### 1.5: Update App to Use Adapter

Modify `app.go` to:
- Instantiate WolfMUDAdapter instead of parser
- Consume protocol messages
- Update mapper to receive structured data
- **Ensure backward compatibility**

**Time Estimate:** 2-3 hours

#### 1.6: Testing & Validation

- Unit tests for adapter
- Integration tests with real WolfMUD output
- Regression tests to ensure nothing broke

**Time Estimate:** 2 hours

### Success Criteria

- ✅ WolfMUD works identically to before
- ✅ All existing features functional
- ✅ Code is cleaner and more modular
- ✅ Tests pass
- ✅ Ready for AI integration

**Total Time Estimate:** 10-13 hours
**Target Completion:** [To be determined]

---

## Phase 2: AI Integration

**Goal:** Add AI-powered classification to WolfMUD adapter.

**Status:** Not started

### Prerequisites
- Phase 1 complete
- Claude API key obtained
- Ollama installed (optional, for fallback)

### Tasks

#### 2.1: AI Client Implementation

Create AI clients in `internal/ai/`:

```go
// claude.go
type ClaudeClient struct {
    apiKey string
    endpoint string
}

func (c *ClaudeClient) Classify(prompt string) (string, error) {
    // Call Claude API
}

// ollama.go
type OllamaClient struct {
    endpoint string
    model string
}

func (o *OllamaClient) Classify(prompt string) (string, error) {
    // Call local Ollama
}
```

**Time Estimate:** 3 hours

#### 2.2: Add Context Tracking

Enhance WolfMUD adapter with:
- History buffer (last 50 lines)
- Classified context (zone, atmosphere)
- Context cache (30 second TTL)

**Time Estimate:** 2 hours

#### 2.3: Implement Classification

Add AI classification methods to WolfMUD adapter:

```go
func (w *WolfMUDAdapter) ClassifyContext(history []string) (*ClassifiedContext, error) {
    // Build prompt
    // Call AI
    // Parse response
    // Cache result
}
```

**Time Estimate:** 3 hours

#### 2.4: Knowledge Base

Implement pattern learning:

```go
type KnowledgeBase struct {
    RoomPatterns map[string]*RoomPattern
    ZoneTriggers map[string][]TriggerPhrase
}

func (kb *KnowledgeBase) Learn(ctx *ClassifiedContext, history []string) {
    // Extract patterns
    // Update confidence
    // Save to disk
}
```

**Time Estimate:** 4 hours

#### 2.5: Integration & Testing

- Integrate AI calls into parsing flow
- Test with real WolfMUD sessions
- Validate zone detection
- Verify metadata enrichment
- Monitor AI call frequency and costs

**Time Estimate:** 3 hours

### Success Criteria

- ✅ Zones detected automatically
- ✅ Room metadata enriches image generation
- ✅ Knowledge base learns patterns
- ✅ AI calls stay under 10/minute
- ✅ Fast path used >50% of the time after initial learning

**Total Time Estimate:** 15 hours
**Target Completion:** [To be determined]

---

## Phase 3: Astaria Support

**Goal:** Add support for Astaria MUD.

**Status:** Not started

### Prerequisites
- Phase 2 complete
- Astaria connection details obtained
- Test account created

### Research Phase (2-3 hours)

#### 3.1: Connect & Capture

- Connect to Astaria
- Capture raw output samples
- Document room formats
- Identify exit patterns
- Note entity formats
- Record zone transitions

**Deliverable:** `internal/adapters/astaria/RESEARCH.md` with findings

#### 3.2: Analyze Patterns

- Compare to WolfMUD
- Identify unique quirks
- Plan parsing strategy
- Document differences

### Implementation Phase (8-12 hours)

#### 3.3: Create Astaria Adapter

```
internal/adapters/astaria/
├── astaria.go           # Main adapter
├── patterns.go          # Astaria-specific patterns
├── knowledge.go         # Knowledge base
├── RESEARCH.md          # Research findings
└── README.md            # Usage guide
```

#### 3.4: Implement Parsing

- Room title detection
- Exit parsing
- Entity detection
- Zone classification
- **Leverage AI heavily initially**

**Time Estimate:** 5 hours

#### 3.5: Add Adapter Selection

Update `app.go` to support multiple adapters:

```go
type AdapterType string

const (
    AdapterWolfMUD AdapterType = "wolfmud"
    AdapterAstaria AdapterType = "astaria"
)

func (a *App) ConnectToMUD(host, port string, adapterType AdapterType) error {
    switch adapterType {
    case AdapterWolfMUD:
        a.adapter = wolfmud.NewAdapter()
    case AdapterAstaria:
        a.adapter = astaria.NewAdapter()
    }
    // ...
}
```

**Time Estimate:** 2 hours

#### 3.6: UI for Adapter Selection

Add dropdown/selector in frontend:

```jsx
<select onChange={handleAdapterChange}>
  <option value="wolfmud">WolfMUD</option>
  <option value="astaria">Astaria</option>
</select>
```

**Time Estimate:** 1 hour

### Validation Phase (4-6 hours)

#### 3.7: Playtest Astaria

- Navigate familiar areas
- Verify map accuracy
- Check room images
- Test zone transitions
- Validate entity detection

#### 3.8: Refine Patterns

- Adjust regex patterns
- Tune AI prompts
- Build knowledge base
- **Iterate until it "feels right"**

### Success Criteria

- ✅ Can connect to Astaria
- ✅ Rooms map correctly
- ✅ Familiar areas look right
- ✅ Images reflect the game world
- ✅ Zone detection works
- ✅ Adapter learns over time

**Total Time Estimate:** 14-21 hours
**Target Completion:** [To be determined]

---

## Phase 4: Polish & Optimization

**Goal:** Production-ready system with good UX.

**Status:** Not started

### Tasks

#### 4.1: Performance Optimization

- Profile AI call frequency
- Optimize caching strategy
- Batch related requests
- Reduce redundant parsing
- Benchmark adapter performance

**Time Estimate:** 3 hours

#### 4.2: Cost Management

- Implement cost tracking
- Add usage warnings
- Create budget limits
- Log cost analytics
- Optimize prompt sizes

**Time Estimate:** 2 hours

#### 4.3: Error Handling

- Graceful degradation when AI unavailable
- Fallback to pattern matching
- User-friendly error messages
- Reconnection logic
- State recovery

**Time Estimate:** 3 hours

#### 4.4: User Feedback System

Implement correction commands:

```
/zone correct <old> <new>
/room merge <id1> <id2>
/learn "<phrase>" → zone:<zone>
/confidence <room_id> <high|medium|low>
```

**Time Estimate:** 4 hours

#### 4.5: Knowledge Base Management

- Export/import knowledge bases
- Share knowledge between users
- Version knowledge bases
- Reset/clear learned patterns
- Backup/restore functionality

**Time Estimate:** 3 hours

#### 4.6: Configuration UI

Add settings panel:

- AI provider selection
- API key management
- Rate limit settings
- Cache configuration
- Cost alerts

**Time Estimate:** 4 hours

#### 4.7: Analytics & Insights

- Session statistics
- AI usage dashboard
- Pattern confidence metrics
- Cost breakdown
- Adapter performance

**Time Estimate:** 3 hours

### Success Criteria

- ✅ Stable and reliable
- ✅ Good performance (<100ms average parsing time)
- ✅ Costs are reasonable (<$0.10 per hour of gameplay)
- ✅ Users can correct mistakes easily
- ✅ Knowledge bases improve over time
- ✅ Professional UX

**Total Time Estimate:** 22 hours
**Target Completion:** [To be determined]

---

## Phase 5: Community & Expansion

**Goal:** Make it easy for others to add MUD support.

**Status:** Not started

### Tasks

#### 5.1: Documentation

- Adapter development guide
- Porting guide (from scratch)
- Best practices
- Example adapter walkthrough
- Troubleshooting guide

**Time Estimate:** 6 hours

#### 5.2: Adapter Template

Create template/boilerplate:

```
internal/adapters/template/
├── adapter.go.template
├── patterns.go.template
├── README.md.template
└── CHECKLIST.md
```

**Time Estimate:** 3 hours

#### 5.3: Testing Tools

- Capture tool (record MUD sessions)
- Replay tool (test adapters offline)
- Validation tool (check protocol compliance)
- Benchmark tool (measure performance)

**Time Estimate:** 5 hours

#### 5.4: Knowledge Sharing

- Knowledge base repository
- Community-contributed patterns
- Adapter registry
- Share best practices

**Time Estimate:** 3 hours

#### 5.5: Third Adapter (Proof of Concept)

Add support for another popular MUD (e.g., Achaea, Aardwolf) to validate the architecture and demonstrate ease of implementation.

**Time Estimate:** 10-15 hours

### Success Criteria

- ✅ Clear documentation
- ✅ Easy to add new MUDs
- ✅ Template accelerates development
- ✅ Community can contribute
- ✅ Knowledge bases are shareable

**Total Time Estimate:** 27-32 hours
**Target Completion:** [To be determined]

---

## Summary

### Total Time Investment

| Phase | Time Estimate | Dependencies |
|-------|--------------|--------------|
| Phase 0: Documentation | 3 hours | None |
| Phase 1: Foundation | 10-13 hours | Phase 0 |
| Phase 2: AI Integration | 15 hours | Phase 1 |
| Phase 3: Astaria Support | 14-21 hours | Phase 2 |
| Phase 4: Polish | 22 hours | Phase 3 |
| Phase 5: Community | 27-32 hours | Phase 4 |
| **Total** | **91-106 hours** | |

### Milestones

**Milestone 1: Working Adapter Architecture** (Phase 1)
- Clean architecture
- WolfMUD works
- Ready for AI

**Milestone 2: Intelligent Parsing** (Phase 2)
- AI-powered classification
- Learning knowledge base
- Reliable zone detection

**Milestone 3: Multi-MUD Support** (Phase 3)
- Astaria working
- Adapter selection UI
- Validated architecture

**Milestone 4: Production Ready** (Phase 4)
- Polished UX
- Cost-effective
- Reliable and stable

**Milestone 5: Community Platform** (Phase 5)
- Easy to extend
- Shareable knowledge
- Documentation complete

### Critical Path

```
Phase 0 → Phase 1 → Phase 2 → Phase 3
                                  ↓
                              Phase 4 → Phase 5
```

**Minimum Viable Product (MVP):** Phases 1-3
**Production Release:** Phases 1-4
**Community Platform:** All phases

### Risk Mitigation

**Risk 1: AI costs too high**
- Mitigation: Aggressive caching, pattern learning, fast path optimization
- Fallback: Use Ollama (local LLM)

**Risk 2: Astaria format too different**
- Mitigation: Extensive research phase, AI-heavy initially
- Fallback: Start with simpler MUD (e.g., another DikuMUD)

**Risk 3: Performance issues**
- Mitigation: Benchmark early, optimize patterns, cache aggressively
- Fallback: Async processing, background classification

**Risk 4: Knowledge base doesn't improve accuracy**
- Mitigation: Monitor metrics, iterate on learning algorithm
- Fallback: Rely more on AI, less on patterns

### Success Metrics

**Phase 1:**
- 0 regressions
- 100% feature parity with current implementation

**Phase 2:**
- Zone detection accuracy > 90%
- AI calls < 10/minute after 1 hour of gameplay
- Fast path usage > 50% after learning

**Phase 3:**
- Astaria map accuracy > 85% (based on known areas)
- Images match player's mental model
- Zone transitions feel natural

**Phase 4:**
- Parsing time < 100ms average
- Cost < $0.10/hour
- User satisfaction with correction system

**Phase 5:**
- 3+ adapters available
- Community contribution (1+ external adapter)
- Documentation rating > 8/10

### Next Steps

1. **Review and approve this roadmap**
2. **Set up development environment**
3. **Begin Phase 1, Task 1.1**

---

## Notes

### Why This Order?

- **Phase 1 first:** Establishes solid foundation
- **Phase 2 before Phase 3:** Prove AI strategy with known MUD before adding complexity
- **Phase 3 validates architecture:** If Astaria works, any MUD will work
- **Phase 4 polish:** Make it production-ready before community
- **Phase 5 last:** Need proven architecture before scaling to community

### Flexibility

This roadmap is a guide, not a strict plan. We can:
- Skip Phase 5 if community isn't a goal
- Do Phase 4 tasks incrementally throughout other phases
- Add new phases (e.g., "Phase 3.5: Third MUD") if needed
- Adjust time estimates based on actual progress

### Dependencies

- Claude API access (for Phase 2)
- Astaria connection details (for Phase 3)
- Stable Diffusion running (existing)
- Go 1.21+ (existing)
- Wails (existing)

### Out of Scope

These are explicitly NOT part of this roadmap:

- Multi-user / proxy server architecture
- Authentication / account management
- Cloud-hosted adapters
- Mobile clients
- VR integration
- Game automation / bots
- Server-side modifications

This roadmap focuses on single-player, local adapter architecture only.

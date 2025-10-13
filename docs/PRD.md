# SeeMUD: Visual MUD Client Product Requirements Document

## Executive Summary

SeeMUD is an innovative MUD (Multi-User Dungeon) client that enhances the traditional text-based gaming experience by automatically generating visual representations of game environments, items, and characters. The system uses AI-powered image generation and intelligent caching to create immersive, dynamic visuals while maintaining the classic MUD gameplay experience.

## Project Vision

Transform text-based MUD gaming into a hybrid visual-textual experience, preserving the imagination-driven nature of MUDs while offering optional visual enhancement that adapts to each player's preferences.

## Core Features

### 1. Intelligent Text Parsing & Classification

**Description**: Parse MUD output to identify and classify different content types.

**Components**:
- Room descriptions
- Item descriptions
- Mob/NPC descriptions
- Player actions
- System messages

**Technical Approach**:
- Use local or cloud-based LLM for classification
- Pattern matching for common MUD formats
- Context-aware parsing (maintains game state)

### 2. Dynamic Image Generation

**Description**: Generate images on-demand based on parsed text descriptions.

**Key Features**:
- Real-time generation triggered by room entry
- Background pre-generation for visible items/mobs
- User-guided regeneration with custom prompts
- Support for multiple art styles/themes

**Supported Generators** (Pluggable Architecture):
- Midjourney API
- DALL-E (OpenAI)
- Stable Diffusion (local or API)
- Flux
- Future generators via plugin system

### 3. Intelligent Caching System

**Description**: Multi-tier caching strategy for efficient resource management.

**Cache Layers**:
1. **Room Cache**: Backgrounds/environments
2. **Entity Cache**: Mobs, NPCs, creatures
3. **Item Cache**: Objects, equipment, treasures
4. **Composite Cache**: Pre-rendered room+entity combinations

**Cache Features**:
- Content-addressable storage (hash-based)
- Version management (multiple variations per prompt)
- User preference tracking
- Automatic cache pruning (LRU)
- Export/import cache libraries

### 4. Layered Rendering System

**Description**: Composite multiple image layers for final scene.

**Layers** (back to front):
1. Background (room/environment)
2. Static objects (furniture, decorations)
3. Items (lootable/interactable)
4. Entities (mobs, NPCs)
5. Effects (weather, lighting)

**Compositing Features**:
- Smart positioning based on description
- Transparency and blending
- Dynamic scaling
- Occlusion handling

### 5. User Interaction & Control

**Image Management**:
- Browse generated variations
- Select preferred version
- Request regeneration
- Add custom prompt modifiers
- Save favorites to personal library

**Look Command Integration**:
- Popup modal for examined objects
- Detailed view generation
- Context-sensitive rendering

**UI Controls**:
- Toggle visual mode on/off
- Adjust image update frequency
- Set art style preferences
- Configure cache behaviour

### 6. AI Classification System

**Purpose**: Intelligently parse and understand MUD output.

**Classification Tasks**:
- Separate room description from contents
- Identify items vs. mobs vs. scenery
- Extract key visual elements
- Determine spatial relationships
- Detect description changes

**Implementation Options**:
- **Local**: Ollama with Llama/Mistral models
- **Cloud**: OpenAI GPT, Anthropic Claude
- **Hybrid**: Local for speed, cloud for complex scenes

## Technical Architecture

### System Components

```
┌─────────────────┐
│   MUD Server    │
└────────┬────────┘
         │ Telnet/TCP
┌────────▼────────┐
│  Connection     │
│    Manager      │
└────────┬────────┘
         │
┌────────▼────────┐
│  Text Parser    │
└────────┬────────┘
         │
┌────────▼────────┐
│  AI Classifier  │
└───┬────────┬────┘
    │        │
┌───▼──┐ ┌──▼────┐
│Cache │ │Gen    │
│System│ │Manager│
└───┬──┘ └──┬────┘
    │       │
┌───▼───────▼────┐
│   Compositor    │
└────────┬────────┘
         │
┌────────▼────────┐
│    UI Layer     │
└─────────────────┘
```

### Data Flow

1. **Input**: MUD server sends text output
2. **Parsing**: Extract description components
3. **Classification**: Identify content types via AI
4. **Cache Check**: Look for existing renders
5. **Generation**: Create missing images
6. **Composition**: Layer images into scene
7. **Display**: Present to user with controls

### Module Interfaces

#### Image Generator Interface
```go
type ImageGenerator interface {
    Generate(prompt string, options GenerationOptions) (*Image, error)
    GenerateVariations(prompt string, count int, options GenerationOptions) ([]*Image, error)
    Upscale(image *Image) (*Image, error)
    GetCapabilities() Capabilities
}
```

#### Classifier Interface
```go
type Classifier interface {
    ClassifyText(input string) (*Classification, error)
    ExtractEntities(description string) ([]Entity, error)
    DetermineRelationships(entities []Entity) (*SpatialMap, error)
}
```

#### Cache Interface
```go
type Cache interface {
    Get(key string) (*CachedImage, error)
    Put(key string, image *Image, metadata Metadata) error
    GetVariations(baseKey string) ([]*CachedImage, error)
    Prune(strategy PruneStrategy) error
}
```

## User Stories

### As a MUD Player

1. **I want** to see visual representations of rooms **so that** I can better immerse myself in the game world
2. **I want** to regenerate images I don't like **so that** visuals match my imagination
3. **I want** to save favourite images **so that** consistent locations look the same
4. **I want** to toggle visuals on/off **so that** I can play traditionally when desired
5. **I want** to customize art styles **so that** visuals match my aesthetic preferences

### As a Power User

1. **I want** to export my image cache **so that** I can share with other players
2. **I want** to use local AI models **so that** I maintain privacy
3. **I want** to configure multiple generators **so that** I can choose quality vs. speed
4. **I want** to script custom parsing rules **so that** uncommon MUD formats work

## Non-Functional Requirements

### Performance
- Image generation: < 5 seconds for initial render
- Cache retrieval: < 50ms
- Text parsing: Real-time (< 100ms)
- Memory usage: < 500MB baseline
- Cache size: Configurable limit (default 10GB)

### Reliability
- Graceful degradation without image service
- Offline mode with cached content
- Automatic reconnection to MUD
- Error recovery for failed generations

### Compatibility
- Support major MUD protocols (Telnet, MXP)
- Cross-platform (Windows, macOS, Linux)
- Multiple MUD codebases (CircleMUD, ROM, etc.)
- Configurable parsing rules

### Security
- Secure API key storage
- Content filtering for generated images
- Privacy mode (no external API calls)
- Local-only operation option

## Configuration

### Example Configuration File
```yaml
# seemud.config.yaml
connection:
  host: "example.mud.com"
  port: 4000
  protocol: "telnet"

classifier:
  provider: "local"  # local, openai, anthropic
  model: "llama2-7b"
  endpoint: "http://localhost:11434"

generators:
  primary:
    provider: "stable-diffusion"
    endpoint: "http://localhost:7860"
    model: "sd-xl-base-1.0"
  fallback:
    provider: "openai"
    api_key: "${OPENAI_API_KEY}"
    model: "dall-e-3"

cache:
  directory: "~/.seemud/cache"
  max_size: "10GB"
  strategy: "lru"

rendering:
  default_style: "fantasy-realism"
  enable_compositing: true
  layer_opacity:
    background: 1.0
    items: 0.9
    entities: 0.95

ui:
  start_with_visuals: true
  popup_on_look: true
  variation_count: 4
```

## Development Phases

### Phase 1: Foundation (Weeks 1-2)
- [ ] Basic MUD connection (telnet client)
- [ ] Text output capture and buffering
- [ ] Simple pattern-based parser
- [ ] File-based caching system

### Phase 2: Intelligence (Weeks 3-4)
- [ ] LLM integration for classification
- [ ] Entity extraction from descriptions
- [ ] Spatial relationship detection
- [ ] Smart prompt generation

### Phase 3: Generation (Weeks 5-6)
- [ ] Image generator interface
- [ ] First generator implementation (Stable Diffusion)
- [ ] Basic caching with hash keys
- [ ] Simple UI for image display

### Phase 4: Sophistication (Weeks 7-8)
- [ ] Multi-layer compositing
- [ ] Variation generation and management
- [ ] User preference learning
- [ ] Advanced cache strategies

### Phase 5: Polish (Weeks 9-10)
- [ ] Additional generator plugins
- [ ] UI improvements and controls
- [ ] Performance optimisation
- [ ] Documentation and testing

## Success Metrics

### Quantitative
- Generation time < 5 seconds (P95)
- Cache hit rate > 70% after 1 hour of play
- Memory usage < 500MB baseline
- User retention > 50% after 1 week

### Qualitative
- Images accurately represent descriptions
- Consistent style within game sessions
- Minimal disruption to gameplay flow
- Positive user feedback on immersion

## Technical Decisions

### Language: Go
**Rationale**:
- Excellent concurrency for parallel image generation
- Single binary deployment
- Strong networking libraries for MUD connection
- Good performance with lower memory overhead
- Mature ecosystem for web services
- Faster learning curve than Rust

### Image Storage: Content-Addressable
**Rationale**:
- Deduplication of similar prompts
- Consistent addressing across sessions
- Easy cache sharing between users
- Natural versioning support

### AI Classification: Hybrid Approach
**Rationale**:
- Local models for speed and privacy
- Cloud models for complex scenes
- Fallback ensures availability
- User choice for privacy preferences

## Risks & Mitigations

### Risk: Image Generation Latency
**Mitigation**:
- Aggressive pre-generation
- Multiple cache tiers
- Progressive rendering

### Risk: Inappropriate Content Generation
**Mitigation**:
- Content filtering APIs
- User reporting system
- Configurable safety levels

### Risk: MUD Format Variations
**Mitigation**:
- Configurable parsing rules
- Community rule sharing
- Fallback to pure text

### Risk: API Cost Overruns
**Mitigation**:
- Local model options
- Aggressive caching
- Rate limiting
- Cost alerts

## Future Enhancements

### Version 2.0
- Multi-player cache sharing
- Custom model fine-tuning
- Animation support (idle animations)
- Sound generation from descriptions

### Version 3.0
- VR/AR viewing modes
- Real-time style transfer
- Collaborative world building
- Plugin marketplace

## Appendices

### A. Glossary
- **MUD**: Multi-User Dungeon
- **Mob**: Mobile object (NPC/creature)
- **LLM**: Large Language Model
- **LRU**: Least Recently Used

### B. References
- MUD Protocol Specifications
- Image Generation API Documentation
- Go Concurrency Patterns
- Content-Addressable Storage Patterns

### C. Example MUD Output
```
The Grand Hall
You stand in a magnificent hall with vaulted ceilings that disappear into
shadow above. Massive pillars of white marble support the structure, each
carved with intricate battle scenes. Tattered banners hang from the walls,
their colours faded but still showing hints of gold and crimson.

A ornate wooden chest sits in the corner.
A silver goblet gleams on a nearby pedestal.

A spectral guardian hovers near the northern exit.
A small mouse scurries along the baseboards.

Exits: north, south, east
```

### D. Example Classification Output
```json
{
  "room": {
    "title": "The Grand Hall",
    "description": "magnificent hall with vaulted ceilings...",
    "atmosphere": "ancient, grand, mysterious"
  },
  "items": [
    {"name": "ornate wooden chest", "position": "corner"},
    {"name": "silver goblet", "position": "pedestal"}
  ],
  "entities": [
    {"name": "spectral guardian", "position": "north exit", "type": "hostile"},
    {"name": "small mouse", "position": "baseboards", "type": "ambient"}
  ],
  "exits": ["north", "south", "east"]
}
```

---

*Document Version: 1.0*
*Last Updated: 2024*
*Status: Initial Draft*
# MUD Adapter Design

## Core Concept

Adapters are **stateful AI-powered parsers** that transform unstructured MUD text into structured SeeMUD Protocol messages.

Each MUD gets its own adapter because:
- Text formats differ significantly between MUDs
- Room naming conventions vary
- Exit formats are MUD-specific
- Special features are unique per MUD

## Adapter State

An adapter is **not stateless**. It maintains context across multiple lines of input, building up an understanding of where the player is and what's happening.

### 1. History Buffer

```go
type HistoryBuffer struct {
    Lines    []string      // Last 50 lines of raw output
    MaxSize  int           // Maximum buffer size
    Cursor   int           // Current position in circular buffer
}

func (h *HistoryBuffer) Add(line string) {
    // Add to circular buffer
}

func (h *HistoryBuffer) Last(n int) []string {
    // Get last N lines for AI context
}

func (h *HistoryBuffer) Clear() {
    // Reset buffer
}
```

**Purpose:**
- Provides context for AI classification
- Helps detect zone transitions
- Enables multi-line pattern matching
- Allows "looking back" at recent events

**Example:**
```
Line 1: "You leave the bustling city streets behind."
Line 2: "The smell of salt air grows stronger."
Line 3: "You arrive at the docks."
Line 4: ""
Line 5: "[Dockside warehouse]"
Line 6: "This weathered building..."
```

AI can see lines 1-3 to understand: "Player just transitioned from city → docks"

### 2. Classified Context

```go
type ClassifiedContext struct {
    CurrentZone      string            // "city", "docks", "forest", "dungeon"
    Atmosphere       string            // "urban, bustling" or "dark, dangerous"
    LocationType     string            // "shop", "street", "dungeon_room", "wilderness"
    RecentEvents     []string          // ["entered_shop", "combat_started"]
    ZoneMetadata     map[string]string // Arbitrary zone-specific data
    LastClassified   time.Time         // When we last ran AI classification
    Confidence       string            // "high", "medium", "low"
}

func (c *ClassifiedContext) IsStale() bool {
    return time.Since(c.LastClassified) > 30*time.Second
}

func (c *ClassifiedContext) Update(ai AIClassification) {
    c.CurrentZone = ai.Zone
    c.Atmosphere = ai.Atmosphere
    c.LocationType = ai.LocationType
    c.LastClassified = time.Now()
    c.Confidence = ai.Confidence
}
```

**Purpose:**
- Persists across multiple rooms
- Enriches room metadata
- Reduces need for repeated AI calls
- Provides context for image generation

**Example Lifecycle:**
```
Room 1: City gate
    → AI classifies: zone="city", atmosphere="urban, guarded"

Rooms 2-5: City streets
    → Reuse context (still in city, < 30 seconds)

Room 6: "You pass through the city walls"
    → AI classifies: zone="wilderness", atmosphere="open, rural"

Rooms 7-20: Forest
    → Reuse context (still in wilderness)
```

### 3. Knowledge Base

```go
type KnowledgeBase struct {
    // Room patterns learned over time
    RoomSignatures   map[string]RoomPattern

    // Zone trigger phrases
    ZoneTriggers     map[string][]string  // zone → trigger phrases

    // Pattern confidence scores
    ConfidenceScores map[string]float64

    // Disambiguation rules
    DisambRules      []DisambiguationRule

    // User corrections (learning feedback)
    Corrections      []Correction
}

type RoomPattern struct {
    RoomID      string   // Stable room ID
    NamePattern string   // Regex or exact match
    KeyPhrases  []string // Description must contain these
    Zone        string   // Expected zone
    Confidence  float64  // How reliable is this pattern?
}

type DisambiguationRule struct {
    NamePattern string   // Rooms matching this name
    Rules       []Rule   // If description contains X → room Y
}

type Correction struct {
    Timestamp   time.Time
    InputText   string
    AIOutput    interface{}
    UserCorrect interface{}
    Reason      string
}
```

**Purpose:**
- Learn patterns over time
- Reduce AI dependency
- Speed up classification
- Improve accuracy through feedback

**Example:**
```go
// After several visits, adapter learns:
kb.RoomSignatures["astaria_city_market_square"] = RoomPattern{
    RoomID: "astaria_city_market_square_01",
    NamePattern: "Market [Ss]quare",
    KeyPhrases: []string{"bustling", "vendors", "stalls"},
    Zone: "city",
    Confidence: 0.95,
}

// Next time we see "Market Square" with "vendors" in description:
// → Skip AI, use this pattern (fast path)
```

## AI Classification Strategy

### When to Call AI

**1. Zone Transition Suspected**
```go
func (a *Adapter) mightBeZoneTransition(line string) bool {
    triggers := []string{
        "leave", "depart", "enter", "arrive",
        "pass through", "walk into", "head toward",
    }

    for _, trigger := range triggers {
        if strings.Contains(strings.ToLower(line), trigger) {
            return true
        }
    }

    return false
}
```

**2. Ambiguous Room Identity**
- Multiple rooms with similar names
- Description doesn't match known pattern
- Low confidence from pattern matching

**3. Context is Stale**
- Last AI call was > 30 seconds ago
- Player might have moved zones

**4. Explicit User Trigger**
- User runs `/classify` command
- Manual override needed

### When NOT to Call AI

**1. High-Confidence Pattern Match**
```go
// Exact room name + key phrases present
if pattern := kb.Match(roomName, description); pattern != nil {
    if pattern.Confidence > 0.9 {
        return immediateClassification(pattern)
    }
}
```

**2. Recent Classification**
```go
if !context.IsStale() {
    // Use cached context (< 30 seconds old)
    return context
}
```

**3. Clear Structural Markers**
```go
// Room name in brackets: [Market Square]
// Obvious exit line: "Obvious exits: north, south, east, west"
if hasStructuralMarkers(line) {
    return fastParse(line)
}
```

**4. Rate Limited**
```go
// Protect against API costs
if aiCallsThisMinute > 10 {
    return fallbackParse(line)
}
```

## AI Prompt Templates

### Zone Detection Prompt

```
You are analyzing output from a MUD (text-based role-playing game).

Recent output:
"""
{last_20_lines}
"""

Current beliefs:
- Zone: {current_zone}
- Atmosphere: {atmosphere}
- Location type: {location_type}

Analyze the output and answer:

1. Has the player moved to a different zone (area/region)?
   - If no change, say "no_change"
   - If changed, provide the new zone name

2. What is the atmosphere/mood of the current location?
   - Provide 2-3 descriptive tags (e.g., "dark, dangerous" or "peaceful, rural")

3. What type of location is this?
   - Options: street, shop, tavern, dungeon_room, wilderness, indoor, outdoor

4. Are there any persistent contextual details to track?
   - E.g., "player is now in a combat zone" or "entered a restricted area"

5. How confident are you in this classification?
   - high: Very clear indicators
   - medium: Reasonable inference
   - low: Guessing based on limited info

Respond ONLY with valid JSON:
{
  "zone_changed": boolean,
  "new_zone": "zone_name or null",
  "atmosphere": "comma, separated, tags",
  "location_type": "type from list above",
  "context_notes": "any important contextual info",
  "confidence": "high|medium|low",
  "reasoning": "brief explanation of your classification"
}
```

### Room Identity Prompt

```
You are helping identify if a room in a MUD is new or a return to a known location.

Current room:
Name: {room_name}
Description: {room_description}
Exits: {exits}

Known rooms with similar names:
{for each candidate room:}
  - ID: {room_id}
    Name: {name}
    Description snippet: {first_50_chars}
    Zone: {zone}
{end for}

Question: Is this room one of the known rooms above, or is it a new room?

Consider:
- Exact name match isn't enough (many rooms share names)
- Look for distinctive phrases in the description
- Consider the zone context
- Maze rooms might be identical but separate instances

Respond ONLY with valid JSON:
{
  "is_known_room": boolean,
  "matched_room_id": "room_id or null",
  "confidence": "high|medium|low",
  "reasoning": "why you think this is/isn't a known room",
  "create_new_instance": boolean
}

If create_new_instance is true, this is a duplicate room (maze) that needs a new ID.
```

### Context Enrichment Prompt

```
You are enriching metadata for a room in a MUD game to help with visual image generation.

Room information:
Name: {room_name}
Description: {room_description}
Zone: {current_zone}
Current atmosphere: {atmosphere}

Provide additional metadata that would help generate an accurate visual image:

- Time of day hints (if any)
- Weather (if mentioned)
- Lighting (bright, dim, dark, etc.)
- Architectural style (if indoor)
- Natural features (if outdoor)
- Mood/tone for image generation
- Any distinctive visual elements

Respond ONLY with valid JSON:
{
  "time_of_day": "day|night|dawn|dusk|unclear",
  "weather": "clear|rain|snow|fog|storm|unclear",
  "lighting": "bright|dim|dark|flickering|natural",
  "style": "medieval|rustic|elegant|ruined|natural|unclear",
  "visual_features": ["list", "of", "distinctive", "visual", "elements"],
  "mood": "descriptive mood for image generation",
  "image_prompt_hints": "additional context for stable diffusion"
}
```

## Adapter Interface

```go
package adapters

import "time"

// MUDAdapter is the interface that all MUD-specific adapters implement
type MUDAdapter interface {
    // Metadata
    Name() string
    Version() string

    // Connection
    Connect(host, port string) error
    Disconnect() error
    IsConnected() bool

    // Core parsing
    ParseLine(line string) []protocol.Message

    // AI-assisted classification
    ClassifyContext(history []string) (*ClassifiedContext, error)
    ClassifyRoom(name, description string, exits []string) (*protocol.RoomEntry, error)

    // Knowledge management
    UpdateKnowledge(room *protocol.RoomEntry, wasCorrect bool)
    SaveKnowledge(path string) error
    LoadKnowledge(path string) error

    // State management
    GetContext() *ClassifiedContext
    ResetContext()
    GetHistory() *HistoryBuffer

    // User feedback
    CorrectZone(oldZone, newZone string)
    CorrectRoom(roomID string, corrections map[string]string)
    LearnTrigger(phrase, zone string)
}
```

## Example Implementation Flow

### 1. Fast Path (Pattern Matching)

```go
func (a *AstariaAdapter) ParseLine(line string) []protocol.Message {
    // Add to history
    a.History.Add(line)

    // Try known patterns first (fast)
    if room := a.tryKnownPattern(line); room != nil {
        return []protocol.Message{room}
    }

    // Check if it's a room title line
    if a.isRoomTitle(line) {
        return a.handleRoomEntry(line)
    }

    // Check for exits
    if exits := a.parseExits(line); exits != nil {
        return []protocol.Message{exits}
    }

    // Check for entities
    if entities := a.parseEntities(line); entities != nil {
        return []protocol.Message{entities}
    }

    // Not a special line, just narrative text
    return nil
}
```

### 2. Room Entry with AI Classification

```go
func (a *AstariaAdapter) handleRoomEntry(titleLine string) []protocol.Message {
    // Wait for description (next few lines)
    description := a.waitForDescription()
    exits := a.waitForExits()

    roomName := a.extractRoomName(titleLine)

    // Check knowledge base
    if pattern := a.KB.Match(roomName, description); pattern != nil {
        if pattern.Confidence > 0.9 {
            // High confidence - use pattern
            return a.buildRoomMessage(pattern, exits)
        }
    }

    // Check if zone transition might have happened
    if a.mightBeZoneTransition(a.History.Last(5)) {
        // Call AI to classify context
        ctx, _ := a.ClassifyContext(a.History.Last(20))

        if ctx.CurrentZone != a.Context.CurrentZone {
            // Zone changed!
            a.Context = ctx

            // Learn this trigger for future
            a.KB.LearnZoneTrigger(ctx.CurrentZone, a.History.Last(3))
        }
    }

    // Check if this is a known room or new
    roomID := a.identifyRoom(roomName, description, exits)

    // Build protocol message with enriched context
    return a.buildRoomMessage(roomID, roomName, description, exits, a.Context)
}
```

### 3. Learning from Feedback

```go
func (a *AstariaAdapter) UpdateKnowledge(room *protocol.RoomEntry, wasCorrect bool) {
    if wasCorrect {
        // Increase confidence
        pattern := a.KB.GetPattern(room.RoomID)
        if pattern != nil {
            pattern.Confidence = min(1.0, pattern.Confidence + 0.1)
        }
    } else {
        // Decrease confidence
        pattern := a.KB.GetPattern(room.RoomID)
        if pattern != nil {
            pattern.Confidence = max(0.0, pattern.Confidence - 0.2)
        }
    }

    // Save knowledge base
    a.KB.Save()
}
```

## Performance Considerations

### Caching Strategy

```go
type ClassificationCache struct {
    entries map[string]CachedEntry
    ttl     time.Duration
    mu      sync.RWMutex
}

type CachedEntry struct {
    Context   ClassifiedContext
    Timestamp time.Time
}

func (c *ClassificationCache) Get(key string) (*ClassifiedContext, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, exists := c.entries[key]
    if !exists || time.Since(entry.Timestamp) > c.ttl {
        return nil, false
    }

    return &entry.Context, true
}
```

### Rate Limiting

```go
type RateLimiter struct {
    callsPerMinute int
    calls          []time.Time
    mu             sync.Mutex
}

func (r *RateLimiter) CanCall() bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-time.Minute)

    // Remove old calls
    r.calls = filterAfter(r.calls, cutoff)

    if len(r.calls) >= r.callsPerMinute {
        return false
    }

    r.calls = append(r.calls, now)
    return true
}
```

### Batch Processing

For multiple rooms in quick succession, batch the AI calls:

```go
func (a *Adapter) ProcessBatch(rooms []RoomData) error {
    if len(rooms) == 0 {
        return nil
    }

    // Single AI call for all rooms
    classifications := a.ai.ClassifyBatch(rooms, a.Context)

    for i, room := range rooms {
        room.Context = classifications[i]
        a.emit(room)
    }

    return nil
}
```

## Testing Strategies

### Unit Tests

```go
func TestZoneTransitionDetection(t *testing.T) {
    adapter := NewAstariaAdapter()

    // Simulate zone transition
    adapter.History.Add("You leave the city behind.")
    adapter.History.Add("The wilderness stretches before you.")

    if !adapter.mightBeZoneTransition(adapter.History.Last(2)) {
        t.Error("Should detect zone transition")
    }
}
```

### Integration Tests

```go
func TestRoomParsing(t *testing.T) {
    adapter := NewAstariaAdapter()

    // Feed real MUD output
    lines := loadTestData("astaria_city_market.txt")

    var messages []protocol.Message
    for _, line := range lines {
        messages = append(messages, adapter.ParseLine(line)...)
    }

    // Verify protocol messages
    assertHasRoomEntry(t, messages)
    assertHasCorrectExits(t, messages)
}
```

### Capture & Replay

```go
func TestWithCapturedSession(t *testing.T) {
    // Capture real session output
    session := loadSession("astaria_session_2024_01_15.log")

    adapter := NewAstariaAdapter()

    for _, line := range session.Lines {
        messages := adapter.ParseLine(line)
        // Verify against known good results
        verifyMessages(t, messages, session.Expected[line])
    }
}
```

## MUD-Specific Considerations

### WolfMUD Specifics

- Room titles in brackets: `[Market Square]`
- Exits format: `You see exits: north, south, east`
- Entity format: `You see a baker here.`
- Combat format: (varies)

### Astaria Specifics

- Room titles: (research needed)
- Exits format: (research needed)
- Zone transitions: (research needed)
- Special commands: (research needed)

Each adapter needs its own documentation in its subdirectory:
- `internal/adapters/wolfmud/README.md`
- `internal/adapters/astaria/README.md`

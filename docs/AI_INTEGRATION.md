# AI Integration Strategy

## Problem Statement

MUDs output unstructured narrative text, not structured data. We need AI to:

1. **Classify zones and atmosphere** from descriptive text
2. **Infer room identity** when descriptions are ambiguous
3. **Track context** across multiple rooms
4. **Learn patterns** to reduce future AI dependency
5. **Enrich metadata** for better image generation

## Core Challenge

AI is expensive (time + money). We must be **strategic** about when and how we use it.

**Goal:** Maximum intelligence with minimum AI calls.

## AI Model Selection

### Primary: Claude API (Anthropic)

**Pros:**
- Long context window (100K+ tokens) - excellent for history analysis
- Strong reasoning capabilities
- Native JSON mode
- Good at following complex instructions
- Fast response times

**Cons:**
- API costs (~$0.01 per 1K tokens)
- Rate limits
- Requires API key
- Internet dependency

**Use Cases:**
- Zone classification
- Room disambiguation
- Context enrichment
- Complex pattern learning

### Secondary: GPT-4 (OpenAI)

**Pros:**
- Very strong reasoning
- Good JSON support
- Well-documented API

**Cons:**
- More expensive than Claude
- Shorter context window than Claude
- Rate limits

**Use Cases:**
- Fallback if Claude unavailable
- Specific tasks that GPT-4 excels at

### Tertiary: Local LLM (Ollama)

**Pros:**
- No API costs
- No rate limits
- Works offline
- Privacy (no data leaves machine)

**Cons:**
- Slower (much slower on CPU)
- Lower quality reasoning
- Requires local setup
- Resource intensive

**Use Cases:**
- Offline mode
- Privacy-conscious users
- Development/testing
- Simple classification tasks

### Recommendation

**Default Stack:**
1. Claude API (primary)
2. Ollama (fallback for offline)
3. Pattern matching (no AI when confident)

**Configuration:**
```go
type AIConfig struct {
    PrimaryProvider   string  // "claude", "gpt4", "ollama"
    FallbackProvider  string  // "ollama", "none"
    ClaudeAPIKey      string
    GPT4APIKey        string
    OllamaEndpoint    string
    MaxCallsPerMinute int     // Rate limiting
    CacheTTL          int     // Seconds to cache results
}
```

## Classification Workflow

### Decision Tree

```
New input line received
    ↓
Does it match a known pattern? (Knowledge Base)
    ↓ YES → Use pattern (FAST PATH) ✓
    ↓ NO
Is context < 30 seconds old? (Cache)
    ↓ YES → Reuse cached context ✓
    ↓ NO
Is this a zone transition trigger?
    ↓ NO → Continue with current context
    ↓ YES
        ↓
    Check rate limit
        ↓ EXCEEDED → Use fallback/degraded mode
        ↓ OK
            ↓
        Call AI (Claude)
            ↓
        Cache result (30s TTL)
            ↓
        Update knowledge base
            ↓
        Emit protocol message ✓
```

### Fast Path (No AI)

**High-confidence pattern match:**

```go
func (a *Adapter) tryFastPath(roomName, description string) *Room {
    // Check knowledge base
    pattern := a.KB.FindPattern(roomName, description)

    if pattern != nil && pattern.Confidence > 0.9 {
        // High confidence - use immediately
        log.Printf("[Fast Path] Matched pattern: %s (confidence: %.2f)",
            pattern.RoomID, pattern.Confidence)

        return &Room{
            RoomID: pattern.RoomID,
            Zone: pattern.Zone,
            // ... other fields from pattern
        }
    }

    return nil
}
```

**When to use fast path:**
- Exact room name + description match
- Room visited multiple times before
- Pattern confidence > 90%

### Cached Context (No AI)

**Reuse recent classification:**

```go
func (a *Adapter) getCachedContext() *ClassifiedContext {
    if a.Context == nil {
        return nil
    }

    age := time.Since(a.Context.LastClassified)

    if age < 30*time.Second {
        log.Printf("[Cache Hit] Reusing context (age: %v)", age)
        return a.Context
    }

    log.Printf("[Cache Miss] Context too old: %v", age)
    return nil
}
```

**When to use cache:**
- Moving within same zone (< 30 seconds)
- Consecutive rooms with no zone indicators
- Room descriptions don't suggest transition

### AI Classification (Expensive)

**When all else fails:**

```go
func (a *Adapter) classifyWithAI(history []string) (*ClassifiedContext, error) {
    // Check rate limit
    if !a.rateLimiter.CanCall() {
        return nil, errors.New("rate limit exceeded")
    }

    // Build prompt
    prompt := a.buildZonePrompt(history, a.Context)

    // Call AI
    start := time.Now()
    response, err := a.ai.Classify(prompt)
    duration := time.Since(start)

    log.Printf("[AI Call] Duration: %v, Cost: ~$%.4f",
        duration, estimateCost(prompt, response))

    if err != nil {
        return nil, err
    }

    // Parse response
    ctx := parseAIResponse(response)

    // Cache result
    a.contextCache.Set(ctx, 30*time.Second)

    // Update knowledge base
    a.KB.Learn(ctx, history)

    return ctx, nil
}
```

## Prompt Engineering

### Principle: Clear, Structured, Examples

Good prompts are:
- **Specific** - Exactly what you want
- **Structured** - Clear input/output format
- **Exemplified** - Show examples when helpful
- **Constrained** - Limit possible outputs

### Zone Detection Prompt

```
You are analyzing output from a MUD (text-based role-playing game).
Your task is to determine if the player has transitioned to a new zone.

# Context

Recent output (last 20 lines):
"""
{last_20_lines}
"""

Current beliefs:
- Current zone: {current_zone}
- Atmosphere: {atmosphere}
- Location type: {location_type}

# Task

Analyze the recent output and answer these questions:

1. **Zone Transition**: Has the player moved to a different zone (geographic area)?
   - Look for phrases like "leave the city", "enter the forest", "approach the docks"
   - Look for environmental changes: urban → wilderness, indoor → outdoor
   - Look for description shifts: bustling → quiet, safe → dangerous

2. **New Zone Name**: If the zone changed, what is it?
   - Options: city, wilderness, forest, dungeon, mountain, ocean, desert, ruins, castle
   - Be specific: "city_center", "dark_forest", "ancient_ruins"

3. **Atmosphere**: What's the mood/feeling of the current location?
   - 2-3 descriptive tags: "dark, dangerous", "peaceful, rural", "busy, commercial"

4. **Location Type**: What kind of place is this?
   - Options: street, shop, tavern, temple, dungeon_room, wilderness, indoor, outdoor, cave

5. **Confidence**: How certain are you?
   - high: Very clear indicators, multiple confirming signals
   - medium: Reasonable inference, some ambiguity
   - low: Guessing, unclear signals

# Response Format

Respond with ONLY valid JSON (no markdown, no explanation):

{
  "zone_changed": boolean,
  "new_zone": "zone_name or null if unchanged",
  "atmosphere": "comma, separated, tags",
  "location_type": "type from list above",
  "context_notes": "brief note about what triggered classification",
  "confidence": "high|medium|low",
  "reasoning": "one sentence explaining your decision"
}

# Examples

Example 1 - Zone Change:
Input: "You pass through the city gates. The sounds of the marketplace fade behind you. Ahead, the forest stretches endlessly."
Output: {"zone_changed": true, "new_zone": "forest", "atmosphere": "wild, natural, quiet", "location_type": "wilderness", "context_notes": "player left city and entered forest", "confidence": "high", "reasoning": "clear transition through gates, explicit mention of forest"}

Example 2 - No Change:
Input: "You walk down another street. Buildings line both sides. A few citizens hurry past."
Output: {"zone_changed": false, "new_zone": null, "atmosphere": "urban, busy", "location_type": "street", "context_notes": "still in city", "confidence": "high", "reasoning": "continued urban environment, no transition indicators"}

Now analyze the provided output.
```

### Room Identity Prompt

```
You are helping identify whether a room in a MUD is new or a return to a previously visited room.

# Current Room

Name: {room_name}
Description:
"""
{room_description}
"""
Exits: {comma_separated_exits}

# Candidate Known Rooms

{for each candidate room:}
## Candidate {index}: {room_id}
- Name: {name}
- Description snippet: {first_100_chars}...
- Zone: {zone}
- Previously seen: {visit_count} times

{end for}

# Task

Determine if the current room matches one of the known rooms, or if it's a new room.

**Important Considerations:**
- Rooms in MUDs often share names (e.g., "Dark Corridor" might appear 50 times in a maze)
- Look for distinctive phrases in the description
- Consider the context (zone, recent rooms visited)
- Maze rooms are intentionally identical - if you can't distinguish, create a new instance

# Response Format

Respond with ONLY valid JSON:

{
  "is_known_room": boolean,
  "matched_room_id": "room_id or null if new/ambiguous",
  "confidence": "high|medium|low",
  "reasoning": "brief explanation of your decision",
  "create_new_instance": boolean
}

**Field Explanations:**
- `is_known_room`: true if this matches a candidate
- `matched_room_id`: which candidate (or null)
- `create_new_instance`: true if this is a maze room (identical but separate location)

# Examples

Example 1 - Clear Match:
Name: "Market Square"
Description: "This bustling square is filled with vendors hawking their wares..."
Candidate 1: Market Square - "This bustling square is filled with vendors..."

Output: {"is_known_room": true, "matched_room_id": "astaria_city_market_square_01", "confidence": "high", "reasoning": "exact name and description match", "create_new_instance": false}

Example 2 - Maze Room:
Name: "Dark Corridor"
Description: "A dark corridor stretches before you."
Candidate 1: Dark Corridor - "A dark corridor stretches before you."

Output: {"is_known_room": false, "matched_room_id": null, "confidence": "low", "reasoning": "identical maze room, cannot distinguish from existing instances", "create_new_instance": true}

Now analyze the provided room.
```

### Context Enrichment Prompt

```
You are enriching metadata for a MUD room to improve visual image generation.

# Room Information

Name: {room_name}
Description:
"""
{room_description}
"""
Zone: {current_zone}
Atmosphere: {atmosphere}

# Task

Extract visual and atmospheric details that would help generate an accurate image with Stable Diffusion.

Look for:
- Time of day hints (morning light, nighttime, dusk, etc.)
- Weather (rain, snow, fog, storm, clear skies)
- Lighting (bright sun, dim torchlight, darkness, flickering flames)
- Architectural style (medieval stone, wooden rustic, elegant marble, ruins)
- Natural features (trees, water, mountains, caves)
- Mood/tone (eerie, peaceful, chaotic, grand)
- Specific visual details (colors, textures, prominent objects)

# Response Format

Respond with ONLY valid JSON:

{
  "time_of_day": "day|night|dawn|dusk|unclear",
  "weather": "clear|rain|snow|fog|storm|unclear",
  "lighting": "bright|dim|dark|flickering|natural|unclear",
  "architectural_style": "medieval|rustic|elegant|ruined|natural|unclear",
  "visual_features": ["list", "of", "specific", "visual", "elements"],
  "mood": "descriptive mood for image generation",
  "color_palette": "dominant colors if mentioned",
  "image_prompt_hints": "additional context for stable diffusion prompt"
}

# Example

Input:
Name: "Temple Courtyard"
Description: "Sunlight streams through gaps in the crumbling stone columns. Moss-covered statues of forgotten gods line the walls. A fountain, long dry, stands in the center surrounded by wildflowers."

Output:
{
  "time_of_day": "day",
  "weather": "clear",
  "lighting": "bright",
  "architectural_style": "ruined",
  "visual_features": ["crumbling columns", "moss-covered statues", "dry fountain", "wildflowers"],
  "mood": "ancient, peaceful, abandoned",
  "color_palette": "stone gray, green moss, colorful wildflowers",
  "image_prompt_hints": "ancient ruined temple courtyard, sunlight streaming through broken columns, moss and wildflowers reclaiming the space, fantasy art"
}

Now analyze the provided room.
```

## Knowledge Base Structure

The knowledge base learns patterns over time to reduce AI dependency.

### Pattern Storage

```go
type KnowledgeBase struct {
    // Room patterns
    RoomPatterns map[string]*RoomPattern

    // Zone triggers
    ZoneTriggers map[string][]TriggerPhrase

    // Confidence tracking
    PatternStats map[string]*PatternStats

    // User corrections
    Corrections []*Correction

    // Last saved
    LastSaved time.Time
}

type RoomPattern struct {
    RoomID       string
    NamePattern  string   // Can be regex
    KeyPhrases   []string // Description must contain these
    Zone         string
    Atmosphere   string
    LocationType string
    Confidence   float64  // 0.0 to 1.0
    SeenCount    int      // How many times we've seen this
    LastSeen     time.Time
}

type TriggerPhrase struct {
    Text       string
    Zone       string
    TriggersTo string   // Zone we're transitioning TO
    Confidence float64
    SeenCount  int
}

type PatternStats struct {
    Attempts  int     // How many times we tried this pattern
    Successes int     // How many were correct
    Failures  int     // How many were wrong
    AvgTime   float64 // Average match time
}

type Correction struct {
    Timestamp   time.Time
    RoomID      string
    FieldName   string  // "zone", "room_id", "atmosphere"
    AIValue     string
    UserValue   string
    Reason      string
}
```

### Learning Algorithm

```go
func (kb *KnowledgeBase) Learn(ctx *ClassifiedContext, history []string) {
    // Extract potential trigger phrases
    for _, line := range history {
        if seemsLikeZoneTrigger(line) {
            kb.addZoneTrigger(line, ctx.CurrentZone)
        }
    }

    // Update pattern confidence
    if pattern := kb.findMatchingPattern(ctx); pattern != nil {
        pattern.SeenCount++
        pattern.LastSeen = time.Now()

        // If AI agrees with pattern, increase confidence
        if aiAgrees(pattern, ctx) {
            pattern.Confidence = min(1.0, pattern.Confidence + 0.05)
        }
    } else {
        // Create new pattern
        kb.addPattern(ctx, history)
    }
}

func (kb *KnowledgeBase) addZoneTrigger(phrase, zone string) {
    // Normalize phrase
    phrase = strings.ToLower(strings.TrimSpace(phrase))

    // Add or update trigger
    trigger := kb.findTrigger(phrase, zone)
    if trigger != nil {
        trigger.SeenCount++
        trigger.Confidence = min(1.0, trigger.Confidence + 0.1)
    } else {
        kb.ZoneTriggers[zone] = append(kb.ZoneTriggers[zone], TriggerPhrase{
            Text:       phrase,
            Zone:       zone,
            Confidence: 0.5, // Start at medium confidence
            SeenCount:  1,
        })
    }

    kb.save()
}
```

### Pattern Matching

```go
func (kb *KnowledgeBase) FindPattern(roomName, description string) *RoomPattern {
    bestMatch := (*RoomPattern)(nil)
    bestScore := 0.0

    for _, pattern := range kb.RoomPatterns {
        score := pattern.Match(roomName, description)

        if score > bestScore && score > 0.7 {
            bestScore = score
            bestMatch = pattern
        }
    }

    return bestMatch
}

func (p *RoomPattern) Match(name, description string) float64 {
    score := 0.0

    // Name match (40% weight)
    if matched, _ := regexp.MatchString(p.NamePattern, name); matched {
        score += 0.4
    }

    // Key phrases (50% weight)
    phraseScore := 0.0
    for _, phrase := range p.KeyPhrases {
        if strings.Contains(strings.ToLower(description), strings.ToLower(phrase)) {
            phraseScore += 1.0
        }
    }
    if len(p.KeyPhrases) > 0 {
        phraseScore /= float64(len(p.KeyPhrases))
    }
    score += phraseScore * 0.5

    // Confidence multiplier (10% weight)
    score += p.Confidence * 0.1

    return score
}
```

## Cost Management

### Strategies

1. **Cache aggressively** - 30 second TTL for context
2. **Use fast path when possible** - Pattern matching first
3. **Batch requests** - Group multiple rooms if possible
4. **Rate limit** - Max 10 calls per minute
5. **Fallback gracefully** - Use degraded mode if limits hit
6. **Learn patterns** - Reduce future AI dependency

### Cost Estimation

```go
func estimateCost(prompt, response string) float64 {
    // Claude pricing (approximate)
    inputTokens := len(prompt) / 4  // Rough estimate
    outputTokens := len(response) / 4

    inputCost := float64(inputTokens) * 0.00001   // $0.01 per 1K tokens
    outputCost := float64(outputTokens) * 0.00003 // $0.03 per 1K tokens

    return inputCost + outputCost
}

func (a *Adapter) logCosts() {
    totalCost := 0.0
    for _, call := range a.aiCallLog {
        totalCost += call.Cost
    }

    log.Printf("[AI Stats] Total calls: %d, Total cost: $%.2f, Avg cost: $%.4f",
        len(a.aiCallLog), totalCost, totalCost/float64(len(a.aiCallLog)))
}
```

### Rate Limiting

```go
type RateLimiter struct {
    maxPerMinute int
    calls        []time.Time
    mu           sync.Mutex
}

func (r *RateLimiter) CanCall() bool {
    r.mu.Lock()
    defer r.mu.Unlock()

    now := time.Now()
    cutoff := now.Add(-time.Minute)

    // Remove old calls
    validCalls := []time.Time{}
    for _, t := range r.calls {
        if t.After(cutoff) {
            validCalls = append(validCalls, t)
        }
    }
    r.calls = validCalls

    // Check limit
    if len(r.calls) >= r.maxPerMinute {
        log.Printf("[Rate Limit] Hit limit: %d calls in last minute",
            len(r.calls))
        return false
    }

    // Record call
    r.calls = append(r.calls, now)
    return true
}
```

### Caching

```go
type ClassificationCache struct {
    entries map[string]*CachedEntry
    ttl     time.Duration
    mu      sync.RWMutex
}

type CachedEntry struct {
    Context   *ClassifiedContext
    Timestamp time.Time
}

func (c *ClassificationCache) Get(key string) (*ClassifiedContext, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()

    entry, exists := c.entries[key]
    if !exists {
        return nil, false
    }

    age := time.Since(entry.Timestamp)
    if age > c.ttl {
        return nil, false
    }

    log.Printf("[Cache Hit] Age: %v, TTL: %v", age, c.ttl)
    return entry.Context, true
}

func (c *ClassificationCache) Set(ctx *ClassifiedContext, ttl time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()

    key := c.makeKey(ctx)
    c.entries[key] = &CachedEntry{
        Context:   ctx,
        Timestamp: time.Now(),
    }

    c.ttl = ttl
}

func (c *ClassificationCache) makeKey(ctx *ClassifiedContext) string {
    return fmt.Sprintf("%s:%s:%s",
        ctx.CurrentZone, ctx.Atmosphere, ctx.LocationType)
}
```

## Feedback Loop

Users can correct AI mistakes, which improves the knowledge base.

### User Commands

```
/zone correct docks → harbor
  → "Actually, we're at the harbor, not the docks"

/room merge this_room that_room
  → "These two room IDs are actually the same room"

/learn "salty air" → zone:docks
  → "When you see 'salty air', that means we're at the docks"

/confidence room_id high
  → "This room pattern is definitely correct"
```

### Processing Corrections

```go
func (a *Adapter) CorrectZone(oldZone, newZone string) {
    // Record correction
    correction := &Correction{
        Timestamp: time.Now(),
        FieldName: "zone",
        AIValue:   oldZone,
        UserValue: newZone,
        Reason:    "user_correction",
    }

    a.KB.Corrections = append(a.KB.Corrections, correction)

    // Update current context
    a.Context.CurrentZone = newZone

    // Re-classify recent rooms with new zone
    for _, roomID := range a.recentRooms {
        room := a.getRoomData(roomID)
        if room.Zone == oldZone {
            room.Zone = newZone
            a.updateRoom(room)
        }
    }

    // Learn from this
    a.KB.Learn(a.Context, a.History.Last(20))

    log.Printf("[User Correction] Zone: %s → %s", oldZone, newZone)
}

func (a *Adapter) LearnTrigger(phrase, zone string) {
    a.KB.addZoneTrigger(phrase, zone)
    log.Printf("[User Taught] Phrase '%s' indicates zone: %s", phrase, zone)
}
```

### Improving Over Time

```go
// After each session, evaluate performance
func (a *Adapter) EvaluateSession() {
    totalClassifications := len(a.sessionLog)
    correctClassifications := 0

    for _, log := range a.sessionLog {
        if log.UserCorrected {
            // User had to correct this
            pattern := a.KB.GetPattern(log.RoomID)
            if pattern != nil {
                pattern.Confidence *= 0.8 // Reduce confidence
            }
        } else {
            // User didn't correct, assume correct
            correctClassifications++
            pattern := a.KB.GetPattern(log.RoomID)
            if pattern != nil {
                pattern.Confidence = min(1.0, pattern.Confidence * 1.1)
            }
        }
    }

    accuracy := float64(correctClassifications) / float64(totalClassifications)
    log.Printf("[Session Stats] Accuracy: %.1f%%, Patterns: %d, AI Calls: %d",
        accuracy*100, len(a.KB.RoomPatterns), a.aiCallCount)
}
```

## Testing Strategies

### Mocking AI Responses

```go
type MockAI struct {
    responses map[string]string
}

func (m *MockAI) Classify(prompt string) (string, error) {
    // Return canned response based on prompt content
    if strings.Contains(prompt, "Market Square") {
        return `{"zone": "city", "atmosphere": "urban, busy"}`, nil
    }

    return `{"zone": "unknown", "confidence": "low"}`, nil
}

func TestZoneClassification(t *testing.T) {
    adapter := NewAdapter(MockAI{})

    history := []string{
        "You leave the city behind.",
        "The wilderness stretches before you.",
    }

    ctx, err := adapter.ClassifyContext(history)
    assert.NoError(t, err)
    assert.Equal(t, "wilderness", ctx.CurrentZone)
}
```

### Capture & Replay

```go
// Capture real AI calls during dev
func (a *Adapter) captureAICall(prompt, response string) {
    capture := AICapture{
        Timestamp: time.Now(),
        Prompt:    prompt,
        Response:  response,
        Cost:      estimateCost(prompt, response),
    }

    // Save to file
    f, _ := os.OpenFile("ai_captures.jsonl", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    json.NewEncoder(f).Encode(capture)
    f.Close()
}

// Replay for testing
func TestWithCapturedAI(t *testing.T) {
    captures := loadCaptures("ai_captures.jsonl")

    for _, capture := range captures {
        // Test adapter with captured prompt/response
        result := processAIResponse(capture.Response)
        // Assert expectations
    }
}
```

## Configuration

### User-Facing Settings

```yaml
ai:
  provider: "claude"  # claude, gpt4, ollama
  fallback: "ollama"  # fallback provider

  # API keys
  claude_api_key: "sk-..."
  gpt4_api_key: "sk-..."

  # Rate limiting
  max_calls_per_minute: 10
  cache_ttl_seconds: 30

  # Behavior
  enable_learning: true
  enable_corrections: true
  auto_classify_zones: true
  confidence_threshold: 0.7  # Require this confidence before using pattern

  # Cost management
  warn_at_cost: 1.00  # Warn user at $1
  stop_at_cost: 5.00  # Stop AI calls at $5

ollama:
  endpoint: "http://localhost:11434"
  model: "llama2"
```

### Developer Settings

```go
type AIConfig struct {
    Debug          bool  // Log all AI calls
    CaptureMode    bool  // Save calls for replay
    MockResponses  bool  // Use mock AI
    ForceProvider  string // Force specific provider
    DisableCache   bool  // Disable caching (for testing)
}
```

## Monitoring & Observability

```go
type AIMetrics struct {
    TotalCalls       int
    SuccessfulCalls  int
    FailedCalls      int
    CacheHits        int
    CacheMisses      int
    AvgLatency       time.Duration
    TotalCost        float64
    CallsThisSession int
}

func (a *Adapter) logMetrics() {
    log.Printf(`
[AI Metrics]
  Total Calls: %d
  Cache Hit Rate: %.1f%%
  Avg Latency: %v
  Total Cost: $%.2f
  Fast Path Usage: %.1f%%
`,
        a.metrics.TotalCalls,
        float64(a.metrics.CacheHits)/float64(a.metrics.TotalCalls)*100,
        a.metrics.AvgLatency,
        a.metrics.TotalCost,
        float64(a.fastPathHits)/float64(a.metrics.TotalCalls)*100,
    )
}
```

## Best Practices

1. **Always try fast path first** - Don't call AI if you don't need to
2. **Cache everything** - 30 seconds is enough for most zones
3. **Learn continuously** - Update knowledge base after every session
4. **Monitor costs** - Log every AI call and track spending
5. **Fail gracefully** - If AI fails, fall back to pattern matching
6. **Trust confidence scores** - Low confidence = prompt user to verify
7. **Batch when possible** - Group related classifications
8. **Test with captures** - Use real AI responses for testing
9. **Let users correct** - Feedback improves the system
10. **Document patterns** - Keep notes on what works per MUD

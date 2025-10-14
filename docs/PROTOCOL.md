# SeeMUD Protocol Specification

## Overview

The SeeMUD Protocol is a JSON-based message format that MUD adapters emit and the SeeMUD client consumes. It's completely MUD-agnostic, meaning the client has no knowledge of specific MUD implementations.

**Key Principle:** The protocol carries **structured, classified information** about what's happening in the game world, not raw MUD output.

## Message Format

All messages follow this structure:

```json
{
  "type": "message_type",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    // Message-specific data
  }
}
```

The `timestamp` is added by the adapter and helps with ordering and debugging.

## Message Types

### 1. Room Entry

Sent when the player enters a new room or returns to a known room.

```json
{
  "type": "room_entry",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "room_id": "astaria_city_market_square_01",
    "name": "Market Square",
    "description": "This bustling square is filled with vendors hawking their wares. Colorful stalls line the cobblestone plaza, and the smell of fresh bread mingles with exotic spices. To the north, the city's main temple rises above the crowd.",
    "exits": ["north", "south", "east", "west"],
    "metadata": {
      "zone": "city",
      "atmosphere": "urban, bustling, commercial",
      "location_type": "outdoor",
      "terrain": "urban",
      "danger_level": "safe",
      "time_of_day": "day",
      "lighting": "bright",
      "weather": "clear"
    },
    "confidence": "high",
    "is_new_room": true
  }
}
```

**Field Descriptions:**

- `room_id`: Stable, unique identifier for this room. Format: `{mud}_{zone}_{normalized_name}_{instance}`
- `name`: Display name of the room
- `description`: Full room description (cleaned, no ANSI codes)
- `exits`: Array of direction names (normalized: "n", "s", "e", "w", "ne", etc.)
- `metadata`: Enriched contextual information (see Metadata section)
- `confidence`: "high", "medium", or "low" - adapter's confidence in classification
- `is_new_room`: Boolean - first time visiting this room

### 2. Room Update

Sent when information about the current room changes (entities appear/disappear, etc.)

```json
{
  "type": "room_update",
  "timestamp": "2024-01-15T10:30:05Z",
  "data": {
    "room_id": "astaria_city_market_square_01",
    "changes": {
      "entities_added": ["city guard"],
      "entities_removed": ["beggar"],
      "description_changed": false
    }
  }
}
```

### 3. Entity Update

Sent when items or NPCs are detected in the room.

```json
{
  "type": "entities",
  "timestamp": "2024-01-15T10:30:01Z",
  "data": {
    "items": [
      {
        "name": "rusty sword",
        "description": "A worn blade covered in rust",
        "type": "weapon",
        "takeable": true
      },
      {
        "name": "wooden crate",
        "description": "A sturdy crate bound with iron",
        "type": "container",
        "takeable": false
      }
    ],
    "npcs": [
      {
        "name": "city guard",
        "description": "A stern-looking guard in chainmail",
        "state": "standing",
        "hostile": false,
        "level": 15
      },
      {
        "name": "beggar",
        "description": "A ragged figure holding out a hand",
        "state": "sitting",
        "hostile": false,
        "level": 1
      }
    ]
  }
}
```

**Note:** Adapters should include as much detail as they can extract, but many fields may be null/missing.

### 4. Context Update

Sent when the environmental context changes (zone transition, atmosphere shift).

```json
{
  "type": "context_update",
  "timestamp": "2024-01-15T10:31:00Z",
  "data": {
    "zone": "docks",
    "zone_changed": true,
    "previous_zone": "city",
    "atmosphere": "maritime, industrial, busy",
    "notes": "Player has entered the dockside trading area. Salty air and sounds of ships creaking.",
    "confidence": "high"
  }
}
```

### 5. System Event

Sent when significant game events occur.

```json
{
  "type": "system_event",
  "timestamp": "2024-01-15T10:32:00Z",
  "data": {
    "event": "combat_start",
    "details": {
      "opponent": "city guard",
      "initiated_by": "player",
      "location": "astaria_city_market_square_01"
    }
  }
}
```

**Common Event Types:**
- `combat_start`
- `combat_end`
- `death`
- `level_up`
- `quest_update`
- `trade_initiated`
- `spell_cast`

### 6. Narrative Text

Sent for text that doesn't fit other categories (general narrative, NPC speech, etc.)

```json
{
  "type": "narrative",
  "timestamp": "2024-01-15T10:30:02Z",
  "data": {
    "text": "The baker looks up from his work and nods at you.",
    "category": "npc_action",
    "speaker": "baker"
  }
}
```

**Categories:**
- `npc_action` - NPC does something
- `npc_speech` - NPC says something
- `environment` - Environmental description
- `player_action` - Result of player action
- `system` - System message
- `other` - Uncategorized

### 7. Command Result

Sent as response to player commands (inventory, stats, etc.)

```json
{
  "type": "command_result",
  "timestamp": "2024-01-15T10:33:00Z",
  "data": {
    "command": "inventory",
    "success": true,
    "output": "You are carrying: rusty sword, health potion (x3), 150 gold coins"
  }
}
```

### 8. Error

Sent when the adapter encounters an error or ambiguous situation.

```json
{
  "type": "error",
  "timestamp": "2024-01-15T10:34:00Z",
  "data": {
    "level": "warning",
    "message": "Could not determine room identity with high confidence",
    "context": {
      "room_name": "Dark Corridor",
      "possible_matches": 3,
      "action_taken": "created_new_instance"
    }
  }
}
```

**Error Levels:**
- `info` - Informational, no action needed
- `warning` - Something uncertain happened, adapter made best guess
- `error` - Parsing failed, data might be incorrect
- `critical` - Adapter is in bad state, may need reset

## Room ID Generation

Room IDs must be **stable** (same room always gets same ID) and **unique** (different rooms get different IDs).

### Format

```
{mud_name}_{zone}_{room_name_normalized}_{instance_number}
```

**Examples:**
- `astaria_city_market_square_01`
- `astaria_docks_warehouse_01`
- `astaria_dungeon_dark_corridor_03` (3rd instance in a maze)
- `wolfmud_tavern_dragons_breath_01`

### Normalization Rules

1. Convert to lowercase
2. Remove articles: "the", "a", "an"
3. Replace spaces with underscores
4. Remove special characters except underscores
5. Limit to 50 characters

```go
func NormalizeRoomName(name string) string {
    // "The Dark Corridor" → "dark_corridor"
    name = strings.ToLower(name)
    name = removeArticles(name)
    name = regexp.MustCompile(`[^a-z0-9_]`).ReplaceAllString(name, "_")
    name = regexp.MustCompile(`_+`).ReplaceAllString(name, "_")
    name = strings.Trim(name, "_")

    if len(name) > 50 {
        name = name[:50]
    }

    return name
}
```

### Instance Numbers

For maze rooms or multiple rooms with the same name:

```
dark_corridor_01  # First instance
dark_corridor_02  # Second instance
dark_corridor_03  # Third instance
```

The adapter decides when to create new instances vs. recognizing a return to a known room (see ADAPTER_DESIGN.md for details).

## Metadata Fields

Metadata enriches room information for better mapping and image generation.

### Zone

Broad area classification. Examples:
- `city` - Urban settlement
- `wilderness` - Outdoor natural area
- `dungeon` - Underground complex
- `ocean` - Water/maritime
- `mountain` - Mountainous terrain
- `forest` - Forested area
- `desert` - Arid wasteland
- `castle` - Fortified structure
- `ruins` - Abandoned/destroyed location

### Atmosphere

Comma-separated descriptive tags for mood/environment:
- `urban, bustling, commercial` - Busy city marketplace
- `dark, dangerous, foreboding` - Scary dungeon
- `peaceful, rural, quiet` - Calm countryside
- `maritime, industrial, busy` - Active docks
- `mystical, ancient, sacred` - Temple or shrine

### Location Type

Specific type of place:
- `street` - Outdoor city path
- `shop` - Commercial establishment
- `tavern` - Drinking establishment
- `temple` - Religious building
- `dungeon_room` - Underground chamber
- `wilderness` - Natural outdoor area
- `indoor` - Generic indoor space
- `outdoor` - Generic outdoor space
- `cave` - Natural underground
- `bridge` - Crossing structure

### Terrain

Physical environment:
- `urban` - City/town
- `indoor` - Inside a building
- `outdoor` - Outside
- `underground` - Below ground
- `water` - On/in water
- `forest` - Wooded area
- `mountain` - High altitude
- `desert` - Arid terrain

### Danger Level

Subjective risk assessment:
- `safe` - No danger expected
- `low` - Minimal threat
- `medium` - Moderate danger
- `high` - Significant threat
- `extreme` - Very dangerous

### Visual Hints (for Image Generation)

Additional fields to guide Stable Diffusion:

```json
{
  "time_of_day": "day|night|dawn|dusk|unclear",
  "weather": "clear|rain|snow|fog|storm|unclear",
  "lighting": "bright|dim|dark|flickering|natural",
  "architectural_style": "medieval|rustic|elegant|ruined|natural",
  "visual_features": ["stone walls", "flickering torches", "vaulted ceiling"],
  "mood": "tense and foreboding",
  "image_prompt_hints": "dark stone dungeon corridor, lit by flickering torches, medieval fantasy"
}
```

## Confidence Levels

Adapters indicate how certain they are about their classification:

### High Confidence
- Exact room name match with known pattern
- Clear structural markers in text
- Zone confirmed by multiple indicators
- Recent AI classification validated by subsequent output

### Medium Confidence
- Room name similar to known pattern
- Zone inferred from context
- AI classification with some ambiguity
- Missing some expected information

### Low Confidence
- Ambiguous text
- Multiple possible interpretations
- Guessing based on limited information
- Pattern match but weak signals

**Client Behavior:**
- **High:** Trust the data, use for mapping and image generation
- **Medium:** Use data but allow user corrections
- **Low:** Mark as uncertain, prompt user to verify or flag for review

## Protocol Versioning

The protocol includes a version number for future compatibility:

```json
{
  "protocol_version": "1.0",
  "type": "room_entry",
  ...
}
```

**Current Version:** 1.0

## Transport Layer

The protocol is transport-agnostic. Current implementations:

### WebSocket (Preferred)
```
Adapter → WebSocket Server → Client
```

Messages sent as JSON strings over WebSocket.

### Local IPC (Alternative)
```
Adapter → Local Socket/Pipe → Client
```

For embedded adapters running in the same process.

### File-Based (Development/Testing)
```
Adapter → JSON Lines File → Client Replay
```

For capturing sessions and testing.

## Example Full Exchange

```json
// Player enters new room
{
  "type": "room_entry",
  "timestamp": "2024-01-15T10:30:00Z",
  "data": {
    "room_id": "astaria_city_market_square_01",
    "name": "Market Square",
    "description": "This bustling square is filled with vendors...",
    "exits": ["north", "south", "east", "west"],
    "metadata": {
      "zone": "city",
      "atmosphere": "urban, bustling",
      "location_type": "outdoor",
      "danger_level": "safe"
    },
    "confidence": "high",
    "is_new_room": true
  }
}

// Entities detected
{
  "type": "entities",
  "timestamp": "2024-01-15T10:30:01Z",
  "data": {
    "items": [
      {"name": "rusty sword", "type": "weapon"}
    ],
    "npcs": [
      {"name": "city guard", "hostile": false}
    ]
  }
}

// NPC action
{
  "type": "narrative",
  "timestamp": "2024-01-15T10:30:02Z",
  "data": {
    "text": "The city guard nods at you.",
    "category": "npc_action",
    "speaker": "city guard"
  }
}

// Player moves north
{
  "type": "room_entry",
  "timestamp": "2024-01-15T10:30:15Z",
  "data": {
    "room_id": "astaria_city_temple_entrance_01",
    "name": "Temple Entrance",
    "description": "Massive stone steps lead up to ornate doors...",
    "exits": ["south", "east", "west", "up"],
    "metadata": {
      "zone": "city",
      "atmosphere": "sacred, peaceful, grand",
      "location_type": "temple",
      "danger_level": "safe"
    },
    "confidence": "high",
    "is_new_room": true
  }
}
```

## Client Implementation Notes

### Handling Messages

```javascript
// Client receives protocol message
function handleMessage(message) {
  switch (message.type) {
    case 'room_entry':
      updateMap(message.data);
      generateRoomImage(message.data);
      displayRoom(message.data);
      break;

    case 'entities':
      updateEntityList(message.data);
      break;

    case 'context_update':
      updateZoneDisplay(message.data);
      break;

    case 'narrative':
      appendToTextOutput(message.data.text);
      break;

    // ... etc
  }
}
```

### Confidence Handling

```javascript
function updateMap(roomData) {
  if (roomData.confidence === 'low') {
    // Mark room as uncertain
    room.uncertain = true;
    // Show warning icon in UI
    showUncertaintyWarning(roomData.room_id);
  } else {
    // Trust the data
    mapper.addRoom(roomData);
  }
}
```

## Adapter Implementation Notes

### Emitting Messages

```go
func (a *Adapter) emitRoomEntry(room *Room) {
  message := protocol.Message{
    Type: "room_entry",
    Timestamp: time.Now(),
    Data: room,
  }

  json.NewEncoder(a.output).Encode(message)
}
```

### Batching

For performance, adapters can batch related messages:

```go
func (a *Adapter) emitBatch(messages []protocol.Message) {
  batch := protocol.Batch{
    Messages: messages,
  }

  json.NewEncoder(a.output).Encode(batch)
}
```

## Future Extensions

Possible future additions to the protocol:

- `combat` message type with detailed combat info
- `quest` message type for quest tracking
- `map_correction` messages (user feedback)
- `pathfinding` requests from client to adapter
- `skill_tree` for character progression
- `economy` for shop/trading data

These would be added in new protocol versions (1.1, 2.0, etc.) with backward compatibility.

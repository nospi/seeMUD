package mapper

import (
	"log"
	"strings"
	"sync"
)

// Mapper handles automatic mapping of the MUD world
type Mapper struct {
	Graph         *RoomGraph
	CurrentRoomID string
	PreviousRoomID string
	LastDirection string // Last movement direction taken
	mutex         sync.RWMutex
}

// NewMapper creates a new mapper instance
func NewMapper() *Mapper {
	return &Mapper{
		Graph: NewRoomGraph(),
	}
}

// DirectionOffsets defines coordinate changes for each direction
var DirectionOffsets = map[string][3]int{
	"n":     {0, 1, 0},
	"north": {0, 1, 0},
	"s":     {0, -1, 0},
	"south": {0, -1, 0},
	"e":     {1, 0, 0},
	"east":  {1, 0, 0},
	"w":     {-1, 0, 0},
	"west":  {-1, 0, 0},
	"ne":    {1, 1, 0},
	"northeast": {1, 1, 0},
	"nw":    {-1, 1, 0},
	"northwest": {-1, 1, 0},
	"se":    {1, -1, 0},
	"southeast": {1, -1, 0},
	"sw":    {-1, -1, 0},
	"southwest": {-1, -1, 0},
	"u":     {0, 0, 1},
	"up":    {0, 0, 1},
	"d":     {0, 0, -1},
	"down":  {0, 0, -1},
}

// OppositeDirection returns the reverse of a direction
var OppositeDirection = map[string]string{
	"n":     "s",
	"north": "south",
	"s":     "n",
	"south": "north",
	"e":     "w",
	"east":  "west",
	"w":     "e",
	"west":  "east",
	"ne":    "sw",
	"northeast": "southwest",
	"nw":    "se",
	"northwest": "southeast",
	"se":    "nw",
	"southeast": "northwest",
	"sw":    "ne",
	"southwest": "northeast",
	"u":     "d",
	"up":    "down",
	"d":     "u",
	"down":  "up",
}

// OnRoomEntered should be called when the player enters a room
func (m *Mapper) OnRoomEntered(name, description string, exits []string) string {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Generate room ID
	roomID := GenerateRoomID(name, description)

	// Check if this room exists
	existingRoom := m.Graph.GetRoom(roomID)

	if existingRoom != nil {
		// Room already mapped, update visit info
		log.Printf("[Mapper] Returned to known room: %s (ID: %s)", name, roomID[:8])
		m.PreviousRoomID = m.CurrentRoomID
		m.CurrentRoomID = roomID
		existingRoom.Visited = existingRoom.Visited // Will be updated by AddRoom
		m.Graph.AddRoom(existingRoom)

		// Link from previous room if we moved
		if m.PreviousRoomID != "" && m.LastDirection != "" {
			m.linkRooms(m.PreviousRoomID, m.LastDirection, roomID)
		}

		m.LastDirection = "" // Reset after use
		return roomID
	}

	// New room - need to calculate coordinates
	x, y, z := 0, 0, 0

	if m.CurrentRoomID != "" && m.LastDirection != "" {
		// Calculate position based on previous room and direction
		prevRoom := m.Graph.GetRoom(m.CurrentRoomID)
		if prevRoom != nil {
			offset, known := DirectionOffsets[strings.ToLower(m.LastDirection)]
			if known {
				x = prevRoom.X + offset[0]
				y = prevRoom.Y + offset[1]
				z = prevRoom.Z + offset[2]
			} else {
				// Unknown direction - place randomly offset
				log.Printf("[Mapper] Unknown direction: %s", m.LastDirection)
				x = prevRoom.X + 1
				y = prevRoom.Y
				z = prevRoom.Z
			}

			// Check for coordinate collision
			if collision := m.Graph.FindRoomAt(x, y, z); collision != nil {
				log.Printf("[Mapper] Coordinate collision at (%d,%d,%d) for new room %s", x, y, z, name)
				// Offset slightly - this needs manual review
				x += 1
			}
		}
	}

	// Create new room
	newRoom := &Room{
		ID:          roomID,
		Name:        name,
		Description: description,
		X:           x,
		Y:           y,
		Z:           z,
		Exits:       make(map[string]string),
	}

	// Add exits (initially unexplored)
	for _, exit := range exits {
		newRoom.Exits[strings.ToLower(exit)] = "" // Empty string = unexplored
	}

	m.Graph.AddRoom(newRoom)
	log.Printf("[Mapper] Mapped new room: %s at (%d,%d,%d) [ID: %s]", name, x, y, z, roomID[:8])

	// Link from previous room if we moved
	if m.PreviousRoomID != "" && m.LastDirection != "" {
		m.linkRooms(m.PreviousRoomID, m.LastDirection, roomID)
	}

	m.PreviousRoomID = m.CurrentRoomID
	m.CurrentRoomID = roomID
	m.LastDirection = "" // Reset after use

	return roomID
}

// OnMovement should be called when the player issues a movement command
func (m *Mapper) OnMovement(direction string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.LastDirection = direction
	log.Printf("[Mapper] Movement command: %s", direction)
}

// linkRooms creates bidirectional links between rooms
func (m *Mapper) linkRooms(fromID, direction, toID string) {
	fromRoom := m.Graph.GetRoom(fromID)
	toRoom := m.Graph.GetRoom(toID)

	if fromRoom == nil || toRoom == nil {
		return
	}

	// Forward link
	normalizedDir := strings.ToLower(direction)
	fromRoom.Exits[normalizedDir] = toID
	m.Graph.AddExit(fromID, normalizedDir, toID)

	// Reverse link
	reverseDir := OppositeDirection[normalizedDir]
	if reverseDir != "" {
		toRoom.Exits[reverseDir] = fromID
		m.Graph.AddExit(toID, reverseDir, fromID)
	}

	log.Printf("[Mapper] Linked rooms: %s -[%s]-> %s", fromRoom.Name, normalizedDir, toRoom.Name)
}

// GetCurrentRoom returns the current room object
func (m *Mapper) GetCurrentRoom() *Room {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.Graph.GetRoom(m.CurrentRoomID)
}

// GetNeighbours returns neighbouring rooms with their directions
func (m *Mapper) GetNeighbours() map[string]*Room {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.CurrentRoomID == "" {
		return nil
	}

	return m.Graph.GetNeighbours(m.CurrentRoomID)
}

// GetGraph returns the room graph (for serialisation)
func (m *Mapper) GetGraph() *RoomGraph {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return m.Graph
}

// SetGraph replaces the current graph (for deserialisation)
func (m *Mapper) SetGraph(graph *RoomGraph) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.Graph = graph
	log.Printf("[Mapper] Loaded graph with %d rooms", len(graph.Rooms))
}

// GetMapStats returns statistics about the mapped area
func (m *Mapper) GetMapStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	minX, maxX, minY, maxY, minZ, maxZ := m.Graph.GetBounds()

	return map[string]interface{}{
		"total_rooms": m.Graph.GetRoomCount(),
		"bounds": map[string]int{
			"min_x": minX,
			"max_x": maxX,
			"min_y": minY,
			"max_y": maxY,
			"min_z": minZ,
			"max_z": maxZ,
		},
		"current_room": m.CurrentRoomID,
	}
}

// IsMovementCommand checks if a command is a movement command
func IsMovementCommand(command string) (bool, string) {
	cmd := strings.ToLower(strings.TrimSpace(command))

	// Check direct direction commands
	if _, exists := DirectionOffsets[cmd]; exists {
		return true, cmd
	}

	// Check "go <direction>" format
	if strings.HasPrefix(cmd, "go ") {
		direction := strings.TrimPrefix(cmd, "go ")
		if _, exists := DirectionOffsets[direction]; exists {
			return true, direction
		}
	}

	return false, ""
}

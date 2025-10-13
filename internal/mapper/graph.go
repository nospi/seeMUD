package mapper

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// Room represents a location in the MUD world
type Room struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	X           int               `json:"x"`
	Y           int               `json:"y"`
	Z           int               `json:"z"`
	Exits       map[string]string `json:"exits"`        // direction -> room ID (nil if unexplored)
	ImagePath   string            `json:"image_path"`   // Path to cached image
	Visited     time.Time         `json:"visited"`      // Last visit time
	VisitCount  int               `json:"visit_count"`  // Number of times visited
	Uncertain   bool              `json:"uncertain"`    // Flag for coordinate uncertainty
	Notes       string            `json:"notes"`        // User notes
}

// Exit represents a directional connection between rooms
type Exit struct {
	From      string `json:"from"`      // Source room ID
	Direction string `json:"direction"` // n, s, e, w, ne, nw, se, sw, u, d, etc.
	To        string `json:"to"`        // Destination room ID (empty if unexplored)
}

// RoomGraph represents the spatial graph of rooms
type RoomGraph struct {
	Rooms map[string]*Room `json:"rooms"` // room ID -> Room
	Exits []*Exit          `json:"exits"` // All exits in the graph
}

// NewRoomGraph creates a new empty room graph
func NewRoomGraph() *RoomGraph {
	return &RoomGraph{
		Rooms: make(map[string]*Room),
		Exits: make([]*Exit, 0),
	}
}

// GenerateRoomID creates a unique identifier for a room based on name and description
func GenerateRoomID(name, description string) string {
	// Use first 100 chars of description to handle dynamic content
	desc := description
	if len(desc) > 100 {
		desc = desc[:100]
	}

	hash := sha256.Sum256([]byte(name + "|" + desc))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter ID
}

// AddRoom adds a room to the graph or updates existing room
func (g *RoomGraph) AddRoom(room *Room) {
	if existing, exists := g.Rooms[room.ID]; exists {
		// Update visit information
		existing.Visited = time.Now()
		existing.VisitCount++

		// Update description if changed (but keep same ID)
		if existing.Description != room.Description {
			existing.Description = room.Description
		}

		// Merge exits (don't overwrite explored exits)
		for dir, roomID := range room.Exits {
			if _, exists := existing.Exits[dir]; !exists {
				existing.Exits[dir] = roomID
			}
		}
	} else {
		// New room
		room.Visited = time.Now()
		room.VisitCount = 1
		g.Rooms[room.ID] = room
	}
}

// GetRoom retrieves a room by ID
func (g *RoomGraph) GetRoom(id string) *Room {
	return g.Rooms[id]
}

// AddExit adds or updates an exit in the graph
func (g *RoomGraph) AddExit(from, direction, to string) {
	// Check if exit already exists
	for _, exit := range g.Exits {
		if exit.From == from && exit.Direction == direction {
			// Update destination
			exit.To = to
			return
		}
	}

	// New exit
	g.Exits = append(g.Exits, &Exit{
		From:      from,
		Direction: direction,
		To:        to,
	})
}

// GetNeighbours returns all neighbouring rooms (1 hop away) from the given room
func (g *RoomGraph) GetNeighbours(roomID string) map[string]*Room {
	room := g.GetRoom(roomID)
	if room == nil {
		return nil
	}

	neighbours := make(map[string]*Room)

	for direction, neighbourID := range room.Exits {
		if neighbourID != "" {
			if neighbour := g.GetRoom(neighbourID); neighbour != nil {
				neighbours[direction] = neighbour
			}
		}
	}

	return neighbours
}

// FindRoomsByName searches for rooms with matching names
func (g *RoomGraph) FindRoomsByName(name string) []*Room {
	var results []*Room

	for _, room := range g.Rooms {
		if room.Name == name {
			results = append(results, room)
		}
	}

	return results
}

// FindRoomAt returns the room at specific coordinates
func (g *RoomGraph) FindRoomAt(x, y, z int) *Room {
	for _, room := range g.Rooms {
		if room.X == x && room.Y == y && room.Z == z {
			return room
		}
	}
	return nil
}

// GetRoomCount returns the total number of mapped rooms
func (g *RoomGraph) GetRoomCount() int {
	return len(g.Rooms)
}

// GetBounds returns the min/max coordinates of the mapped area
func (g *RoomGraph) GetBounds() (minX, maxX, minY, maxY, minZ, maxZ int) {
	if len(g.Rooms) == 0 {
		return 0, 0, 0, 0, 0, 0
	}

	first := true
	for _, room := range g.Rooms {
		if first {
			minX, maxX = room.X, room.X
			minY, maxY = room.Y, room.Y
			minZ, maxZ = room.Z, room.Z
			first = false
		} else {
			if room.X < minX {
				minX = room.X
			}
			if room.X > maxX {
				maxX = room.X
			}
			if room.Y < minY {
				minY = room.Y
			}
			if room.Y > maxY {
				maxY = room.Y
			}
			if room.Z < minZ {
				minZ = room.Z
			}
			if room.Z > maxZ {
				maxZ = room.Z
			}
		}
	}

	return
}

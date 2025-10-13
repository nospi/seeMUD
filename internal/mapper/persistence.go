package mapper

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// MapData represents the serialisable map structure
type MapData struct {
	Version       string     `json:"version"`
	ServerName    string     `json:"server_name"`
	Graph         *RoomGraph `json:"graph"`
	CurrentRoomID string     `json:"current_room_id"`
}

const (
	MapVersion     = "1.0"
	MapCacheDir    = "cache/maps"
	DefaultMapFile = "default.json"
)

// SaveMap saves the current map to disk
func (m *Mapper) SaveMap(serverName string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Ensure cache directory exists
	if err := os.MkdirAll(MapCacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create map cache directory: %w", err)
	}

	// Create map data
	mapData := &MapData{
		Version:       MapVersion,
		ServerName:    serverName,
		Graph:         m.Graph,
		CurrentRoomID: m.CurrentRoomID,
	}

	// Determine filename
	filename := sanitiseFilename(serverName)
	if filename == "" {
		filename = DefaultMapFile
	} else {
		filename = filename + ".json"
	}

	filepath := filepath.Join(MapCacheDir, filename)

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(mapData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal map data: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write map file: %w", err)
	}

	log.Printf("[Mapper] Saved map with %d rooms to %s", m.Graph.GetRoomCount(), filepath)
	return nil
}

// LoadMap loads a map from disk
func (m *Mapper) LoadMap(serverName string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Determine filename
	filename := sanitiseFilename(serverName)
	if filename == "" {
		filename = DefaultMapFile
	} else {
		filename = filename + ".json"
	}

	filepath := filepath.Join(MapCacheDir, filename)

	// Check if file exists
	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		log.Printf("[Mapper] No existing map found at %s", filepath)
		return nil // Not an error, just no map to load
	}

	// Read file
	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read map file: %w", err)
	}

	// Unmarshal JSON
	var mapData MapData
	if err := json.Unmarshal(data, &mapData); err != nil {
		return fmt.Errorf("failed to unmarshal map data: %w", err)
	}

	// Version check
	if mapData.Version != MapVersion {
		log.Printf("[Mapper] Warning: Map version mismatch (file: %s, expected: %s)",
			mapData.Version, MapVersion)
		// Continue anyway - we can handle minor version differences
	}

	// Load graph
	m.Graph = mapData.Graph
	m.CurrentRoomID = mapData.CurrentRoomID

	log.Printf("[Mapper] Loaded map with %d rooms from %s", len(m.Graph.Rooms), filepath)
	return nil
}

// AutoSave saves the map periodically (should be called in a goroutine)
func (m *Mapper) AutoSave(serverName string, intervalSeconds int) {
	// Implement if needed - for now manual save on disconnect
}

// ExportMap exports the map to a specific file path (for sharing)
func (m *Mapper) ExportMap(filepath, serverName string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	mapData := &MapData{
		Version:       MapVersion,
		ServerName:    serverName,
		Graph:         m.Graph,
		CurrentRoomID: m.CurrentRoomID,
	}

	data, err := json.MarshalIndent(mapData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal map data: %w", err)
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write map file: %w", err)
	}

	log.Printf("[Mapper] Exported map to %s", filepath)
	return nil
}

// ImportMap imports a map from a specific file path
func (m *Mapper) ImportMap(filepath string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	data, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("failed to read map file: %w", err)
	}

	var mapData MapData
	if err := json.Unmarshal(data, &mapData); err != nil {
		return fmt.Errorf("failed to unmarshal map data: %w", err)
	}

	// Merge with existing map (don't overwrite)
	if m.Graph == nil {
		m.Graph = NewRoomGraph()
	}

	for id, room := range mapData.Graph.Rooms {
		if _, exists := m.Graph.Rooms[id]; !exists {
			m.Graph.Rooms[id] = room
		}
	}

	for _, exit := range mapData.Graph.Exits {
		m.Graph.AddExit(exit.From, exit.Direction, exit.To)
	}

	log.Printf("[Mapper] Imported map from %s (now %d rooms)", filepath, len(m.Graph.Rooms))
	return nil
}

// sanitiseFilename removes unsafe characters from filename
func sanitiseFilename(name string) string {
	// Simple sanitisation - remove path separators and special chars
	safe := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		   (r >= '0' && r <= '9') || r == '_' || r == '-' {
			safe += string(r)
		}
	}
	return safe
}

package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"seemud-gui/internal/parser"
	"seemud-gui/internal/renderer"
	"seemud-gui/internal/telnet"
)

// App struct
type App struct {
	ctx            context.Context
	mudClient      *telnet.Client
	mudParser      *parser.WolfMUDParser
	sdClient       *renderer.StableDiffusionClient
	outputBuf      []string
	outputMux      sync.RWMutex
	connected      bool
	currentRoom    *parser.ParsedOutput
	roomMux        sync.RWMutex
	roomImageCache map[string]string // Map of room name to image file path
	imageCacheMux  sync.RWMutex
	currentItems   []string // Items in current room
	currentMobs    []string // Mobs/NPCs in current room
	entityMux      sync.RWMutex
}

const defaultSDEndpoint = "http://127.0.0.1:7860"

func resolveSDEndpoint() string {
	if value := strings.TrimSpace(os.Getenv("SEEMUD_SD_ENDPOINT")); value != "" {
		if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
			return value
		}
		return "http://" + value
	}

	return defaultSDEndpoint
}

// NewApp creates a new App application struct
func NewApp() *App {
	sdEndpoint := resolveSDEndpoint()
	log.Printf("Stable Diffusion endpoint: %s", sdEndpoint)

	// Ensure cache directory exists
	cacheDir := filepath.Join("cache", "room_images")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		log.Printf("Warning: Failed to create cache directory: %v", err)
	}

	// Load existing image cache
	imageCache := loadImageCache(cacheDir)

	return &App{
		mudParser:      parser.NewWolfMUDParser(),
		sdClient:       renderer.NewStableDiffusionClient(sdEndpoint),
		outputBuf:      make([]string, 0, 1000), // Buffer last 1000 lines
		roomImageCache: imageCache,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// ConnectToMUD connects to the WolfMUD server
func (a *App) ConnectToMUD(host, port string) error {
	if a.mudClient != nil && a.mudClient.IsConnected() {
		return fmt.Errorf("already connected")
	}

	a.mudClient = telnet.NewClient(host, port)
	err := a.mudClient.Connect()
	if err != nil {
		return err
	}

	a.connected = true

	// Start processing output
	go a.processOutput()

	return nil
}

// DisconnectFromMUD disconnects from the MUD server
func (a *App) DisconnectFromMUD() error {
	if a.mudClient == nil {
		return nil
	}

	a.connected = false
	return a.mudClient.Disconnect()
}

// SendCommand sends a command to the MUD
func (a *App) SendCommand(command string) error {
	if a.mudClient == nil || !a.mudClient.IsConnected() {
		return fmt.Errorf("not connected to MUD")
	}

	return a.mudClient.SendCommand(command)
}

// GetOutput returns new output since last call and clears the buffer
func (a *App) GetOutput() []string {
	a.outputMux.Lock()
	defer a.outputMux.Unlock()

	if len(a.outputBuf) == 0 {
		return []string{}
	}

	// Return current buffer and clear it
	result := make([]string, len(a.outputBuf))
	copy(result, a.outputBuf)
	a.outputBuf = a.outputBuf[:0] // Clear the buffer

	return result
}

// GetConnectionStatus returns whether we're connected to MUD
func (a *App) GetConnectionStatus() bool {
	return a.connected && a.mudClient != nil && a.mudClient.IsConnected()
}

// processOutput handles incoming MUD output
func (a *App) processOutput() {
	if a.mudClient == nil {
		return
	}

	outputChan := a.mudClient.GetOutput()
	for {
		select {
		case <-a.ctx.Done():
			return
		case line, ok := <-outputChan:
			if !ok {
				a.connected = false
				return
			}

			// Parse the line
			parsed := a.mudParser.ParseLine(line)

			// Add to output buffer
			a.outputMux.Lock()
			a.outputBuf = append(a.outputBuf, line)

			// Keep buffer size manageable
			if len(a.outputBuf) > 1000 {
				a.outputBuf = a.outputBuf[1:]
			}
			a.outputMux.Unlock()

			// Log parsed content for debugging
			log.Printf("Parsed: Type=%d, Content=%s", parsed.Type, parsed.CleanText)

			// Trigger image generation for room content
			if parsed.Type == parser.TypeRoomTitle {
				a.roomMux.Lock()
				a.currentRoom = parsed
				a.roomMux.Unlock()

				// Clear entities when entering new room
				a.entityMux.Lock()
				a.currentItems = []string{}
				a.currentMobs = []string{}
				a.entityMux.Unlock()

				log.Printf("Room title detected: %s", parsed.RoomName)
			} else if parsed.Type == parser.TypeRoomDescription {
				a.roomMux.Lock()
				if a.currentRoom != nil && a.currentRoom.Type == parser.TypeRoomTitle {
					// Only add description if we have a valid room title
					a.currentRoom.Content += " " + parsed.Content
					log.Printf("Room description added: %s", parsed.Content)
				}
				a.roomMux.Unlock()
			} else if parsed.Type == parser.TypeInventory && len(parsed.Items) > 0 {
				// Add items to current room inventory
				a.entityMux.Lock()
				a.currentItems = append(a.currentItems, parsed.Items...)
				a.entityMux.Unlock()
				log.Printf("Items detected: %v", parsed.Items)
			} else if parsed.Type == parser.TypeMobs && len(parsed.Mobs) > 0 {
				// Add mobs to current room
				a.entityMux.Lock()
				a.currentMobs = append(a.currentMobs, parsed.Mobs...)
				a.entityMux.Unlock()
				log.Printf("Mobs detected: %v", parsed.Mobs)
			}
		}
	}
}

// GenerateRoomImage generates an image for the current room (uses cache if available)
func (a *App) GenerateRoomImage() (string, error) {
	a.roomMux.RLock()
	currentRoom := a.currentRoom
	a.roomMux.RUnlock()

	if currentRoom == nil || currentRoom.RoomName == "" {
		return "", fmt.Errorf("no room data available")
	}

	// Check cache first
	if base64Image, exists := a.loadImageFromCache(currentRoom.RoomName); exists {
		log.Printf("Returning cached image for room: %s", currentRoom.RoomName)
		return base64Image, nil
	}

	// No cached image, generate new one
	return a.generateNewRoomImage(currentRoom, "")
}

// RegenerateRoomImage forces generation of a new image for the current room
func (a *App) RegenerateRoomImage() (string, error) {
	a.roomMux.RLock()
	currentRoom := a.currentRoom
	a.roomMux.RUnlock()

	if currentRoom == nil || currentRoom.RoomName == "" {
		return "", fmt.Errorf("no room data available")
	}

	// Always generate new image, ignoring cache
	return a.generateNewRoomImage(currentRoom, "")
}

// RegenerateRoomImageWithPrompt regenerates with custom user prompt additions
func (a *App) RegenerateRoomImageWithPrompt(customPrompt string) (string, error) {
	a.roomMux.RLock()
	currentRoom := a.currentRoom
	a.roomMux.RUnlock()

	if currentRoom == nil || currentRoom.RoomName == "" {
		return "", fmt.Errorf("no room data available")
	}

	// Always generate new image with custom prompt, ignoring cache
	return a.generateNewRoomImage(currentRoom, customPrompt)
}

// generateNewRoomImage is a helper that actually generates a new image
func (a *App) generateNewRoomImage(currentRoom *parser.ParsedOutput, customPrompt string) (string, error) {
	// Check if SD is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.sdClient.CheckHealth(ctx); err != nil {
		return "", fmt.Errorf("Stable Diffusion not available: %w", err)
	}

	// Generate new image
	log.Printf("Generating new image for room: %s", currentRoom.RoomName)
	var prompt string
	if customPrompt != "" {
		log.Printf("Using custom prompt additions: %s", customPrompt)
		prompt = renderer.RoomImagePromptWithCustom(currentRoom.RoomName, currentRoom.Content, customPrompt)
	} else {
		prompt = renderer.RoomImagePrompt(currentRoom.RoomName, currentRoom.Content)
	}
	req := &renderer.Txt2ImgRequest{
		Prompt:         prompt,
		NegativePrompt: renderer.GetNegativePrompt(),
		Width:          512,
		Height:         512,
		Steps:          20,
		CFGScale:       7.0,
	}

	ctx, cancel = context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	resp, err := a.sdClient.GenerateImage(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to generate image: %w", err)
	}

	if len(resp.Images) == 0 {
		return "", fmt.Errorf("no images generated")
	}

	base64Image := resp.Images[0]

	// Save to cache (overwrites existing)
	if err := a.saveImageToCache(currentRoom.RoomName, base64Image); err != nil {
		log.Printf("Warning: Failed to save image to cache: %v", err)
		// Don't fail the operation, just warn
	}

	// Return base64 encoded image
	return base64Image, nil
}

// GetCurrentRoom returns the current room information
func (a *App) GetCurrentRoom() map[string]string {
	a.roomMux.RLock()
	defer a.roomMux.RUnlock()

	if a.currentRoom == nil || a.currentRoom.Type != parser.TypeRoomTitle {
		return map[string]string{}
	}

	// Only return room info if we have a valid room title
	return map[string]string{
		"name":        a.currentRoom.RoomName,
		"description": a.currentRoom.Content,
	}
}

// GetCurrentEntities returns items and mobs in the current room
func (a *App) GetCurrentEntities() map[string][]string {
	a.entityMux.RLock()
	defer a.entityMux.RUnlock()

	return map[string][]string{
		"items": a.currentItems,
		"mobs":  a.currentMobs,
	}
}

// CheckSDStatus checks if Stable Diffusion is available
func (a *App) CheckSDStatus() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return a.sdClient.CheckHealth(ctx) == nil
}

// Greet returns a greeting for the given name (keeping for now)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to SeeMUD!", name)
}

// Helper functions for image caching

// sanitizeRoomName converts a room name to a safe filename
func sanitizeRoomName(roomName string) string {
	// Remove or replace characters that aren't safe for filenames
	reg := regexp.MustCompile(`[^a-zA-Z0-9_\-]`)
	sanitized := reg.ReplaceAllString(strings.ToLower(roomName), "_")
	// Remove multiple underscores
	reg = regexp.MustCompile(`_+`)
	sanitized = reg.ReplaceAllString(sanitized, "_")
	// Trim underscores from ends
	sanitized = strings.Trim(sanitized, "_")

	if sanitized == "" {
		sanitized = "unknown_room"
	}

	return sanitized
}

// loadImageCache scans the cache directory and builds the cache map
func loadImageCache(cacheDir string) map[string]string {
	cache := make(map[string]string)

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		log.Printf("Could not read cache directory: %v", err)
		return cache
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".png") {
			// Store the full path in the cache
			cache[entry.Name()] = filepath.Join(cacheDir, entry.Name())
			log.Printf("Loaded cached image: %s", entry.Name())
		}
	}

	return cache
}

// saveImageToCache saves a base64 image to the cache directory
func (a *App) saveImageToCache(roomName string, base64Image string) error {
	sanitized := sanitizeRoomName(roomName)
	filename := sanitized + ".png"
	filepath := filepath.Join("cache", "room_images", filename)

	// Decode base64 image
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		return fmt.Errorf("failed to decode base64 image: %w", err)
	}

	// Write to file
	if err := os.WriteFile(filepath, imageData, 0644); err != nil {
		return fmt.Errorf("failed to save image to cache: %w", err)
	}

	// Update cache map
	a.imageCacheMux.Lock()
	a.roomImageCache[filename] = filepath
	a.imageCacheMux.Unlock()

	log.Printf("Saved image to cache: %s", filepath)
	return nil
}

// loadImageFromCache loads an image from cache if it exists
func (a *App) loadImageFromCache(roomName string) (string, bool) {
	sanitized := sanitizeRoomName(roomName)
	filename := sanitized + ".png"

	a.imageCacheMux.RLock()
	filepath, exists := a.roomImageCache[filename]
	a.imageCacheMux.RUnlock()

	if !exists {
		return "", false
	}

	// Read the file
	imageData, err := os.ReadFile(filepath)
	if err != nil {
		log.Printf("Failed to read cached image %s: %v", filepath, err)
		// Remove from cache if file doesn't exist
		a.imageCacheMux.Lock()
		delete(a.roomImageCache, filename)
		a.imageCacheMux.Unlock()
		return "", false
	}

	// Encode to base64
	base64Image := base64.StdEncoding.EncodeToString(imageData)
	return base64Image, true
}

// GetRoomImage returns a cached image for the current room or empty string if none exists
func (a *App) GetRoomImage() string {
	a.roomMux.RLock()
	currentRoom := a.currentRoom
	a.roomMux.RUnlock()

	if currentRoom == nil || currentRoom.RoomName == "" {
		return ""
	}

	// Try to load from cache
	if base64Image, exists := a.loadImageFromCache(currentRoom.RoomName); exists {
		log.Printf("Returning cached image for room: %s", currentRoom.RoomName)
		return base64Image
	}

	return ""
}

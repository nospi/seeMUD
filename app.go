package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"see-mud-gui/internal/parser"
	"see-mud-gui/internal/renderer"
	"see-mud-gui/internal/telnet"
)

// App struct
type App struct {
	ctx        context.Context
	mudClient  *telnet.Client
	mudParser  *parser.WolfMUDParser
	sdClient   *renderer.StableDiffusionClient
	outputBuf  []string
	outputMux  sync.RWMutex
	connected  bool
	currentRoom *parser.ParsedOutput
	roomMux     sync.RWMutex
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		mudParser: parser.NewWolfMUDParser(),
		sdClient:  renderer.NewStableDiffusionClient("http://127.0.0.1:7860"),
		outputBuf: make([]string, 0, 1000), // Buffer last 1000 lines
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
				log.Printf("Room title detected: %s", parsed.Content)
			} else if parsed.Type == parser.TypeRoomDescription && a.currentRoom != nil {
				a.roomMux.Lock()
				if a.currentRoom != nil {
					// Combine room title and description for image generation
					a.currentRoom.Content += " " + parsed.Content
					log.Printf("Room description added: %s", parsed.Content)
				}
				a.roomMux.Unlock()
			}
		}
	}
}

// GenerateRoomImage generates an image for the current room
func (a *App) GenerateRoomImage() (string, error) {
	a.roomMux.RLock()
	currentRoom := a.currentRoom
	a.roomMux.RUnlock()

	if currentRoom == nil {
		return "", fmt.Errorf("no room data available")
	}

	// Check if SD is available
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.sdClient.CheckHealth(ctx); err != nil {
		return "", fmt.Errorf("Stable Diffusion not available: %w", err)
	}

	// Generate image
	prompt := renderer.RoomImagePrompt(currentRoom.RoomName, currentRoom.Content)
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

	// Return base64 encoded image
	return resp.Images[0], nil
}

// GetCurrentRoom returns the current room information
func (a *App) GetCurrentRoom() map[string]string {
	a.roomMux.RLock()
	defer a.roomMux.RUnlock()

	if a.currentRoom == nil {
		return map[string]string{}
	}

	return map[string]string{
		"name":        a.currentRoom.RoomName,
		"description": a.currentRoom.Content,
	}
}

// CheckSDStatus checks if Stable Diffusion is available
func (a *App) CheckSDStatus() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return a.sdClient.CheckHealth(ctx) == nil
}

// Greet returns a greeting for the given name (keeping for now)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to See-MUD!", name)
}

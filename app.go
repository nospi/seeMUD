package main

import (
	"context"
	"fmt"
	"log"
	"sync"

	"see-mud-gui/internal/parser"
	"see-mud-gui/internal/telnet"
)

// App struct
type App struct {
	ctx        context.Context
	mudClient  *telnet.Client
	mudParser  *parser.WolfMUDParser
	outputBuf  []string
	outputMux  sync.RWMutex
	connected  bool
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{
		mudParser: parser.NewWolfMUDParser(),
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

// GetOutput returns the current output buffer
func (a *App) GetOutput() []string {
	a.outputMux.RLock()
	defer a.outputMux.RUnlock()

	// Return a copy of the buffer
	result := make([]string, len(a.outputBuf))
	copy(result, a.outputBuf)
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

			// TODO: Here we'll add image generation triggers
			if parsed.Type == parser.TypeRoomTitle || parsed.Type == parser.TypeRoomDescription {
				log.Printf("Room content detected: %s", parsed.Content)
				// Future: trigger image generation
			}
		}
	}
}

// Greet returns a greeting for the given name (keeping for now)
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, Welcome to See-MUD!", name)
}

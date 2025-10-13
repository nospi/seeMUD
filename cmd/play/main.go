package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"seemud-gui/internal/parser"
	"seemud-gui/internal/telnet"
)

func main() {
	fmt.Println("🎮 SeeMUD Interactive Client")
	fmt.Println("==============================")
	fmt.Println("Connecting to WolfMUD on localhost:4001...")

	// Create telnet client
	client := telnet.NewClient("localhost", "4001")
	mudParser := parser.NewWolfMUDParser()

	// Connect
	err := client.Connect()
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	fmt.Println("✓ Connected! Creating/logging into account...")
	fmt.Println()
	fmt.Println("Instructions:")
	fmt.Println("1. Press ENTER to create a new account")
	fmt.Println("2. Follow prompts to create character")
	fmt.Println("3. Type 'quit' to exit client")
	fmt.Println("4. Type '/quit' to quit from MUD")
	fmt.Println()
	fmt.Println("----------------------------------------")

	// Track if we're in game
	inGame := false

	// Start output processing
	go func() {
		outputChan := client.GetOutput()
		for line := range outputChan {
			// Skip ANSI control sequences for now
			if strings.Contains(line, "\x1b[") || strings.Contains(line, "[2J") {
				continue
			}

			// Parse the line
			parsed := mudParser.ParseLine(line)

			// Check if we're in game (crude detection)
			if strings.Contains(parsed.CleanText, "Fireplace") ||
			   strings.Contains(parsed.CleanText, "Common room") ||
			   strings.Contains(parsed.CleanText, "Exits:") {
				inGame = true
			}

			// Clean display based on content
			if parsed.CleanText != "" {
				if inGame && parsed.Type == parser.TypeRoomTitle {
					fmt.Printf("\n🏠 === %s ===\n", parsed.CleanText)
					fmt.Println("🎨 [Image would generate here]")
				} else if inGame && parsed.Type == parser.TypeRoomDescription {
					fmt.Printf("📝 %s\n", parsed.CleanText)
				} else if parsed.Type == parser.TypeExits && len(parsed.Exits) > 0 {
					fmt.Printf("🚪 Exits: %s\n", strings.Join(parsed.Exits, ", "))
				} else if parsed.Type == parser.TypeInventory {
					fmt.Printf("📦 %s\n", parsed.CleanText)
				} else {
					// Regular output
					fmt.Println(parsed.CleanText)
				}
			}
		}
	}()

	// Handle user input
	scanner := bufio.NewScanner(os.Stdin)
	reader := bufio.NewReader(os.Stdin)

	for {
		// Use reader for single character input when needed
		input := ""

		if scanner.Scan() {
			input = scanner.Text()
		} else {
			break
		}

		command := strings.TrimSpace(input)

		// Check for client quit
		if command == "quit" {
			fmt.Println("\n👋 Disconnecting from MUD...")
			break
		}

		// Check for MUD quit command
		if command == "/quit" {
			command = "QUIT"
		}

		// Send command to MUD
		err := client.SendCommand(command)
		if err != nil {
			fmt.Printf("⚠️  Error sending command: %v\n", err)
		}

		// Give output time to process
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("✓ Session ended. Goodbye!")
}
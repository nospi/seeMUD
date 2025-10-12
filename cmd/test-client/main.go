package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"see-mud-gui/internal/parser"
	"see-mud-gui/internal/telnet"
)

func main() {
	fmt.Println("See-MUD Test Client")
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

	fmt.Println("Connected! Type 'quit' to exit.")

	// Start output processing
	go func() {
		outputChan := client.GetOutput()
		for line := range outputChan {
			// Parse the line
			parsed := mudParser.ParseLine(line)

			// Color-code output based on type
			switch parsed.Type {
			case parser.TypeRoomTitle:
				fmt.Printf("\033[1;36m[ROOM] %s\033[0m\n", parsed.CleanText)
			case parser.TypeRoomDescription:
				fmt.Printf("\033[0;32m[DESC] %s\033[0m\n", parsed.CleanText)
			case parser.TypeExits:
				fmt.Printf("\033[1;33m[EXITS] %s\033[0m\n", parsed.CleanText)
			case parser.TypeInventory:
				fmt.Printf("\033[0;35m[ITEM] %s\033[0m\n", parsed.CleanText)
			case parser.TypePrompt:
				fmt.Printf("\033[1;32m%s\033[0m ", parsed.CleanText)
			default:
				fmt.Printf("\033[0;37m%s\033[0m\n", parsed.CleanText)
			}
		}
	}()

	// Handle user input
	scanner := bufio.NewScanner(os.Stdin)
	for {
		if !scanner.Scan() {
			break
		}

		command := strings.TrimSpace(scanner.Text())
		if command == "quit" {
			fmt.Println("Disconnecting...")
			break
		}

		if command != "" {
			err := client.SendCommand(command)
			if err != nil {
				fmt.Printf("Error sending command: %v\n", err)
			}
		}

		// Give a moment for output to process
		time.Sleep(10 * time.Millisecond)
	}
}
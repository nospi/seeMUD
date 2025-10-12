package main

import (
	"fmt"
	"log"
	"time"

	"see-mud-gui/internal/parser"
	"see-mud-gui/internal/telnet"
)

func main() {
	fmt.Println("Testing See-MUD components...")

	// Test parser
	p := parser.NewWolfMUDParser()
	testLine := "You are in the corner of the common room in the dragon's breath tavern."
	parsed := p.ParseLine(testLine)
	fmt.Printf("Parser test: Type=%d, Content='%s'\n", parsed.Type, parsed.CleanText)

	// Test telnet connection
	fmt.Println("Testing connection to WolfMUD...")
	client := telnet.NewClient("localhost", "4001")

	err := client.Connect()
	if err != nil {
		log.Printf("Connection failed: %v", err)
		return
	}
	defer client.Disconnect()

	fmt.Println("Connected! Sending 'look' command...")

	// Start output reader
	go func() {
		outputChan := client.GetOutput()
		for i := 0; i < 10; i++ { // Read 10 lines then stop
			select {
			case line := <-outputChan:
				parsed := p.ParseLine(line)
				fmt.Printf("[%d] %s -> Type=%d\n", parsed.Type, line, parsed.Type)
			case <-time.After(1 * time.Second):
				fmt.Println("Timeout waiting for output")
				return
			}
		}
	}()

	// Send a look command
	time.Sleep(100 * time.Millisecond) // Give connection time to establish
	err = client.SendCommand("look")
	if err != nil {
		log.Printf("Send failed: %v", err)
	}

	// Wait a bit for responses
	time.Sleep(2 * time.Second)
	fmt.Println("Test complete!")
}
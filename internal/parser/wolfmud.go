package parser

import (
	"regexp"
	"strings"
)

// WolfMUDParser handles parsing of WolfMUD specific output format
type WolfMUDParser struct {
	// Regular expressions for different content types
	promptRegex    *regexp.Regexp
	exitRegex      *regexp.Regexp
	inventoryRegex *regexp.Regexp
	colorCodeRegex *regexp.Regexp
}

// OutputType represents the type of parsed content
type OutputType int

const (
	TypeUnknown OutputType = iota
	TypeRoomDescription
	TypeRoomTitle
	TypeExits
	TypeInventory
	TypeMobs
	TypePrompt
	TypeSystem
	TypeSay
	TypeTell
)

// ParsedOutput represents a parsed line from the MUD
type ParsedOutput struct {
	Type        OutputType
	Content     string
	CleanText   string
	RawText     string
	RoomName    string
	Exits       []string
	Items       []string
	Mobs        []string
	IsRoomEntry bool
}

// NewWolfMUDParser creates a new parser for WolfMUD
func NewWolfMUDParser() *WolfMUDParser {
	return &WolfMUDParser{
		promptRegex:    regexp.MustCompile(`^[\[<].*[\]>]\s*$`),
		exitRegex:      regexp.MustCompile(`^Exits?:\s*(.+)$`),
		inventoryRegex: regexp.MustCompile(`^(A|An|The)\s+.*\s+(is|are|sits?|lies?|stands?|rests?)\s+.*\.$`),
		colorCodeRegex: regexp.MustCompile(`\x1b\[[0-9;]*m`),
	}
}

// ParseLine parses a single line of MUD output
func (p *WolfMUDParser) ParseLine(line string) *ParsedOutput {
	output := &ParsedOutput{
		RawText: line,
		Type:    TypeUnknown,
	}

	// Remove color codes for processing
	cleaned := p.stripColorCodes(line)
	output.CleanText = cleaned

	// Skip empty lines
	if strings.TrimSpace(cleaned) == "" {
		return output
	}

	// Check for prompt
	if p.promptRegex.MatchString(cleaned) {
		output.Type = TypePrompt
		output.Content = cleaned
		return output
	}

	// Check for exits
	if matches := p.exitRegex.FindStringSubmatch(cleaned); matches != nil {
		output.Type = TypeExits
		output.Content = matches[1]
		output.Exits = p.parseExits(matches[1])
		return output
	}

	// Check for inventory items (things in the room)
	if p.inventoryRegex.MatchString(cleaned) {
		output.Type = TypeInventory
		output.Content = cleaned
		output.Items = []string{p.extractItemName(cleaned)}
		return output
	}

	// Check if this looks like a room title (usually short, no punctuation at end)
	if p.isRoomTitle(cleaned) {
		output.Type = TypeRoomTitle
		output.Content = cleaned
		output.RoomName = cleaned
		output.IsRoomEntry = true
		return output
	}

	// Check for system messages
	if p.isSystemMessage(cleaned) {
		output.Type = TypeSystem
		output.Content = cleaned
		return output
	}

	// Default to room description or general content
	output.Type = TypeRoomDescription
	output.Content = cleaned

	return output
}

// ParseMultipleLines parses multiple lines and groups them logically
func (p *WolfMUDParser) ParseMultipleLines(lines []string) []*ParsedOutput {
	var results []*ParsedOutput
	var currentRoom *ParsedOutput

	for _, line := range lines {
		parsed := p.ParseLine(line)

		// Track room context
		if parsed.Type == TypeRoomTitle {
			currentRoom = parsed
		} else if currentRoom != nil && parsed.Type == TypeRoomDescription {
			// Associate description with current room
			currentRoom.Content += " " + parsed.Content
			continue
		}

		results = append(results, parsed)
	}

	return results
}

// stripColorCodes removes ANSI escape sequences from text
func (p *WolfMUDParser) stripColorCodes(text string) string {
	// Remove color codes
	cleaned := p.colorCodeRegex.ReplaceAllString(text, "")

	// Remove other common ANSI escape sequences
	// ESC[ followed by any number of digits, semicolons, and letters
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	cleaned = ansiRegex.ReplaceAllString(cleaned, "")

	// Remove cursor position sequences like ESC[255;255H
	cursorRegex := regexp.MustCompile(`\x1b\[[0-9]+;[0-9]+[HfABCDEFGJKST]`)
	cleaned = cursorRegex.ReplaceAllString(cleaned, "")

	// Remove other escape sequences (ESC followed by single character)
	escRegex := regexp.MustCompile(`\x1b[78]`)
	cleaned = escRegex.ReplaceAllString(cleaned, "")

	// Remove screen clear sequences
	clearRegex := regexp.MustCompile(`\x1b\[2J`)
	cleaned = clearRegex.ReplaceAllString(cleaned, "")

	return cleaned
}

// parseExits parses the exits string into a slice
func (p *WolfMUDParser) parseExits(exitStr string) []string {
	// Handle common exit formats: "north, south, east" or "n, s, e"
	exits := strings.Split(exitStr, ",")
	var result []string

	for _, exit := range exits {
		exit = strings.TrimSpace(exit)
		if exit != "" {
			result = append(result, exit)
		}
	}

	return result
}

// extractItemName extracts the item name from an inventory line
func (p *WolfMUDParser) extractItemName(line string) string {
	// Simple extraction - gets the noun after articles
	words := strings.Fields(line)
	if len(words) < 2 {
		return line
	}

	// Skip articles and find the main noun
	start := 0
	for i, word := range words {
		lower := strings.ToLower(word)
		if lower == "a" || lower == "an" || lower == "the" {
			start = i + 1
			continue
		}
		if lower == "is" || lower == "are" || lower == "sits" ||
		   lower == "lies" || lower == "stands" || lower == "rests" {
			// Found verb, item name is between start and here
			if i > start {
				return strings.Join(words[start:i], " ")
			}
			break
		}
	}

	// Fallback: return first few words after articles
	if start < len(words) {
		end := start + 3 // Take up to 3 words for item name
		if end > len(words) {
			end = len(words)
		}
		return strings.Join(words[start:end], " ")
	}

	return line
}

// isRoomTitle checks if a line looks like a room title
func (p *WolfMUDParser) isRoomTitle(line string) bool {
	// Room titles are typically:
	// - Short (< 50 chars)
	// - Don't end with punctuation (except maybe :)
	// - Often title case
	if len(line) > 50 {
		return false
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	// Check if it ends with sentence punctuation
	lastChar := line[len(line)-1]
	if lastChar == '.' || lastChar == '!' || lastChar == '?' {
		return false
	}

	// Check if it has multiple sentences (likely description)
	if strings.Contains(line, ". ") {
		return false
	}

	return true
}

// isSystemMessage checks if a line is a system message
func (p *WolfMUDParser) isSystemMessage(line string) bool {
	// Common system message patterns
	systemPrefixes := []string{
		"You can't",
		"You don't",
		"You aren't",
		"You are",
		"You have",
		"You see",
		"There is",
		"There are",
		"It is",
		"You feel",
		"You hear",
		"You smell",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(line, prefix) {
			return true
		}
	}

	return false
}
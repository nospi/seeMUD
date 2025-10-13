package renderer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// StableDiffusionClient handles communication with Stable Diffusion WebUI API
type StableDiffusionClient struct {
	baseURL string
	client  *http.Client
}

// NewStableDiffusionClient creates a new SD client
func NewStableDiffusionClient(baseURL string) *StableDiffusionClient {
	return &StableDiffusionClient{
		baseURL: baseURL,
		client: &http.Client{
			Timeout: 120 * time.Second, // Generous timeout for image generation
		},
	}
}

// Txt2ImgRequest represents a text-to-image generation request
type Txt2ImgRequest struct {
	Prompt         string  `json:"prompt"`
	NegativePrompt string  `json:"negative_prompt,omitempty"`
	Width          int     `json:"width"`
	Height         int     `json:"height"`
	Steps          int     `json:"steps"`
	CFGScale       float64 `json:"cfg_scale"`
	Seed           int64   `json:"seed,omitempty"`
	SamplerName    string  `json:"sampler_name,omitempty"`
}

// Txt2ImgResponse represents the API response
type Txt2ImgResponse struct {
	Images []string `json:"images"`
	Info   string   `json:"info"`
}

// GenerateImage sends a text-to-image request to Stable Diffusion WebUI
func (sd *StableDiffusionClient) GenerateImage(ctx context.Context, req *Txt2ImgRequest) (*Txt2ImgResponse, error) {
	// Set defaults if not specified
	if req.Width == 0 {
		req.Width = 512
	}
	if req.Height == 0 {
		req.Height = 512
	}
	if req.Steps == 0 {
		req.Steps = 20
	}
	if req.CFGScale == 0 {
		req.CFGScale = 7.0
	}
	if req.SamplerName == "" {
		req.SamplerName = "Euler"
	}

	// Marshal request to JSON
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", sd.baseURL+"/sdapi/v1/txt2img", bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := sd.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result Txt2ImgResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// CheckHealth checks if the SD API is available
func (sd *StableDiffusionClient) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", sd.baseURL+"/sdapi/v1/options", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := sd.client.Do(req)
	if err != nil {
		return fmt.Errorf("SD API not available: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SD API returned status %d", resp.StatusCode)
	}

	return nil
}

// RoomImagePrompt generates an optimized prompt for room generation
func RoomImagePrompt(roomName, description string) string {
	basePrompt := fmt.Sprintf("Fantasy medieval environment, %s. %s", roomName, description)

	// Add quality and style modifiers
	stylePrompt := ", highly detailed, atmospheric lighting, fantasy art style, cinematic composition, 8k, masterpiece"

	return basePrompt + stylePrompt
}

// RoomImagePromptWithCustom generates a room prompt with custom user additions
func RoomImagePromptWithCustom(roomName, description, customAdditions string) string {
	basePrompt := RoomImagePrompt(roomName, description)

	if customAdditions != "" {
		return basePrompt + ", " + customAdditions
	}

	return basePrompt
}

// GetNegativePrompt returns a standard negative prompt for fantasy environments
func GetNegativePrompt() string {
	return "blurry, low quality, text, watermark, signature, people, characters, figures, humans, animals, modern objects, cars, buildings, technology"
}
package ai

import (
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/thatrajaryan/web-server/common"
)

type AIBlock struct {
	Provider    string
	APIKey      string
	Model       string
	MaxTokens   int
	Temperature float64
	
	client *resty.Client
}

func (b *AIBlock) Create(config map[string]interface{}) error {
	fmt.Println("[AI] Initializing AI Service Block...")
	b.client = resty.New()
	return b.Update(config)
}

func (b *AIBlock) Connect(target common.Block) error {
	fmt.Printf("[AI] %s (%s) connected to target block\n", b.Provider, b.Model)
	return nil
}

func (b *AIBlock) Update(config map[string]interface{}) error {
	if val, ok := config["provider"].(string); ok {
		b.Provider = val
	}
	if val, ok := config["api_key"].(string); ok {
		b.APIKey = val
	}
	if val, ok := config["model"].(string); ok {
		b.Model = val
	}
	if val, ok := config["max_tokens"].(float64); ok {
		b.MaxTokens = int(val)
	}
	if val, ok := config["temperature"].(float64); ok {
		b.Temperature = val
	}

	fmt.Printf("[AI] Configured Provider: %s, Model: %s, MaxTokens: %d\n", 
		b.Provider, b.Model, b.MaxTokens)

	return nil
}

func (b *AIBlock) Delete() error {
	fmt.Printf("[AI] AI Service (%s) deleted\n", b.Provider)
	return nil
}

// Predict is a placeholder for actual inference logic
func (b *AIBlock) Predict(prompt string) (string, error) {
	if b.Provider == "openrouter" {
		resp, err := b.client.R().
			SetHeader("Authorization", "Bearer "+b.APIKey).
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]interface{}{
				"model": b.Model,
				"messages": []map[string]string{
					{"role": "user", "content": prompt},
				},
				"max_tokens": b.MaxTokens,
				"temperature": b.Temperature,
			}).
			Post("https://openrouter.ai/api/v1/chat/completions")

		if err != nil {
			return "", err
		}
		return resp.String(), nil
	}
	return "", fmt.Errorf("unsupported provider: %s", b.Provider)
}

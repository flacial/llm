package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type OpenRouterModelsResponse struct {
	Data []OpenRouterModel `json:"data"`
}

type OpenRouterModel struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Created       int64             `json:"created"`
	Description   string            `json:"description"`
	ContextLength int               `json:"context_length"`
	Architecture  ModelArchitecture `json:"architecture"`
	Pricing       ModelPricing      `json:"pricing"`
	TopProvider   ModelTopProvider  `json:"top_provider"`
}

type ModelArchitecture struct {
	InputModalities  []string `json:"input_modalities"`
	OutputModalities []string `json:"output_modalities"`
	Tokenizer        string   `json:"tokenizer"`
	InstructType     string   `json:"instruct_type"`
}

type ModelPricing struct {
	Prompt            string `json:"prompt"`
	Completion        string `json:"completion"`
	Image             string `json:"image"`
	Request           string `json:"request"`
	InputCacheRead    string `json:"input_cache_read"`
	InputCacheWrite   string `json:"input_cache_write"`
	WebSearch         string `json:"web_search"`
	InternalReasoning string `json:"internal_reasoning"`
}

type ModelTopProvider struct {
	IsModerated bool `json:"is_moderated"`
}

var ModelsCmd = &cobra.Command{
	Use:   "models",
	Short: "List available LLM models from OpenRouter.ai",
	Long:  `Fetches and displays a list of available LLM models and their details from the OpenRouter.ai API.`,
	RunE:  runModelsCommand,
}

func runModelsCommand(cmd *cobra.Command, args []string) error {
	apiKey := viper.GetString("api_key")
	if apiKey == "" {
		return fmt.Errorf("API key not set. Please set LLM_API_KEY environment variable or 'api_key' in config to query OpenRouter.ai models.")
	}

	openRouterAPIURL := "https://openrouter.ai/api/v1/models"

	req, err := http.NewRequest("GET", openRouterAPIURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request to OpenRouter API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("OpenRouter API returned non-200 status: %d %s, Body: %s", resp.StatusCode, resp.Status, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var modelsResponse OpenRouterModelsResponse
	err = json.Unmarshal(bodyBytes, &modelsResponse)
	if err != nil {
		return fmt.Errorf("failed to parse JSON response from OpenRouter API: %w", err)
	}

	if len(modelsResponse.Data) == 0 {
		fmt.Println("No models found from openrouter.ai—AI took over and we're now doomed.")
		return nil
	}

	fmt.Println("Available Models from openrouter.ai:")
	fmt.Println("------------------------------------")
	for _, model := range modelsResponse.Data {
		fmt.Printf("ID: %s\n", model.ID)
		fmt.Printf("Name: %s\n", model.Name)
		fmt.Printf("Description: %s\n", truncateString(model.Description, 100)+"\n") // Truncate long descriptions
		fmt.Printf("Context Length: %d tokens\n", model.ContextLength)
		if model.Pricing.Prompt != "" {
			fmt.Printf("Pricing (per 1M tokens): Input=$%s, Output=$%s\n",
				model.Pricing.Prompt,
				model.Pricing.Completion)
		}
		if len(model.Architecture.InputModalities) > 0 {
			fmt.Printf("Input Modalities: %s\n", strings.Join(model.Architecture.InputModalities, ", "))
		}
		if model.TopProvider.IsModerated {
			fmt.Println("Moderated: Yes")
		} else {
			fmt.Println("Moderated: No")
		}

		fmt.Printf("Added: %s\n", time.Unix(model.Created, 0).Format("2006-01-02"))
		fmt.Println("------------------------------------")
	}

	return nil
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	return s[:maxLen] + "..."
}

func init() {
	rootCmd.AddCommand(ModelsCmd)
}

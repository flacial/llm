package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/flacial/llm/internal/log"
)

const (
	OpenRouterAPIURL = "https://openrouter.ai/api/v1/chat/completions"
	// TODO: Allow user to configure the timeout because some LLM responses
	// are pretty lengthy
	DefaultTimeout = (2 * time.Minute)
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type LLMClient struct {
	APIKey     string
	HTTPClient HTTPClient
	BaseURL    string
}

func NewLLMClient(apiKey string, client HTTPClient, baseURL string) *LLMClient {
	if client == nil {
		client = &http.Client{Timeout: DefaultTimeout}
	}

	if baseURL == "" {
		baseURL = OpenRouterAPIURL
	}

	return &LLMClient{
		APIKey:     apiKey,
		HTTPClient: client,
		BaseURL:    baseURL,
	}
}

type ChatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []ChatCompletionMessage `json:"messages"`
	Stream      bool                    `json:"stream"`
	Temperature *float64                `json:"temperature"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponseChoices struct {
	Index   int                           `json:"index"`
	Message ChatCompletionResponseMessage `json:"message"`
}

type ChatCompletionResponseMessage struct {
	Role         string `json:"role"`
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
	Reasoning    string `json:"reasoning,omitempty"`
}

type ChatCompletionResponse struct {
	Id      string                          `json:"id"`
	Choices []ChatCompletionResponseChoices `json:"choices"`
}

type ChatCompletionStreamResponseMessageDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionStreamResponseChoices struct {
	Index        int                                      `json:"index"`
	Delta        ChatCompletionStreamResponseMessageDelta `json:"delta"`
	FinishReason string                                   `json:"finish_reason"`
}

type ChatCompletionStreamResponse struct {
	Id      string                                `json:"id"`
	Created int64                                 `json:"created"`
	Model   string                                `json:"model"`
	Choices []ChatCompletionStreamResponseChoices `json:"choices"`
}

func (c *LLMClient) GetChatCompletion(ctx context.Context, reqBody ChatCompletionRequest) (*ChatCompletionResponse, error) {
	log.Logger.Debug().Interface("request_body", reqBody).Msg("Sending chat completion request.")

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Error encoding completion JSON.")
		return nil, fmt.Errorf("error encoding completion JSON: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create HTTP request.")
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Logger.Info().Msg("HTTP request cancelled by context.")
			return nil, context.Canceled
		}
		log.Logger.Error().Err(err).Msg("Error sending request to LLM API.")
		return nil, fmt.Errorf("error sending request to LLM API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Logger.Error().Err(err).Msg("Failed to close response body.")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Logger.Error().
			Int("status_code", resp.StatusCode).
			Bytes("response_body", bodyBytes).
			Msg("LLM API returned non-OK status.")
		return nil, fmt.Errorf("LLM API returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var completionResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		log.Logger.Error().Err(err).Msg("Error decoding LLM API response.")
		return nil, fmt.Errorf("error decoding LLM API response: %w", err)
	}

	log.Logger.Debug().Msg("Successfully received chat completion response.")
	return &completionResp, nil
}

func (c *LLMClient) GetStreamingChatCompletion(ctx context.Context, reqBody ChatCompletionRequest, outputWriter io.Writer) (string, error) {
	log.Logger.Debug().Interface("request_body", reqBody).Msg("Sending streaming chat completion request.")

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Logger.Error().Err(err).Msg("Error encoding streaming completion JSON.")
		return "", fmt.Errorf("error encoding completion JSON: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Logger.Error().Err(err).Msg("Failed to create HTTP request for streaming.")
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			log.Logger.Info().Msg("Streaming HTTP request cancelled by context.")
			return "", context.Canceled
		}
		log.Logger.Error().Err(err).Msg("Error sending streaming request to LLM API.")
		return "", fmt.Errorf("error sending request to LLM API: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Logger.Error().Err(err).Msg("Failed to close streaming response body.")
		}
	}()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Logger.Error().
			Int("status_code", resp.StatusCode).
			Bytes("response_body", bodyBytes).
			Msg("LLM API returned non-OK status for streaming.")
		return "", fmt.Errorf("LLM API returned non-OK status: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		log.Logger.Trace().Str("raw_line", line).Msg("Received stream line.")

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			log.Logger.Debug().Msg("Streaming complete (DONE signal received).")
			break
		}

		var chunk ChatCompletionStreamResponse
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Logger.Error().Err(err).Str("data", data).Msg("Error unmarshalling streaming chunk.")
			return "", fmt.Errorf("error unmarshalling streaming chunk: %w", err)
		}

		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				fmt.Fprintf(outputWriter, "%s", choice.Delta.Content)
				fullContent.WriteString(choice.Delta.Content)
			}

			if choice.FinishReason != "" {
				fmt.Fprintf(outputWriter, "\n\n")
				log.Logger.Debug().Str("finish_reason", choice.FinishReason).Msg("Stream finished.")
			}
		}
	}

	if err := scanner.Err(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Logger.Info().Msg("Streaming response read cancelled by context.")
			return fullContent.String(), context.Canceled
		}
		log.Logger.Error().Err(err).Msg("Error reading streaming response.")
		return fullContent.String(), fmt.Errorf("error reading streaming response: %w", err)
	}

	log.Logger.Debug().Msg("Streaming session completed successfully.")
	return fullContent.String(), nil
}

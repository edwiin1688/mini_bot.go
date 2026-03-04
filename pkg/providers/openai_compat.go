package providers

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAICompatProvider struct {
	BaseURL string
	APIKey  string
	client  *http.Client
}

func NewOpenAICompatProvider(baseURL, apiKey string) *OpenAICompatProvider {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &OpenAICompatProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		client: &http.Client{
			Transport: transport,
			Timeout:   60 * time.Second,
		},
	}
}

func (p *OpenAICompatProvider) GetDefaultModel() string {
	return "gpt-4"
}

func (p *OpenAICompatProvider) Chat(
	ctx context.Context,
	messages []Message,
	tools []ToolDefinition,
	model string,
	options map[string]any,
) (*LLMResponse, error) {

	url := p.BaseURL
	// Automatically append /chat/completions if the URL just ends with /v1 or similar
	if !bytes.HasSuffix([]byte(url), []byte("/chat/completions")) {
		if url[len(url)-1] != '/' {
			url += "/"
		}
		url += "chat/completions"
	}

	payload := map[string]any{
		"model":    model,
		"messages": messages,
	}

	if len(tools) > 0 {
		payload["tools"] = tools
	}

	// Apply options (temperature, max_tokens, etc.)
	for k, v := range options {
		payload[k] = v
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	client := p.client
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(bodyText))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content   string     `json:"content"`
				ToolCalls []ToolCall `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(bodyText, &response); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w - body: %s", err, string(bodyText))
	}

	if len(response.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	msg := response.Choices[0].Message
	return &LLMResponse{
		Content:   msg.Content,
		ToolCalls: msg.ToolCalls,
		Usage: UsageInfo{
			PromptTokens:     response.Usage.PromptTokens,
			CompletionTokens: response.Usage.CompletionTokens,
			TotalTokens:      response.Usage.TotalTokens,
		},
	}, nil
}

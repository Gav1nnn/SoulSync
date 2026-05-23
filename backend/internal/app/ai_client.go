package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIClient(baseURL string) *AIClient {
	return &AIClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *AIClient) Generate(ctx context.Context, request AIGenerateRequest) (AIGenerateResponse, error) {
	payload, err := json.Marshal(request)
	if err != nil {
		return AIGenerateResponse{}, fmt.Errorf("marshal ai request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/generate",
		bytes.NewReader(payload),
	)
	if err != nil {
		return AIGenerateResponse{}, fmt.Errorf("build ai request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return AIGenerateResponse{}, fmt.Errorf("call ai engine: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return AIGenerateResponse{}, fmt.Errorf("ai engine returned status %d", response.StatusCode)
	}

	var result AIGenerateResponse
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		return AIGenerateResponse{}, fmt.Errorf("decode ai response: %w", err)
	}

	return result, nil
}

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

const defaultAIEngineTimeout = 120 * time.Second

type AIClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewAIClient(baseURL string) *AIClient {
	return NewAIClientWithTimeout(baseURL, defaultAIEngineTimeout)
}

func NewAIClientWithTimeout(baseURL string, timeout time.Duration) *AIClient {
	if timeout <= 0 {
		timeout = defaultAIEngineTimeout
	}

	return NewAIClientWithHTTPClient(baseURL, &http.Client{
		Timeout: timeout,
	})
}

func NewAIClientWithHTTPClient(baseURL string, httpClient *http.Client) *AIClient {
	return &AIClient{
		baseURL:    strings.TrimRight(baseURL, "/"),
		httpClient: httpClient,
	}
}

func (c *AIClient) Generate(ctx context.Context, request AIGenerateRequest) (AIGenerateResponse, error) {
	var result AIGenerateResponse
	if err := c.postJSON(ctx, "/generate", request, &result); err != nil {
		return AIGenerateResponse{}, err
	}

	return result, nil
}

func (c *AIClient) Plan(ctx context.Context, request AIAgentPlanRequest) (AIAgentPlanResponse, error) {
	var result AIAgentPlanResponse
	if err := c.postJSON(ctx, "/agent/plan", request, &result); err != nil {
		return AIAgentPlanResponse{}, err
	}

	return result, nil
}

func (c *AIClient) postJSON(ctx context.Context, path string, requestPayload any, result any) error {
	payload, err := json.Marshal(requestPayload)
	if err != nil {
		return fmt.Errorf("marshal ai request: %w", err)
	}

	httpRequest, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+path,
		bytes.NewReader(payload),
	)
	if err != nil {
		return fmt.Errorf("build ai request: %w", err)
	}
	httpRequest.Header.Set("Content-Type", "application/json")

	response, err := c.httpClient.Do(httpRequest)
	if err != nil {
		return fmt.Errorf("call ai engine: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("ai engine returned status %d", response.StatusCode)
	}

	if err := json.NewDecoder(response.Body).Decode(result); err != nil {
		return fmt.Errorf("decode ai response: %w", err)
	}

	return nil
}

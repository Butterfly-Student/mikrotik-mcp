package zai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

const DefaultBaseURL = "https://api.z.ai/api/paas/v4"

type Client struct {
	apiKey     string
	baseURL    string
	model      string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewClient(apiKey, baseURL, model string, logger *zap.Logger) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger,
	}
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	if req.Model == "" {
		req.Model = c.model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx, http.MethodPost,
		c.baseURL+"/chat/completions",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept-Language", "en-US,en")

	c.logger.Debug("calling Z.AI",
		zap.String("model", req.Model),
		zap.Int("messages", len(req.Messages)),
		zap.Int("tools", len(req.Tools)),
	)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request to Z.AI: %w", err)
	}
	defer resp.Body.Close()

	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("Z.AI HTTP %d: decode response: %w", resp.StatusCode, err)
	}

	// Cek error dari body dulu (Z.AI bisa return error dengan status 200)
	if result.Error != nil {
		return nil, fmt.Errorf("Z.AI error [%v]: %s", result.Error.Code, result.Error.Message)
	}
	// Fallback: cek HTTP status untuk 4xx/5xx tanpa body error
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("Z.AI HTTP %d", resp.StatusCode)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("Z.AI returned empty choices")
	}

	c.logger.Debug("Z.AI responded",
		zap.String("finish_reason", result.Choices[0].FinishReason),
		zap.Int("tokens", result.Usage.TotalTokens),
	)

	return &result, nil
}

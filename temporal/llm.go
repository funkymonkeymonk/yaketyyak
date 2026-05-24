package temporal

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type llmClient struct {
	inner *openai.Client
	model string
}

func newLLM(cfg LLMConfig) *llmClient {
	key := cfg.APIKey
	if key == "" {
		key = "sk-placeholder"
	}
	oc := openai.DefaultConfig(key)
	if cfg.BaseURL != "" {
		oc.BaseURL = cfg.BaseURL
	}
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	return &llmClient{
		inner: openai.NewClientWithConfig(oc),
		model: model,
	}
}

func (c *llmClient) chatJSON(ctx context.Context, system, user string, result interface{}) error {
	resp, err := c.inner.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: c.model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: system},
			{Role: openai.ChatMessageRoleUser, Content: user},
		},
		Temperature: 0.0,
	})
	if err != nil {
		return fmt.Errorf("llm chat: %w", err)
	}

	content := resp.Choices[0].Message.Content
	content = extractJSON(content)

	if err := json.Unmarshal([]byte(content), result); err != nil {
		return fmt.Errorf("llm parse response: %w\nraw: %s", err, resp.Choices[0].Message.Content)
	}
	return nil
}

func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		if idx := strings.LastIndex(s, "```"); idx >= 0 {
			s = s[:idx]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start >= 0 && end >= 0 && end > start {
		s = s[start : end+1]
	}
	return strings.TrimSpace(s)
}

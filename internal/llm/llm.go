package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	url  string
	http *http.Client
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func System(content string) Message { return Message{"system", content} }

func User(content string) Message { return Message{"user", content} }

func Assistant(content string) Message { return Message{"assistant", content} }

func New(url string) *Client {
	return &Client{url: strings.TrimRight(url, "/"), http: &http.Client{}}
}

func (c *Client) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("no LLM server at %s: %w", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("LLM server at %s isn't ready: %s", c.url, failure(resp))
	}
	return nil
}

func (c *Client) Prompt(ctx context.Context, system, user string) (string, error) {
	return c.Chat(ctx, []Message{System(system), User(user)})
}

type request struct {
	Messages    []Message      `json:"messages"`
	Temperature float64        `json:"temperature"`
	Template    map[string]any `json:"chat_template_kwargs"`
}

type response struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

func (c *Client) Chat(ctx context.Context, messages []Message) (string, error) {
	body, err := json.Marshal(request{
		Messages:    messages,
		Temperature: 0,
		Template:    map[string]any{"enable_thinking": false},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("no LLM server at %s: %w", c.url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("LLM server at %s failed: %s", c.url, failure(resp))
	}

	var out response
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", fmt.Errorf("LLM server at %s sent an unreadable answer: %w", c.url, err)
	}

	if len(out.Choices) == 0 {
		return "", errors.New("LLM server sent no answer")
	}

	choice := out.Choices[0]
	if choice.FinishReason == "length" {
		return choice.Message.Content, errors.New("LLM answer was cut off at the token limit")
	}
	return choice.Message.Content, nil
}

func failure(resp *http.Response) string {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))

	var body struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if json.Unmarshal(data, &body) == nil && body.Error.Message != "" {
		return body.Error.Message
	}

	if text := strings.TrimSpace(string(data)); text != "" {
		return resp.Status + ": " + text
	}
	return resp.Status
}

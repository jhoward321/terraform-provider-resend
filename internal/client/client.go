package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

const defaultBaseURL = "https://api.resend.com"

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

type Option func(*Client)

func WithBaseURL(url string) Option {
	return func(c *Client) {
		c.baseURL = url
	}
}

func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:     apiKey,
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// --- Request/Response types ---

type CreateAPIKeyRequest struct {
	Name       string `json:"name"`
	Permission string `json:"permission,omitempty"`
	DomainID   string `json:"domain_id,omitempty"`
}

type CreateAPIKeyResponse struct {
	ID    string `json:"id"`
	Token string `json:"token"`
}

type CreateDomainRequest struct {
	Name   string `json:"name"`
	Region string `json:"region,omitempty"`
}

type Domain struct {
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Status    string      `json:"status"`
	Region    string      `json:"region"`
	CreatedAt string      `json:"created_at"`
	Records   []DNSRecord `json:"records"`
}

type DNSRecord struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Priority string `json:"priority,omitempty"`
	TTL      string `json:"ttl,omitempty"`
	Status   string `json:"status,omitempty"`
}

type CreateWebhookRequest struct {
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
}

type UpdateWebhookRequest struct {
	URL        string   `json:"url,omitempty"`
	EventTypes []string `json:"event_types,omitempty"`
}

type Webhook struct {
	ID         string   `json:"id"`
	URL        string   `json:"url"`
	EventTypes []string `json:"event_types"`
	CreatedAt  string   `json:"created_at"`
}

type APIError struct {
	StatusCode int
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("resend API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// --- HTTP helpers ---

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+"/"+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close() //nolint:errcheck
		apiErr := &APIError{StatusCode: resp.StatusCode}
		_ = json.NewDecoder(resp.Body).Decode(apiErr)
		return nil, apiErr
	}

	return resp, nil
}

func decodeResponse[T any](resp *http.Response) (*T, error) {
	defer resp.Body.Close() //nolint:errcheck
	var result T
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}
	return &result, nil
}

// --- API Key methods ---

func (c *Client) CreateAPIKey(ctx context.Context, req CreateAPIKeyRequest) (*CreateAPIKeyResponse, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "api-keys", req)
	if err != nil {
		return nil, err
	}
	return decodeResponse[CreateAPIKeyResponse](resp)
}

func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "api-keys/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close() //nolint:errcheck
	return nil
}

// --- Domain methods ---

func (c *Client) CreateDomain(ctx context.Context, req CreateDomainRequest) (*Domain, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "domains", req)
	if err != nil {
		return nil, err
	}
	return decodeResponse[Domain](resp)
}

func (c *Client) GetDomain(ctx context.Context, id string) (*Domain, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "domains/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[Domain](resp)
}

func (c *Client) DeleteDomain(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "domains/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close() //nolint:errcheck
	return nil
}

// --- Webhook methods ---

func (c *Client) CreateWebhook(ctx context.Context, req CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodPost, "webhooks", req)
	if err != nil {
		return nil, err
	}
	return decodeResponse[Webhook](resp)
}

func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodGet, "webhooks/"+id, nil)
	if err != nil {
		return nil, err
	}
	return decodeResponse[Webhook](resp)
}

func (c *Client) UpdateWebhook(ctx context.Context, id string, req UpdateWebhookRequest) (*Webhook, error) {
	resp, err := c.doRequest(ctx, http.MethodPut, "webhooks/"+id, req)
	if err != nil {
		return nil, err
	}
	return decodeResponse[Webhook](resp)
}

func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, http.MethodDelete, "webhooks/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close() //nolint:errcheck
	return nil
}

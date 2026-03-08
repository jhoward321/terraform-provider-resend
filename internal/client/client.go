package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"time"
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
	Record   string      `json:"record"`
	Type     string      `json:"type"`
	Name     string      `json:"name"`
	Value    string      `json:"value"`
	Priority json.Number `json:"priority,omitempty"`
	TTL      string      `json:"ttl,omitempty"`
	Status   string      `json:"status,omitempty"`
}

type CreateWebhookRequest struct {
	Endpoint string   `json:"endpoint"`
	Events   []string `json:"events"`
}

type UpdateWebhookRequest struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Events   []string `json:"events,omitempty"`
}

type Webhook struct {
	ID            string   `json:"id"`
	Endpoint      string   `json:"endpoint"`
	Events        []string `json:"events"`
	CreatedAt     string   `json:"created_at"`
	SigningSecret string   `json:"signing_secret,omitempty"`
}

type APIError struct {
	StatusCode int
	Message    string `json:"message"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("resend API error (HTTP %d): %s", e.StatusCode, e.Message)
}

// --- HTTP helpers ---

const maxRetries = 5

func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request: %w", err)
		}
		bodyBytes = b
	}

	for attempt := range maxRetries {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
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

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close() //nolint:errcheck
			if attempt == maxRetries-1 {
				return nil, &APIError{StatusCode: resp.StatusCode, Message: "rate limited after max retries"}
			}
			delay := retryDelay(resp, attempt)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
				continue
			}
		}

		if resp.StatusCode >= 400 {
			defer resp.Body.Close() //nolint:errcheck
			apiErr := &APIError{StatusCode: resp.StatusCode}
			_ = json.NewDecoder(resp.Body).Decode(apiErr)
			return nil, apiErr
		}

		return resp, nil
	}

	return nil, fmt.Errorf("unexpected: exhausted retries")
}

func retryDelay(resp *http.Response, attempt int) time.Duration {
	if v := resp.Header.Get("Retry-After"); v != "" {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
	}
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
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
	resp, err := c.doRequest(ctx, http.MethodPatch, "webhooks/"+id, req)
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

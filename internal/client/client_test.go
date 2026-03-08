package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api-keys" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %s", r.Header.Get("Authorization"))
		}

		var body CreateAPIKeyRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "test" {
			t.Errorf("unexpected name: %s", body.Name)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CreateAPIKeyResponse{
			ID:    "key-123",
			Token: "re_test_token",
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "key-123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
	if resp.Token != "re_test_token" {
		t.Errorf("unexpected token: %s", resp.Token)
	}
}

func TestDeleteAPIKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/api-keys/key-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	err := c.DeleteAPIKey(context.Background(), "key-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/domains" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body CreateDomainRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "example.com" {
			t.Errorf("unexpected name: %s", body.Name)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Domain{
			ID:     "domain-123",
			Name:   "example.com",
			Status: "pending",
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.CreateDomain(context.Background(), CreateDomainRequest{Name: "example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "domain-123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
}

func TestGetDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/domains/domain-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"id": "domain-123",
			"name": "example.com",
			"status": "verified",
			"region": "us-east-1",
			"created_at": "2023-04-26T20:21:26.347412+00:00",
			"records": [
				{"record": "SPF", "name": "send", "type": "MX", "ttl": "Auto", "status": "verified", "value": "feedback-smtp.us-east-1.amazonses.com", "priority": 10},
				{"record": "SPF", "name": "send", "type": "TXT", "ttl": "Auto", "status": "verified", "value": "v=spf1 include:amazonses.com ~all"},
				{"record": "DKIM", "name": "resend._domainkey", "type": "CNAME", "ttl": "Auto", "status": "verified", "value": "dkim.us-east-1.resend.com"}
			]
		}`))
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.GetDomain(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "verified" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
	if len(resp.Records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(resp.Records))
	}
	if resp.Records[0].Record != "SPF" {
		t.Errorf("expected first record to be SPF, got %s", resp.Records[0].Record)
	}
	if resp.Records[0].Priority.String() != "10" {
		t.Errorf("expected priority 10, got %s", resp.Records[0].Priority.String())
	}
	if resp.Records[2].Record != "DKIM" {
		t.Errorf("expected third record to be DKIM, got %s", resp.Records[2].Record)
	}
}

func TestDeleteDomain(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/domains/domain-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	err := c.DeleteDomain(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/webhooks" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}

		var body CreateWebhookRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Endpoint != "https://example.com/webhook" {
			t.Errorf("unexpected Endpoint: %s", body.Endpoint)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Webhook{
			ID:            "wh-123",
			Endpoint:      "https://example.com/webhook",
			Events:        []string{"email.sent", "email.delivered"},
			SigningSecret: "whsec_test123",
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.CreateWebhook(context.Background(), CreateWebhookRequest{
		Endpoint: "https://example.com/webhook",
		Events:   []string{"email.sent", "email.delivered"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "wh-123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
	if resp.SigningSecret != "whsec_test123" {
		t.Errorf("unexpected signing secret: %s", resp.SigningSecret)
	}
}

func TestGetWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Webhook{
			ID:       "wh-123",
			Endpoint: "https://example.com/webhook",
			Events:   []string{"email.sent"},
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.GetWebhook(context.Background(), "wh-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Endpoint != "https://example.com/webhook" {
		t.Errorf("unexpected Endpoint: %s", resp.Endpoint)
	}
}

func TestUpdateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Webhook{
			ID:       "wh-123",
			Endpoint: "https://example.com/webhook-updated",
			Events:   []string{"email.sent", "email.bounced"},
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.UpdateWebhook(context.Background(), "wh-123", UpdateWebhookRequest{
		Endpoint: "https://example.com/webhook-updated",
		Events:   []string{"email.sent", "email.bounced"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Endpoint != "https://example.com/webhook-updated" {
		t.Errorf("unexpected Endpoint: %s", resp.Endpoint)
	}
}

func TestDeleteWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	err := c.DeleteWebhook(context.Background(), "wh-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRateLimitRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(CreateAPIKeyResponse{
			ID:    "key-123",
			Token: "re_test_token",
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "key-123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRateLimitExhausted(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	_, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if attempts != maxRetries {
		t.Errorf("expected %d attempts, got %d", maxRetries, attempts)
	}
}

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"message":"Invalid API key"}`))
	}))
	defer server.Close()

	c := New("bad-key", WithBaseURL(server.URL))
	_, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

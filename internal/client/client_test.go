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
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "test" {
			t.Errorf("unexpected name: %s", body.Name)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(CreateAPIKeyResponse{
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
		json.NewDecoder(r.Body).Decode(&body)
		if body.Name != "example.com" {
			t.Errorf("unexpected name: %s", body.Name)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Domain{
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
		json.NewEncoder(w).Encode(Domain{
			ID:     "domain-123",
			Name:   "example.com",
			Status: "verified",
			Records: []DNSRecord{
				{Type: "MX", Name: "example.com", Value: "feedback-smtp.us-east-1.amazonses.com", Priority: "10"},
			},
		})
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
	if len(resp.Records) != 1 {
		t.Errorf("unexpected records count: %d", len(resp.Records))
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
		json.NewDecoder(r.Body).Decode(&body)
		if body.URL != "https://example.com/webhook" {
			t.Errorf("unexpected URL: %s", body.URL)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Webhook{
			ID:         "wh-123",
			URL:        "https://example.com/webhook",
			EventTypes: []string{"email.sent", "email.delivered"},
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.CreateWebhook(context.Background(), CreateWebhookRequest{
		URL:        "https://example.com/webhook",
		EventTypes: []string{"email.sent", "email.delivered"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ID != "wh-123" {
		t.Errorf("unexpected ID: %s", resp.ID)
	}
}

func TestGetWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Webhook{
			ID:         "wh-123",
			URL:        "https://example.com/webhook",
			EventTypes: []string{"email.sent"},
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.GetWebhook(context.Background(), "wh-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.URL != "https://example.com/webhook" {
		t.Errorf("unexpected URL: %s", resp.URL)
	}
}

func TestUpdateWebhook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(Webhook{
			ID:         "wh-123",
			URL:        "https://example.com/webhook-updated",
			EventTypes: []string{"email.sent", "email.bounced"},
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.UpdateWebhook(context.Background(), "wh-123", UpdateWebhookRequest{
		URL:        "https://example.com/webhook-updated",
		EventTypes: []string{"email.sent", "email.bounced"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.URL != "https://example.com/webhook-updated" {
		t.Errorf("unexpected URL: %s", resp.URL)
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

func TestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"message":"Invalid API key"}`))
	}))
	defer server.Close()

	c := New("bad-key", WithBaseURL(server.URL))
	_, err := c.CreateAPIKey(context.Background(), CreateAPIKeyRequest{Name: "test"})
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

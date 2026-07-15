# Resend API Catch-Up Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Model the new webhook `status` field and modernize the `resend_domain` resource with in-place PATCH updates plus the new domain settings (tracking, tls, capabilities, custom_return_path).

**Architecture:** Extend the thin `internal/client` HTTP wrapper with the new request/response fields (pointer fields for optional values so PATCH bodies omit unset fields), then wire the two Framework resources to the new client surface. Webhook `status` and the readable domain fields round-trip via GET for drift detection; `tls`/`custom_return_path` are write-only (API never returns them) and are preserved from config/state.

**Tech Stack:** Go, `terraform-plugin-framework` v1.18.0, `net/http`, `net/http/httptest` for unit tests, `terraform-plugin-testing` for acceptance tests.

## Global Constraints

- Module path: `github.com/jhoward321/terraform-provider-resend` (use in all imports).
- Go 1.25+ (`go.mod` declares `go 1.25.7`).
- **One new dependency:** `github.com/hashicorp/terraform-plugin-framework-validators`, used for `stringvalidator.OneOf` on the closed enums only — webhook `status`, domain `tls`, and `capabilities.sending`/`receiving`. Leave `region` and webhook `events` unvalidated (they grow over time). The dependency is added in Task 2 (`go get ...@latest` + `go mod tidy`) and reused in Task 4.
- The "at least one capability enabled" cross-field rule is NOT expressible with per-field `OneOf`; leave it to the API to enforce and surface the API error.
- Optional booleans/strings/objects in **request** structs use pointer types + `omitempty` so unset fields are omitted from JSON and a deliberate `false` is distinguishable from "unset".
- `tls` and `custom_return_path` are write-only in the Resend API: never read them back; leave their state untouched in Read.
- Follow existing code style: table-free `//nolint:errcheck` on `resp.Body.Close()`, `decodeResponse[T]` generic helper, one httptest server per unit test.
- Run `go build ./...` and `go test ./...` (which run without env vars) after each implementation step. Acceptance tests are gated behind `TF_ACC` + `RESEND_API_KEY` and are not expected to run in this environment.
- Work happens on branch `feat/api-catchup` (already created off `master`). Commit per task.

## File Structure

- `internal/client/client.go` — add `Capabilities` struct; extend `Webhook`, `UpdateWebhookRequest`, `CreateDomainRequest`, `Domain`; add `UpdateDomainRequest` + `UpdateDomain` method.
- `internal/client/client_test.go` — unit tests for the new client behavior.
- `internal/resources/webhook.go` — `status` attribute + create/read/update wiring.
- `internal/resources/webhook_test.go` — acceptance test extension for `status`.
- `internal/resources/domain.go` — new attributes, capability helpers, real `Update`.
- `internal/resources/domain_test.go` — acceptance test extension for new fields.
- `docs/resources/webhook.md`, `docs/resources/domain.md` — document new attributes.

---

## COMMIT 1 — Webhook `status`

### Task 1: Client — webhook `status` field

**Files:**
- Modify: `internal/client/client.go` (the `Webhook` and `UpdateWebhookRequest` types, ~lines 85-96)
- Test: `internal/client/client_test.go`

**Interfaces:**
- Consumes: existing `New`, `WithBaseURL`, `doRequest`, `decodeResponse`.
- Produces:
  - `Webhook.Status string` (JSON `status,omitempty`)
  - `UpdateWebhookRequest.Status string` (JSON `status,omitempty`)

- [ ] **Step 1: Write the failing tests**

Add these two tests to `internal/client/client_test.go`:

```go
func TestGetWebhookStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Webhook{
			ID:       "wh-123",
			Endpoint: "https://example.com/webhook",
			Events:   []string{"email.sent"},
			Status:   "disabled",
		})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.GetWebhook(context.Background(), "wh-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "disabled" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}

func TestUpdateWebhookStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/webhooks/wh-123" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body UpdateWebhookRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		if body.Status != "disabled" {
			t.Errorf("expected status disabled in request, got %q", body.Status)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(Webhook{ID: "wh-123", Status: "disabled"})
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.UpdateWebhook(context.Background(), "wh-123", UpdateWebhookRequest{Status: "disabled"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "disabled" {
		t.Errorf("unexpected status: %s", resp.Status)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/client/ -run 'TestGetWebhookStatus|TestUpdateWebhookStatus' -v`
Expected: FAIL — compile error `unknown field 'Status' in struct literal of type Webhook` (and same for `UpdateWebhookRequest`).

- [ ] **Step 3: Add the `Status` fields**

In `internal/client/client.go`, update the two structs:

```go
type UpdateWebhookRequest struct {
	Endpoint string   `json:"endpoint,omitempty"`
	Events   []string `json:"events,omitempty"`
	Status   string   `json:"status,omitempty"`
}

type Webhook struct {
	ID            string   `json:"id"`
	Endpoint      string   `json:"endpoint"`
	Events        []string `json:"events"`
	CreatedAt     string   `json:"created_at"`
	SigningSecret string   `json:"signing_secret,omitempty"`
	Status        string   `json:"status,omitempty"`
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/client/ -run 'TestGetWebhookStatus|TestUpdateWebhookStatus' -v`
Expected: PASS (both).

- [ ] **Step 5: Run the full client suite**

Run: `go test ./internal/client/ -v`
Expected: PASS (all existing tests + the two new ones).

- [ ] **Step 6: Commit**

```bash
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat(client): add webhook status field"
```

---

### Task 2: Webhook resource — manage `status`

**Files:**
- Modify: `internal/resources/webhook.go`
- Modify (acceptance test): `internal/resources/webhook_test.go`
- Modify (docs): `docs/resources/webhook.md`

**Interfaces:**
- Consumes: `client.Webhook.Status`, `client.UpdateWebhookRequest.Status`, `client.Client.{CreateWebhook,GetWebhook,UpdateWebhook}`.
- Produces: `resend_webhook` resource with a managed `status` attribute (Optional+Computed, default `enabled`).

- [ ] **Step 1: Add the `Status` field to the model**

In `internal/resources/webhook.go`, add to `WebhookResourceModel`:

```go
type WebhookResourceModel struct {
	ID            types.String `tfsdk:"id"`
	URL           types.String `tfsdk:"url"`
	EventTypes    types.List   `tfsdk:"event_types"`
	CreatedAt     types.String `tfsdk:"created_at"`
	SigningSecret types.String `tfsdk:"signing_secret"`
	Status        types.String `tfsdk:"status"`
}
```

- [ ] **Step 2: Add the validators dependency**

Run:
```bash
go get github.com/hashicorp/terraform-plugin-framework-validators@latest
go mod tidy
```
Expected: `go.mod` gains `github.com/hashicorp/terraform-plugin-framework-validators` as a direct dependency. Then `go build ./...` still succeeds.

- [ ] **Step 3: Add the `status` schema attribute, its imports, and its validator**

Add these imports to the import block:

```go
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
```

Then add this attribute inside the `Attributes` map in `Schema` (after `signing_secret`):

```go
"status": schema.StringAttribute{
	Optional:            true,
	Computed:            true,
	MarkdownDescription: "Webhook status. One of `enabled` or `disabled`. Defaults to `enabled`.",
	Default:             stringdefault.StaticString("enabled"),
	Validators: []validator.String{
		stringvalidator.OneOf("enabled", "disabled"),
	},
},
```

- [ ] **Step 4: Set `status` from the API in Create (with create-then-PATCH for `disabled`)**

In `Create`, after `data.SigningSecret = types.StringValue(result.SigningSecret)` and before the "read back full state" GET, insert the conditional PATCH; then set `data.Status` from the follow-up GET. The relevant part of `Create` becomes:

```go
	data.ID = types.StringValue(result.ID)
	data.SigningSecret = types.StringValue(result.SigningSecret)

	// POST does not accept status; if a non-default status was requested, PATCH it.
	if data.Status.ValueString() == "disabled" {
		if _, err := r.client.UpdateWebhook(ctx, result.ID, client.UpdateWebhookRequest{
			Status: "disabled",
		}); err != nil {
			resp.Diagnostics.AddError("Error setting webhook status", err.Error())
			return
		}
	}

	// Create response only returns id and signing_secret, so read back full state.
	webhook, err := r.client.GetWebhook(ctx, result.ID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading webhook after create", err.Error())
		return
	}
	data.URL = types.StringValue(webhook.Endpoint)
	data.CreatedAt = types.StringValue(webhook.CreatedAt)
	data.Status = types.StringValue(webhook.Status)
```

- [ ] **Step 5: Set `status` from the API in Read**

In `Read`, after `data.CreatedAt = types.StringValue(result.CreatedAt)`, add:

```go
	data.Status = types.StringValue(result.Status)
```

- [ ] **Step 6: Send and set `status` in Update**

In `Update`, add `Status` to the PATCH request and set it after the read-back. The `UpdateWebhook` call becomes:

```go
	_, err := r.client.UpdateWebhook(ctx, data.ID.ValueString(), client.UpdateWebhookRequest{
		Endpoint: data.URL.ValueString(),
		Events:   eventTypes,
		Status:   data.Status.ValueString(),
	})
```

and after `data.CreatedAt = types.StringValue(webhook.CreatedAt)` add:

```go
	data.Status = types.StringValue(webhook.Status)
```

- [ ] **Step 7: Build to verify it compiles**

Run: `go build ./...`
Expected: no output (success).

- [ ] **Step 8: Extend the acceptance test**

In `internal/resources/webhook_test.go`, update the first step's config and checks to include `status`, and add a third config step that toggles it. Replace the two config `TestStep`s (the create step and the update step) with:

```go
			{
				Config: `
resource "resend_webhook" "test" {
  url         = "https://example.com/webhook-test"
  event_types = ["email.sent"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("resend_webhook.test", "id"),
					resource.TestCheckResourceAttr("resend_webhook.test", "url", "https://example.com/webhook-test"),
					resource.TestCheckResourceAttr("resend_webhook.test", "status", "enabled"),
					resource.TestCheckResourceAttrSet("resend_webhook.test", "signing_secret"),
				),
			},
			{
				Config: `
resource "resend_webhook" "test" {
  url         = "https://example.com/webhook-updated"
  event_types = ["email.sent", "email.delivered"]
  status      = "disabled"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("resend_webhook.test", "url", "https://example.com/webhook-updated"),
					resource.TestCheckResourceAttr("resend_webhook.test", "status", "disabled"),
				),
			},
```

- [ ] **Step 9: Run unit tests and vet**

Run: `go test ./... && go vet ./...`
Expected: `Go test: ... passed`, no vet errors. (Acceptance tests skip without `RESEND_API_KEY`.)

- [ ] **Step 10: Update docs**

In `docs/resources/webhook.md`, add `status` to the documented attributes. Add this row/entry in the same style as the existing optional attributes (adjust wording to match the file's format):

```markdown
- `status` (String) Webhook status. One of `enabled` or `disabled`. Defaults to `enabled`.
```

- [ ] **Step 11: Commit**

```bash
git add go.mod go.sum internal/resources/webhook.go internal/resources/webhook_test.go docs/resources/webhook.md
git commit -m "feat(webhook): manage enabled/disabled status"
```

---

## COMMIT 2 — Domain modernization

### Task 3: Client — domain fields + `UpdateDomain`

**Files:**
- Modify: `internal/client/client.go`
- Test: `internal/client/client_test.go`

**Interfaces:**
- Consumes: `doRequest`, `decodeResponse`, existing `GetDomain`.
- Produces:
  - `type Capabilities struct { Sending string; Receiving string }`
  - `CreateDomainRequest` extended with `CustomReturnPath *string`, `OpenTracking *bool`, `ClickTracking *bool`, `TrackingSubdomain *string`, `TLS *string`, `Capabilities *Capabilities`
  - `type UpdateDomainRequest struct { OpenTracking *bool; ClickTracking *bool; TrackingSubdomain *string; TLS *string; Capabilities *Capabilities }`
  - `Domain` extended with `OpenTracking bool`, `ClickTracking bool`, `TrackingSubdomain string`, `Capabilities Capabilities`
  - `func (c *Client) UpdateDomain(ctx context.Context, id string, req UpdateDomainRequest) (*Domain, error)` — PATCH then GET

- [ ] **Step 1: Write the failing tests**

Add to `internal/client/client_test.go`:

```go
func TestGetDomainNewFields(t *testing.T) {
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
			"open_tracking": true,
			"click_tracking": false,
			"tracking_subdomain": "track",
			"capabilities": {"sending": "enabled", "receiving": "disabled"},
			"records": []
		}`))
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	resp, err := c.GetDomain(context.Background(), "domain-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.OpenTracking {
		t.Error("expected open_tracking true")
	}
	if resp.ClickTracking {
		t.Error("expected click_tracking false")
	}
	if resp.TrackingSubdomain != "track" {
		t.Errorf("unexpected tracking_subdomain: %s", resp.TrackingSubdomain)
	}
	if resp.Capabilities.Sending != "enabled" || resp.Capabilities.Receiving != "disabled" {
		t.Errorf("unexpected capabilities: %+v", resp.Capabilities)
	}
}

func TestCreateDomainSendsNewFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/domains" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var raw map[string]any
		_ = json.NewDecoder(r.Body).Decode(&raw)
		if raw["tls"] != "enforced" {
			t.Errorf("expected tls enforced, got %v", raw["tls"])
		}
		if raw["click_tracking"] != true {
			t.Errorf("expected click_tracking true, got %v", raw["click_tracking"])
		}
		if _, ok := raw["open_tracking"]; ok {
			t.Errorf("open_tracking should be omitted when unset, got %v", raw["open_tracking"])
		}
		caps, _ := raw["capabilities"].(map[string]any)
		if caps == nil || caps["receiving"] != "enabled" {
			t.Errorf("expected capabilities.receiving enabled, got %v", raw["capabilities"])
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":"domain-123","name":"example.com","status":"pending","capabilities":{"sending":"enabled","receiving":"enabled"}}`))
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	clickTrue := true
	tlsEnforced := "enforced"
	_, err := c.CreateDomain(context.Background(), CreateDomainRequest{
		Name:          "example.com",
		ClickTracking: &clickTrue,
		TLS:           &tlsEnforced,
		Capabilities:  &Capabilities{Sending: "enabled", Receiving: "enabled"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDomain(t *testing.T) {
	var patched bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/domains/domain-123":
			patched = true
			var body UpdateDomainRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body.OpenTracking == nil || *body.OpenTracking != true {
				t.Errorf("expected open_tracking true in PATCH body, got %+v", body.OpenTracking)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"object":"domain","id":"domain-123"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/domains/domain-123":
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"domain-123","name":"example.com","status":"verified","open_tracking":true,"capabilities":{"sending":"enabled","receiving":"disabled"},"records":[]}`))
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	c := New("test-key", WithBaseURL(server.URL))
	openTrue := true
	resp, err := c.UpdateDomain(context.Background(), "domain-123", UpdateDomainRequest{OpenTracking: &openTrue})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patched {
		t.Error("expected a PATCH request to be made")
	}
	if !resp.OpenTracking {
		t.Error("expected open_tracking true from follow-up GET")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/client/ -run 'TestGetDomainNewFields|TestCreateDomainSendsNewFields|TestUpdateDomain' -v`
Expected: FAIL — compile errors for unknown fields (`OpenTracking`, `Capabilities`, etc.) and undefined `UpdateDomainRequest` / `c.UpdateDomain`.

- [ ] **Step 3: Add the `Capabilities` type and extend the domain types**

In `internal/client/client.go`, replace the `CreateDomainRequest`, `Domain`, and `DNSRecord` block's `CreateDomainRequest`/`Domain` (lines ~56-68) so it reads:

```go
type Capabilities struct {
	Sending   string `json:"sending,omitempty"`
	Receiving string `json:"receiving,omitempty"`
}

type CreateDomainRequest struct {
	Name              string        `json:"name"`
	Region            string        `json:"region,omitempty"`
	CustomReturnPath  *string       `json:"custom_return_path,omitempty"`
	OpenTracking      *bool         `json:"open_tracking,omitempty"`
	ClickTracking     *bool         `json:"click_tracking,omitempty"`
	TrackingSubdomain *string       `json:"tracking_subdomain,omitempty"`
	TLS               *string       `json:"tls,omitempty"`
	Capabilities      *Capabilities `json:"capabilities,omitempty"`
}

type UpdateDomainRequest struct {
	OpenTracking      *bool         `json:"open_tracking,omitempty"`
	ClickTracking     *bool         `json:"click_tracking,omitempty"`
	TrackingSubdomain *string       `json:"tracking_subdomain,omitempty"`
	TLS               *string       `json:"tls,omitempty"`
	Capabilities      *Capabilities `json:"capabilities,omitempty"`
}

type Domain struct {
	ID                string       `json:"id"`
	Name              string       `json:"name"`
	Status            string       `json:"status"`
	Region            string       `json:"region"`
	CreatedAt         string       `json:"created_at"`
	OpenTracking      bool         `json:"open_tracking"`
	ClickTracking     bool         `json:"click_tracking"`
	TrackingSubdomain string       `json:"tracking_subdomain"`
	Capabilities      Capabilities `json:"capabilities"`
	Records           []DNSRecord  `json:"records"`
}
```

(Leave the existing `DNSRecord` struct unchanged.)

- [ ] **Step 4: Add the `UpdateDomain` method**

In `internal/client/client.go`, in the "Domain methods" section (after `GetDomain`), add:

```go
func (c *Client) UpdateDomain(ctx context.Context, id string, req UpdateDomainRequest) (*Domain, error) {
	resp, err := c.doRequest(ctx, http.MethodPatch, "domains/"+id, req)
	if err != nil {
		return nil, err
	}
	resp.Body.Close() //nolint:errcheck
	// PATCH returns only {object, id}; GET for full state.
	return c.GetDomain(ctx, id)
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/client/ -run 'TestGetDomainNewFields|TestCreateDomainSendsNewFields|TestUpdateDomain' -v`
Expected: PASS (all three).

- [ ] **Step 6: Run the full client suite**

Run: `go test ./internal/client/ -v`
Expected: PASS (existing + new; existing `TestGetDomain` still passes because the new `Domain` fields default to zero values).

- [ ] **Step 7: Commit**

```bash
git add internal/client/client.go internal/client/client_test.go
git commit -m "feat(client): domain settings fields and UpdateDomain (PATCH)"
```

---

### Task 4: Domain resource — new attributes + real Update

**Files:**
- Modify: `internal/resources/domain.go`
- Modify (acceptance test): `internal/resources/domain_test.go`
- Modify (docs): `docs/resources/domain.md`

**Interfaces:**
- Consumes: `client.Capabilities`, extended `client.CreateDomainRequest`/`Domain`, `client.UpdateDomainRequest`, `client.Client.{CreateDomain,GetDomain,UpdateDomain}`, existing `setDNSRecordState`.
- Produces: `resend_domain` with `open_tracking`, `click_tracking`, `tracking_subdomain`, `tls`, `custom_return_path`, and nested `capabilities` attributes, plus a working `Update`.

- [ ] **Step 1: Extend the model**

In `internal/resources/domain.go`, replace `DomainResourceModel`:

```go
type DomainResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Name              types.String `tfsdk:"name"`
	Region            types.String `tfsdk:"region"`
	Status            types.String `tfsdk:"status"`
	CreatedAt         types.String `tfsdk:"created_at"`
	CustomReturnPath  types.String `tfsdk:"custom_return_path"`
	OpenTracking      types.Bool   `tfsdk:"open_tracking"`
	ClickTracking     types.Bool   `tfsdk:"click_tracking"`
	TrackingSubdomain types.String `tfsdk:"tracking_subdomain"`
	TLS               types.String `tfsdk:"tls"`
	Capabilities      types.Object `tfsdk:"capabilities"`
	SPFMXRecord       types.Object `tfsdk:"spf_mx_record"`
	SPFTXTRecord      types.Object `tfsdk:"spf_txt_record"`
	DKIMRecords       types.List   `tfsdk:"dkim_records"`
}
```

- [ ] **Step 2: Add plan-modifier and validator imports**

The `terraform-plugin-framework-validators` dependency was already added in Task 2, so no `go get` is needed here. Add these imports to the import block in `internal/resources/domain.go`:

```go
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
```

- [ ] **Step 3: Add the new schema attributes**

In `Schema`, add these attributes to the `Attributes` map (place them after `created_at`, before `spf_mx_record`):

```go
				"custom_return_path": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "Subdomain for the custom Return-Path (SPF/DMARC). Defaults to `send`. Create-only. Write-only in the Resend API (not returned on read), so drift is not detected.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.RequiresReplace(),
					},
				},
				"open_tracking": schema.BoolAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "Track email opens. Updated in place.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"click_tracking": schema.BoolAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "Track clicks within HTML email bodies. Updated in place.",
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
				},
				"tracking_subdomain": schema.StringAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "Custom subdomain used for open/click tracking. Updated in place.",
					PlanModifiers: []planmodifier.String{
						stringplanmodifier.UseStateForUnknown(),
					},
				},
				"tls": schema.StringAttribute{
					Optional:            true,
					MarkdownDescription: "TLS enforcement policy. One of `opportunistic` or `enforced`. Updated in place. Write-only in the Resend API (not returned on read), so drift is not detected.",
					Validators: []validator.String{
						stringvalidator.OneOf("opportunistic", "enforced"),
					},
				},
				"capabilities": schema.SingleNestedAttribute{
					Optional:            true,
					Computed:            true,
					MarkdownDescription: "Domain sending/receiving capabilities. At least one must be `enabled` (enforced by the API).",
					PlanModifiers: []planmodifier.Object{
						objectplanmodifier.UseStateForUnknown(),
					},
					Attributes: map[string]schema.Attribute{
						"sending": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "`enabled` or `disabled`. Defaults to `enabled`.",
							Validators: []validator.String{
								stringvalidator.OneOf("enabled", "disabled"),
							},
						},
						"receiving": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "`enabled` or `disabled`. Defaults to `disabled`.",
							Validators: []validator.String{
								stringvalidator.OneOf("enabled", "disabled"),
							},
						},
					},
				},
```

- [ ] **Step 4: Add capability helpers and the shared state setter**

At the bottom of `internal/resources/domain.go`, add:

```go
var capabilitiesAttrTypes = map[string]attr.Type{
	"sending":   types.StringType,
	"receiving": types.StringType,
}

// capabilitiesToAPI builds a *client.Capabilities from the nested object, or
// nil when the object is null/unknown (so it is omitted from the request).
func capabilitiesToAPI(obj types.Object) *client.Capabilities {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}
	attrs := obj.Attributes()
	caps := &client.Capabilities{}
	if s, ok := attrs["sending"].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		caps.Sending = s.ValueString()
	}
	if s, ok := attrs["receiving"].(types.String); ok && !s.IsNull() && !s.IsUnknown() {
		caps.Receiving = s.ValueString()
	}
	return caps
}

func capabilitiesToObject(c client.Capabilities) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(capabilitiesAttrTypes, map[string]attr.Value{
		"sending":   types.StringValue(c.Sending),
		"receiving": types.StringValue(c.Receiving),
	})
}

// setDomainState copies all API-readable fields from result into data. It does
// NOT touch TLS or CustomReturnPath, which the API never returns (write-only).
func setDomainState(result *client.Domain, data *DomainResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	data.ID = types.StringValue(result.ID)
	data.Name = types.StringValue(result.Name)
	data.Status = types.StringValue(result.Status)
	data.Region = types.StringValue(result.Region)
	data.CreatedAt = types.StringValue(result.CreatedAt)
	data.OpenTracking = types.BoolValue(result.OpenTracking)
	data.ClickTracking = types.BoolValue(result.ClickTracking)
	data.TrackingSubdomain = types.StringValue(result.TrackingSubdomain)

	capsObj, d := capabilitiesToObject(result.Capabilities)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	data.Capabilities = capsObj

	diags.Append(setDNSRecordState(result.Records, data)...)
	return diags
}
```

- [ ] **Step 5: Rewrite Create to send new fields and use `setDomainState`**

Replace the body of `Create` (from building `apiReq` through the final `State.Set`) with:

```go
	apiReq := client.CreateDomainRequest{
		Name:   data.Name.ValueString(),
		Region: data.Region.ValueString(),
	}
	if !data.CustomReturnPath.IsNull() {
		v := data.CustomReturnPath.ValueString()
		apiReq.CustomReturnPath = &v
	}
	if !data.OpenTracking.IsNull() && !data.OpenTracking.IsUnknown() {
		v := data.OpenTracking.ValueBool()
		apiReq.OpenTracking = &v
	}
	if !data.ClickTracking.IsNull() && !data.ClickTracking.IsUnknown() {
		v := data.ClickTracking.ValueBool()
		apiReq.ClickTracking = &v
	}
	if !data.TrackingSubdomain.IsNull() && !data.TrackingSubdomain.IsUnknown() {
		v := data.TrackingSubdomain.ValueString()
		apiReq.TrackingSubdomain = &v
	}
	if !data.TLS.IsNull() {
		v := data.TLS.ValueString()
		apiReq.TLS = &v
	}
	apiReq.Capabilities = capabilitiesToAPI(data.Capabilities)

	result, err := r.client.CreateDomain(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Error creating domain", err.Error())
		return
	}

	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// TLS and CustomReturnPath are preserved from the plan (write-only in the API).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
```

- [ ] **Step 6: Simplify Read to use `setDomainState`**

Replace the body of `Read` (from the `GetDomain` call to the end) with:

```go
	result, err := r.client.GetDomain(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading domain", err.Error())
		return
	}

	// TLS and CustomReturnPath are left as-is in state (write-only in the API).
	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
```

- [ ] **Step 7: Implement the real Update method**

Replace the no-op `Update` method with:

```go
func (r *DomainResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DomainResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updReq := client.UpdateDomainRequest{}
	if !data.OpenTracking.IsNull() && !data.OpenTracking.IsUnknown() {
		v := data.OpenTracking.ValueBool()
		updReq.OpenTracking = &v
	}
	if !data.ClickTracking.IsNull() && !data.ClickTracking.IsUnknown() {
		v := data.ClickTracking.ValueBool()
		updReq.ClickTracking = &v
	}
	if !data.TrackingSubdomain.IsNull() && !data.TrackingSubdomain.IsUnknown() {
		v := data.TrackingSubdomain.ValueString()
		updReq.TrackingSubdomain = &v
	}
	if !data.TLS.IsNull() {
		v := data.TLS.ValueString()
		updReq.TLS = &v
	}
	updReq.Capabilities = capabilitiesToAPI(data.Capabilities)

	result, err := r.client.UpdateDomain(ctx, data.ID.ValueString(), updReq)
	if err != nil {
		resp.Diagnostics.AddError("Error updating domain", err.Error())
		return
	}

	resp.Diagnostics.Append(setDomainState(result, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// TLS and CustomReturnPath are preserved from the plan (write-only in the API).
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

- [ ] **Step 8: Build to verify it compiles**

Run: `go build ./...`
Expected: no output (success). If the compiler reports the old `Create` still assigns fields removed from the model, ensure the entire old field-by-field block in `Create` (lines that set `data.ID`, `data.Name`, `data.Status`, `data.Region`, `data.CreatedAt`, and the old `setDNSRecordState` call) was replaced by the Step 5 version.

- [ ] **Step 9: Run unit tests and vet**

Run: `go test ./... && go vet ./...`
Expected: `Go test: ... passed`, no vet errors.

- [ ] **Step 10: Extend the acceptance test**

In `internal/resources/domain_test.go`, replace the single config `TestStep` (before the import step) with a create step that sets the new fields and an update step that changes them:

```go
			{
				Config: fmt.Sprintf(`
resource "resend_domain" "test" {
  name           = %q
  open_tracking  = true
  click_tracking = true
  tls            = "enforced"
}
`, domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("resend_domain.test", "id"),
					resource.TestCheckResourceAttr("resend_domain.test", "name", domainName),
					resource.TestCheckResourceAttr("resend_domain.test", "open_tracking", "true"),
					resource.TestCheckResourceAttr("resend_domain.test", "click_tracking", "true"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "capabilities.sending"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_mx_record.type"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "dkim_records.0.type"),
				),
			},
			{
				Config: fmt.Sprintf(`
resource "resend_domain" "test" {
  name           = %q
  open_tracking  = false
  click_tracking = true
  tls            = "enforced"
}
`, domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("resend_domain.test", "open_tracking", "false"),
				),
			},
```

Also update the import `TestStep` to ignore the write-only fields (they are not returned by the API and so cannot be verified on import):

```go
			{
				ResourceName:            "resend_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tls", "custom_return_path"},
			},
```

- [ ] **Step 11: Update docs**

In `docs/resources/domain.md`, document the new attributes in the same style the file uses for existing ones:

```markdown
- `open_tracking` (Boolean) Track email opens. Updated in place.
- `click_tracking` (Boolean) Track clicks within HTML email bodies. Updated in place.
- `tracking_subdomain` (String) Custom subdomain used for open/click tracking. Updated in place.
- `tls` (String) TLS enforcement policy: `opportunistic` or `enforced`. Write-only (not read back).
- `custom_return_path` (String) Subdomain for the custom Return-Path. Defaults to `send`. Create-only, write-only (not read back).
- `capabilities` (Attributes) Domain sending/receiving capabilities (`sending`, `receiving`), each `enabled` or `disabled`. At least one must be `enabled`.
```

- [ ] **Step 12: Commit**

```bash
git add internal/resources/domain.go internal/resources/domain_test.go docs/resources/domain.md
git commit -m "feat(domain): in-place updates and new domain settings"
```

---

## Final verification

- [ ] **Step 1: Full build, test, vet, lint**

Run:
```bash
go build ./... && go test ./... && go vet ./...
make lint
```
Expected: build clean; all unit tests pass; vet clean; `golangci-lint` clean.

- [ ] **Step 2: Confirm the branch history**

Run: `git log --oneline master..HEAD`
Expected: the design/spec commits plus the four feature commits (client webhook, webhook resource, client domain, domain resource), all on `feat/api-catchup`.

---

## Self-Review Notes (for the plan author)

- **Spec coverage:** client webhook `status` (Task 1) ✔; webhook resource managed status incl. create-then-PATCH + drift read, `status` validated (Task 2) ✔; client domain fields + `UpdateDomain` PATCH→GET (Task 3) ✔; domain new attributes with correct Optional/Computed/RequiresReplace split, real Update, write-only handling of `tls`/`custom_return_path`, `tls`/`capabilities` validated (Task 4) ✔; unit tests first throughout ✔; acceptance test extensions ✔; docs updates ✔; one new dependency (`terraform-plugin-framework-validators`, added Task 2) ✔.
- **Naming consistency:** `Capabilities{Sending,Receiving}`, `UpdateDomainRequest`, `UpdateDomain`, `setDomainState`, `capabilitiesToAPI`, `capabilitiesToObject`, `capabilitiesAttrTypes` used identically across Tasks 3–4.
- **Acceptance tests** require `TF_ACC=1` + `RESEND_API_KEY` and a free-plan domain slot; they validate the wire-format assumptions against the live API and are not expected to run in CI without those secrets.

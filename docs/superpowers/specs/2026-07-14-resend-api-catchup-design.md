# Design: Resend API Catch-Up — Webhook `status` + Domain Modernization

**Date:** 2026-07-14
**Status:** Approved (pending spec review)

## Background

The provider's existing endpoints and fields all still match the live Resend
API — nothing is broken. However, the API has grown, and two areas of the
provider have drifted behind new, non-breaking capabilities:

1. **Webhooks** gained a `status` field (`enabled`/`disabled`) that is readable
   via `GET` and settable via `PATCH`. The provider doesn't model it, so a
   webhook disabled outside Terraform is never reconciled.
2. **Domains** gained an in-place update endpoint (`PATCH /domains/{id}`) plus
   several new settings. The provider treats domains as create/delete-only and
   exposes none of the new fields.

This spec covers bringing both resources up to date. Adding entirely new
resources (segments, contacts, topics, templates, broadcasts) is explicitly
**out of scope** for this work.

## Verified API Contracts

Confirmed against the official Resend docs on 2026-07-14:

### Domains

`POST /domains` request accepts:

| Field | Type | Default | Allowed values |
|---|---|---|---|
| `name` | string | required | — |
| `region` | string | `us-east-1` | `us-east-1`, `eu-west-1`, `sa-east-1`, `ap-northeast-1` |
| `custom_return_path` | string | `send` | any subdomain |
| `open_tracking` | boolean | — | true/false |
| `click_tracking` | boolean | — | true/false |
| `tracking_subdomain` | string | — | any subdomain |
| `tls` | string | `opportunistic` | `opportunistic`, `enforced` |
| `capabilities.sending` | string | `enabled` | `enabled`, `disabled` |
| `capabilities.receiving` | string | `disabled` | `enabled`, `disabled` |

At least one capability must remain `enabled` (enforced server-side).

`GET /domains/{id}` response returns: `object`, `id`, `name`, `status`,
`created_at`, `region`, `open_tracking`, `click_tracking`,
`tracking_subdomain`, `capabilities` (`{sending, receiving}`), and `records[]`
(`record`, `type`, `name`, `value`, `priority` [MX only], `ttl`, `status`).

**Write-only fields:** `tls` and `custom_return_path` are accepted in the
request but are **NOT** returned by `GET` or the create response. They cannot be
read back or drift-detected.

`PATCH /domains/{id}` accepts `click_tracking`, `open_tracking`,
`tracking_subdomain`, `tls`, and `capabilities`. It returns only
`{object, id}`, so a follow-up `GET` is required for full state (same pattern
the webhook create already uses). `name`, `region`, and `custom_return_path`
are **not** accepted by `PATCH`.

### Webhooks

- `POST /webhooks` accepts only `endpoint` and `events`; response returns only
  `object`, `id`, `signing_secret`. (No `status` at create time.)
- `GET /webhooks/{id}` returns `object`, `id`, `created_at`, `status`,
  `endpoint`, `events`, `signing_secret`.
- `PATCH /webhooks/{id}` accepts `endpoint`, `events`, and `status`
  (`enabled`/`disabled`); returns only `{object, id}`.

## Design

### Client layer (`internal/client/client.go`)

Add a `Capabilities` struct (`Sending string`, `Receiving string`).

Extend request/response types with **pointer fields** for optional
booleans/strings/objects, so an unset field is omitted from JSON and a
deliberate `false` is distinguishable from "unset" (critical for PATCH):

- `CreateDomainRequest`: add `CustomReturnPath *string`, `OpenTracking *bool`,
  `ClickTracking *bool`, `TrackingSubdomain *string`, `TLS *string`,
  `Capabilities *Capabilities`.
- New `UpdateDomainRequest`: `ClickTracking *bool`, `OpenTracking *bool`,
  `TrackingSubdomain *string`, `TLS *string`, `Capabilities *Capabilities`.
- `Domain` (GET response): add `OpenTracking bool`, `ClickTracking bool`,
  `TrackingSubdomain string`, `Capabilities Capabilities`. Deliberately no
  `TLS`/`CustomReturnPath` (never returned).
- `Webhook`: add `Status string`.
- `UpdateWebhookRequest`: add `Status string` (`json:"status,omitempty"`).
- `CreateWebhookRequest`: unchanged.

New method `UpdateDomain(ctx, id, UpdateDomainRequest) (*Domain, error)`:
issue `PATCH /domains/{id}`, then `GET` for the full object.

### Webhook resource (`internal/resources/webhook.go`)

- Add `Status types.String` to `WebhookResourceModel`.
- Add a `status` schema attribute: **Optional + Computed**, default `enabled`,
  no `RequiresReplace`. Allowed values (`enabled`/`disabled`) are documented in
  the attribute description and enforced by the API — consistent with how the
  existing `resend_api_key` `permission` enum is handled (no client-side
  validator, no new dependency).
- **Create**: `POST` (server creates as `enabled`) → `GET` for full state → if
  the plan's `status` is `disabled`, `PATCH` to disabled and re-read. Populate
  `status` from the final `GET`.
- **Read**: populate `status` from `GET` (enables drift reconciliation).
- **Update**: existing flow re-reads after PATCH; add `Status` to the PATCH
  body.

### Domain resource (`internal/resources/domain.go`)

New attributes, split by how the API treats them:

| Attribute | Schema | Update behavior |
|---|---|---|
| `open_tracking` | Optional + Computed (bool) | in-place PATCH; read-back drift detection |
| `click_tracking` | Optional + Computed (bool) | in-place PATCH; read-back drift detection |
| `tracking_subdomain` | Optional + Computed (string) | in-place PATCH; read-back drift detection |
| `capabilities` | Optional + Computed (nested: `sending`, `receiving`) | in-place PATCH; read-back drift detection |
| `tls` | Optional, **not** Computed (string) | in-place PATCH; **no drift detection** |
| `custom_return_path` | Optional + `RequiresReplace`, **not** Computed (string) | create-only; **no drift detection** |

- `name` and `region` remain `RequiresReplace` (not in the PATCH body).
- The `Update` method (currently a no-op) becomes real: build an
  `UpdateDomainRequest` from the plan's in-place-updatable fields, call
  `UpdateDomain`, and set state from the returned/re-read object. Every
  currently-configured in-place field is sent on each PATCH (idempotent);
  fields that are null/unset in the plan map to nil pointers and are omitted.
  `tls` is sent on update even though it can't be read back.
- **Read** populates the readable new fields from `GET`; it must **not** touch
  `tls` or `custom_return_path` (leaves whatever is in state), since the API
  never returns them.
- No client-side enum validators (consistent with the existing provider, which
  documents `permission` values rather than validating them). Allowed values
  for `tls` (`opportunistic`/`enforced`) and `capabilities.sending`/`receiving`
  (`enabled`/`disabled`) are documented in the attribute descriptions and
  enforced by the API.
- The "at least one capability enabled" invariant is likewise left to the API
  to enforce; the provider surfaces the API error rather than duplicating a
  cross-field rule (single source of truth). `capabilities` is only sent when
  it is known (configured or carried from prior state).
- Existing DNS-record attributes (`spf_mx_record`, `spf_txt_record`,
  `dkim_records`) are unchanged.

### Behavioral caveat (accepted)

Because `tls` and `custom_return_path` are write-only in the Resend API,
Terraform cannot detect out-of-band drift on them; they behave as
set-and-forget configuration. This is a limitation of the API, not the
provider. All other new fields round-trip normally.

## Testing

- **Unit tests** (`internal/client`, `httptest`), written first (TDD):
  - `UpdateDomain` (PATCH → GET round-trip).
  - Domain create/get carrying the new fields (including `capabilities`).
  - Webhook `status`: create-then-PATCH-to-`disabled`, update, and read paths.
- **Acceptance tests** (`internal/resources`, gated on `TF_ACC` +
  `RESEND_API_KEY`, auto-skip when unset): extend existing domain/webhook tests
  to set and then update the new fields. These validate the wire-format
  assumptions against the real API.
- **Docs:** update `docs/resources/domain.md` and `docs/resources/webhook.md`
  (and any `examples/`) for the new attributes.

## Git / PR Structure

One branch `feat/api-catchup` off `master`, with two focused commits:

1. `feat(webhook): manage enabled/disabled status`
2. `feat(domain): in-place updates and new domain settings`

(Separate from the already-committed `chore/dependency-updates` branch.)

## Out of Scope

- New resources: segments, contacts, topics, templates, broadcasts,
  contact-properties, suppressions.
- Any change to `resend_api_key` (API unchanged).
- Enum validation of webhook `events` or domain `region` (intentionally left
  as free-form pass-through, matching current behavior).

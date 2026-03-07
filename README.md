# Terraform Provider for Resend

A Terraform provider for managing [Resend](https://resend.com) resources — API keys, sending domains, and webhooks.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.22 (to build the provider)

## Usage

```hcl
terraform {
  required_providers {
    resend = {
      source = "jhoward321/resend"
    }
  }
}

provider "resend" {
  api_key = var.resend_api_key # or set RESEND_API_KEY env var
}
```

### API Key

```hcl
resource "resend_api_key" "production" {
  name       = "production"
  permission = "sending_access"
  domain_id  = resend_domain.example.id
}
```

### Domain

```hcl
resource "resend_domain" "example" {
  name   = "mail.example.com"
  region = "us-east-1"
}

# DNS records are available as computed attributes
output "dns_records" {
  value = resend_domain.example.records
}
```

### Webhook

```hcl
resource "resend_webhook" "delivery_events" {
  url         = "https://example.com/webhooks/resend"
  event_types = ["email.sent", "email.delivered", "email.bounced"]
}
```

## Authentication

The provider requires a Resend API key. You can provide it in two ways:

1. Set the `api_key` attribute in the provider block
2. Set the `RESEND_API_KEY` environment variable

## Resources

| Resource | Description |
|---|---|
| `resend_api_key` | Create and delete API keys. The key token is stored in state as a sensitive value. |
| `resend_domain` | Register sending domains. Returns DNS records needed for verification. |
| `resend_webhook` | Configure webhook endpoints with specific event type subscriptions. Supports in-place updates. |

## Building

```bash
git clone https://github.com/jake/terraform-provider-resend.git
cd terraform-provider-resend
make build
```

## Testing

Unit tests (no API key required):

```bash
make test
```

Acceptance tests (requires a Resend API key):

```bash
RESEND_API_KEY=re_xxx make testacc
```

## Releasing

Releases are automated via GitHub Actions.

**Via git tag:**

```bash
git tag v0.1.0
git push origin v0.1.0
```

**Via GitHub UI:**

Go to Actions → Release → Run workflow, enter the version tag (e.g., `v0.1.0`).

Both methods run acceptance tests before building. If tests pass, GoReleaser creates a signed GitHub Release that the Terraform Registry syncs automatically.

### First-time setup

1. Generate a GPG key: `gpg --full-generate-key`
2. Export the private key: `gpg --armor --export-secret-keys <KEY_ID>`
3. Add these GitHub repo secrets:
   - `GPG_PRIVATE_KEY` — the exported private key
   - `PASSPHRASE` — the GPG key passphrase
   - `RESEND_API_KEY` — Resend API key for acceptance tests
4. Add the public key to [registry.terraform.io](https://registry.terraform.io) under your account settings
5. Link the repo on the Terraform Registry via Publish → Provider

## License

[Mozilla Public License v2.0](LICENSE)

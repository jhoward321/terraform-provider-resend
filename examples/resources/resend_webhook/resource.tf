resource "resend_webhook" "example" {
  url         = "https://example.com/webhook"
  event_types = ["email.sent", "email.delivered", "email.bounced"]
}

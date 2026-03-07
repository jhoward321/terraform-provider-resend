resource "resend_api_key" "example" {
  name       = "my-api-key"
  permission = "sending_access"
  domain_id  = resend_domain.example.id
}

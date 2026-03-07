# examples/provider/main.tf
terraform {
  required_providers {
    resend = {
      source = "jhoward321/resend"
    }
  }
}

provider "resend" {
  # Set RESEND_API_KEY environment variable or:
  # api_key = "re_..."
}

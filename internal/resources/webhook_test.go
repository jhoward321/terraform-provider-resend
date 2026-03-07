package resources_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccWebhookResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
				),
			},
			{
				Config: `
resource "resend_webhook" "test" {
  url         = "https://example.com/webhook-updated"
  event_types = ["email.sent", "email.delivered"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("resend_webhook.test", "url", "https://example.com/webhook-updated"),
				),
			},
		},
	})
}

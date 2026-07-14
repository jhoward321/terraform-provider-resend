package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccDomainResource(t *testing.T) {
	domainName := fmt.Sprintf("tf-acc-%s.example.com", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
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
			{
				ResourceName:            "resend_domain.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"tls", "custom_return_path"},
			},
		},
	})
}

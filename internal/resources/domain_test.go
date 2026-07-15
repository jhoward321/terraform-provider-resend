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
				// The tracking/tls/capabilities settings are intentionally NOT set
				// here. Resend rejects them on a freshly-created, unverified domain
				// (open/click tracking require a verified tracking subdomain, which
				// cannot be achieved in an acceptance test), so exercising them would
				// require a domain that CI can never verify. The request/response
				// handling for those fields is covered by the client unit tests; the
				// computed values are still asserted below as they round-trip on a
				// bare domain.
				Config: fmt.Sprintf(`
resource "resend_domain" "test" {
  name = %q
}
`, domainName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("resend_domain.test", "id"),
					resource.TestCheckResourceAttr("resend_domain.test", "name", domainName),
					resource.TestCheckResourceAttrSet("resend_domain.test", "status"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "open_tracking"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "click_tracking"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_mx_record.type"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_mx_record.name"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_mx_record.value"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_txt_record.type"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_txt_record.name"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "spf_txt_record.value"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "dkim_records.0.type"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "dkim_records.0.name"),
					resource.TestCheckResourceAttrSet("resend_domain.test", "dkim_records.0.value"),
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

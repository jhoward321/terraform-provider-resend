package resources_test

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/jhoward321/terraform-provider-resend/internal/provider"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"resend": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func testAccPreCheck(t *testing.T) {
	if v := os.Getenv("RESEND_API_KEY"); v == "" {
		t.Skip("RESEND_API_KEY must be set for acceptance tests")
	}
}

func TestAccAPIKeyResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "resend_api_key" "test" {
  name = "tf-acc-test"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("resend_api_key.test", "id"),
					resource.TestCheckResourceAttrSet("resend_api_key.test", "token"),
					resource.TestCheckResourceAttr("resend_api_key.test", "name", "tf-acc-test"),
				),
			},
		},
	})
}

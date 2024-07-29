//go:build acceptance
// +build acceptance

package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/xanzy/go-gitlab"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAcc_GitLabApplication_DataSource_Basic(t *testing.T) {
	name := acctest.RandString(10)
	url := "https://my_website.com"
	scopes := "openid"
	confidential := false

	options := &gitlab.CreateApplicationOptions{
		Name:         gitlab.Ptr(name),
		RedirectURI:  gitlab.Ptr(url),
		Scopes:       gitlab.Ptr(scopes),
		Confidential: gitlab.Ptr(confidential),
	}

	application, _, err := testutil.TestGitlabClient.Applications.CreateApplication(options)
	if err != nil {
		t.Errorf("Unable to create gitlab application. Error: %s", err.Error())
	}
	t.Cleanup(func() {
		_, err := testutil.TestGitlabClient.Applications.DeleteApplication(application.ID)
		if err != nil {
			t.Errorf("Error deleting application created for datasource_gitlab_application: %s", err.Error())
		}
	})
	//lintignore:AT001
	resource.ParallelTest(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: fmt.Sprintf(`data "gitlab_application" "test" {
					id = %q
				}`, strconv.Itoa(application.ID)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify id attribute
					resource.TestCheckResourceAttr("data.gitlab_application.test", "id", strconv.Itoa(application.ID)),
					resource.TestCheckResourceAttr("data.gitlab_application.test", "name", name),
					resource.TestCheckResourceAttr("data.gitlab_application.test", "redirect_url", url),
				),
			},
		},
	})
}

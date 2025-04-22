//go:build acceptance
// +build acceptance

package sdk

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	gitlab "gitlab.com/gitlab-org/api/client-go"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/api"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

func TestAccGitlabTopic_basic(t *testing.T) {
	var topic gitlab.Topic
	rInt := acctest.RandInt()

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		CheckDestroy:             testAccCheckGitlabTopicDestroy,
		Steps: []resource.TestStep{
			// Create a topic with default options
			{
				Config: fmt.Sprintf(`
				resource "gitlab_topic" "foo" {
				  name  = "foo-req-%d"
				  title = "Foo Req %d"
				}`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTopicExists("gitlab_topic.foo", &topic),
					testAccCheckGitlabTopicAttributes(&topic, &testAccGitlabTopicExpectedAttributes{
						Name: fmt.Sprintf("foo-req-%d", rInt),
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_topic.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the topics values
			{
				Config: fmt.Sprintf(`
				resource "gitlab_topic" "foo" {
				  name        = "foo-full-%d"
				  title       = "Foo Req %d"
				  description = "Terraform acceptance tests"
				}`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTopicExists("gitlab_topic.foo", &topic),
					testAccCheckGitlabTopicAttributes(&topic, &testAccGitlabTopicExpectedAttributes{
						Name:        fmt.Sprintf("foo-full-%d", rInt),
						Description: "Terraform acceptance tests",
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_topic.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update the topics values back to their initial state
			{
				Config: fmt.Sprintf(`
				resource "gitlab_topic" "foo" {
				  name  = "foo-req-%d"
				  title = "Foo Req %d"
				}`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTopicExists("gitlab_topic.foo", &topic),
					testAccCheckGitlabTopicAttributes(&topic, &testAccGitlabTopicExpectedAttributes{
						Name: fmt.Sprintf("foo-req-%d", rInt),
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_topic.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Updating the topic to have a description before it is deleted
			{
				Config: fmt.Sprintf(`
				resource "gitlab_topic" "foo" {
				  name        = "foo-full-%d"
				  title       = "Foo Req %d"
				  description = "Terraform acceptance tests"
				}`, rInt, rInt),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckGitlabTopicExists("gitlab_topic.foo", &topic),
					testAccCheckGitlabTopicAttributes(&topic, &testAccGitlabTopicExpectedAttributes{
						Name:        fmt.Sprintf("foo-full-%d", rInt),
						Description: "Terraform acceptance tests",
					}),
				),
			},
			// Verify import
			{
				ResourceName:      "gitlab_topic.foo",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccGitlabTopic_WithoutAvatarHash(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_topic" "test" {
		name  = "%[1]s"
		title = "%[1]s"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithoutAvatarHash(t, "gitlab_topic.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabTopicDestroy
	resource.Test(t, testCase)
}

func TestAccGitlabTopic_WithAvatar(t *testing.T) {
	testConfig := fmt.Sprintf(`
	resource "gitlab_topic" "test" {
		name  = "%[1]s"
		title = "%[1]s"

		{{.AvatarableAttributeConfig}}
	}
	`, acctest.RandomWithPrefix("acctest"))

	testCase := createAvatarableTestCase_WithAvatar(t, "gitlab_topic.test", testConfig)
	testCase.CheckDestroy = testAccCheckGitlabTopicDestroy
	resource.Test(t, testCase)
}

func testAccCheckGitlabTopicExists(n string, assign *gitlab.Topic) resource.TestCheckFunc {
	return func(s *terraform.State) (err error) {
		defer func() {
			if err != nil {
				err = fmt.Errorf("checking for gitlab topic existence failed: %w", err)
			}
		}()

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not Found: %s", n)
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		topic, _, err := testutil.TestGitlabClient.Topics.GetTopic(id)
		*assign = *topic

		return err
	}
}

type testAccGitlabTopicExpectedAttributes struct {
	Name        string
	Description string
}

func testAccCheckGitlabTopicAttributes(topic *gitlab.Topic, want *testAccGitlabTopicExpectedAttributes) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if topic.Name != want.Name {
			return fmt.Errorf("got name %q; want %q", topic.Name, want.Name)
		}

		if topic.Description != want.Description {
			return fmt.Errorf("got description %q; want %q", topic.Description, want.Description)
		}

		return nil
	}
}

func testAccCheckGitlabTopicDestroy(s *terraform.State) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("destroying gitlab topic failed: %w", err)
		}
	}()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "gitlab_topic" {
			continue
		}

		id, err := strconv.Atoi(rs.Primary.ID)
		if err != nil {
			return err
		}

		topic, _, err := testutil.TestGitlabClient.Topics.GetTopic(id)
		if err == nil {
			if topic != nil && fmt.Sprintf("%d", topic.ID) == rs.Primary.ID {
				return fmt.Errorf("topic %s still exists", rs.Primary.ID)
			}
		}
		if !api.Is404(err) {
			return err
		}
		return nil
	}
	return nil
}

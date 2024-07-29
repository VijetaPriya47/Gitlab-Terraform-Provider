//go:build acceptance
// +build acceptance

package sdk

import (
	"bytes"
	"os"
	"testing"
	"text/template"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"gitlab.com/gitlab-org/terraform-provider-gitlab/internal/provider/testutil"
)

type avatarableAttributeConfig struct {
	AvatarableAttributeConfig string
}

func renderTestConfig(t *testing.T, baseConfigTemplate string, avatarableConfig string) string {
	tmpl, err := template.New("config").Parse(baseConfigTemplate)
	if err != nil {
		t.Fatalf("unable to create config based on template testcase: %v", err)
	}

	cfg := avatarableAttributeConfig{AvatarableAttributeConfig: avatarableConfig}

	var config bytes.Buffer
	if err = tmpl.Execute(&config, cfg); err != nil {
		t.Fatalf("unable to render config based on template testcase: %v", err)
	}

	return config.String()

}

func createAvatarableTestCase_WithoutAvatarHash(t *testing.T, resourceName string, baseConfigTemplate string) resource.TestCase {
	testConfig := renderTestConfig(t, baseConfigTemplate, `avatar = "${path.module}/testdata/avatarable/avatar.png"`)

	// lintignore:AT001
	return resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Create with avatar, but without giving a hash
			{
				Config:             testConfig,
				Check:              resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
				ExpectNonEmptyPlan: true,
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
			// Update the avatar image, but keep the filename to test the `CustomizeDiff` function
			{
				Config:             testConfig,
				Check:              resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
				ExpectNonEmptyPlan: true,
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
		},
	}
}

func createAvatarableTestCase_WithAvatar(t *testing.T, resourceName string, baseConfigTemplate string) resource.TestCase {
	// lintignore:AT001
	return resource.TestCase{
		ProtoV6ProviderFactories: providerFactoriesV6,
		Steps: []resource.TestStep{
			// Create with avatar and providing the hash
			{
				Config: renderTestConfig(t, baseConfigTemplate, `
					avatar      = "${path.module}/testdata/avatarable/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatarable/avatar.png")
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
					resource.TestCheckResourceAttr(resourceName, "avatar_hash", "8d29d9c393facb9d86314eb347a03fde503f2c0422bf55af7df086deb126107e"),
				),
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
			// Update avatar
			{
				Config: renderTestConfig(t, baseConfigTemplate, `
					avatar      = "${path.module}/testdata/avatarable/avatar-update.png"
					avatar_hash = filesha256("${path.module}/testdata/avatarable/avatar-update.png")
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
					resource.TestCheckResourceAttr(resourceName, "avatar_hash", "a58bd926fd3baabd41c56e810f62ade8705d18a4e280fb35764edb4b778444db"),
				),
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
			// Update avatar back to default
			{
				Config: renderTestConfig(t, baseConfigTemplate, `
					avatar      = "${path.module}/testdata/avatarable/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatarable/avatar.png")
				`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
					resource.TestCheckResourceAttr(resourceName, "avatar_hash", "8d29d9c393facb9d86314eb347a03fde503f2c0422bf55af7df086deb126107e"),
				),
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
			// Update the avatar image, but keep the filename to test the `CustomizeDiff` function
			{
				Config: renderTestConfig(t, baseConfigTemplate, `
					avatar      = "${path.module}/testdata/avatarable/avatar.png"
					avatar_hash = filesha256("${path.module}/testdata/avatarable/avatar.png")
				`),
				PreConfig: func() {
					// overwrite the avatar image file
					if err := testutil.CopyFile("testdata/avatarable/avatar.png", "testdata/avatarable/avatar.png.bak"); err != nil {
						t.Fatalf("failed to backup the avatar image file: %v", err)
					}
					if err := testutil.CopyFile("testdata/avatarable/avatar-update.png", "testdata/avatarable/avatar.png"); err != nil {
						t.Fatalf("failed to overwrite the avatar image file: %v", err)
					}
					t.Cleanup(func() {
						if err := os.Rename("testdata/avatarable/avatar.png.bak", "testdata/avatarable/avatar.png"); err != nil {
							t.Fatalf("failed to restore the avatar image file: %v", err)
						}
					})
				},
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet(resourceName, "avatar_url"),
					resource.TestCheckResourceAttr(resourceName, "avatar_hash", "a58bd926fd3baabd41c56e810f62ade8705d18a4e280fb35764edb4b778444db"),
				),
			},
			// Verify import
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"avatar", "avatar_hash", "permanently_remove_on_delete",
				},
			},
		},
	}
}

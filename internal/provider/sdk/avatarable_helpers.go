package sdk

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// avatarableSchema returns a resource schema with the attributes required to support Avatars for GitLab resources.
func avatarableSchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"avatar": {
			Description: "A local path to the avatar image to upload. **Note**: not available for imported resources.",
			Type:        schema.TypeString,
			Optional:    true,
		},
		"avatar_hash": {
			Description:  "The hash of the avatar image. Use `filesha256(\"path/to/avatar.png\")` whenever possible. **Note**: this is used to trigger an update of the avatar. If it's not given, but an avatar is given, the avatar will be updated each time.",
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			RequiredWith: []string{"avatar"},
		},
		"avatar_url": {
			Description: "The URL of the avatar image.",
			Type:        schema.TypeString,
			Computed:    true,
		},
	}
}

// avatarableDiff must be used to properly support the `avatarSchema` attributes in a resource Schema.
func avatarableDiff(ctx context.Context, rd *schema.ResourceDiff, i any) error {
	if _, ok := rd.GetOk("avatar"); ok {
		if v, ok := rd.GetOk("avatar_hash"); !ok || v.(string) == "" {
			if err := rd.SetNewComputed("avatar_hash"); err != nil {
				return err
			}
		}
	}
	return nil
}

type localAvatar struct {
	Filename string
	Image    io.Reader
}

func handleAvatarOnCreate(d *schema.ResourceData) (*localAvatar, error) {
	if v, ok := d.GetOk("avatar"); ok {
		avatarPath := v.(string)
		avatarFile, err := os.Open(avatarPath)
		if err != nil {
			return nil, fmt.Errorf("unable to open avatar file %s: %s", avatarPath, err)
		}

		return &localAvatar{
			Filename: avatarPath,
			Image:    avatarFile,
		}, nil
	}

	return nil, nil
}

func handleAvatarOnUpdate(d *schema.ResourceData) (*localAvatar, error) {
	avatar, isAvatarSet := d.GetOk("avatar")
	if d.HasChanges("avatar", "avatar_hash") || (isAvatarSet && d.Get("avatar_hash").(string) == "") {
		avatarPath := avatar.(string)

		if avatarPath == "" { // the avatar should be removed
			// terraform doesn't care to remove this from state, thus, we do.
			d.Set("avatar_hash", "")

			return &localAvatar{}, nil
		} else { // the avatar should be added or changed
			avatarFile, err := os.Open(avatarPath)
			if err != nil {
				return nil, fmt.Errorf("unable to open avatar file %s: %s", avatarPath, err)
			}

			return &localAvatar{
				Filename: avatarPath,
				Image:    avatarFile,
			}, nil
		}
	}
	return nil, nil
}

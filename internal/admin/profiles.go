package admin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ebrgr/TuneTrail/imagecache"
	"github.com/ebrgr/TuneTrail/internal/db"
	"github.com/ebrgr/TuneTrail/internal/memkv"
	"github.com/ebrgr/TuneTrail/internal/models"
	"github.com/ebrgr/TuneTrail/internal/utils"
	"github.com/google/uuid"
)

var profilePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
var reservedProfiles = map[string]bool{"admin": true, "apis": true, "image": true, "public": true, "assets": true, "web-assets": true, "preview-assets": true, "artist": true, "album": true, "track": true, "chart": true, "listens": true, "rewind": true, "theme-helper": true}

// contextMutex preserves the shared profile lock while allowing image requests to cancel.
type contextMutex chan struct{}

// Lock serializes existing profile operations.
func (m contextMutex) Lock() { m <- struct{}{} }

// Unlock releases the profile operation.
func (m contextMutex) Unlock() { <-m }

// LockContext lets an image request stop waiting for the profile lock.
func (m contextMutex) LockContext(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	select {
	case m <- struct{}{}:
		if err := ctx.Err(); err != nil {
			<-m
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

var profileMu = make(contextMutex, 1)

// ValidateProfile rejects names and slugs that cannot be represented by the public router.
func ValidateProfile(name, slug, icon string) error {
	if strings.TrimSpace(name) == "" || len(name) > 200 || !utf8.ValidString(name) || len(slug) > 64 || !profilePattern.MatchString(slug) || reservedProfiles[slug] {
		return db.ErrInvalidInput
	}
	switch icon {
	case "", "music", "headphones", "disc", "radio", "user":
		return nil
	default:
		return db.ErrInvalidInput
	}
}

// Profile resolves stable identity and detects a stale slug after a rename.
func Profile(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string) (*models.ListeningProfile, error) {
	if err := Authorize(user); err != nil {
		return nil, err
	}
	if id <= 0 || slug == "" {
		return nil, db.ErrInvalidInput
	}
	p, err := store.GetListeningProfileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Slug != slug {
		return nil, db.ErrConflict
	}
	return p, nil
}

// CreateProfile preserves the store's atomic profile and initial-key transaction.
func CreateProfile(ctx context.Context, store db.ProfileStore, user *models.User, name, slug, icon string) (*models.ListeningProfile, error) {
	if err := Authorize(user); err != nil {
		return nil, err
	}
	if err := ValidateProfile(name, slug, icon); err != nil {
		return nil, err
	}
	token, err := utils.GenerateRandomString(48)
	if err != nil {
		return nil, fmt.Errorf("generate key: %w", err)
	}
	return store.CreateListeningProfile(ctx, strings.TrimSpace(name), slug, icon, token)
}

// UpdateProfile keeps the same numeric identity and requires the caller's current slug.
func UpdateProfile(ctx context.Context, store db.ProfileStore, user *models.User, id int32, expected, name, slug string) (*models.ListeningProfile, error) {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, expected); err != nil {
		return nil, err
	}
	if err := ValidateProfile(name, slug, ""); err != nil {
		return nil, err
	}
	if err := store.UpdateListeningProfile(ctx, id, strings.TrimSpace(name), slug); err != nil {
		return nil, err
	}
	return store.GetListeningProfileByID(ctx, id)
}

// CreateKey generates an ingestion key for exactly one already resolved profile.
func CreateKey(ctx context.Context, store interface {
	db.ProfileStore
	db.UserStore
}, user *models.User, id int32, slug, label string) (*models.ApiKey, error) {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return nil, err
	}
	label = strings.TrimSpace(label)
	if label == "" || len(label) > 100 || !utf8.ValidString(label) {
		return nil, db.ErrInvalidInput
	}
	key, err := utils.GenerateRandomString(48)
	if err != nil {
		return nil, err
	}
	return store.SaveApiKey(ctx, db.SaveApiKeyOpts{ProfileID: id, Key: key, Label: label})
}

// RenameKey enforces key/profile ownership in the store's WHERE clause.
func RenameKey(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string, key int32, label string) error {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return err
	}
	label = strings.TrimSpace(label)
	if key <= 0 || label == "" || len(label) > 100 || !utf8.ValidString(label) {
		return db.ErrInvalidInput
	}
	return store.UpdateProfileApiKeyLabel(ctx, key, id, label)
}

// RevokeKey cannot remove a key belonging to another profile.
func RevokeKey(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string, key int32) error {
	profileMu.Lock()
	defer profileMu.Unlock()
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return err
	}
	if key <= 0 {
		return db.ErrInvalidInput
	}
	return store.DeleteProfileApiKey(ctx, key, id)
}

// AvatarStore supplies both profile writes and global reference checks.
type AvatarStore interface {
	db.ProfileStore
	db.ImageStore
}

// SetAvatar serializes profile avatar updates and never deletes a previously published image during a concurrent catalog write.
func SetAvatar(ctx context.Context, store AvatarStore, user *models.User, id int32, slug, icon string, image io.ReadSeeker) (*models.ListeningProfile, error) {
	// Authorize before occupying a limited administrative image slot.
	if err := Authorize(user); err != nil {
		return nil, err
	}
	ctx, release, err := AcquireImageAdmission(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	if err := profileMu.LockContext(ctx); err != nil {
		return nil, err
	}
	defer profileMu.Unlock()
	if err := Authorize(user); err != nil {
		return nil, err
	}
	if err := ValidateProfile("avatar", slug, icon); err != nil {
		return nil, err
	}
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return nil, err
	}
	unlockPublication, err := imagecache.LockPublicationContext(ctx)
	if err != nil {
		return nil, err
	}
	defer unlockPublication()
	var imageID *string
	var iconValue *string
	if icon != "" {
		iconValue = &icon
	}
	if image != nil {
		// Rewind direct callers before the shared pipeline validates the content.
		if _, err := image.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("seek avatar: %w", err)
		}
		uid := uuid.New()
		if err := imagecache.SaveImageContext(ctx, uid, image); err != nil {
			return nil, fmt.Errorf("save avatar image: %w", err)
		}
		v := uid.String()
		imageID = &v
	}
	if err := store.SetProfileAvatar(ctx, id, imageID, iconValue); err != nil {
		if imageID != nil {
			uid := uuid.MustParse(*imageID)
			used, checkErr := store.ImageHasAssociation(context.WithoutCancel(ctx), uid)
			if checkErr == nil && !used {
				_ = imagecache.DeleteImage(uid)
			}
		}
		return nil, fmt.Errorf("save avatar: %w", err)
	}
	// Prior image files are left for the existing global orphan collector, avoiding a reference-check/delete race.
	return store.GetListeningProfileByID(ctx, id)
}

// Keys returns credentials only for the explicitly resolved profile and administrator.
func Keys(ctx context.Context, store db.ProfileStore, user *models.User, id int32, slug string) ([]models.ApiKey, error) {
	if _, err := Profile(ctx, store, user, id, slug); err != nil {
		return nil, err
	}
	return store.GetApiKeysByProfileID(ctx, id)
}

// DeleteProfile confirms name/count atomically and preserves the shared catalog.
func DeleteProfile(ctx context.Context, store interface {
	db.ProfileStore
	db.ImageStore
}, user *models.User, id int32, slug, name string, count int64) error {
	profileMu.Lock()
	defer profileMu.Unlock()
	p, err := Profile(ctx, store, user, id, slug)
	if err != nil {
		return err
	}
	if name == "" || count < 0 {
		return db.ErrInvalidInput
	}
	err = store.DeleteListeningProfile(ctx, id, name, count)
	if errors.Is(err, db.ErrNotFound) {
		_, check := store.GetListeningProfileByID(ctx, id)
		if check == nil {
			return db.ErrConflict
		}
		return check
	}
	if err != nil {
		return err
	}
	memkv.Store.Delete(fmt.Sprintf("now-playing:%d", id))
	if p.AvatarImageID != nil {
		uid, e := uuid.Parse(*p.AvatarImageID)
		if e == nil {
			unlock := imagecache.LockPublication()
			used, check := store.ImageHasAssociation(context.WithoutCancel(ctx), uid)
			if check == nil && !used {
				_ = imagecache.DeleteImage(uid)
			}
			unlock()
		}
	}
	return nil
}

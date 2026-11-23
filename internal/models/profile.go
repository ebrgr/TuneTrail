package models

// ListeningProfile is a public listening identity, independent of admin sessions.
type ListeningProfile struct {
	// IngestionKeyID is set only by player credential authentication.
	IngestionKeyID int64 `json:"-"`
	RequireLogin   bool  `json:"require_login"`
	// UserID is canonical ownership for server authorization, never a directory field.
	UserID        int32   `json:"-"`
	ID            int32   `json:"id"`
	Slug          string  `json:"slug"`
	DisplayName   string  `json:"display_name"`
	AvatarImageID *string `json:"avatar_image_id,omitempty"`
	AvatarIcon    *string `json:"avatar_icon,omitempty"`
	// AvatarColor is a palette entry (AvatarPalette) chosen as the avatar background; nil keeps the theme colour (ISSUE-202).
	AvatarColor *string `json:"avatar_color,omitempty"`
	CreatedAt   int64   `json:"created_at"`
}

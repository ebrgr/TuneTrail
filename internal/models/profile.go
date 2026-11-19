package models

// ListeningProfile is a public listening identity, independent of admin sessions.
type ListeningProfile struct {
	RequireLogin bool `json:"require_login"`
	// UserID is canonical ownership for server authorization, never a directory field.
	UserID        int32   `json:"-"`
	ID            int32   `json:"id"`
	Slug          string  `json:"slug"`
	DisplayName   string  `json:"display_name"`
	AvatarImageID *string `json:"avatar_image_id,omitempty"`
	AvatarIcon    *string `json:"avatar_icon,omitempty"`
	CreatedAt     int64   `json:"created_at"`
}

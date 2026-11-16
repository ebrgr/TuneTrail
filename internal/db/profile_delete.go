package db

import "context"

// ProfileDeletePlan binds an exact-name selection and its listening snapshot to one profile.
type ProfileDeletePlan struct {
	ProfileID, ItemID       int32
	Kind, Name, Fingerprint string
	Count                   int64
}

// ProfileDeleteStore previews and atomically revalidates personal listening deletion.
type ProfileDeleteStore interface {
	PreviewProfileDelete(context.Context, int32, string, int32) (ProfileDeletePlan, error)
	ApplyProfileDelete(context.Context, ProfileDeletePlan) (int64, error)
}

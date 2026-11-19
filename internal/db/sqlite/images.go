package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

func (s *Sqlite) ImageHasAssociation(ctx context.Context, image uuid.UUID) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM artists WHERE image = ?
			UNION ALL
			SELECT 1 FROM releases WHERE image = ?
 UNION ALL SELECT 1 FROM listening_profiles WHERE avatar_image_id = ?
		)`, image.String(), image.String(), image.String()).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("ImageHasAssociation: %w", err)
	}
	return exists == 1, nil
}

func (s *Sqlite) GetImageSource(ctx context.Context, image uuid.UUID) (string, error) {
	var src sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT image_source FROM artists WHERE image = ?
		UNION ALL
		SELECT image_source FROM releases WHERE image = ?
		LIMIT 1`,
		image.String(), image.String()).Scan(&src)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("GetImageSource: %w", err)
	}
	return src.String, nil
}

func (s *Sqlite) GetUserUploadedImageIDs(ctx context.Context) ([]uuid.UUID, error) {
	ret := make([]uuid.UUID, 0)
	rows, err := s.db.QueryContext(ctx, `
		SELECT image FROM artists WHERE image_source = 'User Upload' AND image IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("GetUserUploadedImageIDs: %w", err)
	}
	for rows.Next() {
		var imgidstr string
		err := rows.Scan(&imgidstr)
		if err != nil {
			return nil, fmt.Errorf("GetUserUploadedImageIDs: rows.Scan: %w", err)
		}
		imgid, err := uuid.Parse(imgidstr)
		if err != nil {
			return nil, fmt.Errorf("GetUserUploadedImageIDs: uuid.Parse: %w", err)
		}
		ret = append(ret, imgid)
	}
	rows, err = s.db.QueryContext(ctx, `
		SELECT image FROM releases WHERE image_source = 'User Upload' AND image IS NOT NULL`)
	if err != nil {
		return nil, fmt.Errorf("GetUserUploadedImageIDs: %w", err)
	}
	for rows.Next() {
		var imgidstr string
		err := rows.Scan(&imgidstr)
		if err != nil {
			return nil, fmt.Errorf("GetUserUploadedImageIDs: rows.Scan: %w", err)
		}
		imgid, err := uuid.Parse(imgidstr)
		if err != nil {
			return nil, fmt.Errorf("GetUserUploadedImageIDs: uuid.Parse: %w", err)
		}
		ret = append(ret, imgid)
	}
	return ret, nil
}

// ListeningImageExists checks association at the origin even for cached variants.
func (s *Sqlite) ListeningImageExists(ctx context.Context, image uuid.UUID, profileID, playingTrackID int32) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `
		WITH matching_tracks(id) AS (
			SELECT t.id FROM releases r JOIN tracks t ON t.release_id=r.id WHERE r.image=?
			UNION ALL
			SELECT at.track_id FROM artists a JOIN artist_tracks at ON at.artist_id=a.id WHERE a.image=?
			UNION ALL
			SELECT t.id FROM artists a JOIN artist_releases ar ON ar.artist_id=a.id
			JOIN tracks t ON t.release_id=ar.release_id WHERE a.image=?
		)
		SELECT EXISTS (
			SELECT 1 FROM matching_tracks t
			WHERE t.id=? OR EXISTS (
				SELECT 1 FROM listens l JOIN listening_profiles p ON p.id=l.profile_id
				WHERE l.track_id=t.id AND (p.id=? OR (?=0 AND p.require_login=0))
			)
		)`, image.String(), image.String(), image.String(), playingTrackID, profileID, profileID).Scan(&exists)
	return exists, err
}

// DirectoryAvatarExists authorizes the deliberately public avatar exception.
func (s *Sqlite) DirectoryAvatarExists(ctx context.Context, image uuid.UUID) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM listening_profiles WHERE avatar_image_id=?)`, image.String()).Scan(&exists)
	return exists, err
}

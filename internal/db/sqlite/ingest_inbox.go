package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/ebrgr/TuneTrail/internal/db"
)

// Count complete days from the projection and read only the two partial days from receipts.
const inboxCountsSQL = `WITH receipts(key_id,client,state,metadata,total) AS (
	SELECT key_id,client,state,metadata,total FROM ingest_inbox_daily
	WHERE profile_id=?1 AND day>=?2 AND day<?3 AND key_id BETWEEN ?4 AND ?5
	UNION ALL
	SELECT inbox_key,health_client,state,inbox_metadata,1
	FROM ingest_events INDEXED BY idx_ingest_inbox_edges
	WHERE profile_id=?1 AND received_at>=?6 AND received_at<?2*86400 AND received_at<=?7 AND inbox_key BETWEEN ?4 AND ?5
	UNION ALL
	SELECT inbox_key,health_client,state,inbox_metadata,1
	FROM ingest_events INDEXED BY idx_ingest_inbox_edges
	WHERE profile_id=?1 AND received_at>=max(?2,?3)*86400 AND received_at<=?7 AND inbox_key BETWEEN ?4 AND ?5
)
SELECT key_id,client,sum(total),
	coalesce(sum(total) FILTER (WHERE state='accepted'),0),
	coalesce(sum(total) FILTER (WHERE state='duplicate'),0),
	coalesce(sum(total) FILTER (WHERE state='invalid'),0),
	coalesce(sum(total) FILTER (WHERE state IN ('received','pending_identity','retryable','pending_duplicate')),0),
	coalesce(sum(total) FILTER (WHERE metadata=1),0),
	coalesce(sum(total) FILTER (WHERE metadata=2),0)
FROM receipts GROUP BY key_id,client ORDER BY key_id,client`

// inboxCountArgs splits the supported windows into complete UTC days and two edge ranges.
func inboxCountArgs(profile int32, first, last int64, since, through time.Time) []any {
	const day = 24 * time.Hour
	fullStart := since.Truncate(day)
	if since.Unix() != fullStart.Unix() {
		fullStart = fullStart.Add(day)
	}
	return []any{profile, fullStart.Unix() / 86400, through.Truncate(day).Unix() / 86400,
		first, last, since.Unix(), through.Unix()}
}

// IngestInbox reads counts and safe examples under a single read snapshot.
func (s *Sqlite) IngestInbox(ctx context.Context, profile int32, key *int64, since, through time.Time) ([]db.IngestInboxGroup, error) {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()
	rows, err := tx.QueryContext(ctx, `SELECT id,label FROM api_keys WHERE profile_id=? ORDER BY id`, profile)
	if err != nil {
		return nil, err
	}
	labels := make(map[int64]string)
	for rows.Next() {
		var id int64
		var label string
		if err := rows.Scan(&id, &label); err != nil {
			rows.Close()
			return nil, err
		}
		labels[id] = label
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()

	// A deleted key retains its ID and may still be selected in the Inbox.
	first, last := int64(-1), int64(1<<63-1)
	if key != nil {
		first, last = *key, *key
	}
	rows, err = tx.QueryContext(ctx, inboxCountsSQL, inboxCountArgs(profile, first, last, since, through)...)
	if err != nil {
		return nil, err
	}
	var groups []db.IngestInboxGroup
	for rows.Next() {
		var group db.IngestInboxGroup
		if err := rows.Scan(&group.Key, &group.Client, &group.Total, &group.Accepted, &group.Duplicate,
			&group.Invalid, &group.Pending, &group.Missing, &group.Incomplete); err != nil {
			rows.Close()
			return nil, err
		}
		group.Label, group.KeyPresent = labels[group.Key]
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	for i := range groups {
		for j, count := range []int64{groups[i].Missing, groups[i].Incomplete} {
			if count == 0 {
				continue
			}
			sample, err := inboxExample(ctx, tx, profile, groups[i], j+1, since.Unix(), through.Unix())
			if err != nil {
				return nil, err
			}
			groups[i].Examples[j] = sample
		}
	}
	return groups, tx.Commit()
}

// inboxExample projects only bounded display fields from one profile-owned receipt.
func inboxExample(ctx context.Context, tx *sql.Tx, profile int32, group db.IngestInboxGroup, metadata int, since, through int64) (*db.IngestInboxExample, error) {
	var sample db.IngestInboxExample
	var seconds int64
	err := tx.QueryRowContext(ctx, `SELECT
		coalesce(CASE WHEN json_valid(original_snapshot) THEN
			CASE WHEN json_type(original_snapshot,'$.title')='text' THEN substr(json_extract(original_snapshot,'$.title'),1,240) END
		END,''),
		coalesce(CASE WHEN json_valid(original_snapshot) THEN
			CASE WHEN json_type(original_snapshot,'$.artist')='text' THEN substr(json_extract(original_snapshot,'$.artist'),1,240) END
		END,''),
		received_at,health_client FROM ingest_events INDEXED BY idx_ingest_inbox_example
		WHERE profile_id=? AND inbox_key=? AND health_client=? AND inbox_metadata=?
		  AND inbox_metadata IN (1,2) AND received_at BETWEEN ? AND ?
		ORDER BY received_at DESC,receipt_id DESC LIMIT 1`, profile, group.Key, group.Client, metadata, since, through).
		Scan(&sample.Title, &sample.Artist, &seconds, &sample.Client)
	sample.At = time.Unix(seconds, 0).UTC()
	return &sample, err
}

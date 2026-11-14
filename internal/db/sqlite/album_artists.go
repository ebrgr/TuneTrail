package sqlite

// albumArtistRelations combines independent/legacy credits with live track credits.
// Excluding persisted pairs from the second arm prevents duplicate artists and listen counts.
// It is a SQL expression only: no view, migration or data rewrite is required.
const albumArtistRelations = `(SELECT artist_id,release_id,is_primary FROM artist_releases
 UNION ALL
 SELECT DISTINCT a.artist_id,t.release_id,0 AS is_primary
 FROM artist_tracks a JOIN tracks t ON t.id=a.track_id
 WHERE NOT EXISTS (SELECT 1 FROM artist_releases r
 WHERE r.artist_id=a.artist_id AND r.release_id=t.release_id))`

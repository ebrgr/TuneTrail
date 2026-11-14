---
title: Editing Data
description: How to edit artist, album, and track information in your Koito instance.
---

Catalog metadata, images and merges are shared across profiles. Editing them requires an administrator session; a player API key cannot authorize edits. Profile history operations belong to the selected profile in Administration.

In order to start editing information on your Koito instance, you need to be logged in. See the [Setting up the Scrobber](/guides/scrobbler) guide if you need to log in for the first time.

Once logged in, navigate to the page of the item you want to edit. For this example, we will use the fantastic Korean dream pop group OurR's track [Bloom](https://www.youtube.com/watch?v=USHrBJRmF-o).
When you are logged in and on an artist, album, or track page, you will see the editing options on the top right.

![A Koito page for the track Bloom by OurR, showing the editing options](../../../assets/ourr_bloom.png)

#### Editing

The first option from the left is the edit menu. The edit menu lets you edit things like aliases, artists, and MusicBrainz IDs.

![a menu showing editing options](../../../assets/edit_menu.png)

In the alias manager section, we can add and remove aliases, as well as set an alias as "primary" to use it as the display name of the item. Koito uses aliases for not only the UI, but also for matching artists, albums, and tracks submitted to the scrobbler, and for searching. So if you are like me and listen to a lot of music from countries with non-latin script, adding aliases makes it easy to search for those items.

We can also add and remove artist associations, as well as update the MusicBrainz ID of the item. Note that MusicBrainz IDs for items must be unique among other items of the same type (artists, albums, and tracks).

For artists and albums, similar editing options will be shown.

#### Editing Images

The easiest way to replace an image is to simply drag an image file from your computer onto the page of the artist, album, or track you want to change the image for. This only works when you are logged in.

The other way to replace images is using the Replace Image option, which is the second editing option in the list, to provide a URL to the image you want to use.

Note that you can only edit images on an artist or album page, as the image shown for a track is the same as the image of the album it belongs to.

#### Merging Items

Koito allows you to merge two items, which means that all of that item's children (for artists: albums, tracks and listens; for albums: tracks and listens; etc.) will be assigned to a different item, and the old item will be deleted.

For this example, we will use the incredible track [Tsumugu](https://www.youtube.com/watch?v=NDwqZIXOvKw) by the Japanese artist とた (Tota). Here we can see there two versions of the track, one in Japanese and one romanized. I only want to keep the Japanese one. So, just navigate to the page for the Japanese-titled track, open the Merge Items menu (the third editing option), search for the track we want to be merged, and click "Merge Items". A text will appear in the UI to clarify which track is the primary and which will be merged into it.

![a menu preparing for the track Tsumugu to be merged into 紡ぐ](../../../assets/merge.png)

If we wanted to merge the other way, and only keep the romanized track, just click the "Reverse merge order" checkbox.

If you are ever confused which item will be kept and which will be dissolved, just remember that by default, **the page you are on is the item that will be kept**.

Once merged, we can see that all of the listen activity for Tsumugu has been asigned to 紡ぐ.

![an activity heatmap showing more listens than were previously there](../../../assets/merged_activity.png)

You can also search for items when merging by their ID using the format `id:1234`.

:::danger

Merge operations are irreversible. Make sure you have the right items selected before you merge them.

:::

#### Deleting Items

To delete at item, just click the trash icon, which is the fourth and final icon in the editing options. Doing so will open a confirmation dialogue. Once confirmed, the item you delete, as well as all of its children and listen activity, will be removed.

:::danger

Deleting items is irreversible.

:::

## Server-rendered management

In the default Go mode, **Manage this item** opens `/<slug>/manage/<kind>/<id>`.
The administrator edits shared catalog metadata; the profile is the navigation
context. Search for an existing artist or merge destination, edit aliases/MBID,
mark an album as a compilation, or upload artwork. Clearing the MBID input keeps
the current value; it does not remove an existing MBID. Track artwork always
targets the track's real album.

Artwork accepts JPEG, PNG and static WebP. Animated images and other formats are
not supported. Artwork is limited to 10 MiB, 10,000 pixels per dimension and
40 million pixels.
HTTP(S) downloads use a ten-second timeout and up to three validated redirects.
Shared old artwork remains available until orphan collection.

Manual listens use an IANA timezone. Nonexistent daylight-saving times are
rejected; ambiguous times require an explicit UTC offset. Offsets with historical
seconds are not supported. The history's **Manage** link selects the exact listen
for confirmation and returns to the filtered history after deletion.

Deletion and merging require a separate global impact review and final POST.
The review names the source, destination and surviving ID. **In the Go editor the
current item starts as the source**; use Reverse direction to change that choice
and always read the review.
A changed catalog/history requires a fresh review. Destination aliases, MBID and
other metadata are retained; source aliases are removed. Track duplicate listens
are consolidated only within the same profile and timestamp. No undo is provided.

## Album participation after track edits

Album artists include independent album credits and artists credited on any of
its tracks. Adding or removing a track artist updates that participation
immediately. Another track can still justify the same artist's participation.
Selecting a participating artist as the album's primary artist creates an
independent album credit that survives later track edits.

This correction keeps SQLite schema v3. Existing album credits are preserved:
older versions did not record whether a credit belonged to the album or came
from a track. Consequently, an old mistaken association can remain visible after
editing a track. There is no automatic cleanup of these ambiguous credits.

### Reviewing an old mistaken association

Album metadata is shared by all profiles. Review the artist and album together,
including their other tracks and album credits, before deciding that an old
association is wrong. Do not delete the artist, album or track to remove a credit.

For an administrator familiar with SQLite, the following is an offline,
**single-association** repair. Stop every application instance, back up the
complete data directory (including SQLite sidecars), and first rehearse on a
copy. Open the database with `sqlite3 tunetrail.db`, enable foreign keys and
replace the example IDs with the pair you reviewed. This is not a bulk cleanup.

```sql
.bail on
.parameter init
.parameter set @artist 123
.parameter set @album 456
PRAGMA foreign_keys=ON;
BEGIN IMMEDIATE;
SELECT ar.artist_id, ar.release_id, ar.is_primary, aa.alias, ra.alias
FROM artist_releases ar
LEFT JOIN artist_aliases aa ON aa.artist_id=ar.artist_id AND aa.is_primary=1
LEFT JOIN release_aliases ra ON ra.release_id=ar.release_id AND ra.is_primary=1
WHERE ar.artist_id=@artist AND ar.release_id=@album;
DELETE FROM artist_releases
WHERE artist_id=@artist AND release_id=@album AND is_primary=0
  AND NOT EXISTS (
    SELECT 1 FROM artist_tracks a JOIN tracks t ON t.id=a.track_id
    WHERE a.artist_id=@artist AND t.release_id=@album
  )
  AND EXISTS (
    SELECT 1 FROM artist_releases other
    WHERE other.release_id=@album AND other.artist_id<>@artist
  );
SELECT changes();
PRAGMA foreign_key_check;
PRAGMA integrity_check;
```

Before committing, confirm exactly one association changed, the foreign-key
check has no rows, and the integrity check returns `ok`. Otherwise enter
`ROLLBACK;`. Enter `COMMIT;` only after these checks. The guards refuse primary
credits, credits supported by another track, and the last stored album credit;
removing that last credit would activate the existing album-deletion trigger.
A refused repair needs further review, not removal of the guards.

Restart and verify the artist page in the affected profiles. To undo a committed
repair, stop the application and restore the complete backup. Restoring an older
backup also removes changes recorded since that backup; export newer listens
before restoring if necessary. There is no database migration or downgrade in
this procedure.

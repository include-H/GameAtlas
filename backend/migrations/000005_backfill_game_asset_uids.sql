-- Backfill legacy game_assets rows that predate asset_uid support.
-- The frontend edit flow treats non-empty UIDs as the keep/reorder identity,
-- so rows without a UID could previously neither be deleted nor reordered.
-- Idempotent: after the first run no NULL/empty rows remain.
UPDATE game_assets
SET asset_uid = (
    lower(hex(randomblob(4))) || '-' ||
    lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))), 2) || '-' ||
    'a' || substr(lower(hex(randomblob(2))), 2) || '-' ||
    lower(hex(randomblob(6)))
)
WHERE asset_uid IS NULL OR trim(asset_uid) = '';

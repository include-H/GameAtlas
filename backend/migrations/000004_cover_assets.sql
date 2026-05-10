-- Migrate existing cover_image values into game_assets as asset_type='cover'.
-- This enables multi-cover support while keeping games.cover_image as a denormalized primary reference.
INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
SELECT
    id,
    lower(hex(randomblob(4))) || '-' ||
    lower(hex(randomblob(2))) || '-4' ||
    substr(lower(hex(randomblob(2))), 2) || '-' ||
    substr('89ab', abs(random()) % 4 + 1, 1) ||
    substr(lower(hex(randomblob(2))), 2) || '-' ||
    lower(hex(randomblob(6))),
    'cover',
    cover_image,
    0
FROM games
WHERE cover_image IS NOT NULL AND TRIM(cover_image) != '';

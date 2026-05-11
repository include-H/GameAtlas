-- Migrate existing banner_image values into game_assets so banners
-- can be managed as multi-row assets (like covers and screenshots).
-- The games.banner_image column is kept as a denormalized cache,
-- synced from the primary banner by sort_order.

INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order, created_at)
SELECT
    id,
    lower(hex(randomblob(4))) || '-' ||
    lower(hex(randomblob(2))) || '-4' ||
    lower(hex(randomblob(2))) || '-' ||
    lower(hex(randomblob(2))) || '-' ||
    lower(hex(randomblob(6))),
    'banner',
    banner_image,
    0,
    CURRENT_TIMESTAMP
FROM games
WHERE banner_image IS NOT NULL AND TRIM(banner_image) != '';

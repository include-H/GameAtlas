-- Normalize public_id to lowercase for consistent lookups
UPDATE games SET public_id = lower(public_id) WHERE public_id != lower(public_id);

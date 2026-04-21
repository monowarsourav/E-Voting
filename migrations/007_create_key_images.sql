-- migrations/007_create_key_images.sql
-- Durable key-image storage for double-vote prevention. The UNIQUE constraint
-- is the authoritative race guard — two concurrent vote attempts with the
-- same key image will collide here regardless of in-memory state.

CREATE TABLE IF NOT EXISTS key_images (
    key_image TEXT PRIMARY KEY,
    used_at   INTEGER NOT NULL
);

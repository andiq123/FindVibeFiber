DROP TRIGGER IF EXISTS update_favorite_songs_updated_at ON favorite_songs;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP INDEX IF EXISTS idx_favorite_songs_created_at;
DROP INDEX IF EXISTS idx_favorite_songs_user_uuid;
DROP INDEX IF EXISTS idx_favorite_songs_user_order;
DROP INDEX IF EXISTS idx_favorite_songs_id_user;
DROP INDEX IF EXISTS idx_users_name;
DROP TABLE IF EXISTS favorite_songs;
DROP TABLE IF EXISTS users;

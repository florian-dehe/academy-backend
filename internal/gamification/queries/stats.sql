-- name: GetUserStats :one
SELECT * FROM user_stats WHERE user_id = $1;

-- name: EnsureUserStats :one
INSERT INTO user_stats (user_id) VALUES ($1)
ON CONFLICT (user_id) DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: ListLevels :many
SELECT * FROM levels ORDER BY level;

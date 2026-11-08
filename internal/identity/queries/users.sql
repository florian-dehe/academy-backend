-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByJobBoardID :one
SELECT * FROM users WHERE job_board_user_id = $1;

-- name: InsertUserIfAbsent :one
-- Returns no row when job_board_user_id already exists (the caller then reads it).
INSERT INTO users (email, job_board_user_id, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (job_board_user_id) DO NOTHING
RETURNING *;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

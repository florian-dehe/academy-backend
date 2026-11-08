-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByJobBoardID :one
SELECT * FROM users WHERE job_board_user_id = $1;

-- name: UpsertUserByJobBoardID :one
INSERT INTO users (email, job_board_user_id, display_name)
VALUES ($1, $2, $3)
ON CONFLICT (job_board_user_id) DO UPDATE
   SET email = EXCLUDED.email, display_name = EXCLUDED.display_name, updated_at = now()
RETURNING *;

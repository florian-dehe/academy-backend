-- name: ListChapterProgress :many
SELECT * FROM user_chapter_progress WHERE user_id = $1;

-- name: ListModuleProgress :many
SELECT * FROM user_module_progress WHERE user_id = $1;

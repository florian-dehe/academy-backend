-- name: ListPublishedChapters :many
SELECT c.id, c.position, c.is_free, v.id AS version_id, v.title, v.description
FROM chapters c
JOIN chapter_versions v ON v.chapter_id = c.id AND v.status = 'published'
WHERE c.deleted_at IS NULL
ORDER BY c.position;

-- name: ListModulesByChapterVersion :many
SELECT * FROM modules WHERE chapter_version_id = $1 ORDER BY position;

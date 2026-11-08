-- name: LatestPersonalityResult :one
SELECT * FROM personality_results
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: InsertPersonalityResult :one
INSERT INTO personality_results (user_id, openness, conscientiousness, extraversion, agreeableness, neuroticism)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetLatestPersonalityResult :one
SELECT * FROM personality_results
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

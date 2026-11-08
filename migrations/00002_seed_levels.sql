-- +goose Up
-- Level 1 must exist so user_stats can always reference a level.
INSERT INTO levels (level, xp_required, title) VALUES (1, 0, 'Beginner')
ON CONFLICT DO NOTHING;

-- +goose Down
DELETE FROM levels WHERE level = 1;

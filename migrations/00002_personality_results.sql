-- +goose Up
-- +goose StatementBegin

-- introtest ---------------------------------------------------------------

-- One row per completed intro test; the latest row is the current result.
-- Questions and answers are not stored: the frontend scores the test.
CREATE TABLE personality_results (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid             NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    openness          double precision NOT NULL CONSTRAINT chk_pr_openness CHECK (openness BETWEEN 1 AND 5),
    conscientiousness double precision NOT NULL CONSTRAINT chk_pr_conscientiousness CHECK (conscientiousness BETWEEN 1 AND 5),
    extraversion      double precision NOT NULL CONSTRAINT chk_pr_extraversion CHECK (extraversion BETWEEN 1 AND 5),
    agreeableness     double precision NOT NULL CONSTRAINT chk_pr_agreeableness CHECK (agreeableness BETWEEN 1 AND 5),
    neuroticism       double precision NOT NULL CONSTRAINT chk_pr_neuroticism CHECK (neuroticism BETWEEN 1 AND 5),
    created_at        timestamptz      NOT NULL DEFAULT now()
);

CREATE INDEX idx_personality_results_user_latest ON personality_results (user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE personality_results;
-- +goose StatementEnd

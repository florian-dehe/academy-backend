-- +goose Up
-- +goose StatementBegin

-- gen_random_uuid() is built in since Postgres 13.

-- identity ---------------------------------------------------------------

CREATE TABLE users (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email             text        NOT NULL UNIQUE,
    job_board_user_id text        UNIQUE,
    display_name      text,
    role              text        NOT NULL DEFAULT 'learner'
                      CONSTRAINT chk_users_role CHECK (role IN ('learner', 'content_editor', 'admin')),
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now()
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd

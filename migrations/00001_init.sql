-- +goose Up
-- +goose StatementBegin

-- gen_random_uuid() is built in since Postgres 13.

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

-- content ---------------------------------------------------------------

CREATE TABLE chapters (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    position   integer     NOT NULL,
    is_free    boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE chapter_versions (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_id     uuid        NOT NULL REFERENCES chapters (id),
    version        integer     NOT NULL,
    status         text        NOT NULL DEFAULT 'draft'
                   CONSTRAINT chk_chapter_versions_status CHECK (status IN ('draft', 'published', 'archived')),
    title          text        NOT NULL,
    description    text,
    milestone_text text,
    created_by     uuid        REFERENCES users (id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    published_at   timestamptz,
    CONSTRAINT uq_chapter_versions_chapter_version UNIQUE (chapter_id, version)
);
CREATE UNIQUE INDEX chapter_versions_one_published
    ON chapter_versions (chapter_id) WHERE status = 'published';

CREATE TABLE modules (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    chapter_version_id uuid    NOT NULL REFERENCES chapter_versions (id) ON DELETE CASCADE,
    module_key         uuid    NOT NULL DEFAULT gen_random_uuid(), -- stable across versions
    position           integer NOT NULL,
    title              text    NOT NULL,
    xp_reward          integer NOT NULL DEFAULT 0
                       CONSTRAINT chk_modules_xp_reward CHECK (xp_reward >= 0),
    CONSTRAINT uq_modules_version_key UNIQUE (chapter_version_id, module_key)
);

CREATE TABLE module_videos (
    module_id        uuid PRIMARY KEY REFERENCES modules (id) ON DELETE CASCADE,
    video_url        text NOT NULL,
    duration_seconds integer,
    transcript       text
);

CREATE TABLE module_exercises (
    module_id     uuid  PRIMARY KEY REFERENCES modules (id) ON DELETE CASCADE,
    exercise_type text  NOT NULL,
    title         text,
    instructions  text,
    config        jsonb NOT NULL DEFAULT '{}'
);

-- Recap quiz (module_id set) or final quiz (chapter_version_id set).
CREATE TABLE quizzes (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    kind               text    NOT NULL
                       CONSTRAINT chk_quizzes_kind CHECK (kind IN ('recap', 'final')),
    module_id          uuid    UNIQUE REFERENCES modules (id) ON DELETE CASCADE,
    chapter_version_id uuid    UNIQUE REFERENCES chapter_versions (id) ON DELETE CASCADE,
    pass_score         integer NOT NULL
                       CONSTRAINT chk_quizzes_pass_score CHECK (pass_score BETWEEN 0 AND 100),
    questions          jsonb   NOT NULL DEFAULT '[]',
    CONSTRAINT chk_quizzes_one_target CHECK (
        (kind = 'recap' AND module_id IS NOT NULL AND chapter_version_id IS NULL) OR
        (kind = 'final' AND chapter_version_id IS NOT NULL AND module_id IS NULL))
);

-- learning --------------------------------------------------------------
-- Learner data cascades on user delete (account deletion removes all Academy data).
-- Open: this includes payments; revisit if payment records must be retained.

CREATE TABLE user_chapter_progress (
    user_id            uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    chapter_id         uuid NOT NULL REFERENCES chapters (id),
    chapter_version_id uuid NOT NULL REFERENCES chapter_versions (id),
    status             text NOT NULL DEFAULT 'locked'
                       CONSTRAINT chk_user_chapter_progress_status
                       CHECK (status IN ('locked', 'unlocked', 'in_progress', 'completed')),
    started_at         timestamptz,
    completed_at       timestamptz,
    PRIMARY KEY (user_id, chapter_id)
);

CREATE TABLE user_module_progress (
    user_id               uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    module_id             uuid NOT NULL REFERENCES modules (id),
    status                text NOT NULL DEFAULT 'locked'
                          CONSTRAINT chk_user_module_progress_status
                          CHECK (status IN ('locked', 'unlocked', 'in_progress', 'completed')),
    video_completed_at    timestamptz,
    quiz_passed_at        timestamptz,
    exercise_completed_at timestamptz,
    completed_at          timestamptz,
    PRIMARY KEY (user_id, module_id)
);

CREATE TABLE quiz_attempts (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    quiz_id      uuid        NOT NULL REFERENCES quizzes (id),
    answers      jsonb       NOT NULL DEFAULT '{}', -- {question_id: option_id}
    score        integer,
    passed       boolean,
    started_at   timestamptz NOT NULL DEFAULT now(),
    submitted_at timestamptz
);
CREATE INDEX quiz_attempts_user_quiz ON quiz_attempts (user_id, quiz_id);

CREATE TABLE exercise_submissions (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    module_id  uuid        NOT NULL REFERENCES modules (id),
    payload    jsonb       NOT NULL DEFAULT '{}',
    score      integer,
    completed  boolean     NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX exercise_submissions_user_module ON exercise_submissions (user_id, module_id);

-- gamification ----------------------------------------------------------

CREATE TABLE levels (
    level       integer PRIMARY KEY,
    xp_required integer NOT NULL UNIQUE
                CONSTRAINT chk_levels_xp_required CHECK (xp_required >= 0),
    title       text
);

CREATE TABLE user_stats (
    user_id            uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    total_xp           integer NOT NULL DEFAULT 0
                       CONSTRAINT chk_user_stats_total_xp CHECK (total_xp >= 0),
    level              integer NOT NULL DEFAULT 1 REFERENCES levels (level),
    current_streak     integer NOT NULL DEFAULT 0,
    longest_streak     integer NOT NULL DEFAULT 0,
    last_activity_date date
);

-- introtest -------------------------------------------------------------

-- Five axes, each 1.0..5.0.
CREATE TABLE personality_results (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           uuid         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    openness          numeric(2,1) NOT NULL CONSTRAINT chk_personality_openness          CHECK (openness          BETWEEN 1.0 AND 5.0),
    conscientiousness numeric(2,1) NOT NULL CONSTRAINT chk_personality_conscientiousness CHECK (conscientiousness BETWEEN 1.0 AND 5.0),
    extraversion      numeric(2,1) NOT NULL CONSTRAINT chk_personality_extraversion      CHECK (extraversion      BETWEEN 1.0 AND 5.0),
    agreeableness     numeric(2,1) NOT NULL CONSTRAINT chk_personality_agreeableness     CHECK (agreeableness     BETWEEN 1.0 AND 5.0),
    neuroticism       numeric(2,1) NOT NULL CONSTRAINT chk_personality_neuroticism       CHECK (neuroticism       BETWEEN 1.0 AND 5.0),
    created_at        timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX personality_results_user_latest ON personality_results (user_id, created_at DESC);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE personality_results, user_stats, levels, exercise_submissions, quiz_attempts,
    user_module_progress, user_chapter_progress, quizzes, module_exercises, module_videos,
    modules, chapter_versions, chapters, users;
-- +goose StatementEnd

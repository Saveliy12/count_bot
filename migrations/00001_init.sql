-- +goose Up
CREATE TABLE families (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL CHECK (btrim(name) <> ''),
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX families_single_default_idx
    ON families (is_default)
    WHERE is_default;

CREATE TABLE users (
    telegram_id BIGINT PRIMARY KEY,
    family_id BIGINT NOT NULL REFERENCES families (id) ON DELETE RESTRICT,
    display_name TEXT NOT NULL CHECK (btrim(display_name) <> ''),
    username TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX users_family_id_idx ON users (family_id);

CREATE TABLE expenses (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (telegram_id) ON DELETE CASCADE,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    category TEXT NOT NULL CHECK (
        category IN (
            'food',
            'transport',
            'housing',
            'entertainment',
            'health',
            'shopping',
            'other'
        )
    ),
    spent_at DATE NOT NULL,
    comment TEXT CHECK (
        comment IS NULL
        OR (btrim(comment) <> '' AND char_length(comment) <= 500)
    ),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX expenses_user_id_spent_at_idx
    ON expenses (user_id, spent_at DESC);

CREATE INDEX expenses_category_spent_at_idx
    ON expenses (category, spent_at DESC);

CREATE INDEX expenses_spent_at_idx
    ON expenses (spent_at DESC);

INSERT INTO families (name, is_default)
VALUES ('Моя семья', TRUE);

-- +goose Down
DROP TABLE IF EXISTS expenses;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS families;

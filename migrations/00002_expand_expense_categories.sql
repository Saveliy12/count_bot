-- +goose Up
ALTER TABLE expenses
    DROP CONSTRAINT IF EXISTS expenses_category_check;

ALTER TABLE expenses
    ADD CONSTRAINT expenses_category_check CHECK (
        category IN (
            'food',
            'transport',
            'housing',
            'entertainment',
            'health',
            'shopping',
            'education',
            'travel',
            'subscriptions',
            'gifts',
            'pets',
            'personal_care',
            'sport',
            'other'
        )
    );

-- +goose Down
ALTER TABLE expenses
    DROP CONSTRAINT IF EXISTS expenses_category_check;

ALTER TABLE expenses
    ADD CONSTRAINT expenses_category_check CHECK (
        category IN (
            'food',
            'transport',
            'housing',
            'entertainment',
            'health',
            'shopping',
            'other'
        )
    );

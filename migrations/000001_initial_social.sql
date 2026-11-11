-- +goose Up
CREATE TABLE users(
    id BIGSERIAL PRIMARY KEY,
    username text NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE posts(
    id BIGSERIAL PRIMARY KEY,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text text NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE likes(
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX posts_created_at_idx ON posts(created_at);
CREATE INDEX likes_post_id_idx ON likes(post_id);

-- +goose Down
DROP TABLE likes;
DROP TABLE posts;
DROP TABLE users;
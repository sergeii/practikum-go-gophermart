CREATE TABLE users (
    id serial PRIMARY KEY,
    login TEXT UNIQUE NOT NULL,
    password TEXT NOT NULL,
    CHECK (login <> ''),
    CHECK (password <> '')
);
CREATE UNIQUE INDEX users_login_lower_uniq_idx ON users (lower(login));

BEGIN;
CREATE TABLE users (
    "id" serial NOT NULL PRIMARY KEY,
    "login" TEXT UNIQUE NOT NULL CHECK ("login" <> ''),
    "password" TEXT NOT NULL CHECK ("password" <> '')
);
CREATE UNIQUE INDEX "users_login_lower_uniq_idx" ON users (lower("login"));
COMMIT;

BEGIN;
CREATE TYPE order_status AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');
CREATE TABLE orders (
    "id"          serial NOT NULL PRIMARY KEY,
    "uploaded_at" timestamp with time zone NOT NULL,
    "number"      text NOT NULL UNIQUE,
    "accrual"     decimal(7,2),
    "user_id"     integer NOT NULL,
    "status"      order_status NOT NULL DEFAULT 'NEW',
    CHECK ("accrual" >= 0)
);
ALTER TABLE orders ADD CONSTRAINT "orders_user_id_fk_users" FOREIGN KEY ("user_id") REFERENCES users ("id") DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX orders_user_id_idx ON orders ("user_id");
COMMIT;

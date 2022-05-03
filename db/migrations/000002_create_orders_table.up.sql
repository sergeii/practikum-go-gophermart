BEGIN;
CREATE TYPE order_status AS ENUM ('new', 'processing', 'invalid', 'processed');
CREATE TABLE orders (
    "id"          serial NOT NULL PRIMARY KEY,
    "uploaded_at" timestamp with time zone NOT NULL,
    "number"      text NOT NULL UNIQUE,
    "user_id"     integer NOT NULL,
    "status"      order_status NOT NULL DEFAULT 'new'
);
ALTER TABLE orders ADD CONSTRAINT "orders_user_id_fk_users" FOREIGN KEY ("user_id") REFERENCES users ("id") DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX orders_user_id_idx ON orders ("user_id");
COMMIT;

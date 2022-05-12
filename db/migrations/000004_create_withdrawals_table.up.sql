BEGIN;
CREATE TABLE withdrawals (
    "id"           serial NOT NULL PRIMARY KEY,
    "user_id"      integer NOT NULL,
    "processed_at" timestamp with time zone NOT NULL,
    "number"       text NOT NULL UNIQUE,
    "sum"          decimal(9,2) NOT NULL DEFAULT 0 CHECK ("sum" > 0)
);
ALTER TABLE withdrawals ADD CONSTRAINT "withdrawals_user_id_fk_users" FOREIGN KEY ("user_id") REFERENCES users ("id") DEFERRABLE INITIALLY DEFERRED;
CREATE INDEX withdrawals_user_id_idx ON orders ("user_id");
COMMIT;

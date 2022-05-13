BEGIN;
ALTER TABLE users ADD COLUMN "balance_current" decimal(9,2) NOT NULL DEFAULT 0 CHECK ("balance_current" >= 0);
ALTER TABLE users ADD COLUMN "balance_withdrawn" decimal(9,2) NOT NULL DEFAULT 0 CHECK ("balance_withdrawn" >= 0);
COMMIT;

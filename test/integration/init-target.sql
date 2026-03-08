-- Init Script for Target DB

-- 1. Extension (Source had uuid-ossp, maybe target doesn't yet, so diff should CREATE it)
-- Note: It requires superuser usually, but postgres inside docker is superuser.
-- Let's NOT create it here so pg-diff detects it missing in target.

-- Target ONLY extension (should be dropped by diff, but actually our diff handles target-missing as drop. If target has it but source doesn't, target should DROP it. Let's create one target has but source doesn't).
CREATE EXTENSION IF NOT EXISTS "citext";

-- 2. Types & Enums (Missing user_status, so diff should CREATE it)
CREATE TYPE extra_status AS ENUM ('new', 'old');

-- 3. Sequences
-- Missing user_id_seq inside target, so diff should CREATE it.
-- dropping_seq is missing in target, so diff should CREATE it. Actually, wait. Target missing means Target DB lacks it. So DIFF from source->target says "Target needs to catch up". So Target creates it.
-- Let's establish: Target DB is the one we want to mutate to look like Source DB.
-- If Source has X, Target needs X (CREATE X).
-- If Target has Y but Source does NOT, Target needs to DROP Y.

CREATE SEQUENCE target_only_seq START 10 INCREMENT 1;

-- 4. Tables
CREATE TABLE users (
    id integer PRIMARY KEY, -- Missing default seq
    email varchar(255) UNIQUE NOT NULL,
    -- missing status column
    last_login timestamp -- Extra column that should be dropped
);

COMMENT ON TABLE users IS 'Modernized user table';
COMMENT ON COLUMN users.id IS 'Primary identifier - modernized';

CREATE TABLE extra_target_table (
    id int
);

-- Note: old_legacy_table is in Source but NOT in Target -> Diff should issue CREATE TABLE old_legacy_table

-- 5. Views (None in target yet, diff should CREATE active_users)

-- 6. Functions (None in target yet, diff should CREATE get_user_status, getting legacy data)
CREATE OR REPLACE FUNCTION get_target_only_data() RETURNS integer AS $$ 
BEGIN RETURN 2; END; 
$$ LANGUAGE plpgsql;

-- 7. Privileges (Target has different privs)
CREATE ROLE read_only_bob LOGIN PASSWORD 'bobpass';
-- Missing SELECT ON users
GRANT DELETE ON users TO read_only_bob; -- Should be revoked

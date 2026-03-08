-- Init Script for Source DB

-- 1. Extension (Assume already loaded, but let's test isolation if we want, or just basic user objects)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 2. Types & Enums
CREATE TYPE user_status AS ENUM ('active', 'inactive', 'banned');

-- 3. Sequences
CREATE SEQUENCE user_id_seq START 1 INCREMENT 1;
CREATE SEQUENCE dropping_seq START 100 INCREMENT 5;

-- 4. Tables with Columns, Constraints, and Indices
CREATE TABLE users (
    id integer PRIMARY KEY DEFAULT nextval('user_id_seq'),
    email varchar(255) UNIQUE NOT NULL,
    status user_status DEFAULT 'active',
    created_at timestamp DEFAULT current_timestamp
);

COMMENT ON TABLE users IS 'Legacy user table';
COMMENT ON COLUMN users.id IS 'Primary identifier';
COMMENT ON COLUMN users.email IS 'User email address';

CREATE TABLE roles (
    id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

-- Table to be dropped in target
CREATE TABLE old_legacy_table (
    id int
);

-- 5. Views
CREATE VIEW active_users AS
SELECT id, email FROM users WHERE status = 'active';

-- 6. Functions
CREATE OR REPLACE FUNCTION get_user_status(uid integer) RETURNS user_status AS $$
BEGIN
    RETURN (SELECT status FROM users WHERE id = uid);
END;
$$ LANGUAGE plpgsql;

-- Function to be dropped in target
CREATE OR REPLACE FUNCTION get_legacy_data() RETURNS integer AS $$
BEGIN
    RETURN 1;
END;
$$ LANGUAGE plpgsql;

-- 7. Privileges
CREATE ROLE read_only_bob LOGIN PASSWORD 'bobpass';
GRANT SELECT ON users TO read_only_bob;
GRANT INSERT ON roles TO read_only_bob;

-- Tezos Delegation Service Database Schema
-- This schema defines the structure and indexes for storing Tezos delegation operations.

CREATE TABLE IF NOT EXISTS delegations (
    id SERIAL PRIMARY KEY,              -- Surrogate primary key for internal use
    tzkt_id BIGINT UNIQUE NOT NULL,     -- Unique identifier from the Tzkt API to prevent duplicates
    timestamp TIMESTAMP NOT NULL,       -- UTC timestamp of the delegation operation
    amount BIGINT NOT NULL,             -- Amount delegated (in mutez, 1 tez = 1,000,000 mutez)
    delegator TEXT NOT NULL,            -- Sender's (delegator's) address
    level BIGINT NOT NULL               -- Block height of the delegation
);

CREATE INDEX IF NOT EXISTS idx_timestamp_tzkt_id_desc ON delegations (timestamp DESC, tzkt_id DESC);
CREATE INDEX IF NOT EXISTS idx_year_timestamp_tzkt_id_desc ON delegations (EXTRACT(YEAR FROM timestamp), timestamp DESC, tzkt_id DESC);

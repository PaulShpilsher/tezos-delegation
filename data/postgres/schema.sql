CREATE TABLE IF NOT EXISTS delegations (
    id SERIAL PRIMARY KEY,
    tzkt_id BIGINT UNIQUE NOT NULL,     -- Unique identifier from the Tzkt API to prevent duplicates.
    timestamp TIMESTAMP NOT NULL,       -- UTC timestamp of the delegation operation.
    amount BIGINT NOT NULL,             -- Amount delegated (in mutez, as Tezos uses mutez where 1 tez = 1,000,000 mutez).
    delegator TEXT NOT NULL,            -- Sender's address.
    level BIGINT NOT NULL               -- Block height of the delegation.
);

CREATE INDEX idx_timestamp ON delegations (timestamp DESC);
CREATE INDEX idx_year ON delegations (EXTRACT(YEAR FROM timestamp));

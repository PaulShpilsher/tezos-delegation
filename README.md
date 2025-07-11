# Tezos Delegation Microservice

> **Take-Home Assignment**  
> _This project is a solution to the [Senior Backend Exercice](https://kilnfi.notion.site/Senior-Backend-Exercice-1ed0b785cb0f49719a83436998dd0548) for Kiln. Please see the link for the full requirements._

---

## Table of Contents
- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quickstart](#quickstart)
- [API Reference](#api-reference)
- [Architecture & Design](#architecture--design)
- [Implementation Details](#implementation-details)
- [Database Design](#database-design)
- [Assumptions & Limitations](#assumptions--limitations)
- [Possible Enhancements](#possible-enhancements)

---

## Overview
This microservice synchronizes Tezos delegation operations from the [Tzkt API](https://api.tzkt.io/) into a local PostgreSQL database and exposes a REST API for querying delegations with pagination and optional year filtering. It is designed for reliability, performance, and production-readiness, following best practices in Go microservice development.

---

## Prerequisites
- [Go](https://golang.org/) 1.24+
- [Docker](https://www.docker.com/) & [Docker Compose](https://docs.docker.com/compose/)

---

## Quickstart


### 1. Clone the repository
```sh
git clone git@github.com:PaulShpilsher/tezos-delegation.git
cd tezos-delegation
```

### 2. Start the service with Docker Compose
```sh
docker-compose up --build
```
- The API will be available at [http://localhost:3000/xtz/delegations](http://localhost:3000/xtz/delegations)
- PostgreSQL will be available at `localhost:5432`  

   For credentials, see the [`.env.docker`](./.env.docker) file.

### 3. Run tests
```sh
go test ./...
```

### Minimal Docker Image

This project includes a multi-stage Dockerfile that produces a minimal image using `FROM scratch` as the final stage. The resulting image contains only the statically-linked Go binary and CA certificates, yielding a very small and secure container.

**Key points:**
- The final image is built from scratch (no OS layer), minimizing attack surface and image size.
- Only the application binary and CA certificates are included.
- No shell, package manager, or extra files are present.


---

## API Reference

### GET `/xtz/delegations`
Retrieve a paginated list of Tezos delegations, optionally filtered by year. Entries are returned with the most recent first.

#### Query Parameters
| Name      | Type   | Required | Default | Description                                 |
|-----------|--------|----------|---------|---------------------------------------------|
| `page`    | int    | No       | 1       | Page number (must be >= 1)                  |
| `pageSize`| int    | No       | 50      | Items per page (1-1000)                     |
| `year`    | int    | No       | -       | Filter by year (>= 2018)                    |

#### Response
- **200 OK**
```json
{
  "data": [
    {
        "timestamp": "2022-05-05T06:29:14Z",
        "amount": "125896",
        "delegator": "tz1a1SAaXRt9yoGMx29rh9FsBF4UzmvojdTL",
        "level": "2338084"
    },
    {
        "timestamp": "2021-05-07T14:48:07Z",
        "amount": "9856354",
        "delegator": "KT1JejNYjmQYh8yw95u5kfQDRuxJcaUPjUnf",
        "level": "1461334"
    }
    ...
  ]
}
```
- **400 Bad Request**
```json
{ "error": "Invalid page parameter: too long" }
{ "error": "Invalid page parameter: must be a positive integer" }
{ "error": "Invalid year parameter: too long" }
{ "error": "Invalid year parameter: must be a valid year from 2018 onwards" }
```
- **500 Internal Server Error**
```json
{ "error": "Service temporarily unavailable" }
```

#### Possible Error Responses
| Status | Error Message                                            | Condition                                                        |
|--------|---------------------------------------------------------|------------------------------------------------------------------|
| 400    | Invalid page parameter: too long                        | `page` param > 10 chars                                          |
| 400    | Invalid page parameter: must be a positive integer      | `page` not int, < 1, or missing                                  |
| 400    | Invalid year parameter: too long                        | `year` param > 10 chars                                          |
| 400    | Invalid year parameter: must be a valid year from 2018 onwards | `year` not int, < 2018, or negative                              |
| 500    | Service temporarily unavailable                         | Database or unexpected error in service                          |

#### Example Requests
- **Default (first page, 50 results):**
```sh
curl 'http://localhost:3000/xtz/delegations'
```
- **With all parameters:**
```sh
curl 'http://localhost:3000/xtz/delegations?page=2&year=2022'
```
- **Missing/invalid parameter (error):**
```sh
curl 'http://localhost:3000/xtz/delegations?page=0'
# Response: { "error": "Invalid page parameter: must be a positive integer" }
```
- **Optional year omitted:**
```sh
curl 'http://localhost:3000/xtz/delegations?page=1'
```

---

## Architecture & Design

### High-Level Diagram
```
+-------------------+         +-------------------+         +-------------------+
|  Tzkt API         |<------->|  Poller Service   |<------->|  PostgreSQL DB    |
+-------------------+         +-------------------+         +-------------------+
                                                              ^
                                                              |
                                                      +-------------------+
                                                      | Delegation Service|
                                                      +-------------------+
                                                              ^
                                                              |
                                                      +-------------------+
                                                      |  REST API Server  |
                                                      +-------------------+
```

### Components
- **PollerService**: Periodically fetches new delegations from Tzkt, stores them in the DB. Handles historical sync and polling, with robust retry and rate-limit handling.
- **DelegationService**: Business logic for retrieving delegations with pagination and filtering.
- **API Layer**: Exposes `/xtz/delegations` endpoint, validates input, handles errors, and caches responses for 30s.
- **Repository Layer**: Handles DB access, ensures idempotency and efficient queries.
- **Config Layer**: Loads environment variables, supports different environments.

### Design Decisions
- **Hexagonal/Ports & Adapters**: Clear separation between core logic, infrastructure, and API.
- **Clean Architecture**: The project follows Clean Architecture principles, emphasizing separation of concerns, dependency inversion, and testability. Core business logic is isolated from frameworks and infrastructure, with clear boundaries between domain, application, and external layers (API, DB, external services).
- **Resilience**: Poller handles API rate limits, retries, and context cancellation for graceful shutdown.
- **Security**: Adds security headers to all responses.
- **Testing**: Extensive unit tests and mocks for all layers.

---

## Implementation Details

### Key Packages & Libraries
- [`github.com/kataras/iris/v12`](https://github.com/kataras/iris) — Web framework for REST API
- [`github.com/rs/zerolog`](https://github.com/rs/zerolog) — Structured logging
- [`github.com/lib/pq`](https://github.com/lib/pq) — PostgreSQL driver
- [`github.com/joho/godotenv`](https://github.com/joho/godotenv) — .env config loading
- [`github.com/stretchr/testify`](https://github.com/stretchr/testify) — Testing utilities
- [`github.com/golang/mock`](https://github.com/golang/mock) — Mock generation for interfaces

### Notable Implementation Points
- **PollerService**: 
  - Syncs all historical data on startup, then polls every minute.
  - Handles Tzkt API rate limits (HTTP 429/503), respects `Retry-After`, uses exponential backoff.
  - Graceful shutdown via context cancellation and WaitGroup.
  - Only fetches new delegations (using last Tzkt ID), utilizing cursor-based paging of the Tzkt API, which is indexed by Tzkt ID for efficient incremental sync.
- **API Handler**:
  - Validates and sanitizes all query parameters.
  - Returns clear error messages and status codes.
- **Repository**:
  - Uses transactions and `ON CONFLICT DO NOTHING` to avoid duplicates.
  - Efficiently paginates and filters by year using DB indexes.
- **Config**:
  - Loads from environment, with sensible defaults for local/dev.

---

## Database Design

### Schema
```sql

CREATE TABLE IF NOT EXISTS delegations (
    id SERIAL PRIMARY KEY,              -- Surrogate primary key for internal use
    tzkt_id BIGINT UNIQUE NOT NULL,     -- Unique identifier from the Tzkt API to prevent duplicates
    timestamp TIMESTAMP NOT NULL,       -- UTC timestamp of the delegation operation
    amount BIGINT NOT NULL,             -- Amount delegated (in mutez, 1 tez = 1,000,000 mutez)
    delegator TEXT NOT NULL,            -- Sender's (delegator's) address
    level BIGINT NOT NULL               -- Block height of the delegation
);

-- Constraints for data integrity and security
ALTER TABLE delegations ADD CONSTRAINT chk_amount_non_negative CHECK (amount >= 0);
ALTER TABLE delegations ADD CONSTRAINT chk_level_non_negative CHECK (level >= 0);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_timestamp_tzkt_id_desc ON delegations (timestamp DESC, tzkt_id DESC);
CREATE INDEX IF NOT EXISTS idx_year_timestamp_tzkt_id_desc ON delegations (EXTRACT(YEAR FROM timestamp), timestamp DESC, tzkt_id DESC);


```
- **Indexes**: Support fast pagination and year-based queries.
- **Constraints**: Ensure data integrity (no negative amounts/levels, unique Tzkt IDs).

---

## Assumptions & Limitations
- Only the `/xtz/delegations` endpoint is exposed (read-only API).
- The service assumes the Tzkt API is available and reliable; transient errors are retried.
- No authentication is implemented (could be added for production).
- The year filter is limited to years >= 2018.
- The service is stateless.
- No rate limiting is enforced on the API (TODO in code).

---

## Possible Enhancements
- Add authentication and authorization for API endpoints.
- Implement API rate limiting and request tracing.
- Add metrics and health endpoints for observability.
- Support more advanced filtering (by delegator, amount, etc).
- Add OpenAPI/Swagger documentation.
- Poller's Horizontal scaling
- Use a message queue for decoupling polling and ingestion.
- Add alerting for poller failures or data staleness.
- Optimize DB schema for very large datasets (partitioning, etc).

---


# Banking Simulator — Full Architecture & Flow Documentation

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [High-Level Architecture](#2-high-level-architecture)
3. [Service Breakdown](#3-service-breakdown)
   - [Spring Boot — Core Banking](#31-spring-boot--core-banking-service-port-8080)
   - [Gin — Transfer Processor](#32-gin--transfer-processor-service-port-8081)
   - [FastAPI — Fraud & Analytics](#33-fastapi--fraud--analytics-service-port-8082)
   - [Streamlit — Frontend](#34-streamlit--frontend-port-8501)
   - [Airflow — ETL Pipeline](#35-airflow--etl-pipeline-port-8083)
4. [Database Layer](#4-database-layer)
5. [Transfer Flow — End to End](#5-transfer-flow--end-to-end)
   - [IMPS Flow](#51-imps-flow-instant)
   - [NEFT / RTGS Flow](#52-neft--rtgs-flow-async)
6. [Temporal Workflow Engine](#6-temporal-workflow-engine)
7. [Fraud Detection](#7-fraud-detection)
8. [Airflow ETL Pipeline](#8-airflow-etl-pipeline)
9. [Frontend Pages](#9-frontend-pages)
10. [Docker & Infrastructure](#10-docker--infrastructure)
11. [Environment Variables](#11-environment-variables)
12. [API Reference](#12-api-reference)
13. [Key Design Decisions](#13-key-design-decisions)

---

## 1. Project Overview

This is a **multi-service banking simulator** that mimics real-world banking infrastructure using five distinct services, three databases, a workflow engine, and an ETL pipeline. It is designed to demonstrate how different technologies interact in a distributed system.

| What it simulates | Technology |
|---|---|
| Core banking (accounts, balances, transactions) | Spring Boot + PostgreSQL |
| Fund transfer processing (NEFT, RTGS, IMPS) | Go (Gin) + MongoDB |
| Fraud detection + analytics | Python (FastAPI) + SQLite |
| Durable workflow orchestration | Temporal |
| ETL data reconciliation | Apache Airflow |
| Monitoring dashboard | Streamlit |

---

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        STREAMLIT FRONTEND                           │
│                    http://localhost:8501                            │
│    Accounts | Transfer | Analytics | Pipeline                      │
└──────┬──────────────┬──────────────┬──────────────┬────────────────┘
       │              │              │              │
       ▼              ▼              ▼              ▼
┌──────────┐   ┌──────────┐  ┌──────────┐   ┌──────────────┐
│  SPRING  │   │   GIN    │  │  FASTAPI │   │  PostgreSQL  │
│   BOOT   │   │ SERVICE  │  │ SERVICE  │   │  (direct     │
│  :8080   │◄──┤  :8081   │  │  :8082   │   │   query for  │
│          │   │          │  │          │   │   Pipeline)  │
└────┬─────┘   └────┬─────┘  └────┬─────┘   └──────────────┘
     │              │              │
     ▼              ▼              ▼
┌──────────┐  ┌──────────┐  ┌──────────┐
│PostgreSQL│  │ MongoDB  │  │  SQLite  │
│ banking  │  │ banking  │  │ fraud.db │
│   :5432  │  │  :27017  │  │          │
└──────────┘  └──────────┘  └──────────┘
                   │
                   │  (Airflow reads both MongoDB + Postgres)
                   ▼
┌────────────────────────────────────────┐
│           APACHE AIRFLOW               │
│     http://localhost:8083              │
│   DAG: banking_etl_dag (*/5 * * * *)  │
│   Writes → transfer_pipeline table    │
│              in PostgreSQL             │
└────────────────────────────────────────┘
                   │
        ┌──────────┘
        ▼
┌──────────────────────────────────────┐
│           TEMPORAL                   │
│   (optional) localhost:7233          │
│   Durable workflow for NEFT/RTGS     │
│   settlement with retry & timeout    │
└──────────────────────────────────────┘
```

---

## 3. Service Breakdown

### 3.1 Spring Boot — Core Banking Service (port 8080)

**Language / Framework:** Java 17, Spring Boot 3, Spring Data JPA
**Database:** PostgreSQL (`banking` database)
**Role:** The authoritative source of truth for account balances and settled transactions.

#### What it owns:
- **Account management** — create, fetch, list accounts; stores balance
- **Transaction records** — every settled transfer creates a `Transaction` row
- **Balance updates** — debit from sender, credit to receiver during settlement

#### REST API:

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/accounts` | List all accounts |
| `POST` | `/accounts` | Create a new account |
| `GET` | `/accounts/{id}` | Get account by ID |
| `POST` | `/transactions/settle` | Settle a transfer (debit + credit + record) |
| `GET` | `/transactions` | List all transactions |
| `GET` | `/transactions/{id}` | Get a specific transaction |

#### PostgreSQL Schema:

**`accounts` table:**
```sql
id           UUID PRIMARY KEY
owner_name   VARCHAR
balance      DECIMAL(15,2)
created_at   TIMESTAMP
```

**`transactions` table:**
```sql
id             UUID PRIMARY KEY
correlation_id UUID UNIQUE
from_account   UUID (FK → accounts)
to_account     UUID (FK → accounts)
amount         DECIMAL(15,2)
transfer_mode  VARCHAR   -- IMPS | NEFT | RTGS
status         VARCHAR   -- SUCCESS | FAILED
created_at     TIMESTAMP
```

#### Key behaviour:
- `/transactions/settle` is called by the Gin service after the settlement delay
- It atomically debits `from_account`, credits `to_account`, and inserts a `transactions` row
- The TPIN is validated here — wrong TPIN returns an error, Gin marks the transfer FAILED

---

### 3.2 Gin — Transfer Processor Service (port 8081)

**Language / Framework:** Go 1.21, Gin HTTP framework
**Database:** MongoDB (`banking` database, `transfers` collection)
**Role:** Receives all transfer requests, writes them to MongoDB immediately, then orchestrates settlement (either instantly for IMPS or asynchronously for NEFT/RTGS).

#### REST API:

| Method | Endpoint | Description |
|---|---|---|
| `POST` | `/process-transfer` | Initiate a transfer |
| `GET` | `/transfer/:id` | Get transfer status by ID |
| `GET` | `/transfers` | List all transfers |

#### MongoDB Document Schema (`transfers` collection):

```json
{
  "_id":           "uuid-string",
  "correlation_id":"uuid-string",
  "from_account":  "uuid-string",
  "to_account":    "uuid-string",
  "amount":        200000.00,
  "transfer_mode": "NEFT",
  "status":        "PENDING",
  "created_at":    "2026-06-07T14:25:00Z"
}
```

#### Transfer Mode Behaviour:

| Mode | Initial Status | Settlement | Delay |
|---|---|---|---|
| **IMPS** | `SUCCESS` | Synchronous — settles inline in the request | None (instant) |
| **RTGS** | `PROCESSING` | Async — Temporal workflow or goroutine | 5 minutes |
| **NEFT** | `PENDING` | Async — Temporal workflow or goroutine | 10 minutes |

#### Internal components:

**`client/interface.go`** — `BankingClient` interface:
```go
type BankingClient interface {
    SettleTransfer(fromAccount, toAccount string, amount float64, transferMode, tpin, correlationID string) (*SettleTransferResponse, error)
}
```

**`client/core_banking_client.go`** — HTTP client that calls Spring Boot's `/transactions/settle`.

**`db/database.go`** — `TransferRepository` interface backed by `MongoTransferRepo`. Handles `Create`, `FindByID`, `UpdateStatus`, `FindAll` against MongoDB.

**`handlers/transfer.go`** — Main handler logic:
- For IMPS: calls `bankingClient.SettleTransfer()` inline, updates MongoDB status immediately
- For NEFT/RTGS: if Temporal is connected → starts `SettlementWorkflow`; else → launches `simulateSettlement()` goroutine as fallback

---

### 3.3 FastAPI — Fraud & Analytics Service (port 8082)

**Language / Framework:** Python 3.11, FastAPI, SQLAlchemy
**Database:** SQLite (`fraud.db`)
**Role:** Fraud risk assessment on transactions and aggregated analytics. Acts as a sidecar that reads from Spring Boot's API.

#### REST API:

| Method | Endpoint | Description |
|---|---|---|
| `GET` | `/health` | Health check |
| `GET` | `/fraud/{transaction_id}` | Assess fraud risk for a transaction, log result |
| `GET` | `/fraud/logs/all` | Return all fraud log entries |
| `GET` | `/analytics/summary` | Total count, volume, breakdown by mode and status |
| `GET` | `/analytics/high-value` | All transactions above ₹50,000 |

#### Fraud Risk Logic (`routers/fraud.py`):

```
Amount > ₹50,000  →  HIGH risk    "Transaction exceeds ₹50,000 threshold"
Amount > ₹10,000  →  MEDIUM risk  "Moderate transaction of ₹X"
Amount ≤ ₹10,000  →  LOW risk     "Transaction within normal range"
```

Every call to `/fraud/{transaction_id}`:
1. Fetches the transaction from Spring Boot via `CoreBankingClient`
2. Runs `assess_risk(amount)`
3. Writes a `FraudLog` row to SQLite
4. Returns the risk level and reason

#### SQLite Schema (`fraud.db`):

**`fraud_logs` table:**
```sql
id               UUID PRIMARY KEY
transaction_id   UUID
risk_level       VARCHAR   -- LOW | MEDIUM | HIGH
reason           TEXT
created_at       TIMESTAMP
```

#### Analytics Logic (`routers/analytics.py`):
- Calls Spring Boot `/transactions` to get all settled transactions
- `/analytics/summary` — aggregates count, total volume, breakdown by `transferMode` and `status`
- `/analytics/high-value` — filters transactions above ₹50,000 threshold

---

### 3.4 Streamlit — Frontend (port 8501)

**Language:** Python 3.11, Streamlit
**Role:** Single-page dashboard that acts as the user interface for the entire simulator.

#### Pages:

**Accounts Page:**
- Lists all accounts from Spring Boot (`GET /accounts`)
- Shows account name, balance, account ID (truncated)
- Create new account form → `POST /accounts`
- "Check Risk" button per account → calls FastAPI `/fraud/{txn_id}` for recent transaction

**Transfer Page:**
- Dropdown to pick sender and receiver (account names, not raw IDs)
- Enter amount, select mode (IMPS/NEFT/RTGS), enter TPIN
- Submits to Gin `POST /process-transfer`
- On NEFT/RTGS success: immediately fires `POST /api/v1/dags/banking_etl_dag/dagRuns` to Airflow
- Shows Transfer ID (UUID), amount, mode, status after submission
- High-value fraud check triggered automatically if amount > ₹50,000
- Lists all past transfers fetched from Gin `GET /transfers`
  - Shows Transfer ID, from → to names, mode badge, status badge, amount, date

**Analytics Page:**
- Calls FastAPI `/analytics/summary` and `/analytics/high-value`
- Shows: Total Transactions, Total Volume, Flagged (>50k) count
- Transfer mode breakdown table (IMPS/NEFT/RTGS counts)
- Status breakdown table (SUCCESS/FAILED/PENDING counts)
- High-value transaction cards with fraud risk level

**Pipeline Page:**
- Reads directly from PostgreSQL `transfer_pipeline` table (written by Airflow)
- Shows: Total Tracked, In-Flight count, Settled count, Avg Lag (seconds)
- Yellow `IN-FLIGHT` badge = exists in MongoDB but not yet in Postgres
- Green `SETTLED` badge = exists in both, shows actual settlement lag in seconds
- Shows both `mongo_status` and `pg_status` side by side per transfer
- Link to Airflow UI at `http://localhost:8083`

---

### 3.5 Airflow — ETL Pipeline (port 8083)

**Version:** Apache Airflow 2.9.3
**Schedule:** Every 5 minutes (`*/5 * * * *`)
**DAG location:** `fastapi-service/dags/banking_etl_dag.py`
**Role:** Extracts transfer data from MongoDB, extracts settled transactions from PostgreSQL, joins them, and writes a unified `transfer_pipeline` table back into PostgreSQL. Enables cross-database visibility.

Full detail in [Section 8](#8-airflow-etl-pipeline).

---

## 4. Database Layer

### PostgreSQL (`banking-db` container, port 5433 on host)

Owned by Spring Boot. Tables:
- `accounts` — all bank accounts and their balances
- `transactions` — all settled transfers (written when Spring Boot processes settlement)
- `transfer_pipeline` — written exclusively by Airflow ETL (cross-database join result)
- `airflow` — separate database in the same Postgres instance used by Airflow internals

### MongoDB (`banking-mongo` container, port 27017)

Owned by Gin service. Collections:
- `transfers` — every transfer request from the moment it is initiated. Status evolves: `PENDING` → `SUCCESS` / `FAILED`.

**Key difference from Postgres transactions:** MongoDB gets the record immediately at T+0. Postgres only gets it after the settlement delay (5 or 10 minutes). This gap is what Airflow makes visible.

### SQLite (`fraud.db` inside FastAPI container)

Owned by FastAPI. Tables:
- `fraud_logs` — one row per fraud check performed, with risk level and reason

---

## 5. Transfer Flow — End to End

### 5.1 IMPS Flow (Instant)

```
User submits transfer (IMPS, ₹X)
         │
         ▼
Streamlit POST /process-transfer → Gin
         │
         ▼
Gin: validates request, parses UUIDs
Gin: inserts doc into MongoDB {status: "SUCCESS", ...}
         │
         ▼
Gin: calls bankingClient.SettleTransfer() → Spring Boot POST /transactions/settle
         │
         ├─ TPIN correct → Spring Boot debits from_account, credits to_account
         │                  inserts row into transactions table {status: "SUCCESS"}
         │                  returns {status: "SUCCESS"}
         │                  Gin updates MongoDB {status: "SUCCESS"}
         │
         └─ TPIN wrong →  Spring Boot returns error
                          Gin updates MongoDB {status: "FAILED"}
         │
         ▼
Gin responds 202 to Streamlit with transfer details
Streamlit shows result immediately
```

**Databases touched:** MongoDB (insert + update), PostgreSQL (insert transactions row)
**Total time:** < 1 second

---

### 5.2 NEFT / RTGS Flow (Async)

```
User submits transfer (NEFT/RTGS, ₹X)
         │
         ▼
Streamlit POST /process-transfer → Gin
         │
         ▼
Gin: validates request, parses UUIDs
Gin: inserts doc into MongoDB
     NEFT → {status: "PENDING"}
     RTGS → {status: "PROCESSING"}
         │
         ▼
Gin responds 202 immediately ← user sees "initiated"

         ┌──── Temporal connected? ────┐
         │                             │
     YES │                          NO │
         ▼                             ▼
Temporal workflow starts         goroutine launched
(settlement-{transferID})        go simulateSettlement(...)
         │                             │
         └──────────┬──────────────────┘
                    │
         ▼ (NEFT: wait 10 min / RTGS: wait 5 min)

Streamlit simultaneously fires:
POST /api/v1/dags/banking_etl_dag/dagRuns → Airflow
         │
         ▼
Airflow DAG runs:
  - MongoDB shows PENDING/PROCESSING (in-flight)
  - PostgreSQL has no record yet
  - transfer_pipeline written: {is_in_flight: true}
         │
         ▼
Pipeline page shows: IN-FLIGHT ✅

         ▼ (after delay expires)

SettleTransfer() called → Spring Boot
  - Debit from_account, credit to_account
  - Insert transactions row {status: "SUCCESS"}
  - Return {status: "SUCCESS"}
         │
         ▼
MongoDB updated → {status: "SUCCESS"}
Transfer page now shows: SUCCESS
         │
         ▼
Next Airflow DAG run (scheduled or triggered):
  - MongoDB: SUCCESS ✓
  - PostgreSQL: SUCCESS ✓
  - transfer_pipeline updated: {is_in_flight: false, pg_status: "SUCCESS", lag: Xs}
         │
         ▼
Pipeline page shows: SETTLED with lag time
```

---

## 6. Temporal Workflow Engine

**What it is:** Temporal is a durable workflow orchestration platform. It provides reliability guarantees that a plain goroutine cannot — workflows survive process restarts, have built-in retry logic, and are fully observable.

**Task Queue:** `settlement-task-queue`

**Worker registration (`temporal/worker.go`):**
```go
w.RegisterWorkflow(workflows.SettlementWorkflow)
w.RegisterActivity(act)   // act = &SettlementActivity{BankingClient: ...}
```

**Workflow (`temporal/workflows/settlement_workflow.go`):**
1. Sleeps for the mode-appropriate delay (NEFT=10min, RTGS=5min)
2. Executes `SettleTransfer` activity with retry policy:
   - Max attempts: 3
   - Initial interval: 10 seconds
   - Backoff coefficient: 2.0
   - StartToCloseTimeout: 15 minutes

**Activity (`temporal/activities/settlement_activity.go`):**
1. Calls `bankingClient.SettleTransfer()` → Spring Boot
2. Updates transfer status in MongoDB via `db.Repo.UpdateStatus()`
3. Returns error if Spring Boot fails → Temporal retries automatically

**Fallback behaviour:**
If Temporal is not running (e.g., `temporal server start-dev` not started), Gin falls back to a plain Go goroutine (`simulateSettlement`) with the same delays but no retry/durability guarantees.

**How to run Temporal locally:**
```bash
make temporal-run
# runs: temporal server start-dev --db-filename temporal-data.db
```

---

## 7. Fraud Detection

Fraud detection is triggered in two ways:

**1. Automatic — during transfer submission (Streamlit):**
```
if amount > 50,000:
    Streamlit calls GET /fraud/{transaction_id} on FastAPI
```
This happens client-side in Streamlit right after a transfer is submitted, if the amount exceeds the threshold.

**2. Manual — from Accounts page:**
The "Check Risk" button on each account calls `/fraud/{txn_id}` for a specific transaction.

**Flow:**
```
FastAPI /fraud/{transaction_id}
    │
    ├── Fetch transaction from Spring Boot GET /transactions/{id}
    ├── Extract amount, transfer_mode
    ├── assess_risk(amount):
    │       > 50,000 → HIGH
    │       > 10,000 → MEDIUM
    │       ≤ 10,000 → LOW
    ├── Write FraudLog row to SQLite fraud.db
    └── Return {risk_level, reason, fraud_log_id}
```

**Limitation:** The risk assessment is purely amount-based (threshold rules). No ML model, no velocity checks, no pattern analysis — intentionally simple for the simulator.

---

## 8. Airflow ETL Pipeline

### Why it exists

MongoDB (owned by Gin) and PostgreSQL (owned by Spring Boot) are completely separate databases with no shared schema. After a NEFT/RTGS transfer is initiated:
- MongoDB knows about it immediately (T+0)
- PostgreSQL only knows about it after settlement (T+5min or T+10min)

Airflow bridges this gap by running an ETL job that joins both databases and writes a unified view.

### DAG: `banking_etl_dag`

**File:** `fastapi-service/dags/banking_etl_dag.py`
**Schedule:** Every 5 minutes (`*/5 * * * *`)
**Trigger:** Also fired immediately when a NEFT/RTGS transfer is submitted from Streamlit

**Task graph:**
```
ensure_schema → sync_pipeline
```

**Task 1: `ensure_schema`**
Creates `transfer_pipeline` table in PostgreSQL if it doesn't exist:
```sql
CREATE TABLE IF NOT EXISTS transfer_pipeline (
    mongo_transfer_id    TEXT PRIMARY KEY,
    from_account_id      TEXT,
    to_account_id        TEXT,
    amount               NUMERIC(15,2),
    transfer_mode        TEXT,
    mongo_status         TEXT,
    pg_status            TEXT,
    is_in_flight         BOOLEAN GENERATED ALWAYS AS (pg_status IS NULL) STORED,
    initiated_at         TIMESTAMP,
    settled_at           TIMESTAMP,
    settlement_lag_secs  INTEGER,
    etl_synced_at        TIMESTAMP DEFAULT NOW()
);
```

`is_in_flight` is a **PostgreSQL generated column** — always computed as `(pg_status IS NULL)`. No application code sets this; Postgres computes it automatically.

**Task 2: `sync_pipeline`**

1. **Extract from MongoDB:** `db.transfers.find({})` — gets all transfers regardless of status
2. **Extract from PostgreSQL:** `SELECT * FROM transactions` — gets only settled transfers
3. **Reconcile:** For each MongoDB transfer, find the matching Postgres transaction using the exact `correlation_id` that was generated at initiation and propagated through the system.
4. **Upsert:** `INSERT ... ON CONFLICT (mongo_transfer_id) DO UPDATE` — idempotent, safe to run repeatedly

**Matching logic (Correlation ID):**
MongoDB and Postgres share no common foreign key by default. To solve this, the Gin service generates a unique `correlation_id` when a transfer is initiated. This ID is saved in MongoDB, passed to Spring Boot during settlement, and saved in Postgres. The Airflow DAG joins the two distinct databases precisely using this shared `correlation_id`.

**What a row looks like:**
```
In-Flight transfer (MongoDB knows, Postgres doesn't yet):
  mongo_transfer_id: abc-123
  mongo_status:      PENDING
  pg_status:         NULL          ← Postgres hasn't settled it yet
  is_in_flight:      TRUE          ← generated automatically
  initiated_at:      2026-06-07 14:25:00
  settled_at:        NULL
  settlement_lag_secs: NULL

Settled transfer (both databases agree):
  mongo_transfer_id: abc-123
  mongo_status:      SUCCESS
  pg_status:         SUCCESS       ← Postgres confirmed
  is_in_flight:      FALSE
  initiated_at:      2026-06-07 14:25:00
  settled_at:        2026-06-07 14:35:00
  settlement_lag_secs: 600         ← 10 minutes for NEFT
```

### Airflow Infrastructure

Three containers in docker-compose:

| Container | Role |
|---|---|
| `airflow-init` | One-time setup: creates airflow DB, runs `airflow db migrate`, creates admin user. Exits after completion. |
| `airflow-webserver` | Serves Airflow UI at port 8083. Login: `admin` / `admin` |
| `airflow-scheduler` | Watches DAG files, triggers runs on schedule, manages task execution |

**Build:** All three build from `fastapi-service/airflow.Dockerfile`:
```dockerfile
FROM apache/airflow:2.9.3-python3.11
USER root
RUN apt-get update && apt-get install -y postgresql-client
USER airflow
RUN pip install psycopg2-binary==2.9.9 pymongo==4.6.1
```

**DAG volume mount:** `./fastapi-service/dags:/opt/airflow/dags` — DAG files are not baked into the image; they're live-mounted so changes are picked up immediately.

---

## 9. Frontend Pages

### Streamlit Page Structure

```
app.py
├── CSS & design system (dark theme, CSS variables, card components)
├── Helper functions
│   ├── get()         ─ HTTP GET with error handling
│   ├── post()        ─ HTTP POST with error handling
│   ├── fmt_currency()─ format ₹ amounts
│   ├── fmt_date()    ─ format ISO timestamps
│   ├── fmt_id()      ─ truncate UUID to 8 chars + "..."
│   └── status_badge()─ coloured HTML badge for statuses
├── Sidebar
│   ├── Service health indicators (Spring Boot, Gin, FastAPI)
│   └── Navigation radio (Accounts | Transfer | Analytics | Pipeline)
├── Accounts Page
├── Transfer Page
├── Analytics Page
└── Pipeline Page
```

### API calls by page

| Page | Service called | Endpoint |
|---|---|---|
| Accounts | Spring Boot | `GET /accounts` |
| Accounts — create | Spring Boot | `POST /accounts` |
| Accounts — risk check | FastAPI | `GET /fraud/{txn_id}` |
| Transfer — submit | Gin | `POST /process-transfer` |
| Transfer — DAG trigger | Airflow | `POST /api/v1/dags/banking_etl_dag/dagRuns` |
| Transfer — list | Gin | `GET /transfers` |
| Transfer — fraud auto-check | FastAPI | `GET /fraud/{txn_id}` |
| Analytics | FastAPI | `GET /analytics/summary` |
| Analytics — high value | FastAPI | `GET /analytics/high-value` |
| Pipeline | PostgreSQL | Direct `SELECT * FROM transfer_pipeline` |

---

## 10. Docker & Infrastructure

### Services and ports

| Container name | Image / Build | Port (host→container) | Role |
|---|---|---|---|
| `banking-db` | `postgres:15-alpine` | `5433:5432` | PostgreSQL |
| `banking-mongo` | `mongo:7-jammy` | `27017:27017` | MongoDB |
| `core-banking` | `./spring-boot-service` | `8080:8080` | Spring Boot |
| `transfer-processor` | `./gin-service` | `8081:8081` | Gin |
| `fraud-analytics` | `./fastapi-service` | `8082:8082` | FastAPI |
| `banking-frontend` | `./streamlit-service` | `8501:8501` | Streamlit |
| `airflow-init` | `./fastapi-service/airflow.Dockerfile` | — | One-time init |
| `airflow-webserver` | `./fastapi-service/airflow.Dockerfile` | `8083:8080` | Airflow UI |
| `airflow-scheduler` | `./fastapi-service/airflow.Dockerfile` | — | DAG scheduler |

### Networks and Volumes

```yaml
networks:
  banking-net:        # All containers share this bridge network

volumes:
  pgdata:             # PostgreSQL data persistence
  mongodata:          # MongoDB data persistence
  airflow-logs:       # Airflow task execution logs
```

### Startup dependency chain

```
postgres (healthy) ──► spring-boot-service
                   ──► airflow-init ──► (airflow-init exits)
                                        airflow-webserver (restart:unless-stopped)
                                        airflow-scheduler (restart:unless-stopped)

mongo (healthy) ──► gin-service

spring-boot + gin + fastapi ──► streamlit
```

### Makefile targets

```bash
make up            # docker-compose up -d
make build         # docker-compose up -d --build
make down          # docker-compose down
make temporal-run  # temporal server start-dev --db-filename temporal-data.db
make test-fastapi  # pytest with coverage for FastAPI
make test-gin      # go test ./... with coverage for Gin
make test-spring   # gradlew clean test jacocoTestReport
make clean         # remove all coverage artifacts
```

---

## 11. Environment Variables

Configured via `.env` file in the project root:

| Variable | Used by | Description |
|---|---|---|
| `DB_URL` | Spring Boot | `jdbc:postgresql://banking-db:5432/banking` |
| `DB_USERNAME` | Spring Boot | `postgres` |
| `DB_PASSWORD` | Spring Boot | `password` |
| `SPRING_BOOT_URL` | Gin, Streamlit, FastAPI | `http://core-banking:8080` |
| `GIN_URL` | Streamlit | `http://transfer-processor:8081` |
| `FASTAPI_URL` | Streamlit | `http://fraud-analytics:8082` |
| `DATABASE_URL` | FastAPI | SQLite path for fraud.db |
| `MONGO_URI` | Gin, Airflow | `mongodb://banking-mongo:27017` |
| `MONGO_DB` | Gin, Airflow | `banking` |
| `PG_HOST` | Streamlit, Airflow | `postgres` |
| `PG_USER` | Streamlit, Airflow | `postgres` |
| `PG_PASSWORD` | Streamlit, Airflow | `password` |
| `PG_DB` | Streamlit, Airflow | `banking` |
| `AIRFLOW_URL` | Streamlit | `http://airflow-webserver:8080` |
| `TEMPORAL_HOST` | Gin | `host.docker.internal:7233` |

---

## 12. API Reference

### Gin Service (Transfer Processor) — `:8081`

```
POST /process-transfer
Body: {
  "from_account":  "uuid",
  "to_account":    "uuid",
  "amount":        200000.00,
  "transfer_mode": "NEFT",     // IMPS | NEFT | RTGS
  "tpin":          "1234"
}
Response 202: { "message": "...", "transfer": { Transfer object } }

GET /transfer/:id
Response 200: { Transfer object }

GET /transfers
Response 200: [ Transfer, Transfer, ... ]
```

### Spring Boot (Core Banking) — `:8080`

```
GET  /accounts                  → list all accounts
POST /accounts                  → create account { owner_name, initial_balance }
GET  /accounts/:id              → get account by UUID
POST /transactions/settle       → settle a transfer
GET  /transactions              → list all transactions
GET  /transactions/:id          → get transaction by UUID
```

### FastAPI (Fraud & Analytics) — `:8082`

```
GET /health                     → { status: "fastapi-service running" }
GET /fraud/{transaction_id}     → assess risk, log to SQLite, return result
GET /fraud/logs/all             → all fraud log entries
GET /analytics/summary          → total count, volume, breakdown by mode/status
GET /analytics/high-value       → transactions > ₹50,000
```

### Airflow REST API — `:8083`

```
POST /api/v1/dags/banking_etl_dag/dagRuns
Auth: Basic admin:admin
Body: { "dag_run_id": "unique-run-id" }
→ Triggers an immediate DAG run
```

---

## 13. Key Design Decisions

### Why MongoDB for transfers?
Transfer initiation is a write-heavy, schema-flexible operation. MongoDB allows the Gin service to write a transfer document immediately and evolve its structure without migrations. The status field (`PENDING → SUCCESS/FAILED`) is updated in-place.

### Why PostgreSQL for core banking?
Account balances and settled transactions require ACID guarantees. A double-entry bookkeeping system (debit one account, credit another) must be atomic. PostgreSQL's transaction support ensures no money is created or destroyed during settlement.

### Why SQLite for fraud logs?
Fraud logs are append-only, low-volume, and local to the FastAPI service. SQLite requires zero infrastructure and is perfectly adequate for a simulator.

### Why Temporal for NEFT/RTGS?
A plain goroutine will lose its state if the Gin process crashes mid-settlement. Temporal workflows are durable — if the worker restarts, the workflow resumes from where it left off. The retry policy also handles transient Spring Boot failures automatically.

### Why Airflow for the ETL?
Airflow provides:
1. **Scheduled execution** — runs every 5 minutes without manual intervention
2. **Task dependency graph** — `ensure_schema` must complete before `sync_pipeline`
3. **Observable history** — every DAG run, task duration, and log is visible in the Airflow UI
4. **Idempotent upserts** — running the same DAG twice produces the same result

### The two-database gap
MongoDB and PostgreSQL share no natural foreign key. The reconciliation uses a shared `correlation_id` that is generated by the Gin service at initiation and passed all the way through to Spring Boot's PostgreSQL database during settlement. This allows Airflow to perform a precise, exact join across the two otherwise disconnected databases.

### Why the Pipeline page reads Postgres directly?
The Pipeline page bypasses all microservices and queries `transfer_pipeline` directly from PostgreSQL using `psycopg2`. This is intentional — the table is written by Airflow (not by any service), so there is no microservice API to call. Streamlit talks to Postgres directly in this case.

### ETL latency is a feature, not a bug
The gap between the Transfer page (live from MongoDB) and the Pipeline page (Airflow snapshot) is intentional. It demonstrates a real concept in data engineering: operational systems (MongoDB, Postgres) serve live requests; analytical systems (Airflow + transfer_pipeline) provide reconciled, historically accurate views with inherent latency.

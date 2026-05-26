# Banking Simulator — Codebase Overview

## What is this project?

A **multi-service banking simulator** that mimics a real banking backend. It splits responsibilities across **three backend microservices** (Spring Boot, Gin, FastAPI) and one **Streamlit frontend**, all wired together via HTTP REST calls and containerized with Docker Compose.

---

## The Big Picture — Communication Map

```
                        ┌──────────────────────────────┐
                        │   Streamlit Frontend  :8501  │
                        │   (banking-frontend)         │
                        └──────┬────────┬──────┬───────┘
                               │        │      │
          ┌────────────────────┘        │      └──────────────────┐
          │  /accounts, /transactions   │  /process-transfer      │  /analytics/summary
          │  (account mgmt)             │  /transfer/:id          │  /analytics/high-value
          ▼                             ▼                          ▼  /fraud/{id}
┌─────────────────┐          ┌──────────────────┐      ┌──────────────────────┐
│  Spring Boot    │◄─────────│   Gin Service    │      │   FastAPI Service    │
│  :8080          │ settle   │   :8081          │      │   :8082              │
│  (core-banking) │ transfer │ (transfer-       │      │  (fraud-analytics)   │
└─────────────────┘          │  processor)      │      └──────────┬───────────┘
         ▲                   └──────────────────┘                 │
         │                                                         │ reads transactions
         └─────────────────────────────────────────────────────────┘
```

> **Key insight**: Streamlit talks to **ALL THREE** backend services directly. It's not just one gateway — each page uses a different backend.

---

## Service 1 — Spring Boot (Port 8080) — "Core Banking"

**Role**: The **central database** of the entire system. All accounts and settled transactions live here.

### What it does
- **Account management** — create, read, list accounts (with hashed TPIN stored)
- **Transaction ledger** — store, retrieve, and settle transactions (with actual debit/credit from balances)
- **Balance management** — when a transaction settles, it deducts from sender and adds to receiver

### API Endpoints
| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/accounts` | Create a new account (name, initialBalance, tpin) |
| `GET` | `/accounts` | List all accounts |
| `GET` | `/accounts/{id}` | Get a specific account |
| `GET` | `/accounts/{id}/transactions` | Transaction history for an account |
| `POST` | `/transactions/settle` | **Settle a transfer** — validates TPIN, debits/credits balances |
| `GET` | `/transactions/{id}` | Get a single transaction |
| `GET` | `/transactions/all` | Get all transactions |

### Who talks to it?
- **Gin service** calls `/transactions/settle` to actually move money
- **FastAPI service** calls `/transactions/all` and `/transactions/{id}` for analytics/fraud
- **Streamlit** calls `/accounts` and `/accounts/{id}/transactions` directly for the Accounts page

---

## Service 2 — Gin (Port 8081) — "Transfer Processor"

**Role**: **Orchestrates money transfers**. It's the entry point for initiating a transfer.

### What it does
- Accepts a transfer request (from, to, amount, mode: NEFT/RTGS/IMPS, TPIN)
- Creates a local transfer record in its **own database** (SQLite/Postgres) with an initial status
- Calls Spring Boot's `/transactions/settle` to actually debit/credit money
- **Simulates real banking settlement delays:**
  - **IMPS** → Instant (synchronous, settled immediately)
  - **RTGS** → 15-second delay (async goroutine)
  - **NEFT** → 30-second delay (async goroutine)
- Lets you check transfer status via ID

### API Endpoints
| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/process-transfer` | Initiate a new transfer |
| `GET` | `/transfer/{id}` | Check status of a transfer |
| `GET` | `/health` | Health check |

### Who talks to it?
- **Streamlit** calls `/process-transfer` and `/transfer/{id}` on the Transfer page

---

## Service 3 — FastAPI (Port 8082) — "Fraud & Analytics"

**Role**: **Read-only analytics and fraud detection** layer. It doesn't store banking data — it reads from Spring Boot and adds its own fraud logs.

### What it does
- **Fraud Detection** — given a transaction ID, fetches it from Spring Boot and classifies it:
  - `LOW` → below ₹10,000
  - `MEDIUM` → ₹10,000 – ₹50,000
  - `HIGH` → above ₹50,000
  - Stores a `FraudLog` entry in its own SQLite database
- **Analytics** — aggregates data from Spring Boot:
  - Total transaction count, total volume
  - Breakdown by transfer mode (NEFT/RTGS/IMPS)
  - Breakdown by status (SUCCESS/PENDING/FAILED)
  - High-value transactions list (above ₹50,000 threshold)

### API Endpoints
| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/fraud/{transaction_id}` | Check fraud risk for a transaction |
| `GET` | `/fraud/logs/all` | All historical fraud log entries |
| `GET` | `/analytics/summary` | Transaction stats (count, volume, by-mode, by-status) |
| `GET` | `/analytics/high-value` | Transactions above ₹50,000 |
| `GET` | `/health` | Health check |

### Who talks to it?
- **Streamlit** calls `/analytics/summary` and `/analytics/high-value` on the Analytics page

---

## Streamlit Frontend (Port 8501) — "Banking UI"

**Role**: The dashboard. Connects directly to **all three** backend services.

### Pages
| Page | Talks to | What it shows |
|------|----------|---------------|
| **Accounts** | Spring Boot `:8080` | List accounts, create account, view transaction history per account |
| **Transfer** | Gin `:8081` | Initiate NEFT/RTGS/IMPS transfer, check transfer status by ID |
| **Analytics** | FastAPI `:8082` | Summary stats, high-value transactions list |

### Sidebar
- Shows live health status (green/red dot) for all three services
- Navigation between the three pages

---

## Data Flow — A Full Transfer Example

1. **User clicks "Initiate Transfer" on Streamlit**
2. Streamlit POSTs to **Gin** `/process-transfer` with `{from, to, amount, mode, tpin}`
3. Gin creates a local transfer record (status = PENDING/PROCESSING depending on mode)
4. Gin calls **Spring Boot** `/transactions/settle` — Spring Boot validates TPIN, debits sender, credits receiver, stores transaction
5. Gin updates its local record status to SUCCESS/FAILED
6. Streamlit shows the result (transfer ID, status)
7. Later, user goes to **Analytics page**
8. Streamlit calls **FastAPI** `/analytics/summary` — FastAPI fetches all transactions from Spring Boot and returns aggregated stats

---

## Summary Table

| Service | Language | Port | Database | Role |
|---------|----------|------|----------|------|
| Spring Boot | Java | 8080 | PostgreSQL | Core banking: accounts + transaction ledger |
| Gin | Go | 8081 | PostgreSQL | Transfer processor + settlement simulation |
| FastAPI | Python | 8082 | SQLite | Fraud detection + analytics (read-only) |
| Streamlit | Python | 8501 | None | Frontend dashboard |

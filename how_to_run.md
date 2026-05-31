# How to Run — Banking Simulator

## Prerequisites

| Tool | Purpose |
|------|---------| 
| Docker Desktop | Run everything via docker-compose (easiest) |
| Java 17+ & Maven | Run Spring Boot locally |
| Go 1.21+ | Run Gin locally |
| Python 3.11+ | Run FastAPI & Streamlit locally |
| PostgreSQL | Required by Spring Boot, Gin, and Temporal |

---

## Option 1 — Docker Compose (Recommended)

Everything — including Temporal — starts with a single command.

### Step 1 — Verify the root `.env`

Open [.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/.env) — the defaults already work for Docker:

```env
DB_USERNAME=postgres
DB_PASSWORD=password
DB_NAME=banking
DB_HOST=postgres          # Docker container name — do not change
DB_PORT=5432
DB_SSLMODE=disable

DB_URL=jdbc:postgresql://postgres:5432/banking
DATABASE_URL=sqlite:///./fraud.db

# Inter-service URLs — use Docker container names, not localhost
SPRING_BOOT_URL=http://core-banking:8080
GIN_URL=http://transfer-processor:8081
FASTAPI_URL=http://fraud-analytics:8082
```

> [!IMPORTANT]
> `TEMPORAL_HOST` is hardcoded as `temporal:7233` directly in `docker-compose.yml` — you do **not** need to add it to the `.env` file.

### Step 2 — Build and start all services

```bash
cd "d:\InternProject - Likith\banking-simulator-main"
docker-compose up --build
```

This starts **7 containers** in the correct order:

```
1. postgres         (banking DB — Temporal also uses this)
2. temporal         (waits for postgres to be healthy)
3. temporal-ui      (waits for temporal)
4. spring-boot      (waits for postgres)
5. gin-service      (waits for postgres + spring-boot + temporal)
6. fastapi-service
7. streamlit
```

### Step 3 — Open the app

| Service | URL | What it is |
|---------|-----|------------|
| Streamlit UI | http://localhost:8501 | Banking dashboard |
| Spring Boot API | http://localhost:8080 | Core banking REST API |
| Gin API | http://localhost:8081 | Transfer processor REST API |
| FastAPI | http://localhost:8082 | Fraud & analytics REST API |
| FastAPI Swagger | http://localhost:8082/docs | Interactive API docs |
| **Temporal UI** | **http://localhost:8088** | **View NEFT/RTGS workflow history** |

### Step 4 — Verify Temporal is working

1. Open http://localhost:8088 — you should see the Temporal dashboard
2. Initiate a **NEFT** or **RTGS** transfer from the Streamlit UI
3. Go back to http://localhost:8088 → you'll see a running workflow with a countdown timer
4. After 30s (NEFT) or 15s (RTGS) the workflow completes and status updates to `SUCCESS`

### Useful Docker commands

```bash
# Run in background
docker-compose up --build -d

# View logs for a specific service
docker-compose logs -f gin-service
docker-compose logs -f temporal

# Stop everything (keeps DB data)
docker-compose down

# Stop and wipe ALL data including DB volumes
docker-compose down -v

# Restart just one service
docker-compose restart gin-service
```

---

## Option 2 — Run Each Service Locally (No Docker)

Use this when you want to debug a single service without spinning up everything.

> [!NOTE]
> Each service has its own `.env` file that points to `localhost` instead of Docker container names.

> [!WARNING]
> For Temporal to work locally, you need the Temporal server running. The easiest way is to start **only** the Temporal containers from Docker while running your services locally — see the tip below.

---

### Tip — Run only Temporal via Docker (for local dev)

```bash
cd "d:\InternProject - Likith\banking-simulator-main"
docker-compose up postgres temporal temporal-ui -d
```

This gives you Temporal at `localhost:7233` and its UI at http://localhost:8088 without starting the other services.

> [!NOTE]
> If Temporal is not running, the Gin service **still works** — NEFT/RTGS transfers fall back to the goroutine-based settlement automatically. You'll see this warning in the logs:
> `Warning: could not connect to Temporal — falling back to goroutine settlement`

---

### Spring Boot (Port 8080)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\spring-boot-service"
mvn spring-boot:run
```

Reads `DB_URL`, `DB_USERNAME`, `DB_PASSWORD` from environment. Defaults to `localhost:5432/banking`.

**Verify:** http://localhost:8080/accounts

---

### Gin (Port 8081)

The Gin service loads [gin-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/.env) automatically via `godotenv`.

```bash
cd "d:\InternProject - Likith\banking-simulator-main\gin-service"
go run main.go
```

The `.env` already has `TEMPORAL_HOST=localhost:7233`. If Temporal is running, NEFT/RTGS use durable workflows. If not, they fall back to goroutines.

**Verify:** http://localhost:8081/health

To run tests:
```bash
go test ./...

# With coverage
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

---

### FastAPI (Port 8082)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\fastapi-service"
pip install -r requirements.txt
uvicorn main:app --reload --port 8082
```

**Verify:** http://localhost:8082/health  
**Swagger docs:** http://localhost:8082/docs

---

### Streamlit (Port 8501)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\streamlit-service"
pip install -r requirements.txt
streamlit run app.py
```

**Verify:** http://localhost:8501

---

## .env Files — What Goes Where

| File | Used by | When |
|------|---------|------|
| [.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/.env) | `docker-compose.yml` | Docker mode only |
| [gin-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/.env) | `godotenv` in `main.go` | Local Go run only |
| [fastapi-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/fastapi-service/.env) | `python-dotenv` | Local Python run only |
| [streamlit-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/streamlit-service/.env) | `os.getenv` with defaults | Local Streamlit run only |

> [!WARNING]
> Never commit the `.env` files to Git — they contain DB credentials. The `.gitignore` already excludes them.

---

## Startup Order (local runs only)

Docker Compose handles this automatically. For manual local runs, follow this order:

```
1. PostgreSQL           (must be first — everything depends on it)
2. Temporal server      (needs PostgreSQL)
3. Spring Boot  :8080   (core banking — Gin and FastAPI depend on it)
4. Gin          :8081   (connects to Temporal on startup)
5. FastAPI      :8082   (calls Spring Boot for data)
6. Streamlit    :8501   (calls all three above)
```

---

## How Temporal Settlement Works

When you initiate a **NEFT** or **RTGS** transfer:

```
POST /process-transfer  (Streamlit → Gin)
         │
         ▼
Gin writes transfer to DB  (status = PENDING / PROCESSING)
         │
         ▼
Gin starts a Temporal Workflow  ──→  visible at http://localhost:8088
         │
         ▼
workflow.Sleep(30s for NEFT / 15s for RTGS)   ← durable, survives restarts
         │
         ▼
Activity: call Spring Boot /transactions/settle
         │
         ▼
Gin DB updated to SUCCESS or FAILED
```

**IMPS** is still instant — it calls Spring Boot synchronously with no Temporal involved.

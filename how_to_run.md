# How to Run — Banking Simulator

## Prerequisites

| Tool | Purpose |
|------|---------|
| Docker Desktop | Run everything via docker-compose (easiest) |
| Java 17+ & Maven | Run Spring Boot locally |
| Go 1.21+ | Run Gin locally |
| Python 3.11+ | Run FastAPI & Streamlit locally |
| PostgreSQL | Required by Spring Boot and Gin |

---

## Option 1 — Docker Compose (Recommended)

### Step 1 — Fill in the root `.env`

Open [.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/.env) and set your real Postgres details:

```env
DB_USERNAME=postgres
DB_PASSWORD=yourpassword
DB_NAME=banking
DB_HOST=your-db-host        # e.g. your cloud DB hostname
DB_PORT=5432

DB_URL=jdbc:postgresql://your-db-host:5432/banking
DATABASE_URL=sqlite:///./fraud.db

# These use Docker container names — don't change these
SPRING_BOOT_URL=http://core-banking:8080
GIN_URL=http://transfer-processor:8081
FASTAPI_URL=http://fraud-analytics:8082
```

> [!IMPORTANT]
> `DB_HOST` is your actual Postgres host. `SPRING_BOOT_URL`, `GIN_URL`, `FASTAPI_URL` use Docker container names (not localhost) — **do not change those**.

### Step 2 — Build and start all services

```bash
cd "d:\InternProject - Likith\banking-simulator-main"
docker-compose up --build
```

### Step 3 — Open the app

| Service | URL |
|---------|-----|
| Streamlit UI | http://localhost:8501 |
| Spring Boot API | http://localhost:8080 |
| Gin API | http://localhost:8081 |
| FastAPI | http://localhost:8082 |

### Useful Docker commands

```bash
# Run in background
docker-compose up --build -d

# View logs for a specific service
docker-compose logs -f gin-service

# Stop everything
docker-compose down

# Stop and wipe volumes
docker-compose down -v
```

---

## Option 2 — Run Each Service Locally (No Docker)

Use this when you want to run/debug a single service without spinning up everything.

> [!NOTE]
> Each service has its own `.env` file that points to `localhost` instead of Docker container names.

---

### Spring Boot (Port 8080)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\spring-boot-service"
mvn spring-boot:run
```

The `application.properties` reads `DB_URL`, `DB_USERNAME`, `DB_PASSWORD` from environment. Set them or let it use the defaults (`localhost:5432/banking` / `postgres` / `password`).

**Verify:** http://localhost:8080/accounts

---

### Gin (Port 8081)

The Gin service loads [gin-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/.env) automatically via `godotenv`.

```bash
cd "d:\InternProject - Likith\banking-simulator-main\gin-service"
go run main.go
```

**Verify:** http://localhost:8081/health

To run tests:
```bash
go test ./...
```

---

### FastAPI (Port 8082)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\fastapi-service"
pip install -r requirements.txt
uvicorn main:app --reload --port 8082
```

The [fastapi-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/fastapi-service/.env) is loaded by `python-dotenv` automatically.

**Verify:** http://localhost:8082/health  
**Swagger docs:** http://localhost:8082/docs

---

### Streamlit (Port 8501)

```bash
cd "d:\InternProject - Likith\banking-simulator-main\streamlit-service"
pip install -r requirements.txt
streamlit run app.py
```

The [streamlit-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/streamlit-service/.env) points all URLs to `localhost`.

**Verify:** http://localhost:8501

---

## .env Files — What Goes Where

| File | Used by | When |
|------|---------|------|
| [.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/.env) | `docker-compose.yml` | Docker mode only |
| [gin-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/.env) | `godotenv` in `main.go` | Local Go run only |
| [fastapi-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/fastapi-service/.env) | `python-dotenv` in `database.py` | Local Python run only |
| [streamlit-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/streamlit-service/.env) | `os.getenv` with defaults | Local Streamlit run only |

> [!WARNING]
> Never commit the `.env` files to Git — they contain DB credentials. The `.gitignore` should already exclude them.

---

## Startup Order (matters for local runs)

```
1. PostgreSQL DB           (must be running first)
2. Spring Boot  :8080      (core banking — everything depends on this)
3. Gin          :8081      (calls Spring Boot to settle transfers)
4. FastAPI      :8082      (calls Spring Boot for analytics data)
5. Streamlit    :8501      (calls all three above)
```

Docker Compose handles this automatically via `depends_on`.

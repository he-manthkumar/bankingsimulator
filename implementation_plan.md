# Implementation Plan — Resilient Transfer Settlements with Temporal

This plan introduces Temporal to the `gin-service` to manage NEFT/RTGS settlement delays and spring-boot API calls durably. This prevents transfers from getting stuck in a `PENDING`/`PROCESSING` state if the Gin service crashes during a settlement delay.

---

## Proposed Changes

### Infrastructure

#### [MODIFY] [docker-compose.yml](file:///d:/InternProject%20-%20Likith/banking-simulator-main/docker-compose.yml)
- Add a new lightweight developer Temporal server container using the official `temporalio/temporal:latest` image running `server start-dev`.
- Expose ports `7233` (gRPC SDK port) and `8233` (Web UI port).
- Inject `TEMPORAL_HOST=temporal-server:7233` environment variable into `transfer-processor` (`gin-service`).
- Ensure the `transfer-processor` depends on the `temporal-server` container starting up.

---

### Gin Service (`gin-service`)

#### [MODIFY] [go.mod](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/go.mod)
- Add dependency `go.temporal.io/sdk` to the Go module.

#### [NEW] [workflows.go](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/workflows/workflows.go)
- Create a new package `workflows` containing:
  - **`TransferWorkflow`**: Coordinates the durable `workflow.Sleep` depending on the transfer mode (NEFT = 30s, RTGS = 15s, IMPS = 0s) and executes activities.
  - **`SettleTransferActivity`**: Triggers the Spring Boot `/transactions/settle` HTTP post request.
  - **`UpdateLocalStatusActivity`**: Updates GORM local database record status to `SUCCESS` or `FAILED`.

#### [MODIFY] [main.go](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/main.go)
- Initialize the Temporal client on startup using `TEMPORAL_HOST` (defaulting to `localhost:7233` for local runs).
- Register the workflow and activities.
- Start a Temporal worker in a background goroutine to process transfer queues.
- Share the initialized Temporal client with the `handlers` package.

#### [MODIFY] [transfer.go](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/handlers/transfer.go)
- Declare a package-level `var TemporalClient client.Client`.
- In `ProcessTransferWithClient`, instead of launching `go simulateSettlement(...)`, trigger a new Temporal `TransferWorkflow` using `TemporalClient.ExecuteWorkflow()`.
- Delete the old `simulateSettlement` function entirely.

---

### Environment Configuration

#### [MODIFY] [.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/.env)
- Add `TEMPORAL_HOST=temporal-server:7233` to root `.env`.

#### [MODIFY] [gin-service/.env](file:///d:/InternProject%20-%20Likith/banking-simulator-main/gin-service/.env)
- Add `TEMPORAL_HOST=localhost:7233` to local `.env`.

---

## Verification Plan

### Manual Verification
1. **Verify Temporal UI**: Open `http://localhost:8233` and ensure the Temporal Web UI is serving.
2. **Execute NEFT/RTGS Transfer**: Initiate a transfer (e.g. NEFT/RTGS) from the Streamlit UI.
3. **Verify Temporal Execution**: Check the Temporal UI to see the workflow in the `Running` state executing `workflow.Sleep`.
4. **Crash Resiliency Test**:
   - Trigger a 30s NEFT transfer.
   - Kill/restart the `transfer-processor` docker container mid-transfer.
   - Verify that after the container recovers, the workflow resumes, successfully contacts Spring Boot, and settles the transfer.

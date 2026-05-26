import uuid
from datetime import datetime

from fastapi import FastAPI
from fastapi.testclient import TestClient
from sqlalchemy import create_engine
from sqlalchemy.orm import sessionmaker
from sqlalchemy.pool import StaticPool
from unittest.mock import patch

from db.database import get_db
from models.models import Base, FraudLog
from routers.fraud import router, assess_risk


engine = create_engine(
    "sqlite://",
    connect_args={"check_same_thread": False},
    poolclass=StaticPool,
)

TestingSessionLocal = sessionmaker(autocommit=False, autoflush=False, bind=engine)
Base.metadata.create_all(bind=engine)


def override_get_db():
    db = TestingSessionLocal()
    try:
        yield db
    finally:
        db.close()


app = FastAPI()
app.include_router(router)
app.dependency_overrides[get_db] = override_get_db
client = TestClient(app)


def reset_fraud_logs():
    db = TestingSessionLocal()
    try:
        db.query(FraudLog).delete()
        db.commit()
    finally:
        db.close()


def test_assessRisk_shouldReturnHighWhenAmountExceedsThreshold():
    risk_level, reason = assess_risk(60000.00)

    assert risk_level == "HIGH"
    assert reason == "Transaction exceeds ₹50,000 threshold"

def test_assessRisk_shouldReturnMediumWhenAmountIsFiftyThousand():
    risk_level, reason = assess_risk(50000.00)

    assert risk_level == "MEDIUM"
    assert reason == "Moderate transaction of ₹50,000.00"

def test_assessRisk_shouldReturnMediumWhenAmountExceedsTenThousand():
    risk_level, reason = assess_risk(20000.00)

    assert risk_level == "MEDIUM"
    assert reason == "Moderate transaction of ₹20,000.00"

def test_assessRisk_shouldReturnLowWhenAmountIsTenThousand():
    risk_level, reason = assess_risk(10000.00)

    assert risk_level == "LOW"
    assert reason == "Transaction within normal range"


def test_assessRisk_shouldReturnLowWhenAmountIsWithinNormalRange():
    risk_level, reason = assess_risk(5000.00)

    assert risk_level == "LOW"
    assert reason == "Transaction within normal range"


def test_checkFraud_shouldReturn_400_when_transaction_id_is_invalid():
    response = client.get("/fraud/not-a-uuid")

    assert response.status_code == 400
    assert response.json() == {"detail": "Invalid transaction ID"}


def test_checkFraud_shouldReturn404WhenTransactionIsNotFound():
    transaction_id = str(uuid.uuid4())

    with patch("routers.fraud.CoreBankingClient.get_transaction", return_value=None):
        response = client.get(f"/fraud/{transaction_id}")

    assert response.status_code == 404
    assert response.json() == {"detail": "Transaction not found"}


def test_checkFraud_shouldReturnFraudResultWhenTransactionIsLowRisk():
    reset_fraud_logs()
    transaction_id = str(uuid.uuid4())

    with patch(
        "routers.fraud.CoreBankingClient.get_transaction", return_value={"amount": 5000.00, "transferMode": "IMPS"},):
        response = client.get(f"/fraud/{transaction_id}")

    assert response.status_code == 200
    body = response.json()
    assert body["transaction_id"] == transaction_id
    assert body["amount"] == 5000.0
    assert body["transfer_mode"] == "IMPS"
    assert body["risk_level"] == "LOW"
    assert body["reason"] == "Transaction within normal range"
    assert "fraud_log_id" in body


def test_checkFraud_shouldReturnFraudResultWhenTransactionIsMediumRisk():
    reset_fraud_logs()
    transaction_id = str(uuid.uuid4())

    with patch(
        "routers.fraud.CoreBankingClient.get_transaction",
        return_value={"amount": 20000.00, "transferMode": "NEFT"},
    ):
        response = client.get(f"/fraud/{transaction_id}")

    assert response.status_code == 200
    body = response.json()
    assert body["transaction_id"] == transaction_id
    assert body["amount"] == 20000.0
    assert body["transfer_mode"] == "NEFT"
    assert body["risk_level"] == "MEDIUM"
    assert body["reason"] == "Moderate transaction of ₹20,000.00"
    assert "fraud_log_id" in body


def test_checkFraud_shouldReturnFraudResultWhenTransactionIsHighRisk():
    reset_fraud_logs()
    transaction_id = str(uuid.uuid4())

    with patch(
        "routers.fraud.CoreBankingClient.get_transaction",
        return_value={"amount": 100000.00, "transferMode": "RTGS"},
    ):
        response = client.get(f"/fraud/{transaction_id}")

    assert response.status_code == 200
    body = response.json()
    assert body["transaction_id"] == transaction_id
    assert body["amount"] == 100000.0
    assert body["transfer_mode"] == "RTGS"
    assert body["risk_level"] == "HIGH"
    assert body["reason"] == "Transaction exceeds ₹50,000 threshold"
    assert "fraud_log_id" in body


def test_checkFraud_shouldReturnFraudLogIdWhenLogIsPersisted():
    reset_fraud_logs()
    transaction_id = str(uuid.uuid4())

    with patch(
        "routers.fraud.CoreBankingClient.get_transaction",
        return_value={"amount": 75000.00, "transferMode": "IMPS"},
    ):
        response = client.get(f"/fraud/{transaction_id}")

    assert response.status_code == 200
    body = response.json()

    db = TestingSessionLocal()
    try:
        log = db.query(FraudLog).filter(FraudLog.id == uuid.UUID(body["fraud_log_id"])).first()
        assert log is not None
        assert str(log.transaction_id) == transaction_id
        assert log.risk_level == "HIGH"
        assert log.reason == "Transaction exceeds ₹50,000 threshold"
    finally:
        db.close()


def test_getAllFraudLogs_shouldReturnEmptyListWhenNoLogsExist():
    reset_fraud_logs()

    response = client.get("/fraud/logs/all")

    assert response.status_code == 200
    assert response.json() == []


def test_getAllFraudLogs_shouldReturnAllLogsWhenLogsExist():
    reset_fraud_logs()

    db = TestingSessionLocal()
    try:
        log1 = FraudLog(
            id=uuid.uuid4(),
            transaction_id=uuid.uuid4(),
            risk_level="LOW",
            reason="Transaction within normal range",
            created_at=datetime.utcnow(),
        )
        log2 = FraudLog(
            id=uuid.uuid4(),
            transaction_id=uuid.uuid4(),
            risk_level="HIGH",
            reason="Transaction exceeds ₹50,000 threshold",
            created_at=datetime.utcnow(),
        )
        db.add(log1)
        db.add(log2)
        db.commit()
    finally:
        db.close()

    response = client.get("/fraud/logs/all")

    assert response.status_code == 200
    body = response.json()
    assert len(body) == 2
    assert "id" in body[0]
    assert "transaction_id" in body[0]
    assert "risk_level" in body[0]
    assert "reason" in body[0]
    assert "created_at" in body[0]


def test_getAllFraudLogs_shouldReturnLogsInDescendingCreatedAtOrderWhenMultipleLogsExist():
    reset_fraud_logs()

    older_time = datetime(2024, 1, 1, 10, 0, 0)
    newer_time = datetime(2024, 1, 1, 12, 0, 0)

    db = TestingSessionLocal()
    try:
        older_log = FraudLog(
            id=uuid.uuid4(),
            transaction_id=uuid.uuid4(),
            risk_level="LOW",
            reason="Older log",
            created_at=older_time,
        )
        newer_log = FraudLog(
            id=uuid.uuid4(),
            transaction_id=uuid.uuid4(),
            risk_level="HIGH",
            reason="Newer log",
            created_at=newer_time,
        )
        db.add(older_log)
        db.add(newer_log)
        db.commit()
    finally:
        db.close()

    response = client.get("/fraud/logs/all")

    assert response.status_code == 200
    body = response.json()
    assert len(body) == 2
    assert body[0]["reason"] == "Newer log"
    assert body[1]["reason"] == "Older log"
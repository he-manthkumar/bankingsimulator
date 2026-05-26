from fastapi import FastAPI
from fastapi.testclient import TestClient
from unittest.mock import patch

from routers.analytics import router


app = FastAPI()
app.include_router(router)
client = TestClient(app)


def test_getHighValueTransfers_shouldReturnEmptyListWhenNoTransactionsExist():
    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=[]):
        response = client.get("/analytics/high-value")

    assert response.status_code == 200
    assert response.json() == {"threshold": 50000.0,"count": 0,"transactions": []}


def test_getHighValueTransfers_shouldReturnEmptyListWhenNoTransactionsExceedThreshold():
    transactions = [
        {"id": "1", "amount": 1000},
        {"id": "2", "amount": 49999.99}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/high-value")

    assert response.status_code == 200
    assert response.json() == {"threshold": 50000.0,"count": 0,"transactions": []}


def test_getHighValueTransfers_shouldReturnTransactionsWhenAmountExceedsThreshold():
    transactions = [
        {"id": "1", "amount": 1000},
        {"id": "2", "amount": 75000},
        {"id": "3", "amount": 120000}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/high-value")

    assert response.status_code == 200
    assert response.json() == {"threshold": 50000.0,"count": 2,"transactions": [{"id": "2", "amount": 75000},{"id": "3", "amount": 120000}]}


def test_getHighValueTransfers_shouldReturnOnlyTransactionsGreaterThanThresholdWhenAmountEqualsThreshold():
    transactions = [
        {"id": "1", "amount": 50000.00},
        {"id": "2", "amount": 50000.01}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/high-value")

    assert response.status_code == 200
    assert response.json() == {"threshold": 50000.0,"count": 1,"transactions": [{"id": "2", "amount": 50000.01}]}


def test_getTransactionSummary_shouldReturnEmptySummaryWhenNoTransactionsExist():
    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=[]):
        response = client.get("/analytics/summary")

    assert response.status_code == 200
    assert response.json() == {"total_transactions": 0,"total_volume": 0,"by_transfer_mode": [],"by_status": []}


def test_getTransactionSummary_shouldReturnSummaryWhenTransactionsExist():
    transactions = [
        {"id": "1", "amount": 1000, "transferMode": "IMPS", "status": "SUCCESS"},
        {"id": "2", "amount": 2000, "transferMode": "NEFT", "status": "PENDING"},
        {"id": "3", "amount": 3000, "transferMode": "IMPS", "status": "SUCCESS"}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/summary")

    assert response.status_code == 200
    assert response.json() == {"total_transactions": 3,"total_volume": 6000.0,"by_transfer_mode": [{"mode": "IMPS", "count": 2},{"mode": "NEFT", "count": 1}],
        "by_status": [{"status": "SUCCESS", "count": 2},{"status": "PENDING", "count": 1}]}


def test_getTransactionSummary_shouldReturnUnknownBucketsWhenFieldsAreMissing():
    transactions = [
        {"id": "1", "amount": 1000},
        {"id": "2", "amount": 2000, "transferMode": "RTGS"},
        {"id": "3", "amount": 3000, "status": "FAILED"}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/summary")

    assert response.status_code == 200
    assert response.json() == {
        "total_transactions": 3,
        "total_volume": 6000.0,
        "by_transfer_mode": [
            {"mode": "UNKNOWN", "count": 2},
            {"mode": "RTGS", "count": 1}
        ],
        "by_status": [
            {"status": "UNKNOWN", "count": 2},
            {"status": "FAILED", "count": 1}
        ]
    }


def test_getTransactionSummary_shouldReturnCorrectVolumeWhenAmountsAreStringsandNumbers():
    transactions = [
        {"id": "1", "amount": "1000", "transferMode": "IMPS", "status": "SUCCESS"},
        {"id": "2", "amount": 2500.5, "transferMode": "NEFT", "status": "FAILED"}
    ]

    with patch("routers.analytics.CoreBankingClient.get_all_transactions", return_value=transactions):
        response = client.get("/analytics/summary")

    assert response.status_code == 200
    assert response.json() == {
        "total_transactions": 2,
        "total_volume": 3500.5,
        "by_transfer_mode": [
            {"mode": "IMPS", "count": 1},
            {"mode": "NEFT", "count": 1}
        ],
        "by_status": [
            {"status": "SUCCESS", "count": 1},
            {"status": "FAILED", "count": 1}
        ]
    }
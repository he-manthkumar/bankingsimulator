import pytest
import requests
from fastapi import HTTPException
from unittest.mock import Mock, patch

from clients.core_banking_client import CoreBankingClient


def test_getAccount_shouldReturnAccountWhenResponseIs200():
    account_id = "123"

    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"id": account_id, "name": "Likhith"}

    with patch("clients.core_banking_client.requests.get", return_value=mock_response) as mock_get:
        result = CoreBankingClient.get_account(account_id)

    assert result == {"id": account_id, "name": "Likhith"}
    mock_get.assert_called_once()


def test_getAccount_shouldReturn404WhenAccountNotFound():
    account_id = "123"

    mock_response = Mock()
    mock_response.status_code = 404

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_account(account_id)

    assert exc.value.status_code == 404
    assert exc.value.detail == f"Account {account_id} not found"


def test_getAccount_shouldReturn502WhenCoreBankingServiceReturnsNon200():
    account_id = "123"

    mock_response = Mock()
    mock_response.status_code = 500

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_account(account_id)

    assert exc.value.status_code == 502
    assert exc.value.detail == "Core banking service error"


def test_getAccount_shouldReturn503WhenConnectionErrorOccurs():
    account_id = "123"

    with patch("clients.core_banking_client.requests.get", side_effect=requests.exceptions.ConnectionError):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_account(account_id)

    assert exc.value.status_code == 503
    assert exc.value.detail == "Core banking service unavailable"


def test_getAllAccounts_shouldReturnAccountsWhenResponseIs200():
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = [
        {"id": "1", "name": "Likhith"},
        {"id": "2", "name": "Arigela"},
    ]

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        result = CoreBankingClient.get_all_accounts()

    assert result == [
        {"id": "1", "name": "Likhith"},
        {"id": "2", "name": "Arigela"},
    ]


def test_getAllAccounts_shouldReturn502WhenCoreBankingServiceReturnsNon200():
    mock_response = Mock()
    mock_response.status_code = 500

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_all_accounts()

    assert exc.value.status_code == 502
    assert exc.value.detail == "Core banking service error"


def test_getAllAccounts_shouldReturn503WhenConnectionErrorOccurs():
    with patch("clients.core_banking_client.requests.get", side_effect=requests.exceptions.ConnectionError):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_all_accounts()

    assert exc.value.status_code == 503
    assert exc.value.detail == "Core banking service unavailable"


def test_getTransaction_shouldReturnTransactionWhenResponseIs200():
    transaction_id = "txn-123"

    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {"id": transaction_id, "status": "SUCCESS"}

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        result = CoreBankingClient.get_transaction(transaction_id)

    assert result == {"id": transaction_id, "status": "SUCCESS"}


def test_getTransaction_shouldReturnNoneWhenTransactionNotFound():
    transaction_id = "txn-123"

    mock_response = Mock()
    mock_response.status_code = 404

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        result = CoreBankingClient.get_transaction(transaction_id)

    assert result is None


def test_getTransaction_shouldReturn502WhenCoreBankingServiceReturnsNon200Non404():
    transaction_id = "txn-123"

    mock_response = Mock()
    mock_response.status_code = 500

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_transaction(transaction_id)

    assert exc.value.status_code == 502
    assert exc.value.detail == "Core banking service error"


def test_getTransaction_shouldReturn503WhenConnectionErrorOccurs():
    transaction_id = "txn-123"

    with patch("clients.core_banking_client.requests.get", side_effect=requests.exceptions.ConnectionError):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_transaction(transaction_id)

    assert exc.value.status_code == 503
    assert exc.value.detail == "Core banking service unavailable"


def test_getAllTransactions_shouldReturnTransactionsWhenResponseIs200():
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = [
        {"id": "txn-1", "status": "SUCCESS"},
        {"id": "txn-2", "status": "FAILED"},
    ]

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        result = CoreBankingClient.get_all_transactions()

    assert result == [
        {"id": "txn-1", "status": "SUCCESS"},
        {"id": "txn-2", "status": "FAILED"},
    ]


def test_getAllTransactions_shouldReturnEmptyListWhenCoreBankingServiceReturnsNon200():
    mock_response = Mock()
    mock_response.status_code = 500

    with patch("clients.core_banking_client.requests.get", return_value=mock_response):
        result = CoreBankingClient.get_all_transactions()

    assert result == []


def test_getAllTransactions_shouldReturn503WhenConnectionErrorOccurs():
    with patch("clients.core_banking_client.requests.get", side_effect=requests.exceptions.ConnectionError):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.get_all_transactions()

    assert exc.value.status_code == 503
    assert exc.value.detail == "Core banking service unavailable"


def test_settleTransfer_shouldReturnResponseWhenSettlementSucceeds():
    mock_response = Mock()
    mock_response.status_code = 200
    mock_response.json.return_value = {
        "id": "txn-123",
        "fromAccount": "acc-1",
        "toAccount": "acc-2",
        "amount": 100.0,
        "transferMode": "IMPS",
        "status": "SUCCESS",
    }

    with patch("clients.core_banking_client.requests.post", return_value=mock_response):
        result = CoreBankingClient.settle_transfer("acc-1", "acc-2", 100.0, "IMPS")

    assert result["status"] == "SUCCESS"
    assert result["amount"] == 100.0


def test_settleTransfer_shouldReturn502WhenSettlementFailsInCoreBanking():
    mock_response = Mock()
    mock_response.status_code = 500

    with patch("clients.core_banking_client.requests.post", return_value=mock_response):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.settle_transfer("acc-1", "acc-2", 100.0, "IMPS")

    assert exc.value.status_code == 502
    assert exc.value.detail == "Settlement failed in core banking"


def test_settleTransfer_shouldReturn503WhenConnectionErrorOccurs():
    with patch("clients.core_banking_client.requests.post", side_effect=requests.exceptions.ConnectionError):
        with pytest.raises(HTTPException) as exc:
            CoreBankingClient.settle_transfer("acc-1", "acc-2", 100.0, "IMPS")

    assert exc.value.status_code == 503
    assert exc.value.detail == "Core banking service unavailable"   
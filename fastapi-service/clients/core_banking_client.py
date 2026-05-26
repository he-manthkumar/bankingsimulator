import os
import requests
from typing import Optional
from fastapi import HTTPException

SPRING_BOOT_URL = os.getenv("SPRING_BOOT_URL")
TIMEOUT = 5


class CoreBankingClient:
    @staticmethod
    def get_account(account_id: str) -> dict:
        try:
            response = requests.get(f"{SPRING_BOOT_URL}/accounts/{account_id}",timeout=TIMEOUT)
        except requests.exceptions.ConnectionError:
            raise HTTPException(status_code=503, detail="Core banking service unavailable")

        if response.status_code == 404:
            raise HTTPException(status_code=404, detail=f"Account {account_id} not found")
        if response.status_code != 200:
            raise HTTPException(status_code=502, detail="Core banking service error")

        return response.json()

    @staticmethod
    def get_all_accounts() -> list:
        try:
            response = requests.get(f"{SPRING_BOOT_URL}/accounts",timeout=TIMEOUT)
        except requests.exceptions.ConnectionError:
            raise HTTPException(status_code=503, detail="Core banking service unavailable")

        if response.status_code != 200:
            raise HTTPException(status_code=502, detail="Core banking service error")

        return response.json()

    @staticmethod
    def get_transaction(transaction_id: str) -> Optional[dict]:
        try:
            response = requests.get(f"{SPRING_BOOT_URL}/transactions/{transaction_id}",timeout=TIMEOUT)
        except requests.exceptions.ConnectionError:
            raise HTTPException(status_code=503, detail="Core banking service unavailable")

        if response.status_code == 404:
            return None
        if response.status_code != 200:
            raise HTTPException(status_code=502, detail="Core banking service error")

        return response.json()

    @staticmethod
    def get_all_transactions() -> list:
        try:
            response = requests.get(f"{SPRING_BOOT_URL}/transactions/all",timeout=TIMEOUT)
        except requests.exceptions.ConnectionError:
            raise HTTPException(status_code=503, detail="Core banking service unavailable")

        if response.status_code != 200:
            return []

        return response.json()

    @staticmethod
    def settle_transfer(from_account: str, to_account: str, amount: float, transfer_mode: str) -> dict:
        payload = {
            "fromAccount": from_account,
            "toAccount": to_account,
            "amount": amount,
            "transferMode": transfer_mode,
            "status": "SUCCESS"
        }
        try:
            response = requests.post(f"{SPRING_BOOT_URL}/transactions/settle",json=payload,timeout=TIMEOUT)
        except requests.exceptions.ConnectionError:
            raise HTTPException(status_code=503, detail="Core banking service unavailable")

        if response.status_code != 200:
            raise HTTPException(status_code=502, detail="Settlement failed in core banking")

        return response.json()

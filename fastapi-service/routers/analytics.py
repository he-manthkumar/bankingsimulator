from fastapi import APIRouter
from clients.core_banking_client import CoreBankingClient

router = APIRouter(prefix="/analytics", tags=["Analytics"])

HIGH_VALUE_THRESHOLD = 50000.00

@router.get("/high-value")
def get_highValueTransfers():
    transactions = CoreBankingClient.get_all_transactions()
    high_value = [t for t in transactions if float(t.get("amount", 0)) > HIGH_VALUE_THRESHOLD ]
    return {"threshold": HIGH_VALUE_THRESHOLD,"count": len(high_value), "transactions": high_value}


@router.get("/summary")
def getTransactionSummary():    
    transactions = CoreBankingClient.get_all_transactions()

    total_count = len(transactions)
    total_volume = sum(float(t.get("amount", 0)) for t in transactions)

    by_mode = {}
    by_status = {}
    for t in transactions:
        mode = t.get("transferMode", "UNKNOWN")
        status = t.get("status", "UNKNOWN")
        by_mode[mode] = by_mode.get(mode, 0) + 1
        by_status[status] = by_status.get(status, 0) + 1

    return {
        "total_transactions": total_count,
        "total_volume": total_volume,
        "by_transfer_mode": [{"mode": k, "count": v} for k, v in by_mode.items()],
        "by_status": [{"status": k, "count": v} for k, v in by_status.items()]
    }

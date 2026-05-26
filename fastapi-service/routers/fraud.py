import uuid
from fastapi import APIRouter, Depends, HTTPException
from sqlalchemy.orm import Session

from db.database import get_db
from models.models import FraudLog
from clients.core_banking_client import CoreBankingClient

router = APIRouter(prefix="/fraud", tags=["Fraud Detection"])

HIGH_VALUE_THRESHOLD = 50000.00


@router.get("/{transaction_id}")
def check_fraud(transaction_id: str, db: Session = Depends(get_db)):
    try:
        uuid.UUID(transaction_id)
    except ValueError:
        raise HTTPException(status_code=400, detail="Invalid transaction ID")

    txn = CoreBankingClient.get_transaction(transaction_id)
    if txn is None:
        raise HTTPException(status_code=404, detail="Transaction not found")

    amount = float(txn["amount"])
    transfer_mode = txn["transferMode"]

    risk_level, reason = assess_risk(amount)

    log = FraudLog(id=uuid.uuid4(),transaction_id=uuid.UUID(transaction_id),risk_level=risk_level,reason=reason)
    db.add(log)
    db.commit()
    db.refresh(log)

    return {
        "transaction_id": transaction_id,
        "amount": amount,
        "transfer_mode": transfer_mode,
        "risk_level": risk_level,
        "reason": reason,
        "fraud_log_id": str(log.id)
    }


@router.get("/logs/all")
def get_all_fraud_logs(db: Session = Depends(get_db)):
    logs = db.query(FraudLog).order_by(FraudLog.created_at.desc()).all()
    return [
        {
            "id": str(l.id),
            "transaction_id": str(l.transaction_id),
            "risk_level": l.risk_level,
            "reason": l.reason,
            "created_at": l.created_at.isoformat() if l.created_at else None
        }
        for l in logs
    ]


def assess_risk(amount: float):
    if amount > HIGH_VALUE_THRESHOLD:
        return "HIGH", f"Transaction exceeds ₹{HIGH_VALUE_THRESHOLD:,.0f} threshold"
    elif amount > 10000:
        return "MEDIUM", f"Moderate transaction of ₹{amount:,.2f}"
    else:
        return "LOW", "Transaction within normal range"

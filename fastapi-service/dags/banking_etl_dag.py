import os
import logging
from datetime import datetime, timedelta

import psycopg2
from pymongo import MongoClient

from airflow import DAG
from airflow.operators.python import PythonOperator

PG_HOST     = os.environ.get("POSTGRES_HOST", "postgres")
PG_USER     = os.environ.get("POSTGRES_USER", "postgres")
PG_PASSWORD = os.environ.get("POSTGRES_PASSWORD", "password")
PG_DB       = os.environ.get("POSTGRES_DB", "banking")

MONGO_URI = os.environ.get("MONGO_URI", "mongodb://banking-mongo:27017")
MONGO_DB  = os.environ.get("MONGO_DB",  "banking")


def pg_conn():
    return psycopg2.connect(
        host=PG_HOST, user=PG_USER,
        password=PG_PASSWORD, database=PG_DB,
    )


def ensure_schema(**_ctx):
    conn = pg_conn()
    cur  = conn.cursor()
    cur.execute("""
        CREATE TABLE IF NOT EXISTS transfer_pipeline (
            mongo_transfer_id    TEXT        PRIMARY KEY,
            from_account_id      TEXT,
            to_account_id        TEXT,
            amount               NUMERIC(15,2),
            transfer_mode        TEXT,
            mongo_status         TEXT,
            pg_status            TEXT,
            is_in_flight         BOOLEAN     GENERATED ALWAYS AS (pg_status IS NULL) STORED,
            initiated_at         TIMESTAMP,
            settled_at           TIMESTAMP,
            settlement_lag_secs  INTEGER,
            etl_synced_at        TIMESTAMP   DEFAULT NOW()
        );
    """)
    conn.commit()
    cur.close()
    conn.close()
    logging.info("transfer_pipeline table ready.")

def sync_pipeline(**_ctx):
    client    = MongoClient(MONGO_URI, serverSelectionTimeoutMS=8_000)
    transfers = list(client[MONGO_DB]["transfers"].find({}))
    client.close()
    logging.info("MongoDB: %d transfers", len(transfers))

    conn = pg_conn()
    cur  = conn.cursor()
    cur.execute("""
        SELECT id, from_account, to_account, amount, transfer_mode, status, created_at, correlation_id
        FROM transactions;
    """)
    pg_rows = cur.fetchall()
    logging.info("Postgres: %d settled transactions", len(pg_rows))

    pg_transactions_by_correlation_id = {
        str(row[7]): {
            "id":     str(row[0]),
            "status": row[5],
            "ts":     row[6],
        }
        for row in pg_rows
        if row[7] is not None
    }

    upserted = 0
    for t in transfers:
        mongo_id   = str(t.get("_id", ""))
        correlation_id = str(t.get("correlation_id", ""))
        from_acc   = str(t.get("from_account", ""))
        to_acc     = str(t.get("to_account",   ""))
        amount     = float(t.get("amount",      0))
        mode       = t.get("transfer_mode", "")
        m_status   = t.get("status", "")
        initiated  = t.get("created_at")

        matched = pg_transactions_by_correlation_id.get(correlation_id)

        pg_id      = matched["id"]     if matched else None
        pg_status  = matched["status"] if matched else None
        settled_at = matched["ts"]     if matched else None
        lag        = (
            int((settled_at - initiated).total_seconds())
            if (settled_at and initiated) else None
        )

        cur.execute("""
            INSERT INTO transfer_pipeline
                (mongo_transfer_id, from_account_id, to_account_id,
                 amount, transfer_mode,
                 mongo_status, pg_status,
                 initiated_at, settled_at, settlement_lag_secs,
                 etl_synced_at)
            VALUES (%s,%s,%s,%s,%s,%s,%s,%s,%s,%s, NOW())
            ON CONFLICT (mongo_transfer_id) DO UPDATE SET
                mongo_status        = EXCLUDED.mongo_status,
                pg_status           = EXCLUDED.pg_status,
                settled_at          = EXCLUDED.settled_at,
                settlement_lag_secs = EXCLUDED.settlement_lag_secs,
                etl_synced_at       = NOW()
        """, (
            mongo_id, from_acc, to_acc,
            amount, mode,
            m_status, pg_status,
            initiated, settled_at, lag,
        ))
        upserted += 1

    conn.commit()
    cur.close()
    conn.close()
    logging.info("Upserted %d rows into transfer_pipeline.", upserted)

with DAG(
    dag_id            = "banking_etl_dag",
    description       = "Every 5 min: sync MongoDB transfers + Postgres transactions → transfer_pipeline",
    schedule_interval = "*/5 * * * *",
    start_date        = datetime(2024, 1, 1),
    catchup           = False,
    default_args      = {
        "owner":       "banking-etl",
        "retries":     1,
        "retry_delay": timedelta(minutes=1),
    },
    tags = ["etl", "banking", "mongodb", "postgres"],
) as dag:

    t1 = PythonOperator(task_id="ensure_schema", python_callable=ensure_schema)
    t2 = PythonOperator(task_id="sync_pipeline", python_callable=sync_pipeline)

    t1 >> t2

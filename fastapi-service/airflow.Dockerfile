FROM apache/airflow:2.9.3-python3.11
USER root
RUN apt-get update && apt-get install -y --no-install-recommends postgresql-client && rm -rf /var/lib/apt/lists/*
USER airflow
RUN pip install --no-cache-dir psycopg2-binary==2.9.9 pymongo==4.6.1

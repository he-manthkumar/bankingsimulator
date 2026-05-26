from fastapi import FastAPI
from db.database import Base, engine
from routers import fraud, analytics

Base.metadata.create_all(bind=engine)

app = FastAPI(
    title="Fraud & Analytics Service",
    description="Banking fraud detection and analytics powered by FastAPI",
    version="1.0.0"
)

app.include_router(fraud.router)
app.include_router(analytics.router)


@app.get("/health")
def health():
    return {"status": "fastapi-service running"}

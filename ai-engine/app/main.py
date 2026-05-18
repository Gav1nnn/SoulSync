from fastapi import FastAPI


app = FastAPI(title="SoulSync AI Engine")


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"service": "ai-engine", "status": "ok"}

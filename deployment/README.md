# Deployment

**Docker Compose for local databases** lives at the **repository root**: [`../docker-compose.yml`](../docker-compose.yml).

From the repo root:

```powershell
docker compose up -d
```

The previous `docker-compose.yml` in this folder (Postgres only, mismatched DB name) was removed in favor of the unified file aligned with `backend/cmd/api/main.go` defaults.

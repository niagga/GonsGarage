# GonsGarage Operations Runbook

This runbook documents day-to-day operations for isolated production and staging stacks on `192.168.1.100`.

## Topology

- **Production project name:** `gonsgarage-prod`
- **Staging project name:** `gonsgarage-stg`
- **Production URL:** `http://192.168.1.100:8102`
- **Staging URL:** `http://192.168.1.100:18102`

Health endpoints:

- `GET /health`
- `GET /ready`

## Deploy flow (recommended)

### 1) Deploy staging first

```bash
cd /DATA/AppData/gonsgarage
./scripts/update-server-gonsgarage-stg.sh
```

Validate:

```bash
curl -s -o /dev/null -w "stg_health:%{http_code}\n" http://127.0.0.1:18102/health
curl -s -o /dev/null -w "stg_ready:%{http_code}\n"  http://127.0.0.1:18102/ready
```

### 2) Deploy production after staging passes

```bash
cd /DATA/AppData/gonsgarage
./scripts/update-server-gonsgarage.sh
```

Validate:

```bash
curl -s -o /dev/null -w "prod_health:%{http_code}\n" http://127.0.0.1:8102/health
curl -s -o /dev/null -w "prod_ready:%{http_code}\n"  http://127.0.0.1:8102/ready
```

## Safety invariants

- Production deploy is **isolated**. Do not attach to Arnela network/database.
- `COMPOSE_OVERRIDE` is blocked by production deploy script.
- `.env.prod` must not reference `arnela-postgres`.
- Always deploy through scripts (they pin compose project names).

## Operations

### Container status

```bash
docker ps --format '{{.Names}}|{{.Status}}|{{.Ports}}' | grep '^gonsgarage-'
```

### Logs

```bash
docker logs --tail 200 gonsgarage-api
docker logs --tail 200 gonsgarage-stg-api
```

### Quick restart (single service)

```bash
docker restart gonsgarage-api
docker restart gonsgarage-stg-api
```

## Database checks

### Production DB connectivity through app endpoint

```bash
curl -sS http://127.0.0.1:8102/ready
```

### Verify legacy Arnela coupling is not present

```bash
grep '^DATABASE_URL=' /DATA/AppData/gonsgarage/.env.prod
# expected host: postgres (service), never arnela-postgres
```

## Rollback (fast path)

1. Use latest backup from `/DATA/AppData/gonsgarage/backups/`.
2. Restore previous compose/env files if needed.
3. Re-run deploy script:

```bash
cd /DATA/AppData/gonsgarage
./scripts/update-server-gonsgarage.sh
```

If database rollback is needed, restore latest dump into `gonsgarage-postgres`.

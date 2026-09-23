# Despliegue LAN / Docker (plantilla Arnela)

## Camino recomendado: en el servidor (git pull + compose)

**No copies el árbol desde tu PC** para un deploy normal. En el servidor bajás lo que ya está en Git, reconstruís contenedores y verificás health/ready.

Script canónico: [`scripts/deploy-prod.sh`](../scripts/deploy-prod.sh).

### Deploy

```bash
cd /DATA/AppData/gonsgarage
export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml   # si DATABASE_URL usa arnela-postgres
bash scripts/deploy-prod.sh deploy
```

Qué hace:

1. Guarda el SHA actual (pre-deploy).
2. `git fetch` + `checkout` de `main` (override con `GIT_REF`) + `pull --ff-only`.
3. `docker compose -f docker-compose.prod.yml [-f override] --env-file .env.prod up -d --build`
4. Reintenta `GET /health` y `GET /ready` en `http://127.0.0.1:8102` hasta OK (API + Postgres vía `/ready`).
5. Si ambos dan **200**, escribe `.deploy-last-good` con ese SHA (rollback seguro).

Equivalente a mano (sin script):

```bash
cd /DATA/AppData/gonsgarage
git fetch --all --prune && git checkout main && git pull --ff-only origin main
docker compose -f docker-compose.prod.yml -f docker-compose.prod.arnela-network.yml --env-file .env.prod up -d --build
curl -sS http://127.0.0.1:8102/health
curl -sS http://127.0.0.1:8102/ready
```

### Rollback (último commit estable)

Tras un deploy **exitoso**, el script marca el SHA en `.deploy-last-good` (archivo local del servidor; no va a git).

```bash
cd /DATA/AppData/gonsgarage
export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml
bash scripts/deploy-prod.sh rollback
```

Hace `checkout --detach` de ese SHA, `up -d --build` y vuelve a exigir health/ready 200. Si el deploy nuevo falla health, **no** actualiza last-good: el rollback vuelve al último verde.

### Status

```bash
bash scripts/deploy-prod.sh status
```

### Compatibilidad

[`scripts/update-server-gonsgarage.sh`](../scripts/update-server-gonsgarage.sh) delega en `deploy-prod.sh deploy`.

### Por qué este camino (y no SCP desde el PC)

- Evita subir `node_modules` / `.next` por la red (minutos de SCP innecesarios).
- El artefacto desplegado es exactamente un commit de Git (auditable).
- Rollback = re-build del SHA ya marcado estable, no “adivinar” `HEAD~1`.

**Requisito:** el commit que querés en prod tiene que estar en el remoto (`git push` desde tu máquina de desarrollo antes del deploy en el servidor).

---

## Alternativa desde tu PC (`deploy.ps1` / SCP)

Solo si necesitás empujar el working tree local sin pasar por Git (emergencia). Es más lento y no deja un SHA limpio.

1. **`git pull`** en el clon local.
2. **`.\deploy.ps1`** en PowerShell desde la raíz del repo (copia `backend/`, `frontend/`, `nginx/`, compose; no copia `.git`).

Si `DATABASE_URL` usa `arnela-postgres`, en `deploy.ps1` dejá `$COMPOSE_OVERRIDE = "docker-compose.prod.arnela-network.yml"`.

---

## Script del servidor y Arnela

Si **`DATABASE_URL`** usa el hostname **`arnela-postgres`**, el API tiene que estar en la **misma red Docker** que ese contenedor (ver [Opción B](#opción-b-recomendada-misma-red-docker-que-arnela)). **Antes** de ejecutar el script:

```bash
export COMPOSE_OVERRIDE=docker-compose.prod.arnela-network.yml
```

Sin el segundo `-f` (override vacío), el API suele entrar en **bucle de reinicio** y nginx devuelve **502** en `/health` (*no such host* al resolver `arnela-postgres`).

En **PowerShell** desde tu PC, [`deploy.ps1`](../deploy.ps1) admite la variable **`$COMPOSE_OVERRIDE`** (segundo `-f` en el `docker compose` remoto).

## Paridad Arnela (checklist)

1. **Red Docker** — [Opción B (misma red que Arnela)](#opción-b-recomendada-misma-red-docker-que-arnela); en el servidor, `export COMPOSE_OVERRIDE=…` antes del script bash, o `$COMPOSE_OVERRIDE` en `deploy.ps1`.
2. **`DATABASE_URL`** — Host resoluble desde el contenedor del API; [Postgres compartido con Arnela](#postgres-compartido-con-arnela-mismo-homeos). Si ves *lookup arnela-postgres*, revisá también [Incidente DNS en mvp-next-steps](../docs/mvp-next-steps.md#incidente-dns-arnela-postgres).
3. **`CORS_ORIGINS` / `GIN_MODE` / `JWT_SECRET`** — [CORS, `GIN_MODE=release` y `JWT_SECRET`](#cors-gin_moderelease-y-jwt_secret-checklist-51-52).
4. **`/health` y `/ready`** — [Verificación tras el primer `up`](#verificación-tras-el-primer-up-43--44).
5. **Backup** — [Backup de BD](#backup-de-bd-checklist-53).

Backlog documentado (archivado): [openspec/changes/archive/2026-04-21-arnela-parity/](../openspec/changes/archive/2026-04-21-arnela-parity/).

---

Archivos en la **raíz del repo**:

| Archivo | Uso |
|---------|-----|
| [`scripts/deploy-prod.sh`](../scripts/deploy-prod.sh) | **Canónico en servidor:** `deploy` / `rollback` / `status`. |
| [`scripts/update-server-gonsgarage.sh`](../scripts/update-server-gonsgarage.sh) | Alias → `deploy-prod.sh deploy`. |
| [`docker-compose.prod.yml`](../docker-compose.prod.yml) | Redis + API + Next (standalone) + nginx en **8102**. |
| [`.env.prod.example`](../.env.prod.example) | Plantilla; copiar a **`.env.prod`** en el servidor (no git). |
| [`deploy.ps1`](../deploy.ps1) | Alternativa SCP desde PC (no preferida). |
| [`docker-setup.prod.ps1`](../docker-setup.prod.ps1) | Mismo compose en local para probar antes de subir. |
| [`nginx/default.conf`](../nginx/default.conf) | `/` → front, `/api/` y `/swagger/` → API. |
| [`backend/Dockerfile`](../backend/Dockerfile) | Binario `gonsgarage-api`. |
| [`frontend/Dockerfile`](../frontend/Dockerfile) | Activa `DOCKER_BUILD=1` para `output: "standalone"` (solo en build Linux/Docker; `pnpm build` local en Windows sigue sin standalone). |
| [`docker-compose.prod.arnela-network.yml`](../docker-compose.prod.arnela-network.yml) | **Opción B (recomendada con Arnela):** une el API a la red Docker de Arnela y usá `arnela-postgres` como host en `DATABASE_URL`. |


## Fiscal integration (dark launch)

Fiscal schema, worker, and Cloudware mutations stay **off by default**. Operational runbook:

- [`docs/fiscal-integration-runbook.md`](../docs/fiscal-integration-runbook.md) — rollout stages, switches, monitoring, credential rotation, backup/restore, rollback
- [`docs/fiscal-integration-traceability.md`](../docs/fiscal-integration-traceability.md) — scenario → spec links

Schema apply (one-shot, does not enable finalization):

```bash
docker compose -f docker-compose.prod.yml --profile fiscal --env-file .env.prod run --rm gonsgarage-migrate
```

Do **not** set `FISCAL_FINALIZATION_ENABLED=true` or `FISCAL_WORKER_ENABLED=true` until parent tasks 13.2–13.4 record approved policy, storage, credentials, and gate evidence. Production rejects mock provider and local artifact backends.

Core `/ready` remains PostgreSQL-only; fiscal provider health is exposed at `/fiscal/dependency-status` and must not take the API out of rotation during a provider outage.

## Postgres compartido con Arnela (mismo `homeos`)


Si `docker ps` muestra `arnela-postgres` con **`5432/tcp` sin** `0.0.0.0:5432->…`, el Postgres **no está publicado en el host**: por eso fallan `host.docker.internal`, `172.17.0.1` y la IP LAN.

### Opción **B** (recomendada): misma red Docker que Arnela

- No abrís el puerto 5432 a la LAN.
- El API resuelve el hostname **`arnela-postgres`** (nombre del contenedor de Arnela).

1. Obtené el nombre **real** de la red Docker de Arnela:

   ```bash
   docker inspect arnela-postgres -f '{{range $k,$v := .NetworkSettings.Networks}}{{$k}}{{"\n"}}{{end}}'
   ```

   Si no es `arnela_arnela-network`, editá `name:` en [`docker-compose.prod.arnela-network.yml`](../docker-compose.prod.arnela-network.yml) para que coincida.

2. Dentro de ese Postgres, creá la base **`gonsgarage`** y un usuario/contraseña (o reutilizá uno que ya exista y tenga permiso sobre esa base).

3. En **`.env.prod`**:

   ```env
   DATABASE_URL=postgres://USUARIO:CONTRASEÑA@arnela-postgres:5432/gonsgarage?sslmode=disable
   ```

4. Levantá con **dos** ficheros compose:

   ```bash
   cd /DATA/AppData/gonsgarage
   docker compose -f docker-compose.prod.yml -f docker-compose.prod.arnela-network.yml --env-file .env.prod up -d --build
   ```

### Opción **A** (rápida): publicar 5432 en el compose de Arnela

En el `docker-compose` de Arnela, en el servicio `postgres`, añadí algo como `ports: - "5432:5432"` (o solo `127.0.0.1:5432:5432` si solo querés acceso desde el host). Reiniciá Arnela. Entonces `host.docker.internal` o `192.168.1.100` puede funcionar **si** Postgres escucha y `pg_hba` permite el origen. Suele ser menos limpio que la opción B (puerto expuesto, más superficie de ataque si no filtrás firewall).

### Postgres “solo en el host” (sin Arnela)

Creá rol + base en Postgres nativo, `listen_addresses` / `pg_hba` para Docker o LAN, y `DATABASE_URL` con `host.docker.internal` o la IP del servidor (ver en este mismo archivo la sección **connection refused** / `host.docker.internal`).

## CORS, `GIN_MODE=release` y `JWT_SECRET` (checklist 5.1–5.2)

En **`.env.prod`** del servidor (ver [`.env.prod.example`](../.env.prod.example)):

- **`GIN_MODE=release`** — simula producción; el Gin del API usa CORS restringido (no conviene dejar `debug` en el servidor de pruebas si ya validás el flujo real).
- **`CORS_ORIGINS`** — coma-separado; cada valor debe coincidir con la **URL exacta** que ve el navegador (p. ej. `http://192.168.1.100:8102`, **sin** barra final). Con `release`, no uses `*` salvo que aceptes abrir el API a cualquier origen.
- **`JWT_SECRET`** — obligatorio, largo y aleatorio (p. ej. `openssl rand -base64 32`); nunca el placeholder del example en un servidor accesible.

El API lee estas variables en arranque; detalle de middleware en `backend/cmd/api/main.go`.

## Acceso por hostname (p. ej. DuckDNS) desde la **misma LAN** que el servidor

Si con **`http://192.168.1.100:8102`** todo responde al instante pero con **`http://gonsgarage.duckdns.org:8102`** las peticiones del navegador quedan mucho tiempo en **Pending** y luego acaban:

1. **Ruta de red distinta** — El nombre suele resolver al **IP pública del router** (WAN). Un PC en la WiFi/LAN puede mandar el tráfico “fuera” y volver (**NAT hairpin** / loopback NAT). Muchos routers domésticos lo hacen **mal, lento o con timeouts**; no es un fallo de nginx ni del código de GonsGarage.

2. **IPv6 (AAAA)** — Si DuckDNS publica **AAAA** y la ruta IPv6 hasta tu casa no funciona bien, el navegador puede **esperar** a que falle IPv6 antes de usar IPv4 (**Happy Eyeballs**), lo que se ve como muchos `fetch` en Pending unos **20–75 s**.

**Qué hacer:** desde la LAN, o bien usá la **IP local** (`192.168…`), o bien forzá que el hostname resuelva a la **IP LAN** en ese segmento (**DNS local / split-horizon**, Pi-hole, `hosts` en el PC de prueba, o router con DNS override). Otra opción es revisar en DuckDNS si conviene **quitar AAAA** mientras no tengáis IPv6 estable. Desde **fuera** de la casa, el hostname con WAN suele ir bien.

Incluí **ambos** orígenes en `CORS_ORIGINS` si usáis IP y hostname (ver [`.env.prod.example`](../.env.prod.example)).

## Secretos (checklist 4.2)

- En el servidor solo **`/DATA/AppData/gonsgarage/.env.prod`** (o la ruta que uses); **no** subir `.env.prod` al git (está en [`.gitignore`](../.gitignore)).
- Obligatorios: **`JWT_SECRET`** (largo, aleatorio), **`DATABASE_URL`** (usuario dedicado, no el superusuario de Postgres).
- **`REDIS_URL`** en el compose prod apunta al contenedor Redis interno; no hace falta secret extra para Redis salvo que cambies el diseño.

## Backup de BD (checklist 5.3)

**Política mínima:** antes de migraciones o deploys que toquen esquema, y al menos **semanal** si hay datos que no quieras reconstruir solo desde seed.

1. **Volcado desde el host** (si tenés `psql`/`pg_dump` y red o túnel al Postgres que usa `DATABASE_URL`):

   ```bash
   pg_dump "$DATABASE_URL" -Fc -f "gonsgarage_$(date +%Y%m%d_%H%M).dump"
   ```

   Ajustá `DATABASE_URL` exportada o usá un `.pgpass` / URI completa con usuario dedicado (mismos credenciales que el API, no el rol superusuario).

2. **Postgres solo en Docker** (p. ej. contenedor `arnela-postgres`): ejecutá `pg_dump` **dentro** del contenedor o contra el hostname desde una máquina que tenga acceso de red:

   ```bash
   docker exec -t arnela-postgres pg_dump -U TU_USUARIO -d gonsgarage -Fc > "gonsgarage_$(date +%Y%m%d).dump"
   ```

3. Guardá los `.dump` **fuera** del volumen que podrías borrar en un rollback; probá restaurar en entorno de prueba al menos una vez (`pg_restore -d …`).

La sección **Rollback** más abajo repite un `pg_dump` rápido como paso previo a cambios arriesgados.

## Verificación tras el primer `up` (4.3 / 4.4)

Desde cualquier máquina en la LAN (sustituí host/puerto si cambiaron):

```bash
curl -sS "http://192.168.1.100:8102/health"
curl -sS "http://192.168.1.100:8102/ready"
```

- **4.3:** respuestas JSON OK del API vía nginx.
- **4.4:** abrir `http://192.168.1.100:8102` en el navegador, **login** contra el mismo origen (el front ya fue construido con `NEXT_PUBLIC_API_URL` igual a esa URL pública).

Si `/ready` falla, Postgres no es alcanzable desde el contenedor del API (revisá `DATABASE_URL`, firewall y `host.docker.internal`).

## Login → **502 Bad Gateway** (nginx)

El front carga pero **`POST /api/v1/auth/login`** devuelve 502: nginx **no** recibe respuesta OK del contenedor **`gonsgarage-api`** (caído, reiniciando o no escucha en `8080`).

- **Scripts / CI:** cualquier `docker compose -f docker-compose.prod.yml …` que **no** pase `--env-file .env.prod` puede fallar al interpolar variables (`NEXT_PUBLIC_API_URL`, etc.) o dejarte un estado incoherente. Usá siempre el mismo `--env-file` que en el `up` (ver [`scripts/update-server-gonsgarage.sh`](../scripts/update-server-gonsgarage.sh)).

En el **servidor**:

```bash
cd /DATA/AppData/gonsgarage
docker compose -f docker-compose.prod.yml --env-file .env.prod ps
docker logs gonsgarage-api --tail 120
```

Si **`gonsgarage-api`** aparece como **`Restarting`**, el proceso Go está saliendo al arrancar: los logs suelen mostrar fallo de conexión a Postgres (`DATABASE_URL`), credenciales, o panic de migración. Hasta que el API quede **Up** (no reiniciando), `/health` vía nginx dará **502**.

- Si el API está **Exited** o en bucle de reinicio, casi siempre es **`DATABASE_URL`**: Postgres en el host debe escuchar en una interfaz alcanzable desde Docker (no solo `127.0.0.1`). Revisá `postgresql.conf` → `listen_addresses` y `pg_hba.conf` para permitir el bridge Docker (p. ej. `172.17.0.0/16`) o la IP del host. Alternativa: en `.env.prod`, probá el host real en lugar de `host.docker.internal` si Postgres escucha en `192.168.1.100:5432`.
- Comprobar red interna (desde el contenedor nginx al API):

```bash
docker exec gonsgarage-nginx wget -qO- -S http://gonsgarage-api:8080/health 2>&1 | head -20
```

Si esto falla con *connection refused*, el API no está arriba → prioridad: logs de `gonsgarage-api`.

### Logs: `connection refused` a `172.17.0.1:5432` (`host.docker.internal`)

Suelen convivir **dos** cosas:

1. **Contraseña / usuario de ejemplo**  
   Si en el log aparece `CHANGE_ME_strong_password_here`, tu `.env.prod` **no** está listo: editá **`DATABASE_URL`** con la contraseña **real** del rol `gonsgarage_app` (o el usuario que hayas creado en Postgres).

2. **Postgres solo escucha en `127.0.0.1`**  
   Desde el contenedor, `host.docker.internal` apunta al host (a menudo `172.17.0.1`). Si Postgres está configurado con `listen_addresses = 'localhost'` (solo `127.0.0.1`), **no** atiende en esa IP del bridge → *connection refused*.

**En el servidor (host Linux, fuera de Docker):**

```bash
ss -tlnp | grep 5432
```

- Si ves solo `127.0.0.1:5432`, tenés que ampliar escucha y permisos, por ejemplo:
  - En `postgresql.conf`: `listen_addresses = '*'` (o al menos incluir la IP del host en la LAN si preferís algo más acotado).
  - En `pg_hba.conf` una línea para la red Docker, p. ej.:  
    `host  gonsgarage  gonsgarage_app  172.17.0.0/16  scram-sha-256`  
    (ajustá método si usás `md5`. Reiniciá Postgres después: `systemctl restart postgresql` o el servicio que uses.)

**Alternativa rápida:** si tras `listen_addresses = '*'` Postgres queda en `0.0.0.0:5432`, probá en `.env.prod` sustituir el host por la **IP LAN del servidor** (p. ej. `192.168.1.100`) en lugar de `host.docker.internal`, siempre que no haya firewall bloqueando el contenedor hacia esa IP.

Después de cambiar `.env.prod`:

```bash
cd /DATA/AppData/gonsgarage
docker compose -f docker-compose.prod.yml --env-file .env.prod up -d --build
docker logs gonsgarage-api --tail 30
```

## pgAdmin desde tu PC hacia Postgres del servidor

**Recomendado (sin abrir el puerto 5432 a la LAN): túnel SSH**

1. En **PowerShell** o terminal de tu PC (con `ssh` instalado), dejá abierta una sesión que reenvíe el puerto (elegí un puerto local libre, p. ej. **5433**):

```powershell
ssh -L 5433:127.0.0.1:5432 root@192.168.1.100
```

   Eso asume que **Postgres en el servidor escucha en `127.0.0.1:5432`**. Si tu Postgres solo escucha en otra interfaz, ajustá el segundo tramo (consultá con `ss -tlnp | grep 5432` en el servidor).

2. En **pgAdmin** → *Register* → *Server* → pestaña **General**: nombre libre. Pestaña **Connection**:
   - **Host:** `127.0.0.1` o `localhost`
   - **Port:** `5433` (el local del túnel)
   - **Username / Password:** los de tu base `gonsgarage` (los mismos que usaste en `DATABASE_URL` del `.env.prod`, usuario dedicado, no hace falta ser `postgres`).

3. Mantené la ventana del **ssh** abierta mientras usás pgAdmin.

**Alternativa (directo por red):** solo si en el servidor Postgres tiene `listen_addresses` que incluya la LAN y firewall/pg_hba permiten tu IP. Entonces en pgAdmin: **Host** `192.168.1.100`, **Port** `5432`. No lo recomendamos en WiFi compartido sin TLS ni restricción por IP.

## HTTPS vs HTTP en LAN (4.5)

Para **solo red local** (`192.168.x.x`) es aceptable **HTTP** en el checklist: sin certificado, sin Let’s Encrypt. Si exponés el servidor a **internet**, añadí TLS (nginx + certbot, o TLS en el proxy) y actualizá **`CORS_ORIGINS`** / **`NEXT_PUBLIC_API_URL`** a `https://…`.

## Rollback (4.6)

1. **Antes de cada deploy:** copia del `.env.prod` vigente y, si aplica, volcado rápido de BD (`pg_dump -Fc gonsgarage > backup.dump`).
2. **Volver atrás en contenedores:** en el directorio del deploy, `docker compose -f docker-compose.prod.yml --env-file .env.prod down`, restaurá la carpeta/código anterior (o `git checkout` + rebuild) y `… up -d --build`.
3. **Solo API (binario):** sustituí el binario por la versión anterior y reiniciá el proceso; la base suele seguir compatible salvo que hayas dependido de migraciones destructivas (evitar `RESET_DATABASE` en servidor).

## Checklist MVP

Cerrar tareas **4.x** en [`docs/mvp-solo-checklist.md`](../docs/mvp-solo-checklist.md) cuando hayas hecho la **verificación en servidor** (curl + login). Este documento cubre criterios **4.2–4.6** a nivel de runbook; **4.1** son las URLs ya anotadas en el checklist.

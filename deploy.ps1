# GonsGarage - despliegue remoto (plantilla Arnela).
# Hay que ajustar $SERVER, $USER y $REMOTE_DIR antes de usar.
#
# Este script copia el arbol local (backend/, frontend/, etc.) por scp: no hace git.
# En esta carpeta del repo, ejecutar antes:  git pull
#
# Camino recomendado: scripts/deploy-prod.sh en el servidor (git pull + compose + rollback).
# Este script es la alternativa de emergencia desde tu PC y aplica las MISMAS guardas de
# aislamiento que scripts/deploy-prod.sh (P0-T1: GonsGarage PROD usa su propio `postgres`).
#
# Uso:
#   .\deploy.ps1 -DryRun     # solo guardas + comandos, sin tocar el servidor
#   .\deploy.ps1             # copia + build + up en el servidor
#
# Salida: exit 0 si las guardas pasan (y el deploy termina OK), exit 1 si algo se rechaza.
#
# Nota de encoding: este fichero se mantiene en ASCII puro porque Windows PowerShell 5.1
# lee los .ps1 sin BOM como ANSI y los acentos salen corruptos en pantalla.

[CmdletBinding()]
param(
    # Ejecuta las guardas y muestra los comandos que se enviarian al servidor, sin red.
    [switch]$DryRun,

    # Segundo `-f` opcional para `docker compose` (vacio = solo docker-compose.prod.yml).
    # Un valor que acople Arnela se rechaza; ver deploy/README.md.
    [string]$ComposeOverride = ""
)

$SERVER = "192.168.1.100"
$USER = "root"
$REMOTE_DIR = "/DATA/AppData/gonsgarage"

# Politica P0-T1: Postgres propio del stack, sin red Arnela. Vacio por defecto.
$COMPOSE_OVERRIDE = $ComposeOverride

$ErrorActionPreference = "Stop"

function Fail {
    param([string]$Message)
    Write-Host ""
    Write-Host $Message -ForegroundColor Red
    Write-Host ""
    exit 1
}

# Lista de ficheros compose y argumentos equivalentes para la shell remota.
$composeFiles = @("docker-compose.prod.yml")
if ($COMPOSE_OVERRIDE.Trim().Length -gt 0) {
    $composeFiles += $COMPOSE_OVERRIDE.Trim()
}
$composeArgs = (($composeFiles | ForEach-Object { "-f $_" }) -join " ")

# Cuerpo de la guarda remota (bash). Se ejecuta en el servidor antes de cada `up` y
# tambien en -DryRun: lo que se prueba aca es exactamente lo que corre alla.
# Equivale a assert_gonsgarage_isolation() de scripts/deploy-prod.sh.
$remoteGuardBody = @'
set -eu

# IMPORTANTE: este bloque viaja como argumento de ssh, no como archivo.
# Windows PowerShell 5.1 borra las comillas dobles de un argumento nativo, y una
# comilla doble sin cerrar deja el script remoto invalido (bash: unexpected EOF).
# Por eso aca NO se usan comillas dobles: solo simples y variables desnudas.
#
# Nombres por defecto de produccion. GONS_ENV_FILE / GONS_COMPOSE_FILE permiten
# apuntar la guarda a otros ficheros en pruebas locales, sin tocar los reales.
ENV_FILE=${GONS_ENV_FILE:-.env.prod}
COMPOSE_FILE=${GONS_COMPOSE_FILE:-docker-compose.prod.yml}
OVERRIDE='__OVERRIDE__'

if [ ! -f $COMPOSE_FILE ]; then
  echo 'Error: no se encuentra' $COMPOSE_FILE 'en' $(pwd) >&2
  exit 1
fi

if [ ! -f $ENV_FILE ]; then
  echo 'Error: falta' $ENV_FILE 'en' $(pwd) '- variables de produccion, no versionar' >&2
  exit 1
fi

case $OVERRIDE in
  '') ;;
  *[Aa][Rr][Nn][Ee][Ll][Aa]*)
    echo 'Error: COMPOSE_OVERRIDE='$OVERRIDE' acopla Arnela. GonsGarage PROD usa Postgres propio (P0-T1).' >&2
    echo '       Dejar el override vacio y apuntar DATABASE_URL al servicio postgres de docker-compose.prod.yml.' >&2
    exit 1
    ;;
  *)
    if [ ! -f $OVERRIDE ]; then
      echo 'Error: COMPOSE_OVERRIDE='$OVERRIDE' no existe en' $(pwd) >&2
      exit 1
    fi
    ;;
esac

if grep -qEi '@arnela-postgres[:/]|//arnela-postgres' $ENV_FILE 2>/dev/null; then
  echo 'Error:' $ENV_FILE 'referencia arnela-postgres. Tras el cutover P0-T1 eso esta prohibido.' >&2
  echo '       Esperado: DATABASE_URL=...@postgres:5432/gonsgarage?... - ver deploy/README.md' >&2
  exit 1
fi

if ! grep -qE '@postgres[:/]|//postgres[:/]' $ENV_FILE 2>/dev/null; then
  echo 'WARN:' $ENV_FILE 'no parece usar host postgres - servicio compose de GonsGarage.' >&2
fi

if ! grep -qE '^[[:space:]]*postgres:' $COMPOSE_FILE 2>/dev/null; then
  echo 'Error:' $COMPOSE_FILE 'no declara el servicio postgres. Sin el, el API queda sin DB.' >&2
  exit 1
fi

echo '==> Guarda de aislamiento OK - compose:' $COMPOSE_FILE 'override:' ${OVERRIDE:-ninguno}
'@

# El repo se normaliza a CRLF en Windows (core.autocrlf) y bash en Linux no tolera
# el \r al final de linea, asi que cualquier script que viaje por ssh se pasa a LF.
function ConvertTo-Lf {
    param([string]$Text)
    return $Text.Replace("`r`n", "`n")
}

function Get-RemoteGuardScript {
    return (ConvertTo-Lf $remoteGuardBody).Replace("__OVERRIDE__", $COMPOSE_OVERRIDE.Trim())
}

# --- Guarda local: el override no puede acoplar Arnela ni faltar en el repo ---
function Assert-LocalIsolation {
    if ($COMPOSE_OVERRIDE.Trim().Length -eq 0) {
        return
    }

    $override = $COMPOSE_OVERRIDE.Trim()

    # La guarda remota embute el override en un literal bash con comillas simples y lo
    # usa sin comillas: una comilla simple o un espacio en la ruta rompen esa guarda.
    if ($override -match "['\s]") {
        Fail "COMPOSE_OVERRIDE=$override contiene una comilla simple o un espacio, que rompe la guarda remota. Renombrar el archivo sin comillas ni espacios."
    }

    if ($override -match "(?i)arnela") {
        Fail @"
COMPOSE_OVERRIDE=$override acopla Arnela. GonsGarage PROD usa Postgres propio (P0-T1: cutover de red).

  En su lugar:
    1. DATABASE_URL=...@postgres:5432/gonsgarage?sslmode=disable  en .env.prod del servidor
    2. Dejar el override vacio:  .\deploy.ps1 -ComposeOverride ""
    3. Deploy por el camino recomendado: en el servidor, bash scripts/deploy-prod.sh deploy

  docker-compose.prod.arnela-network.yml queda como legado y este script ya no lo copia.
  Contexto del incidente: docs/mvp-next-steps.md#incidente-dns-arnela-postgres
"@
    }

    if (-not (Test-Path (Join-Path $PSScriptRoot $override))) {
        Fail "COMPOSE_OVERRIDE=$override no existe en $PSScriptRoot"
    }
}

# El override de red Arnela es legado: no se copia al servidor.
$items = @(
    "docker-compose.prod.yml",
    ".env.prod.example",
    "backend",
    "frontend",
    "nginx"
)

Write-Host "=== GonsGarage deploy ===" -ForegroundColor Cyan
Write-Host ""

Assert-LocalIsolation

if ($DryRun) {
    Write-Host "[DryRun] Guarda local OK." -ForegroundColor Yellow
    Write-Host ""
    Write-Host "[DryRun] Guarda remota que se ejecutaria en ${REMOTE_DIR}:" -ForegroundColor Yellow
    Write-Host (Get-RemoteGuardScript)
    Write-Host ""
    Write-Host "[DryRun] Comando de build + up:" -ForegroundColor Yellow
    Write-Host "  docker compose $composeArgs --env-file .env.prod up -d --build"
    Write-Host ""
    Write-Host "[DryRun] Nada se copio ni se ejecuto en ${USER}@${SERVER}." -ForegroundColor Yellow
    exit 0
}

Write-Host "[1/4] Crear directorio remoto..." -ForegroundColor Yellow
ssh "${USER}@${SERVER}" "mkdir -p ${REMOTE_DIR}"

Write-Host "[2/4] Copiar artefactos..." -ForegroundColor Yellow
foreach ($item in $items) {
    $localPath = Join-Path $PSScriptRoot $item
    if (Test-Path $localPath) {
        Write-Host "  -> $item" -ForegroundColor Gray
        scp -r $localPath "${USER}@${SERVER}:${REMOTE_DIR}/"
    }
    else {
        Write-Host "  [SKIP] $item no encontrado" -ForegroundColor Red
    }
}

Write-Host "[3/4] .env.prod en el servidor (si no existe)..." -ForegroundColor Yellow
$envBootstrap = ConvertTo-Lf @"
cd ${REMOTE_DIR}
if [ ! -f .env.prod ]; then
  cp .env.prod.example .env.prod
  echo '.env.prod creado desde example - editar JWT_SECRET y DATABASE_URL antes del up'
else
  echo '.env.prod ya existe'
fi
"@

ssh "${USER}@${SERVER}" $envBootstrap

Write-Host "[4/4] guardas + build + up (Docker)..." -ForegroundColor Yellow
$remoteCommand = (Get-RemoteGuardScript) + (ConvertTo-Lf @"

cd ${REMOTE_DIR}
docker compose ${composeArgs} --env-file .env.prod up -d --build
"@)

ssh "${USER}@${SERVER}" $remoteCommand
if ($LASTEXITCODE -ne 0) {
    Fail "El deploy en el servidor fallo (exit $LASTEXITCODE). Revisar la salida de arriba; rollback: bash scripts/deploy-prod.sh rollback"
}

Write-Host ""
Write-Host "=== Listo ===" -ForegroundColor Green
Write-Host "App:     http://${SERVER}:8102" -ForegroundColor Cyan
Write-Host "Health:  http://${SERVER}:8102/health" -ForegroundColor Cyan
Write-Host "Ready:   http://${SERVER}:8102/ready" -ForegroundColor Cyan
Write-Host "Swagger: http://${SERVER}:8102/swagger/index.html" -ForegroundColor Cyan

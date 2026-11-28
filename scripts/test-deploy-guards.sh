#!/usr/bin/env bash
#
# GonsGarage — pruebas offline de las guardas de deploy.ps1 (aislamiento P0-T1).
#
# Por qué existe: deploy.ps1 no solo valida en tu máquina, también manda un script
# bash al servidor por ssh. Dos trampas de Windows lo rompían en silencio y las dos
# están comentadas en deploy.ps1:
#
#   1. Windows PowerShell 5.1 borra las comillas dobles de los argumentos nativos.
#      Una sola comilla sin cerrar dejaba el script remoto inválido:
#      "bash: unexpected EOF while looking for matching `\"'".
#   2. El repo queda en CRLF al hacer checkout (core.autocrlf=true) y bash en Linux
#      no tolera el \r al final de línea.
#
# Cómo funciona: no usa red. Las funciones ssh y scp se reemplazan por stubs dentro
# de una copia de deploy.ps1, así que las pruebas leen del disco exactamente lo que
# se habría enviado al servidor. Si la inyección de los stubs falla, el script
# aborta: nunca ejecuta ssh ni scp reales contra el host de producción.
#
# Uso:
#   bash scripts/test-deploy-guards.sh
#   PW=pwsh bash scripts/test-deploy-guards.sh     # forzar un PowerShell concreto
#
# Requiere bash y PowerShell (powershell.exe o pwsh). node es opcional: sin él se
# omite la comprobación de marshalling de argumentos, que es la que atrapa la
# trampa 1 en Windows PowerShell 5.1.

# Sin `set -e`: casi todas las comprobaciones esperan un exit code distinto de cero.
set -uo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
DEPLOY_PS1="$ROOT/deploy.ps1"
DEPLOY_PROD="$ROOT/scripts/deploy-prod.sh"

PASS=0
FAIL=0
SKIP=0

ok()   { echo "  PASS  $1"; PASS=$((PASS + 1)); }
bad()  { echo "  FAIL  $1"; [ $# -gt 1 ] && echo "        $2"; FAIL=$((FAIL + 1)); }
skip() { echo "  SKIP  $1"; SKIP=$((SKIP + 1)); }

# chk <esperado> <obtenido> <etiqueta> [detalle]
chk() {
  if [ "$1" = "$2" ]; then ok "$3 (exit $2)"; else bad "$3" "esperado exit $1, obtenido $2${4:+ — $4}"; fi
}

die() { echo "ABORTA: $1" >&2; exit 1; }

# ---------------------------------------------------------------- precondiciones
[ -f "$DEPLOY_PS1" ]  || die "no encuentro $DEPLOY_PS1"
[ -f "$DEPLOY_PROD" ] || die "no encuentro $DEPLOY_PROD"

PW="${PW:-$(command -v pwsh || command -v powershell || true)}"
[ -n "$PW" ] || die "no encuentro PowerShell (instalar pwsh o ejecutar desde Windows)"

NODE="$(command -v node || true)"

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
cd "$WORK" || die "no puedo entrar en $WORK"

echo "PowerShell: $PW"
echo "node:       ${NODE:-no disponible}"
echo

# ------------------------------------------------------------------ copia con stubs
# Los stubs se inyectan justo después de $ErrorActionPreference para que existan
# antes de la primera llamada de red.
MARKER='$ErrorActionPreference = "Stop"'
grep -qF "$MARKER" "$DEPLOY_PS1" || die "deploy.ps1 ya no tiene la línea marcadora '$MARKER'; actualizar la inyección de este test"

cp "$DEPLOY_PS1" ./deploy-under-test.ps1

cat > stubs.ps1 <<'PSEOF'

# --- stubs inyectados por scripts/test-deploy-guards.sh: sin red ---
function ssh { param([Parameter(ValueFromRemainingArguments = $true)][object[]]$A2)
  $n = 0
  if (Test-Path $env:STUB_LOG) { $n = @(Select-String -Path $env:STUB_LOG -Pattern '^CALL ' -ErrorAction SilentlyContinue).Count }
  $idx = $n + 1
  [IO.File]::AppendAllText($env:STUB_LOG, "CALL $idx host=" + $A2[0] + " nargs=" + $A2.Count + "`n")
  [IO.File]::WriteAllText($env:STUB_OUT + "/body-$idx.txt", [string]$A2[1])
  $global:LASTEXITCODE = if ($env:STUB_FAIL -eq '1') { 7 } else { 0 } }

function scp { param([Parameter(ValueFromRemainingArguments = $true)][object[]]$A2)
  [IO.File]::AppendAllText($env:STUB_LOG, "SCP " + ($A2 -join ' ') + "`n")
  $global:LASTEXITCODE = 0 }
PSEOF

sed -i "/^$(printf '%s' "$MARKER" | sed 's/[][\.*^$/]/\\&/g')\$/r stubs.ps1" ./deploy-under-test.ps1
grep -q 'function ssh' ./deploy-under-test.ps1 \
  || die "no se inyectaron los stubs: NO se ejecuta nada para no arriesgar un ssh real"

# Un archivo de override de prueba, presente en el directorio raíz del script.
printf 'services: {}\n' > extra.yml

export STUB_OUT="$WORK"
export STUB_LOG="$WORK/log.txt"
: > "$STUB_LOG"

run_ps() { "$PW" -NoProfile -ExecutionPolicy Bypass -File "$@"; }

echo "== 1. Ejecución completa sin red (stubs) =="
run_ps ./deploy-under-test.ps1 > out.txt 2>&1
chk 0 $? "deploy completo con guarda OK"
SSH_CALLS=$(grep -c '^CALL ' "$STUB_LOG" || true)
SCP_CALLS=$(grep -c '^SCP ' "$STUB_LOG" || true)
chk 3 "$SSH_CALLS" "llamadas ssh (mkdir, .env.prod, guarda+up)"
echo "        scp: $SCP_CALLS (0 si no hay artefactos locales en el sandbox)"

REMOTE_CMD="$(cat "$WORK/body-3.txt" 2>/dev/null || true)"
if [ -z "$REMOTE_CMD" ]; then
  bad "se capturó el comando remoto (guarda + up)"
else
  ok "se capturó el comando remoto (guarda + up)"
fi

echo
echo "== 2. Orden en el servidor: cd -> guarda -> up =="
# La guarda mira archivos relativos, así que el cd tiene que ir antes. Y el up, después.
CD_LINE=$(printf '%s\n' "$REMOTE_CMD" | grep -n "^cd " | head -1 | cut -d: -f1)
G_LINE=$(printf '%s\n' "$REMOTE_CMD" | grep -n "no declara el servicio postgres" | head -1 | cut -d: -f1)
U_LINE=$(printf '%s\n' "$REMOTE_CMD" | grep -n "^docker compose" | head -1 | cut -d: -f1)
if [ -n "$CD_LINE" ] && [ -n "$G_LINE" ] && [ "$CD_LINE" -lt "$G_LINE" ]; then
  ok "cd (línea $CD_LINE) antes de la guarda (línea $G_LINE)"
else
  bad "la guarda corre dentro del directorio de deploy" \
      "cd=$CD_LINE guarda=$G_LINE - si va antes del cd mira el home del ssh y aborta todo deploy"
fi
if [ -n "$G_LINE" ] && [ -n "$U_LINE" ] && [ "$G_LINE" -lt "$U_LINE" ]; then
  ok "guarda (línea $G_LINE) antes de up (línea $U_LINE)"
else
  bad "orden guarda/up" "guarda=$G_LINE up=$U_LINE"
fi
# Un here-string de PowerShell no lleva salto de línea final: si alguien concatena el
# cd sin `n, el cuerpo llega como "cd /DATA/...set -eu" y el remoto no ejecuta nada útil.
FIRST_LINE=$(printf '%s\n' "$REMOTE_CMD" | sed -n '1p')
SECOND_LINE=$(printf '%s\n' "$REMOTE_CMD" | sed -n '2p')
if printf '%s' "$FIRST_LINE" | grep -qE '^cd /' && [ "$SECOND_LINE" = 'set -eu' ]; then
  ok "el cd y el set -eu van en líneas separadas"
else
  bad "el cd no quedó pegado a la guarda" "línea 1: [$FIRST_LINE] línea 2: [$SECOND_LINE]"
fi

echo
echo "== 3. Sin comillas dobles ni CR en lo que viaja por ssh =="
DQ=$(printf '%s' "$REMOTE_CMD" | tr -cd '"' | wc -c | tr -d ' ')
chk 0 "$DQ" "comillas dobles en el comando remoto"
CR=$(printf '%s' "$REMOTE_CMD" | tr -cd '\r' | wc -c | tr -d ' ')
chk 0 "$CR" "retornos de carro en el comando remoto"

echo
echo "== 4. Marshalling de argumentos (trampa de PowerShell 5.1) =="
if [ -z "$NODE" ]; then
  skip "argv intacto al pasar a un .exe nativo (falta node)"
else
  printf '%s' "$REMOTE_CMD" > expect.txt
  cat > marshal.js <<'JSEOF'
const fs = require('fs');
fs.writeFileSync('got.txt', process.argv.slice(2).join('\u0000'));
JSEOF
  cat > marshal.ps1 <<'PSEOF'
& $env:NODE marshal.js ([IO.File]::ReadAllText("expect.txt"))
PSEOF
  NODE="$NODE" run_ps ./marshal.ps1 > /dev/null 2>&1
  if cmp -s expect.txt got.txt; then
    ok "cuerpo byte-idéntico tras pasar por un proceso nativo"
  else
    bad "el cuerpo se corrompe al pasarlo a ssh" "PowerShell está comiendo comillas del argumento"
  fi
fi

echo
echo "== 5. Comportamiento de la guarda remota (archivos falsos) =="
mkdir -p site
printf 'services:\n  postgres:\n    image: pg\n' > site/ok.yml
printf 'services:\n  api:\n    image: x\n' > site/no-pg.yml
printf 'DATABASE_URL=postgres://u:p@arnela-postgres:5432/g\n' > site/bad.env
printf 'DATABASE_URL=postgres://u:p@postgres:5432/gonsgarage\n' > site/good.env
printf 'DATABASE_URL=postgres://u:p@db.internal:5432/g\n' > site/plain.env
printf 'services: {}\n' > site/extra.yml

# Solo el bloque de guarda, sin el cd previo: cada prueba lo corre en su propio
# fixture, donde el directorio de deploy es el directorio actual.
printf '%s\n' "$REMOTE_CMD" | sed -n '/^set -eu$/,/Guarda de aislamiento OK/p' > site/guard.sh
[ -s site/guard.sh ] || die "la guarda extraída quedó vacía; revisar el formato del comando remoto (¿cd pegado a set -eu?)"
grep -q 'no declara el servicio postgres' site/guard.sh \
  || die "la guarda extraída está vacía o incompleta; revisar el formato del comando remoto"

cd "$WORK/site" || die "no puedo entrar en $WORK/site"

# g <etiqueta> <exit esperado> <VAR=valor>...
g() {
  local label="$1" exp="$2"
  shift 2
  local out
  out=$(env "$@" bash guard.sh 2>&1)
  chk "$exp" $? "$label" "$(printf '%s\n' "$out" | grep -E 'Error|WARN' | head -1)"
}

g "DB apuntando a arnela-postgres"        1 GONS_COMPOSE_FILE=ok.yml   GONS_ENV_FILE=bad.env
g "compose sin servicio postgres"         1 GONS_COMPOSE_FILE=no-pg.yml GONS_ENV_FILE=good.env
g "falta el archivo de entorno"           1 GONS_COMPOSE_FILE=ok.yml   GONS_ENV_FILE=missing.env
g "falta el archivo compose"              1 GONS_COMPOSE_FILE=nope.yml GONS_ENV_FILE=good.env
g "combinación válida"                    0 GONS_COMPOSE_FILE=ok.yml   GONS_ENV_FILE=good.env
g "DB con host que no es postgres (WARN)" 0 GONS_COMPOSE_FILE=ok.yml   GONS_ENV_FILE=plain.env

# Las ramas del override se prueban horneando el valor, porque el preflight local
# corta antes de llegar al servidor.
o() {
  local label="$1" exp="$2" ov="$3"
  shift 3
  sed "s|^OVERRIDE=''|OVERRIDE='$ov'|" guard.sh > override.sh
  env "$@" bash override.sh > /dev/null 2>&1
  chk "$exp" $? "$label"
}

o "override que acopla Arnela"  1 docker-compose.prod.arnela-network.yml GONS_COMPOSE_FILE=ok.yml GONS_ENV_FILE=good.env
o "override inexistente"        1 no-existe.yml                          GONS_COMPOSE_FILE=ok.yml GONS_ENV_FILE=good.env
o "override existente"          0 extra.yml                              GONS_COMPOSE_FILE=ok.yml GONS_ENV_FILE=good.env

cd "$WORK" || die "no puedo volver a $WORK"

echo
echo "== 6. Preflight local de deploy.ps1 =="
l() {
  local label="$1" exp="$2"
  shift 2
  run_ps ./deploy-under-test.ps1 "$@" > /dev/null 2>&1
  chk "$exp" $? "$label"
}

l "override que acopla Arnela"      1 -ComposeOverride docker-compose.prod.arnela-network.yml
l "override inexistente"            1 -ComposeOverride no-existe.yml
l "override con comilla simple"     1 -ComposeOverride "it's.yml"
l "override con espacio"            1 -ComposeOverride "mi archivo.yml"
l "override con metacaracteres"     1 -ComposeOverride 'x$(id).yml'
l "-DryRun sin red"                 0 -DryRun
l "-DryRun con override válido"     0 -DryRun -ComposeOverride extra.yml

echo
echo "== 7. -ComposeOverride se copia al servidor =="
# El override se valida contra el repo local, así que también tiene que viajar: si no
# se copia, la guarda remota no lo encuentra y ese modo de deploy nunca funciona.
export STUB_LOG="$WORK/log-ov.txt"
export STUB_OUT="$WORK/ov"
mkdir -p "$STUB_OUT"
: > "$STUB_LOG"
run_ps ./deploy-under-test.ps1 -ComposeOverride extra.yml > /dev/null 2>&1
if grep -q 'extra.yml' "$STUB_LOG"; then
  ok "el override entra en la copia (scp)"
else
  bad "el override se copia al servidor" "no aparece extra.yml en las llamadas scp"
fi
export STUB_LOG="$WORK/log.txt"
export STUB_OUT="$WORK"

echo
echo "== 8. Un fallo en el servidor aborta el deploy =="
export STUB_FAIL=1
: > "$STUB_LOG"
OUT_FAIL=$(run_ps ./deploy-under-test.ps1 2>&1)
CODE_FAIL=$?
unset STUB_FAIL
chk 1 "$CODE_FAIL" "exit code distinto de cero"
if printf '%s\n' "$OUT_FAIL" | grep -q "El deploy en el servidor fallo (exit 7)"; then
  ok "mensaje de fallo con el exit real del servidor"
else
  bad "mensaje de fallo ausente o sin el exit del servidor"
fi

echo
echo "== 9. Copia en CRLF (como queda tras un checkout en Windows) =="
sed 's/$/\r/' ./deploy-under-test.ps1 > deploy-crlf.ps1
grep -q 'function ssh' ./deploy-crlf.ps1 || die "la copia CRLF perdió los stubs"
export STUB_OUT="$WORK/crlf"
export STUB_LOG="$WORK/log-crlf.txt"
mkdir -p "$STUB_OUT"
: > "$STUB_LOG"
run_ps ./deploy-crlf.ps1 > /dev/null 2>&1
chk 0 $? "corre desde un archivo CRLF"
if [ -s "$STUB_OUT/body-3.txt" ]; then
  CR2=$(tr -cd '\r' < "$STUB_OUT/body-3.txt" | wc -c | tr -d ' ')
  chk 0 "$CR2" "sin CR pese al archivo CRLF"
else
  bad "se capturó el comando remoto desde CRLF"
fi

echo
echo "== 10. Paridad estática con scripts/deploy-prod.sh =="
for pattern in '@arnela-postgres[:/]|//arnela-postgres' '^[[:space:]]*postgres:'; do
  if grep -qF "$pattern" "$DEPLOY_PS1" && grep -qF "$pattern" "$DEPLOY_PROD"; then
    ok "ambos caminos de deploy tienen el patrón: $pattern"
  else
    bad "patrón de aislamiento presente en los dos caminos: $pattern" \
        "deploy.ps1=$(grep -cF "$pattern" "$DEPLOY_PS1") deploy-prod.sh=$(grep -cF "$pattern" "$DEPLOY_PROD")"
  fi
done

echo
echo "== RESUMEN: PASS=$PASS FAIL=$FAIL SKIP=$SKIP =="
[ "$FAIL" -eq 0 ] || exit 1

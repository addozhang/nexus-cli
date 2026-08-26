#!/usr/bin/env bash
# Provisions a local Nexus Repository container for the e2e suite:
# starts sonatype/nexus3, waits for readiness, rotates the admin password,
# creates one hosted repository per supported format, and seeds Maven
# components so search/versions/info have indexed data.
#
# Idempotent: safe to re-run; skips work already done.
set -euo pipefail

CONTAINER=${NX_E2E_CONTAINER:-nx-e2e}
PORT=${NX_E2E_PORT:-8081}
DOCKER_PORT=${NX_E2E_DOCKER_PORT:-8082}
PASS=${NX_E2E_PASS:-nx-e2e-Pass1!}
BASE="http://localhost:${PORT}"

if ! docker ps --format '{{.Names}}' | grep -qx "$CONTAINER"; then
  docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
  docker run -d --name "$CONTAINER" \
    -p "${PORT}:8081" -p "${DOCKER_PORT}:8082" \
    sonatype/nexus3:latest >/dev/null
  echo "container ${CONTAINER} started"
else
  echo "container ${CONTAINER} already running"
fi

echo "waiting for Nexus to become ready (this can take a few minutes)..."
code=""
for _ in $(seq 1 120); do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/service/rest/v1/status" || true)
  [ "$code" = "200" ] && break
  sleep 5
done
if [ "$code" != "200" ]; then
  echo "error: Nexus did not become ready in time" >&2
  docker logs --tail 50 "$CONTAINER" >&2 || true
  exit 1
fi

INITIAL=$(docker exec "$CONTAINER" cat /nexus-data/admin.password 2>/dev/null || true)
if [ -n "$INITIAL" ]; then
  change=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${INITIAL}" -X PUT \
    -H 'Content-Type: text/plain' --data "$PASS" \
    "$BASE/service/rest/v1/security/users/admin/change-password")
  if [ "$change" != "204" ] && [ "$change" != "200" ]; then
    echo "error: rotating admin password failed (HTTP $change)" >&2
    exit 1
  fi
  echo "admin password rotated"
else
  check=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" "$BASE/service/rest/v1/status")
  if [ "$check" != "200" ]; then
    echo "error: existing instance rejects NX_E2E_PASS; set it correctly or remove the container" >&2
    exit 1
  fi
fi

disable_anonymous() {
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" -X PUT     -H 'Content-Type: application/json' --data '{"enabled":false}'     "$BASE/service/rest/v1/security/anonymous")
  echo "anonymous access disabled (HTTP ${code})"
}

accept_eula() {
  local accepted
  accepted=$(curl -s -u "admin:${PASS}" "$BASE/service/rest/v1/system/eula" |
    python3 -c 'import sys,json; print("true" if json.load(sys.stdin).get("accepted") else "false")')
  if [ "$accepted" = "true" ]; then
    return 0
  fi
  # Newer CE builds refuse writes until the EULA disclaimer is echoed back.
  curl -s -u "admin:${PASS}" "$BASE/service/rest/v1/system/eula" |
    python3 -c 'import sys,json; print(json.dumps({"accepted": True, "disclaimer": json.load(sys.stdin)["disclaimer"]}))' > /tmp/nx-eula.json
  local code
  code=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" -X POST \
    -H 'Content-Type: application/json' --data @/tmp/nx-eula.json \
    "$BASE/service/rest/v1/system/eula")
  rm -f /tmp/nx-eula.json
  if [ "$code" != "204" ] && [ "$code" != "200" ]; then
    echo "error: accepting EULA failed (HTTP $code)" >&2
    exit 1
  fi
  echo "EULA accepted"
}

create_repo() {
  local endpoint=$1 payload=$2 name
  name=$(printf '%s' "$payload" | sed -n 's/.*"name" *: *"\([^"]*\)".*/\1/p')
  local exists
  exists=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" "$BASE/service/rest/v1/repositories/${name}")
  if [ "$exists" = "200" ]; then
    echo "repository ${name} already exists"
    return 0
  fi
  local created
  # Deletion is async server-side: the name may stay reserved briefly even
  # after GET reports it gone, so retry a few times on 400.
  for _ in $(seq 1 10); do
    created=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" \
      -H 'Content-Type: application/json' -X POST \
      --data "$payload" "$BASE/service/rest/v1/repositories/${endpoint}")
    [ "$created" = "201" ] || [ "$created" = "200" ] && break
    sleep 2
  done
  echo "create ${name} (${endpoint}): HTTP ${created}"
}

STORAGE='{"blobStoreName":"default","strictContentTypeValidation":true,"writePolicy":"ALLOW_ONCE"}'
create_repo maven/hosted '{"name":"e2e-maven","online":true,"storage":'${STORAGE}',"maven":{"versionPolicy":"RELEASE","layoutPolicy":"STRICT"}}'
create_repo npm/hosted '{"name":"e2e-npm","online":true,"storage":'${STORAGE}'}'
create_repo pypi/hosted '{"name":"e2e-pypi","online":true,"storage":'${STORAGE}'}'
create_repo go/hosted '{"name":"e2e-go","online":true,"storage":'${STORAGE}'}'
create_repo cargo/hosted '{"name":"e2e-cargo","online":true,"storage":{"blobStoreName":"default","strictContentTypeValidation":true,"writePolicy":"ALLOW"}}'
create_repo docker/hosted '{"name":"e2e-docker","online":true,"storage":'${STORAGE}',"docker":{"v1Enabled":false,"forceBasicAuth":true,"httpPort":'${DOCKER_PORT}'}}'

accept_eula
disable_anonymous

pom() {
  cat <<XML
<project>
  <modelVersion>4.0.0</modelVersion>
  <groupId>com.example</groupId>
  <artifactId>e2e</artifactId>
  <version>$1</version>
  <packaging>jar</packaging>
</project>
XML
}

seed_maven() {
  local version=$1
  local dir="com/example/e2e/${version}"
  local code
  # Idempotent: if the jar is already deployed, leave the component alone.
  # Re-uploading into an existing RELEASE version returns 409 regardless of
  # write policy.
  if [ "$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" "$BASE/repository/e2e-maven/${dir}/e2e-${version}.jar")" = "200" ]; then
    echo "com.example:e2e:${version} already seeded"
    return 0
  fi
  code=$(printf '%s' "$(pom "$version")" | curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" \
    -H 'Content-Type: application/xml' --data-binary @- \
    -X PUT "$BASE/repository/e2e-maven/${dir}/e2e-${version}.pom")
  if [ "$code" != "201" ] && [ "$code" != "200" ]; then
    echo "error: seeding ${version} pom failed (HTTP $code)" >&2
    exit 1
  fi
  local tmpjar
  tmpjar=$(mktemp /tmp/nx-e2e-jar.XXXXXX.jar)
  python3 - "$version" "$tmpjar" <<'PY'
import sys, zipfile
v, path = sys.argv[1], sys.argv[2]
with zipfile.ZipFile(path, "w") as zf:
    zf.writestr("META-INF/MANIFEST.MF", f"Manifest-Version: 1.0\nImplementation-Version: {v}\n")
    zf.writestr("e2e.txt", f"e2e jar payload {v}")
PY
  code=$(curl -s -o /dev/null -w '%{http_code}' -u "admin:${PASS}" \
    -H 'Content-Type: application/java-archive' --data-binary "@${tmpjar}" \
    -X PUT "$BASE/repository/e2e-maven/${dir}/e2e-${version}.jar")
  rm -f "$tmpjar"
  if [ "$code" != "201" ] && [ "$code" != "200" ]; then
    echo "error: seeding ${version} jar failed (HTTP $code)" >&2
    exit 1
  fi
  echo "seeded com.example:e2e:${version}"
}


seed_maven 1.0.0
seed_maven 1.1.0

# Search indexing is asynchronous on a freshly booted instance; wait until
# the seeded components become visible before letting the suite run.
echo "waiting for search index..."
for _ in $(seq 1 60); do
  count=$(curl -s -u "admin:${PASS}" \
    "$BASE/service/rest/v1/search?q=e2e&format=maven2" |
    python3 -c 'import sys,json; print(len(json.load(sys.stdin).get("items", [])))' || echo 0)
  [ "$count" -ge 2 ] && break
  sleep 2
done
if [ "${count:-0}" -lt 2 ]; then
  echo "error: seeded components never became searchable" >&2
  exit 1
fi

echo "e2e environment ready at ${BASE}"

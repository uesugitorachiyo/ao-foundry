#!/usr/bin/env bash
set -euo pipefail

expected_authorization_sha256="b7e18a1967b31b2806184444ab9aeab5e984e050f66261431ec57ece4cc833ee"
mission_id="mission-4d91b0a9e4ab273e"
foundry_root="$(git rev-parse --show-toplevel)"
workspace_root="${AO_WORKSPACE_ROOT:-$(dirname "$foundry_root")}"

for repository in ao-mission ao-atlas; do
  if [[ ! -d "$workspace_root/$repository" ]]; then
    printf 'missing required workspace repository: %s\n' "$repository" >&2
    exit 2
  fi
done

authorization_rel="ao-mission/.ao-mission/handoffs/$mission_id/blueprint-pack/build-authorization.json"
atlas_blueprint_import_rel="ao-atlas/.atlas-local/$mission_id/atlas-import/blueprint-import.json"
atlas_import_rel="ao-atlas/.atlas-local/$mission_id/atlas-import/foundry-import/foundry-import.json"
atlas_status_rel="ao-atlas/.atlas-local/$mission_id/atlas-import/atlas-status.json"

for artifact in "$authorization_rel" "$atlas_blueprint_import_rel" "$atlas_import_rel" "$atlas_status_rel"; do
  if [[ ! -f "$workspace_root/$artifact" ]]; then
    printf 'missing required artifact: %s\n' "$artifact" >&2
    exit 2
  fi
done

tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/foundry-blueprint-atlas-pulse.XXXXXX")"
trap 'rm -rf "$tmpdir"' EXIT

authorization_sha256() {
  shasum -a 256 "$workspace_root/$authorization_rel" | awk '{print $1}'
}

before_sha256="$(authorization_sha256)"
if [[ "$before_sha256" != "$expected_authorization_sha256" ]]; then
  printf 'authorization SHA-256 mismatch before preflight: %s\n' "$before_sha256" >&2
  exit 1
fi

(
  cd "$foundry_root"
  go build -o "$tmpdir/foundry" ./cmd/foundry
)

(
  cd "$workspace_root"
  "$tmpdir/foundry" pulse intake-preflight \
    --blueprint-authorization "$authorization_rel" \
    --atlas-blueprint-import "$atlas_blueprint_import_rel" \
    --atlas-import "$atlas_import_rel" \
    --atlas-status "$atlas_status_rel" \
    --requires-atlas \
    --json \
    --out "$tmpdir/pulse-intake-preflight.json" > "$tmpdir/pulse-intake-preflight.stdout.json"
)

jq -e '
  .status == "ready" and
  .blueprint_status == "ready" and
  .atlas_blueprint_status == "ready" and
  .atlas_status == "ready" and
  .first_failing_check == ""
' "$tmpdir/pulse-intake-preflight.json" >/dev/null

jq -e '
  .build_authorization.digest == "sha256:b7e18a1967b31b2806184444ab9aeab5e984e050f66261431ec57ece4cc833ee" and
  ([.schedules_work, .executes_work, .approves_work, .mutates_repositories, .calls_providers, .release_or_publish_allowed] | all(. == false))
' "$workspace_root/$atlas_blueprint_import_rel" >/dev/null

after_sha256="$(authorization_sha256)"
if [[ "$after_sha256" != "$expected_authorization_sha256" ]]; then
  printf 'authorization SHA-256 mismatch after preflight: %s\n' "$after_sha256" >&2
  exit 1
fi

set +e
unsafe_output="$(cd "$foundry_root" && "$tmpdir/foundry" pulse intake-preflight --blueprint-authorization "../$authorization_rel" 2>&1)"
unsafe_rc=$?
set -e
if [[ "$unsafe_rc" -eq 0 ]] || [[ "$unsafe_output" != *"unsafe source artifact path"* ]]; then
  printf 'expected parent-path preflight rejection, got exit=%s output=%s\n' "$unsafe_rc" "$unsafe_output" >&2
  exit 1
fi

printf 'blueprint_atlas_pulse_contract=ready\n'
printf 'authorization_sha256=%s\n' "$after_sha256"

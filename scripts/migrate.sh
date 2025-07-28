#!/usr/bin/env bash
#
# migrate.sh
#
# A simple migration runner for PostgreSQL.
# Usage:
#   ./migrate.sh up [target_version]
#   ./migrate.sh down [target_version]
#
# Relies on the following environment variables for DB connection:
#   PGHOST, PGPORT, PGUSER, PGPASSWORD, PGDATABASE
#
# Expects a folder "migrations/" with files named:
#   001_description.up.sql
#   001_description.down.sql
#   002_another.up.sql
#   002_another.down.sql
#   ...
#
# The "migrations" table should be created by migration 001:
#   migrations(
#     id SERIAL PRIMARY KEY,
#     version VARCHAR(255) UNIQUE NOT NULL,
#     run_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
#   );
#
set -eo pipefail

MIGRATIONS_DIR="./migrations"
PSQL="psql -v ON_ERROR_STOP=1 -qAt"

if [[ -z "$PGDATABASE" ]]; then
  echo "Error: PGDATABASE is not set. Please export PGDATABASE, PGHOST, PGPORT, PGUSER, and PGPASSWORD."
  exit 1
fi

function usage() {
  echo "Usage: $0 {up|down} [target_version]"
  echo "  e.g.: $0 up           # migrate all the way to latest"
  echo "        $0 up 002       # migrate up through version 002"
  echo "        $0 down         # rollback all migrations"
  echo "        $0 down 001     # rollback down to version 001 (i.e., undo > 001)"
  exit 1
}

if [[ $# -lt 1 || $# -gt 2 ]]; then
  usage
fi

DIRECTION="$1"            # "up" or "down"
TARGET_VERSION="${2:-}"   # e.g. "002"; if empty, defaults as below

# Helper: Ensure migrations directory exists
if [[ ! -d "$MIGRATIONS_DIR" ]]; then
  echo "Error: migrations directory '$MIGRATIONS_DIR' not found."
  exit 1
fi

# Helper: get list of all versions from filenames (sorted numerically)
function all_versions() {
  ls "$MIGRATIONS_DIR"/*.up.sql 2>/dev/null | \
    sed -E 's#^.*/([0-9]+)_.*\.up\.sql$#\1#;' | \
    sort -n | uniq
}

# Helper: check if migrations table exists
function migrations_table_exists() {
  local count
  count=$($PSQL -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'migrations';")
  if [[ "$count" -eq 1 ]]; then
    echo "yes"
  else
    echo "no"
  fi
}

# Helper: get applied versions from migrations table (sorted numerically)
function applied_versions() {
  $PSQL -c "SELECT version FROM migrations ORDER BY version::int;"
}

# Helper: apply a single UP migration for version $1
function apply_up_migration() {
  local ver="$1"
  local upfile
  upfile=$(ls "$MIGRATIONS_DIR"/"$ver"_*.up.sql 2>/dev/null | head -n1)
  if [[ -z "$upfile" ]]; then
    echo "Error: Up file for version $ver not found."
    exit 1
  fi
  echo "Applying UP migration: $upfile"
  psql -f "$upfile"
  echo "INSERT INTO migrations (version) VALUES ('$ver');" | psql
}

# Helper: apply a single DOWN migration for version $1
function apply_down_migration() {
  local ver="$1"
  local downfile
  downfile=$(ls "$MIGRATIONS_DIR"/"$ver"_*.down.sql 2>/dev/null | head -n1)
  if [[ -z "$downfile" ]]; then
    echo "Error: Down file for version $ver not found."
    exit 1
  fi
  echo "Applying DOWN migration: $downfile"
  psql -f "$downfile"
  echo "DELETE FROM migrations WHERE version = '$ver';" | psql
}

# ------------------------------
# START MIGRATION LOGIC
# ------------------------------
case "$DIRECTION" in
  up)
    # Gather all versions
    VERSIONS=($(all_versions))
    if [[ ${#VERSIONS[@]} -eq 0 ]]; then
      echo "No .up.sql migrations found in $MIGRATIONS_DIR."
      exit 0
    fi

    # Determine target version
    if [[ -z "$TARGET_VERSION" ]]; then
      TARGET_VERSION="${VERSIONS[-1]}"
    fi

    # Check that target exists in the list
    if ! printf '%s\n' "${VERSIONS[@]}" | grep -q "^$TARGET_VERSION$"; then
      echo "Error: target version $TARGET_VERSION not found among migrations."
      exit 1
    fi

    # If migrations table does not exist, apply 001 up first to create it if 001 is defined
    if [[ "$(migrations_table_exists)" == "no" ]]; then
      # Expect version "001" to exist
      if printf '%s\n' "${VERSIONS[@]}" | grep -q "^001$"; then
        apply_up_migration "001"
      else
        echo "Error: migrations table missing & no 001 migration to create it."
        exit 1
      fi
    fi

    # Fetch applied versions into a Bash array
    mapfile -t APPLIED < <(applied_versions)

    # For each version <= TARGET_VERSION, if not applied, apply it
    for ver in "${VERSIONS[@]}"; do
      if [[ "$ver" -le "$TARGET_VERSION" ]]; then
        if ! printf '%s\n' "${APPLIED[@]}" | grep -q "^$ver$"; then
          apply_up_migration "$ver"
        else
          echo "Skipping already applied version: $ver"
        fi
      fi
    done
    ;;

  down)
    # Check if migrations table exists; if not, nothing to rollback
    if [[ "$(migrations_table_exists)" == "no" ]]; then
      echo "Migrations table does not exist; nothing to roll back."
      exit 0
    fi

    # Fetch applied versions in descending order
    mapfile -t APPLIED_DESC < <(
      psql -qAt -c "SELECT version FROM migrations ORDER BY version::int DESC;"
    )
    if [[ ${#APPLIED_DESC[@]} -eq 0 ]]; then
      echo "No applied migrations to roll back."
      exit 0
    fi

    # Determine target for down
    if [[ -z "$TARGET_VERSION" ]]; then
      # Roll back all: target = "000"
      TARGET_VERSION="000"
    else
      # Validate target is a numeric string (matching existing version format)
      if [[ ! "$TARGET_VERSION" =~ ^[0-9]+$ ]]; then
        echo "Error: target version must be a numeric string like 001, 002, etc."
        exit 1
      fi
    fi

    # For each applied version > TARGET_VERSION, roll it back
    for ver in "${APPLIED_DESC[@]}"; do
      if [[ "$ver" > "$TARGET_VERSION" ]]; then
        apply_down_migration "$ver"
      else
        echo "Reached target version $TARGET_VERSION; stopping down migrations."
        break
      fi
    done
    ;;

  *)
    usage
    ;;
esac

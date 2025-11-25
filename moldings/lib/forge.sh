#!/bin/bash
# lib/forge.sh — The Core Logic
set -euo pipefail

# --- LOGGING HELPER ---
log_info()  { echo "   ℹ️  $1"; }
log_succ()  { echo "   ✅ $1"; }
log_warn()  { echo "   ⚠️  $1"; }
log_error() { echo "   ❌ $1"; }

# --- ENVIRONMENT LOADER ---
# Usage: load_and_validate_env "path/to/.env" "VAR1" "VAR2" ...
function load_and_validate_env() {
    local ENV_FILE="$1"
    shift
    local REQUIRED_VARS=("$@")

    if [ ! -f "$ENV_FILE" ]; then
        log_error ".env file not found at $ENV_FILE"
        exit 1
    fi

    log_info "Loading environment from $ENV_FILE"
    set -a
    source "$ENV_FILE"
    set +a

    local missing_vars=()
    for var in "${REQUIRED_VARS[@]}"; do
        if [ -z "${!var:-}" ]; then
            missing_vars+=("$var")
        fi
    done

    if [ ${#missing_vars[@]} -gt 0 ]; then
        log_error "Missing required variables in .env:"
        for var in "${missing_vars[@]}"; do
            echo "      - $var"
        done
        exit 1
    fi
}

# --- THE FORGE WORKER ---
# Usage: forge_component "component_name" "molding_base_dir" "output_dir"
function forge_component() {
    local COMPONENT=$1
    local BASE_MOLDING_DIR=$2
    local BASE_OUTPUT_DIR=$3
    
    local SRC_DIR="${BASE_MOLDING_DIR}/${COMPONENT}"
    local OUT_DIR="${BASE_OUTPUT_DIR}/${COMPONENT}"
    local OVERRIDE_DIR="./overrides/${COMPONENT}"

    # 1. CHECK OVERRIDES (The Escape Hatch)
    if [ -d "$OVERRIDE_DIR" ] && [ "$(ls -A $OVERRIDE_DIR)" ]; then
        log_warn "Forging $COMPONENT using LOCAL OVERRIDES"
        mkdir -p "$OUT_DIR"
        cp -r "$OVERRIDE_DIR/"* "$OUT_DIR/"
        return
    fi

    # 2. STANDARD FORGING (Your Logic)
    if [ ! -d "${SRC_DIR}" ]; then
        log_warn "Skipping ${COMPONENT}: No molding found at ${SRC_DIR}"
        return
    fi

    log_info "Forging $COMPONENT from moldings..."
    mkdir -p "${OUT_DIR}"

    find "${SRC_DIR}" -type f | while read -r file; do
        rel_path="${file#${SRC_DIR}/}"
        out_file="${OUT_DIR}/${rel_path}"
        out_dir_path="$(dirname "${out_file}")"

        mkdir -p "${out_dir_path}"

        # Logic A: Always envsubst .env files
        if [[ "${file}" == *.env ]]; then
            envsubst < "${file}" > "${out_file}"
            continue
        fi

        # Logic B: Optimization - Check if file has variables
        if grep -q '\${' "${file}"; then
            envsubst < "${file}" > "${out_file}"
        else
            cp "${file}" "${out_file}"
        fi
    done
}
#!/bin/bash
# cloudintegration.sh - Deploys the SigNoz cloud integration agent into a cloud
# account, creating its prerequisites first. foundry.sh fetches and runs it.
#
# Usage:
#   curl -fsSL https://signoz.io/foundry.sh | bash -s -- cloudintegration <platform> <flavor> [flags]
#   FOUNDRY_VERSION=v0.1.4 bash cloudintegration.sh <platform> <flavor> [flags]
#   bash cloudintegration.sh -h

set -euo pipefail

readonly NAME="cloudintegration.sh"
readonly REPO="SigNoz/foundry"
readonly AGENT_IMAGE="docker.io/signoz/cloud-integration-agent"

FOUNDRY_VERSION="${FOUNDRY_VERSION:-}"
PLATFORM=""
FLAVOR=""
REGION=""
INTEGRATION_ACCOUNT_ID=""
SIGNOZ_API_URL=""
SIGNOZ_API_KEY=""
INGESTION_URL=""
INGESTION_KEY=""
AGENT_VERSION=""
# Flags parse_args does not know, left for the platform to parse.
PLATFORM_ARGS=()

# https://no-color.org honoured; auto-stripped when stderr is not a TTY.
if [[ -t 2 ]] && [[ -z "${NO_COLOR:-}" ]]; then
  readonly C_INFO=$'\033[32;1m'
  readonly C_WARN=$'\033[33;1m'
  readonly C_ERROR=$'\033[31;1m'
  readonly C_RESET=$'\033[0m'
else
  readonly C_INFO=""
  readonly C_WARN=""
  readonly C_ERROR=""
  readonly C_RESET=""
fi

info() {
  echo "${C_INFO}[INFO]${C_RESET} $*"
}

warn() {
  echo "${C_WARN}[WARN]${C_RESET} $*" >&2
}

err() {
  echo "${C_ERROR}[ERROR]${C_RESET} $*" >&2
}

die() {
  err "$*"
  exit 1
}

help() {
  printf "NAME\n"
  printf "\t%s - Deploy the SigNoz cloud integration agent\n\n" "${NAME}"
  printf "USAGE\n"
  printf "\t%s <platform> <flavor> [flags]\n\n" "${NAME}"
  printf "DESCRIPTION\n"
  printf "\tCreates the prerequisites of the SigNoz cloud integration agent in a cloud account and deploys it.\n"
  printf "\tEvery step is safe to re-run; re-run the same command to retry a failed deployment.\n\n"
  printf "OPTIONS\n"
  printf "\t--region <region>\t\tRegion the integration is deployed to.\n"
  printf "\t--integration-account-id <uuid>\tSigNoz cloud integration account ID.\n"
  printf "\t--signoz-api-url <url>\t\tSigNoz API URL the agent checks in with.\n"
  printf "\t--signoz-api-key <key>\t\tSigNoz API key the agent checks in with.\n"
  printf "\t--ingestion-url <url>\t\tSigNoz ingestion URL.\n"
  printf "\t--ingestion-key <key>\t\tSigNoz ingestion key.\n"
  printf "\t--agent-version <version>\tAgent version (e.g. v0.0.16). [default: the platform's]\n"
  printf "\t-h, --help\t\t\tShow this help message.\n\n"
  printf "ENVIRONMENT\n"
  printf "\tFOUNDRY_VERSION\tFoundry release whose templates are deployed. Set by foundry.sh.\n\n"
  gcp_help
}

require_value() {
  if [[ $# -lt 2 ]] || [[ -z "$2" ]] || [[ "$2" == --* ]]; then
    die "Missing value for $1"
  fi
}

require_flag() {
  if [[ -z "$2" ]]; then
    die "Missing required flag: $1"
  fi
}

# Runs the command after $1 (attempts) until it succeeds, 10s apart. Absorbs
# the eventual consistency of freshly created cloud identities.
retry() {
  local attempts="$1"
  shift

  local attempt
  for ((attempt = 1; attempt < attempts; attempt++)); do
    if "$@"; then
      return
    fi

    warn "Retrying in 10s (attempt ${attempt}/${attempts})"
    sleep 10
  done

  "$@"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -h | --help)
        help
        exit 0
        ;;
      --*)
        require_value "$@"
        case "$1" in
          --region) REGION="$2" ;;
          --integration-account-id) INTEGRATION_ACCOUNT_ID="$2" ;;
          --signoz-api-url) SIGNOZ_API_URL="$2" ;;
          --signoz-api-key) SIGNOZ_API_KEY="$2" ;;
          --ingestion-url) INGESTION_URL="$2" ;;
          --ingestion-key) INGESTION_KEY="$2" ;;
          --agent-version) AGENT_VERSION="$2" ;;
          *) PLATFORM_ARGS+=("$1" "$2") ;;
        esac
        shift 2
        ;;
      -*) die "Unknown flag: $1" ;;
      *)
        if [[ -z "${PLATFORM}" ]]; then
          PLATFORM="$1"
        elif [[ -z "${FLAVOR}" ]]; then
          FLAVOR="$1"
        else
          die "Unexpected argument: $1"
        fi
        shift
        ;;
    esac
  done
}

# Value formats are validated by the deployed templates, not here.
verify_inputs() {
  if [[ -z "${FOUNDRY_VERSION}" ]]; then
    die "FOUNDRY_VERSION is not set. Run through foundry.sh or set it to a foundry release tag."
  fi

  require_flag --region "${REGION}"
  require_flag --integration-account-id "${INTEGRATION_ACCOUNT_ID}"
  require_flag --signoz-api-url "${SIGNOZ_API_URL}"
  require_flag --signoz-api-key "${SIGNOZ_API_KEY}"
  require_flag --ingestion-url "${INGESTION_URL}"
  require_flag --ingestion-key "${INGESTION_KEY}"

  case "${AGENT_VERSION}" in
    "" | v*) ;;
    *) AGENT_VERSION="v${AGENT_VERSION}" ;;
  esac
}

run() {
  case "${PLATFORM}/${FLAVOR}" in
    gcp/terraform)
      gcp_parse_args ${PLATFORM_ARGS[@]+"${PLATFORM_ARGS[@]}"}
      verify_inputs
      gcp_terraform
      ;;
    /) die "Missing <platform> <flavor>. Run with -h for usage." ;;
    *) die "Unsupported cloudintegration: ${PLATFORM}/${FLAVOR}" ;;
  esac
}

fail_trap() {
  local rc=$?
  if [[ ${rc} -ne 0 ]]; then
    err "Failed to deploy the cloud integration."
    err "Re-run the same command to retry; completed steps are reused."
    err "For support, see https://github.com/${REPO}/issues"
  fi
  exit "${rc}"
}

# ---------------------------------------------------------------------------
# GCP (terraform): Infrastructure Manager deploys the module in GCP_MODULE_DIR.
# ---------------------------------------------------------------------------

readonly GCP_MODULE_DIR="docs/examples/cloudintegration/gcp/terraform/pours/cloudintegration"
# Names both the Infrastructure Manager deployment and the service account it runs as.
readonly GCP_BOOTSTRAP="signoz-integration-bootstrap"
readonly GCP_SERVICES=(
  config.googleapis.com
  cloudbuild.googleapis.com
  storage.googleapis.com
  iam.googleapis.com
  serviceusage.googleapis.com
  cloudresourcemanager.googleapis.com
  secretmanager.googleapis.com
)
readonly GCP_BOOTSTRAP_ROLES=(
  roles/config.agent
  roles/serviceusage.serviceUsageAdmin
  roles/resourcemanager.projectIamAdmin
  roles/iam.serviceAccountAdmin
  roles/iam.serviceAccountUser
  roles/run.admin
  roles/pubsub.admin
  roles/logging.configWriter
)

GCP_PROJECT=""
GCP_API_SECRET="signoz-integration-api-credentials"
GCP_INGESTION_SECRET="signoz-integration-ingestion-key-credential"

gcp_help() {
  printf "GCP TERRAFORM\n"
  printf "\tEnables APIs, stores the keys in Secret Manager, creates the %s service account\n" "${GCP_BOOTSTRAP}"
  printf "\tand deploys the terraform module with Infrastructure Manager.\n\n"
  printf "\t--gcp-project <id>\t\tProject the integration is deployed to and monitors.\n"
  printf "\t--gcp-api-secret <id>\t\tSecret holding the API key. [default: %s]\n" "${GCP_API_SECRET}"
  printf "\t--gcp-ingestion-secret <id>\tSecret holding the ingestion key. [default: %s]\n\n" "${GCP_INGESTION_SECRET}"
  printf "\tcurl -fsSL https://signoz.io/foundry.sh | bash -s -- cloudintegration gcp terraform \\\\\n"
  printf "\t  --region us-central1 --gcp-project my-project --integration-account-id <uuid> \\\\\n"
  printf "\t  --signoz-api-url <url> --signoz-api-key <key> --ingestion-url <url> --ingestion-key <key>\n"
}

gcp_parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --gcp-project) GCP_PROJECT="$2" ;;
      --gcp-api-secret) GCP_API_SECRET="$2" ;;
      --gcp-ingestion-secret) GCP_INGESTION_SECRET="$2" ;;
      *) die "Unknown flag for gcp terraform: $1" ;;
    esac
    shift 2
  done
}

gcp_verify_inputs() {
  require_flag --gcp-project "${GCP_PROJECT}"

  if [[ "${GCP_API_SECRET}" == "${GCP_INGESTION_SECRET}" ]]; then
    die "--gcp-api-secret and --gcp-ingestion-secret must differ"
  fi
}

gcp_verify_prereqs() {
  if ! command -v gcloud >/dev/null 2>&1; then
    die "Missing prerequisite: gcloud (https://cloud.google.com/sdk/docs/install)"
  fi

  local account
  account="$(gcloud config get-value account 2>/dev/null || true)"
  if [[ -z "${account}" ]]; then
    die "gcloud is not signed in. Run 'gcloud auth login' and retry."
  fi

  if ! gcloud projects describe "${GCP_PROJECT}" --format='value(projectId)' >/dev/null; then
    die "Cannot access project ${GCP_PROJECT} as ${account}"
  fi

  info "Deploying to project ${GCP_PROJECT} in ${REGION} as ${account}"
}

gcp_enable_services() {
  local enabled
  enabled="$(gcloud services list --enabled --project="${GCP_PROJECT}" --format='value(config.name)')"

  local missing=()
  local service
  for service in "${GCP_SERVICES[@]}"; do
    if ! grep -qxF "${service}" <<<"${enabled}"; then
      missing+=("${service}")
    fi
  done

  if [[ ${#missing[@]} -eq 0 ]]; then
    info "Required APIs are already enabled"
    return
  fi

  info "Enabling APIs: ${missing[*]}"
  gcloud services enable "${missing[@]}" --project="${GCP_PROJECT}" --quiet

  info "Waiting 60s for the APIs to propagate"
  sleep 60
}

# Creates secret $1 unless it exists, adds $2 as a new version and stores the
# version number in the variable named $3. Refuses secrets SigNoz does not own.
gcp_put_secret() {
  local secret="$1"
  local value="$2"

  local label
  if label="$(gcloud secrets describe "${secret}" --project="${GCP_PROJECT}" --format='value(labels.managed-by)' 2>/dev/null)"; then
    if [[ "${label}" != "signoz" ]]; then
      die "Secret ${secret} exists but is not labeled managed-by=signoz. Pass another secret name."
    fi
    info "Reusing secret ${secret}"
  else
    info "Creating secret ${secret}"
    gcloud secrets create "${secret}" \
      --project="${GCP_PROJECT}" \
      --replication-policy=user-managed \
      --locations="${REGION}" \
      --labels=managed-by=signoz \
      --quiet
  fi

  local version
  version="$(printf '%s' "${value}" | gcloud secrets versions add "${secret}" \
    --project="${GCP_PROJECT}" \
    --data-file=- \
    --format='value(name.basename())')"
  info "Added version ${version} to secret ${secret}"
  printf -v "$3" '%s' "${version}"
}

gcp_bootstrap_email() {
  echo "${GCP_BOOTSTRAP}@${GCP_PROJECT}.iam.gserviceaccount.com"
}

gcp_create_bootstrap() {
  local email
  email="$(gcp_bootstrap_email)"

  local created="false"
  if gcloud iam service-accounts describe "${email}" --project="${GCP_PROJECT}" >/dev/null 2>&1; then
    info "Reusing service account ${email}"
  else
    info "Creating service account ${email}"
    gcloud iam service-accounts create "${GCP_BOOTSTRAP}" \
      --project="${GCP_PROJECT}" \
      --display-name="SigNoz bootstrap executor" \
      --quiet
    created="true"
  fi

  info "Granting roles to ${email}"
  local role
  for role in "${GCP_BOOTSTRAP_ROLES[@]}"; do
    retry 6 gcloud projects add-iam-policy-binding "${GCP_PROJECT}" \
      --member="serviceAccount:${email}" \
      --role="${role}" \
      --condition=None \
      --quiet >/dev/null
  done

  local secret
  for secret in "${GCP_API_SECRET}" "${GCP_INGESTION_SECRET}"; do
    retry 6 gcloud secrets add-iam-policy-binding "${secret}" \
      --project="${GCP_PROJECT}" \
      --member="serviceAccount:${email}" \
      --role=roles/secretmanager.admin \
      --condition=None \
      --quiet >/dev/null
  done

  local account
  account="$(gcloud config get-value account 2>/dev/null)"
  local member="user:${account}"
  if [[ "${account}" == *.gserviceaccount.com ]]; then
    member="serviceAccount:${account}"
  fi

  retry 6 gcloud iam service-accounts add-iam-policy-binding "${email}" \
    --project="${GCP_PROJECT}" \
    --member="${member}" \
    --role=roles/iam.serviceAccountUser \
    --condition=None \
    --quiet >/dev/null

  if [[ "${created}" == "true" ]]; then
    info "Waiting 60s for the new grants to propagate"
    sleep 60
  fi
}

# $1 and $2 are the API and ingestion secret versions.
gcp_deploy() {
  local deployment="projects/${GCP_PROJECT}/locations/${REGION}/deployments/${GCP_BOOTSTRAP}"

  local inputs="project_id=${GCP_PROJECT}"
  inputs+=",deployment_region=${REGION}"
  inputs+=",integration_account_id=${INTEGRATION_ACCOUNT_ID}"
  inputs+=",signoz_api_url=${SIGNOZ_API_URL}"
  inputs+=",ingestion_url=${INGESTION_URL}"
  inputs+=",api_secret_id=${GCP_API_SECRET}"
  inputs+=",api_secret_version=$1"
  inputs+=",ingestion_secret_id=${GCP_INGESTION_SECRET}"
  inputs+=",ingestion_secret_version=$2"
  if [[ -n "${AGENT_VERSION}" ]]; then
    inputs+=",agent_image=${AGENT_IMAGE}:${AGENT_VERSION}"
  fi

  info "Deploying ${GCP_MODULE_DIR} at ${FOUNDRY_VERSION} to ${deployment}"
  gcloud infra-manager deployments apply "${deployment}" \
    --project="${GCP_PROJECT}" \
    --service-account="projects/${GCP_PROJECT}/serviceAccounts/$(gcp_bootstrap_email)" \
    --git-source-repo="https://github.com/${REPO}.git" \
    --git-source-directory="${GCP_MODULE_DIR}" \
    --git-source-ref="${FOUNDRY_VERSION}" \
    --tf-version-constraint="=1.5.7" \
    --input-values="${inputs}" \
    --quiet

  echo
  info "Deployed ${deployment}"
}

gcp_terraform() {
  gcp_verify_inputs
  gcp_verify_prereqs
  gcp_enable_services

  local api_key="${SIGNOZ_API_KEY//\\/\\\\}"
  api_key="${api_key//\"/\\\"}"

  local api_secret_version ingestion_secret_version
  gcp_put_secret "${GCP_API_SECRET}" "{\"api_key\":\"${api_key}\"}" api_secret_version
  gcp_put_secret "${GCP_INGESTION_SECRET}" "${INGESTION_KEY}" ingestion_secret_version

  gcp_create_bootstrap
  gcp_deploy "${api_secret_version}" "${ingestion_secret_version}"
}

trap fail_trap EXIT

parse_args "$@"
run
trap - EXIT

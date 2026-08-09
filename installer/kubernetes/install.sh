#!/usr/bin/env bash
set -euo pipefail

RENOS_NAMESPACE="${RENOS_NAMESPACE:-renos}"
RELEASE_NAME="${RELEASE_NAME:-renos}"
CHART_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HELM_VALUES="${HELM_VALUES:-}"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }
header() {
  echo ""
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
  echo -e "${CYAN}  $*${NC}"
  echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

check_prerequisites() {
  header "Checking prerequisites"

  if ! command -v kubectl &>/dev/null; then
    error "kubectl is not installed. Please install kubectl first."
    exit 1
  fi
  info "kubectl: $(kubectl version --client --short 2>/dev/null || kubectl version --client)"

  if ! command -v helm &>/dev/null; then
    error "helm is not installed. Please install Helm first."
    exit 1
  fi
  info "helm: $(helm version --short)"

  if ! kubectl cluster-info &>/dev/null; then
    error "Cannot connect to Kubernetes cluster. Check your kubeconfig."
    exit 1
  fi
  info "Kubernetes cluster: connected"
}

prompt_config() {
  header "Configuration"

  if [[ -z "${HELM_VALUES}" ]]; then
    read -r -p "Enter the domain for RenOS (e.g., renos.example.com) [renos.local]: " DOMAIN
    DOMAIN="${DOMAIN:-renos.local}"

    read -r -p "Enable TLS/HTTPS? (y/N): " TLS_ENABLED
    TLS_ENABLED="${TLS_ENABLED:-n}"

    read -r -p "Enable ingress? (Y/n): " INGRESS_ENABLED
    INGRESS_ENABLED="${INGRESS_ENABLED:-y}"

    read -r -p "Enable backups? (y/N): " BACKUP_ENABLED
    BACKUP_ENABLED="${BACKUP_ENABLED:-n}"

    read -r -p "Enable monitoring (Prometheus ServiceMonitor)? (y/N): " MONITORING_ENABLED
    MONITORING_ENABLED="${MONITORING_ENABLED:-n}"

    read -r -p "Enable autoscaling? (Y/n): " AUTOSCALING_ENABLED
    AUTOSCALING_ENABLED="${AUTOSCALING_ENABLED:-y}"

    read -r -p "Core replicas [2]: " CORE_REPLICAS
    CORE_REPLICAS="${CORE_REPLICAS:-2}"

    read -r -p "Hub replicas [2]: " HUB_REPLICAS
    HUB_REPLICAS="${HUB_REPLICAS:-2}"

    read -r -p "Enable NATS cluster (3 replicas)? (Y/n): " NATS_CLUSTER
    NATS_CLUSTER="${NATS_CLUSTER:-y}"

    echo ""
    info "Configuration complete. Writing values file..."
  fi
}

install_chart() {
  header "Installing RenOS"

  VALUES_FILE=$(mktemp)
  trap 'rm -f "${VALUES_FILE}"' EXIT

  if [[ -n "${HELM_VALUES}" ]]; then
    cp "${HELM_VALUES}" "${VALUES_FILE}"
    info "Using provided values file: ${HELM_VALUES}"
  else
    cat > "${VALUES_FILE}" <<EOF
global:
  imagePullPolicy: IfNotPresent

ingress:
  enabled: ${INGRESS_ENABLED}
  hosts:
    - host: ${DOMAIN}
      paths:
        - path: /api
          pathType: Prefix
          service: core
          port: 8080
        - path: /
          pathType: Prefix
          service: hub
          port: 3000
EOF

    if [[ "${TLS_ENABLED,,}" =~ ^(yes|y)$ ]]; then
      cat >> "${VALUES_FILE}" <<EOF
  tls:
    - hosts:
        - ${DOMAIN}
      secretName: renos-tls
EOF
    fi

    if [[ -n "${CORE_REPLICAS}" ]]; then
      cat >> "${VALUES_FILE}" <<EOF
core:
  replicaCount: ${CORE_REPLICAS}
EOF
    fi

    if [[ -n "${HUB_REPLICAS}" ]]; then
      cat >> "${VALUES_FILE}" <<EOF
hub:
  replicaCount: ${HUB_REPLICAS}
EOF
    fi

    if [[ "${BACKUP_ENABLED,,}" =~ ^(yes|y)$ ]]; then
      cat >> "${VALUES_FILE}" <<EOF
backup:
  enabled: true
EOF
    fi

    if [[ "${MONITORING_ENABLED,,}" =~ ^(yes|y)$ ]]; then
      cat >> "${VALUES_FILE}" <<EOF
monitoring:
  serviceMonitor:
    enabled: true
EOF
    fi

    if [[ "${AUTOSCALING_ENABLED,,}" =~ ^(yes|y)$ ]]; then
      cat >> "${VALUES_FILE}" <<EOF
autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
EOF
    fi

    if [[ "${NATS_CLUSTER,,}" =~ ^(yes|y)$ ]]; then
      cat >> "${VALUES_FILE}" <<EOF
nats:
  cluster:
    enabled: true
    replicas: 3
EOF
    else
      cat >> "${VALUES_FILE}" <<EOF
nats:
  cluster:
    enabled: false
    replicas: 1
EOF
    fi
  fi

  info "Creating namespace '${RENOS_NAMESPACE}' if it doesn't exist..."
  kubectl create namespace "${RENOS_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

  info "Installing/Upgrading Helm release '${RELEASE_NAME}'..."
  helm upgrade --install "${RELEASE_NAME}" "${CHART_DIR}" \
    --namespace "${RENOS_NAMESPACE}" \
    --values "${VALUES_FILE}" \
    --create-namespace \
    --wait \
    --timeout 10m

  info "Helm installation complete!"
}

wait_for_pods() {
  header "Waiting for all pods to be ready"

  local retries=60
  local interval=10
  local count=0

  while [[ $count -lt $retries ]]; do
    local not_ready
    not_ready=$(kubectl get pods -n "${RENOS_NAMESPACE}" --no-headers 2>/dev/null | \
      awk '{print $3}' | grep -c -v -E "^(Running|Completed|Succeeded)$" || true)

    if [[ "${not_ready}" -eq 0 ]]; then
      info "All pods are ready!"
      echo ""
      kubectl get pods -n "${RENOS_NAMESPACE}" -o wide
      return 0
    fi

    echo -n "."
    sleep "${interval}"
    count=$((count + 1))
  done

  echo ""
  warn "Some pods are not ready after $((retries * interval)) seconds."
  kubectl get pods -n "${RENOS_NAMESPACE}" -o wide
}

create_admin_user() {
  header "Creating admin user"

  local core_pod
  core_pod=$(kubectl get pods -n "${RENOS_NAMESPACE}" \
    -l "app.kubernetes.io/component=core" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)

  if [[ -z "${core_pod}" ]]; then
    warn "No core pod found. Skipping admin user creation."
    warn "You can create the admin user manually by running:"
    echo "  kubectl exec -n ${RENOS_NAMESPACE} -it <core-pod> -- renos admin create"
    return
  fi

  local admin_user="${RENOS_ADMIN_USER:-admin}"
  local admin_email="${RENOS_ADMIN_EMAIL:-admin@${DOMAIN:-renos.local}}"
  local admin_password="${RENOS_ADMIN_PASSWORD:-}"

  if [[ -z "${admin_password}" ]]; then
    admin_password=$(tr -dc 'A-Za-z0-9!@#$%^&*()_+' < /dev/urandom | head -c 20 || true)
    if [[ -z "${admin_password}" ]]; then
      admin_password="renos-$(date +%s)"
    fi
  fi

  info "Creating admin user '${admin_user}'..."
  kubectl exec -n "${RENOS_NAMESPACE}" "${core_pod}" -- \
    renos admin create \
    --username "${admin_user}" \
    --email "${admin_email}" \
    --password "${admin_password}" 2>/dev/null || {

    warn "Could not create admin user automatically."
    warn "The core pod may still be initializing. You can create the admin user later."
    return
  }

  echo ""
  info "Admin user created successfully!"
  SAVED_ADMIN_PASSWORD="${admin_password}"
}

print_summary() {
  header "Installation Summary"

  echo -e "${GREEN}RenOS has been installed successfully!${NC}"
  echo ""

  echo -e "  Namespace:    ${CYAN}${RENOS_NAMESPACE}${NC}"
  echo -e "  Release:      ${CYAN}${RELEASE_NAME}${NC}"
  echo ""

  local hub_svc
  hub_svc=$(kubectl get svc -n "${RENOS_NAMESPACE}" -l "app.kubernetes.io/component=hub" \
    -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "renos-hub")

  echo -e "  Access URLs:"
  echo -e "    Hub:     ${CYAN}http://${hub_svc}.${RENOS_NAMESPACE}.svc.cluster.local:3000${NC}"
  echo -e "    Core:    ${CYAN}http://${hub_svc}.${RENOS_NAMESPACE}.svc.cluster.local:8080${NC}"
  echo ""

  if [[ "${INGRESS_ENABLED:-n}" =~ ^(yes|y)$ ]] && [[ -n "${DOMAIN:-}" ]]; then
    echo -e "  Ingress:"
    if [[ "${TLS_ENABLED:-n}" =~ ^(yes|y)$ ]]; then
      echo -e "    ${CYAN}https://${DOMAIN}${NC}"
    else
      echo -e "    ${CYAN}http://${DOMAIN}${NC}"
    fi
    echo ""
  fi

  if [[ -n "${SAVED_ADMIN_PASSWORD:-}" ]]; then
    echo -e "  Admin Credentials:"
    echo -e "    Username: ${CYAN}admin${NC}"
    echo -e "    Password: ${CYAN}${SAVED_ADMIN_PASSWORD}${NC}"
    echo -e "    ${YELLOW}Please save these credentials and change the password after first login.${NC}"
    echo ""
  fi

  echo -e "  Useful commands:"
  echo -e "    ${YELLOW}kubectl get pods -n ${RENOS_NAMESPACE}${NC}"
  echo -e "    ${YELLOW}kubectl logs -n ${RENOS_NAMESPACE} -l app.kubernetes.io/component=core${NC}"
  echo -e "    ${YELLOW}helm list -n ${RENOS_NAMESPACE}${NC}"
  echo -e "    ${YELLOW}helm status ${RELEASE_NAME} -n ${RENOS_NAMESPACE}${NC}"
  echo ""
}

main() {
  echo ""
  echo -e "${CYAN}╔══════════════════════════════════════════════════╗${NC}"
  echo -e "${CYAN}║         RenOS Installer for Kubernetes          ║${NC}"
  echo -e "${CYAN}║              Version: 1.0.0-rc1                 ║${NC}"
  echo -e "${CYAN}╚══════════════════════════════════════════════════╝${NC}"
  echo ""

  check_prerequisites
  prompt_config
  install_chart
  wait_for_pods
  create_admin_user
  print_summary
}

main "$@"

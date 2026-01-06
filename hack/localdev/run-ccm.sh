#!/bin/bash

# Set paths (adjust as needed)
ORIGINAL_DIR="$(pwd)"  # Assumes you run this from the project root
TEMP_DIR="/tmp/oci-cloud-controller-manager-local"
PATCH_FILE="$ORIGINAL_DIR/hack/localdev/local-run.patch"
SIDECAR_PATCH_FILE="$ORIGINAL_DIR/patches/0001-Modify-sidecar-upstream-to-use-versiond-feature-gate.patch"

# Default flags
CLEAN=false
SKIP_VENDOR=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --skip-vendor)
            SKIP_VENDOR=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            echo "Usage: $0 [--clean] [--skip-vendor]"
            exit 1
            ;;
    esac
done

# Clean up any existing temp dir if --clean is specified
if [ "$CLEAN" = true ]; then
    rm -rf "$TEMP_DIR"
fi

# Build rsync exclusions
EXCLUSIONS=(--exclude='.git' --exclude='venv3')
if [ "$SKIP_VENDOR" = true ]; then
    EXCLUSIONS+=(--exclude='vendor')
fi

#Runs mock imds server
go run hack/localdev/run_imds.go ccm &
IMDS_PID=$!
# Trap to kill the server on script exit or interrupt
trap "kill $IMDS_PID 2>/dev/null; echo 'Mock IMDS server terminated.'" EXIT INT TERM
# Give the server a moment to start 
sleep 1

# Copy the project with exclusions
rsync -av "${EXCLUSIONS[@]}" "$ORIGINAL_DIR/" "$TEMP_DIR/"

# Apply patch in temp dir
cd "$TEMP_DIR"
if [ -f .git ]; then
    git apply "$PATCH_FILE"
    git apply "$SIDECAR_PATCH_FILE"
else
    patch -p1 < "$PATCH_FILE"
    patch -p1 < "$SIDECAR_PATCH_FILE"
fi

# Run the Make target
export CONFIG_YAML_FILENAME=$HOME/cloudconfig_local
export PROVISIONER_TYPE="oracle.com/oci"
export CE_ENDPOINT_OVERRIDE="https://containerengine-integ.us-phoenix-1.oci.oraclecloud.com"
export ENABLE_OCI_SERVICE_CONTROLLER=false
export CPO_ENABLE_RESOURCE_ATTRIBUTION=false
export ENABLE_NOR_CONTROLLER=false
export ENABLE_POD_READINESS_CONTROLLER=false
export ENABLE_MIXED_CLUSTERS_SUPPORT=false

GOFIPS140=latest go run cmd/cloud-provider-oci/main.go  \
	  --kubeconfig=$HOME/.kube/config                 \
	  --cloud-config=$CONFIG_YAML_FILENAME     \
	  --cluster-cidr=10.244.0.0/16                   \
	  --csi-enabled=false							 \
	  --cloud-provider=oci                           \
	  --use-resource-principal=false				 \
	  --metrics-endpoint=0.0.0.0:9008	\
	  --enable-volume-provisioning=false \
	  --concurrent-service-syncs=3 \
	  -v=4

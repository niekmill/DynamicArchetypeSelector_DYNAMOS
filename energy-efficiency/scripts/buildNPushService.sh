#!/bin/bash

# Change this to the path of the DYNAMOS repository on your disk
echo "Setting up paths..."
# Path to root of DYNAMOS project on local machine
DYNAMOS_ROOT="/Users/niekbremer/Downloads/solverDEMO/dynamos-collin"

# Exit on errors
set -e

# Function to display usage instructions
usage() {
  echo "Usage: $0 -s <service-name> -p <service-path-from-DYNAMOSroot> [-b <branch-name>] [-r <docker-registry>]"
  echo "  -s: Service name (required)"
  echo "  -p: Path to service from DYNAMOS root (required)"
  echo "  -b: Branch name (default: current git branch)"
  echo "  -r: Docker registry (default: docker.io/poetoec)"
  exit 1
}

# Default values
DOCKER_REGISTRY="docker.io/poetoec"
BRANCH_NAME=$(git rev-parse --abbrev-ref HEAD)

# Parse arguments
while getopts "s:p:b:r:" opt; do
  case $opt in
    s) SERVICE_NAME="$OPTARG" ;;
    p) SERVICE_PATH_FROM_ROOT="$OPTARG" ;;
    b) BRANCH_NAME="$OPTARG" ;;
    r) DOCKER_REGISTRY="$OPTARG" ;;
    *) usage ;;
  esac
done

# Validate required arguments
if [ -z "$SERVICE_NAME" ] || [ -z "$SERVICE_PATH_FROM_ROOT" ]; then
  echo "Error: Service name and service path are required."
  usage
fi

# Set the path from DYNAMOS root
SERVICE_PATH="${DYNAMOS_ROOT}${SERVICE_PATH_FROM_ROOT}"

# Ensure the provided path exists
if [ ! -d "$SERVICE_PATH" ]; then
  echo "Error: The provided DYNAMOS root path does not exist: $SERVICE_PATH"
  exit 1
fi

# Build Docker image tag
IMAGE_TAG="${DOCKER_REGISTRY}/${SERVICE_NAME}:${BRANCH_NAME}"

# Display settings
echo "Service Name: $SERVICE_NAME"
echo "Branch Name: $BRANCH_NAME"
echo "Docker Registry: $DOCKER_REGISTRY"
echo "Image Tag: $IMAGE_TAG"

# Change to the service directory
cd "$SERVICE_PATH"

# Ensure the Dockerfile exists
if [ ! -f "Dockerfile" ]; then
  echo "Error: Dockerfile not found in the service path: $SERVICE_PATH"
  exit 1
fi

# Build the Docker image
echo "Building Docker image..."
docker build --build-arg NAME="${SERVICE_NAME}" -t "${IMAGE_TAG}" .

# Push the Docker image
echo "Pushing Docker image to registry..."
docker push "${IMAGE_TAG}"

# Deployment completion message (exit on errors is enabled, so if errors occur this message will not be shown)
echo "Docker image ${IMAGE_TAG} built and pushed successfully."

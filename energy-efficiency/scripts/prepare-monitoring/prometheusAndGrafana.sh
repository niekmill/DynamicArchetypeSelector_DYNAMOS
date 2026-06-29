#!/bin/bash

# Change this to the path of the DYNAMOS repository on your disk
echo "Setting up paths..."
# Path to root of DYNAMOS project on local machine
DYNAMOS_ROOT="/Users/niekbremer/Downloads/solverDEMO/dynamos-collin"
# Charts
charts_path="${DYNAMOS_ROOT}/charts"
monitoring_chart="${charts_path}/monitoring"

# Create the namespace in the Kubernetes cluster (if not exists)
kubectl create namespace monitoring

# Install and add Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

# Additional information to helm chart used: https://artifacthub.io/packages/helm/prometheus-community/kube-prometheus-stack
# It is a widely used chart and maintained extensively (at least this was at the time of creating: 2025). Formerly named: prometheus-operator

# Install prometheus stack (this may take a while before the pods are running (sometimes even up to minutes))
# -i flag allows helm to install it if it does not exist yet, otherwise upgrade it
# Use the monitoring namcespace for prometheus (and use config file with the -f flag)
# Using upgrade ensures that helm manages it correctly, this will upgrade or install if not exists
# This names the release 'prometheus'. This is VERY IMPORTANT, because this release will be used by Kepler and others to create ServiceMonitors for example
# Use specific version to ensure compatability (this version has worked in previous setups)
helm upgrade -i prometheus prometheus-community/kube-prometheus-stack \
    --namespace monitoring \
    --version 68.1.0 \
    -f "$monitoring_chart/prometheus-config.yaml"
# Prometheus stack already includes grafana itself with a default setup (saves time to set it up yourself)
# Uninstall the release using helm to rollback changes: helm uninstall prometheus --namespace monitoring
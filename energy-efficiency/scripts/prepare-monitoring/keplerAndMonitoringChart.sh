#!/bin/bash

# Change this to the path of the DYNAMOS repository on your disk
echo "Setting up paths..."
# Path to root of DYNAMOS project on local machine
DYNAMOS_ROOT="/Users/niekbremer/Downloads/solverDEMO/dynamos-collin"
# Charts
charts_path="${DYNAMOS_ROOT}/charts"
monitoring_chart="${charts_path}/monitoring"
monitoring_values="$monitoring_chart/values.yaml"

# More information on Kepler installation: https://sustainable-computing.io/installation/kepler-helm/
# Installing Prometheus (and Grafana) can be skipped, this is already done earlier 

# Install and add Kepler
helm repo add kepler https://sustainable-computing-io.github.io/kepler-helm-chart
helm repo update
# Install Kepler
# This also creates a service monitor for the prometheus stack
# Use specific version to ensure compatability (this version has worked in previous setups)
helm upgrade -i kepler kepler/kepler \
    --namespace monitoring \
    --version 0.5.12 \
    --set serviceMonitor.enabled=true \
    --set serviceMonitor.labels.release=prometheus \
# Uninstall the release using helm to rollback changes: helm uninstall kepler -n monitoring

# Apply/install the monitoring helm release (will use the monitoring charts,
# which includes the deamonset, service and sesrvicemonitor for cadvisor for example)
# Optional: enable debug flag to output logs for potential errors (add --debug to the end of the next line)
helm upgrade -i monitoring $monitoring_chart --namespace monitoring -f "$monitoring_values"
# Uninstall the release using helm to rollback changes: helm uninstall monitoring --namespace monitoring
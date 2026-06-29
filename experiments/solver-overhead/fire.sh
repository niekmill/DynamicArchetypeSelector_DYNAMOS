#!/bin/bash
set -u

STATE=${1:-on}
N=${2:-300}

kubectl set env deploy/orchestrator -n orchestrator SOLVER_ENABLED=$STATE >/dev/null
kubectl rollout status deploy/orchestrator -n orchestrator --timeout=180s >/dev/null
sleep 10

START=$(date -u +%s)
for i in $(seq 1 $N); do
  curl -s -o /dev/null -X POST http://localhost:8080/api/v1/requestApproval \
    -H 'Content-Type: application/json' --max-time 10 \
    -d '{"type":"sqlDataRequest","user":{"id":"1234","userName":"jorrit.stutterheim@cloudnation.nl"},"dataProviders":["UVA"]}' &
  if (( i % 25 == 0 )); then wait; fi
  sleep 0.2
done
wait
END=$(date -u +%s)

echo "state=$STATE start=$START end=$END"

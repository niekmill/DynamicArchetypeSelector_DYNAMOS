#!/bin/bash
set -eu

put() {
  local key="$1"
  local value="$2"
  echo ">> $key"
  kubectl exec -n core etcd-0 -c etcd -- /usr/local/bin/etcdctl put "$key" "$value" >/dev/null
}

put /archetypes/computeToData-cached     '{"name":"computeToData-cached","computeProvider":"dataProvider","resultRecipient":"requestor","weight":100}'
put /archetypes/computeToData-compressed '{"name":"computeToData-compressed","computeProvider":"dataProvider","resultRecipient":"requestor","weight":100}'
put /archetypes/dataThroughTtp-cached    '{"name":"dataThroughTtp-cached","computeProvider":"other","resultRecipient":"requestor","weight":200}'
put /archetypes/dataThroughTtp-compressed '{"name":"dataThroughTtp-compressed","computeProvider":"other","resultRecipient":"requestor","weight":200}'

echo
echo "=== final archetype keys ==="
kubectl exec -n core etcd-0 -c etcd -- /usr/local/bin/etcdctl get --prefix /archetypes --keys-only

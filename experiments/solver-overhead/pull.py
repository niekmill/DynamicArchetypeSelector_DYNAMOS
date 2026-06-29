#!/usr/bin/env python3
import sys
import json
import urllib.parse
import urllib.request

start, end = int(sys.argv[1]), int(sys.argv[2])

QUERIES = {
    "cpu_cores":     'sum(rate(container_cpu_usage_seconds_total{pod=~"orchestrator.*",namespace="orchestrator"}[1m]))',
    "energy_joules": 'sum(rate(kepler_container_joules_total{container_name="orchestrator",mode="dynamic"}[1m]))',
}

def fetch(query):
    url = f"http://localhost:9090/api/v1/query_range?query={urllib.parse.quote(query)}&start={start}&end={end}&step=5"
    data = json.load(urllib.request.urlopen(url))
    vals = [float(v) for s in data["data"]["result"] for _, v in s["values"]]
    return sum(vals) / len(vals) if vals else 0

for name, q in QUERIES.items():
    print(f"{name}: {fetch(q):.4g}")

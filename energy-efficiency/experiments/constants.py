# Data folder types
DATA_TYPE_NORMAL = "normal"
DATA_TYPE_FABRIC = "fabric"
DATA_TYPE_FOLDERS = {
    DATA_TYPE_NORMAL: "data",
    DATA_TYPE_FABRIC: "data-fabric"
}

# Experiment script values
# All prefixes, i.e. implementations
IMPLEMENTATIONS_PREFIXES = ["baseline", "compression", "caching"]
OPTIMIZATIONS_PREFIXES = ["compression", "caching"]
ARCHETYPES = ["ComputeToData", "DataThroughTTP"]
ARCHETYPE_ACRONYMS = {
    "ComputeToData": "CtD", 
    "DataThroughTTP": "DtTTP"
}

# Prometheus 
PROMETHEUS_URL = "http://prometheus-kube-prometheus-prometheus.monitoring.svc.cluster.local:9090"
PROM_CONTAINERS = "{container_name=~\"system_processes|uva|vu|surf|sql.*|policy.*|orchestrator|sidecar|rabbitmq|api-gateway\"}"
PROM_KEPLER_ENERGY_METRIC = "kepler_container_joules_total"
PROM_KEPLER_CONTAINER_LABEL = "container_name"
PROM_ENERGY_QUERY_TOTAL = f"sum({PROM_KEPLER_ENERGY_METRIC}{PROM_CONTAINERS}) by ({PROM_KEPLER_CONTAINER_LABEL})"
PROM_ENERGY_QUERY_RANGE = f"sum(increase({PROM_KEPLER_ENERGY_METRIC}{PROM_CONTAINERS}[2m])) by ({PROM_KEPLER_CONTAINER_LABEL})"

# Experiment configurations
NUM_EXP_ACTIONS = 7  # Number of actions per experiment
IDLE_PERIOD = 45  # Idle period in seconds
ACTIVE_PERIOD = 120  # Active period in seconds

# DYNAMOS requests
REQUEST_URLS = {
    "uva":  "http://uva.uva.svc.cluster.local:8080/agent/v1/sqlDataRequest/uva",
    "surf": "http://surf.surf.svc.cluster.local:8080/agent/v1/sqlDataRequest/surf",
}
HEADERS = {
    "Content-Type": "application/json",
    # Access token required for data requests in DYNAMOS
    "Authorization": "bearer 1234"
}
INITIAL_REQUEST_BODY = {
    "type": "sqlDataRequest",
    "query": "SELECT DISTINCT p.Unieknr, p.Geslacht, p.Gebdat, s.Aanst_22, s.Functcat, s.Salschal as Salary FROM Personen p JOIN Aanstellingen s ON p.Unieknr = s.Unieknr LIMIT 30000",
    "algorithm": "",
    "options": {"graph": False, "aggregate": False},
    "user": {"id": "12324", "userName": "jorrit.stutterheim@cloudnation.nl"},
}
APPROVAL_URL = "http://api-gateway.api-gateway.svc.cluster.local:8080/api/v1/requestApproval"
HEADERS_APPROVAL = {"Content-Type": "application/json"}
REQUEST_BODY_APPROVAL = {
    "type": "sqlDataRequest",
    "user": {
        "id": "12324",
        "userName": "jorrit.stutterheim@cloudnation.nl"
    },
    "dataProviders": ["UVA"]
}

# Update archetypes
UPDATE_ARCH_URL = "http://orchestrator.orchestrator.svc.cluster.local:8080/api/v1/archetypes/agreements"
INITIAL_REQUEST_BODY_ARCH = {
    "name": "computeToData",
    "computeProvider": "dataProvider",
    "resultRecipient": "requestor",
}
WEIGHTS = {
    "ComputeToData": 100,
    "DataThroughTTP": 300
}
ARCH_DATA_STEWARDS = {
    # Each archetype has a different data steward it should request the data from
    "ComputeToData": "uva",
    "DataThroughTTP": "surf"
}
HEADERS_UPDATE_ARCH = { "Content-Type": "application/json" }
package solver

type Archetype struct {
	Name       string
	Attributes map[string]any
}

type RequestContext struct {
	SQLLimit            int
	HasSQLLimit         bool
	OptionalServiceHops int
	DataProviderCount   int
	IsFirstRequest bool
}

type Criterion struct {
	Name     string
	Weight   float64
	Lower    float64
	Upper    float64
	Minimize bool

	Score func(a Archetype, ctx RequestContext) float64
}

type Operator string

const (
	LessThan           Operator = "<"
	LessThanOrEqual    Operator = "<="
	GreaterThan        Operator = ">"
	GreaterThanOrEqual Operator = ">="
	Equal              Operator = "=="
)

type Constraint struct {
	Criterion string
	Operator  Operator
	Threshold float64
}

type ArchetypeResult struct {
	Name  string
	Score float64
}

type SolverResult struct {
	Ranking    []ArchetypeResult
	Normalised map[string][]float64
	Dropped    map[string]string
}

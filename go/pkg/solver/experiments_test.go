package solver

import (
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)


const experimentsDir = "../../../experiments_data"


func criteriaWithWeights(weights map[string]float64) []Criterion {
	defaults := DefaultCriteria()
	out := make([]Criterion, 0, len(defaults))
	for _, c := range defaults {
		if w, ok := weights[c.Name]; ok {
			c.Weight = w
		}
		out = append(out, c)
	}
	return out
}

func openCSV(t *testing.T, filename string) (*csv.Writer, *os.File) {
	t.Helper()
	if err := os.MkdirAll(experimentsDir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", experimentsDir, err)
	}
	path := filepath.Join(experimentsDir, filename)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}
	t.Logf("writing %s", path)
	return csv.NewWriter(f), f
}

func ftoa(x float64) string {
	return strconv.FormatFloat(x, 'f', 6, 64)
}


func TestExperiment_WeightSweep(t *testing.T) {
	w, f := openCSV(t, "weight_sweep.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"sql_limit", "varied", "weight", "archetype", "score"}); err != nil {
		t.Fatal(err)
	}

	criteriaNames := []string{"energy", "latency", "cost", "privacy"}
	arches := DefaultArchetypes()
	sqlLimits := []int{10000, 20000, 30000}

	for _, limit := range sqlLimits {
		ctx := RequestContext{
			SQLLimit:          limit,
			HasSQLLimit:       true,
			DataProviderCount: 2,
			IsFirstRequest:    false,
		}
		for _, varied := range criteriaNames {
			for step := 0; step <= 20; step++ {
				vw := float64(step) / 20.0
				remaining := (1.0 - vw) / 3.0
				weights := map[string]float64{}
				for _, c := range criteriaNames {
					if c == varied {
						weights[c] = vw
					} else {
						weights[c] = remaining
					}
				}
				res, err := Topsis(arches, criteriaWithWeights(weights), nil, ctx)
				if err != nil {
					t.Fatalf("topsis at limit=%d %s=%.2f: %v", limit, varied, vw, err)
				}
				for _, r := range res.Ranking {
					_ = w.Write([]string{strconv.Itoa(limit), varied, ftoa(vw), r.Name, ftoa(r.Score)})
				}
			}
		}
	}
}


func TestExperiment_WorkloadScaling(t *testing.T) {
	w, f := openCSV(t, "workload_scaling.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"cache_state", "expected_rows", "archetype", "score"}); err != nil {
		t.Fatal(err)
	}

	arches := DefaultArchetypes()
	weights := map[string]float64{"energy": 0.4, "latency": 0.3, "cost": 0.2, "privacy": 0.1}
	criteria := criteriaWithWeights(weights)

	// Strictly linear ladder, step = 3000, up to the DYNAMOS architectural
	// cap (~30k rows). Same convention as the scenarios-workload experiment
	// so figures share a consistent x-axis.
	sqlLimits := []int{3000, 6000, 9000, 12000, 15000, 18000, 21000, 24000, 27000, 30000}
	for _, expRows := range sqlLimits {
		for _, cold := range []bool{true, false} {
			ctx := RequestContext{
				SQLLimit:          expRows,
				HasSQLLimit:       true,
				DataProviderCount: 2,
				IsFirstRequest:    cold,
			}
			res, err := Topsis(arches, criteria, nil, ctx)
			if err != nil {
				t.Fatalf("topsis rows=%d cold=%v: %v", expRows, cold, err)
			}
			state := "warm"
			if cold {
				state = "cold"
			}
			for _, r := range res.Ranking {
				_ = w.Write([]string{state, strconv.Itoa(expRows), r.Name, ftoa(r.Score)})
			}
		}
	}
}


func TestExperiment_CacheDynamic(t *testing.T) {
	w, f := openCSV(t, "cache_dynamic.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"archetype", "score_cold", "score_warm"}); err != nil {
		t.Fatal(err)
	}

	arches := DefaultArchetypes()
	weights := map[string]float64{"energy": 0.8, "latency": 0.1, "cost": 0.05, "privacy": 0.05}
	criteria := criteriaWithWeights(weights)

	scoresFor := func(cold bool) map[string]float64 {
		ctx := RequestContext{
			SQLLimit:          20000,
			HasSQLLimit:       true,
			DataProviderCount: 2,
			IsFirstRequest:    cold,
		}
		res, err := Topsis(arches, criteria, nil, ctx)
		if err != nil {
			t.Fatalf("topsis cold=%v: %v", cold, err)
		}
		out := map[string]float64{}
		for _, r := range res.Ranking {
			out[r.Name] = r.Score
		}
		return out
	}

	cold := scoresFor(true)
	warm := scoresFor(false)
	for _, a := range arches {
		_ = w.Write([]string{a.Name, ftoa(cold[a.Name]), ftoa(warm[a.Name])})
	}
}

func TestExperiment_ConstraintFiltering(t *testing.T) {
	w, f := openCSV(t, "constraint_filtering.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"scenario", "archetype", "survived", "score"}); err != nil {
		t.Fatal(err)
	}

	arches := DefaultArchetypes()
	weights := map[string]float64{"energy": 0.4, "latency": 0.3, "cost": 0.2, "privacy": 0.1}
	criteria := criteriaWithWeights(weights)
	ctx := RequestContext{
		SQLLimit:          20000,
		HasSQLLimit:       true,
		DataProviderCount: 2,
		IsFirstRequest:    false,
	}

	scenarios := []struct {
		name        string
		constraints []Constraint
	}{
		{"no_constraints", nil},
		{"energy_le_15", []Constraint{
			{Criterion: "energy", Operator: LessThanOrEqual, Threshold: 15},
		}},
		{"energy_le_15_latency_le_5000", []Constraint{
			{Criterion: "energy", Operator: LessThanOrEqual, Threshold: 15},
			{Criterion: "latency", Operator: LessThanOrEqual, Threshold: 5000},
		}},
		{"all_three_tight", []Constraint{
			{Criterion: "energy", Operator: LessThanOrEqual, Threshold: 15},
			{Criterion: "latency", Operator: LessThanOrEqual, Threshold: 5000},
			{Criterion: "privacy", Operator: LessThanOrEqual, Threshold: 4},
		}},
	}

	for _, s := range scenarios {
		res, err := Topsis(arches, criteria, s.constraints, ctx)
		if err != nil {
			t.Logf("scenario %q: topsis err %v (treated as all dropped)", s.name, err)
		}
		survived := map[string]float64{}
		for _, r := range res.Ranking {
			survived[r.Name] = r.Score
		}
		for _, a := range arches {
			score, kept := survived[a.Name]
			_ = w.Write([]string{
				s.name,
				a.Name,
				fmt.Sprintf("%v", kept),
				ftoa(score),
			})
		}
	}
}

func TestExperiment_ScenariosWorkload(t *testing.T) {
	w, f := openCSV(t, "scenarios_workload.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"scenario", "cache_state", "sql_limit", "archetype", "score"}); err != nil {
		t.Fatal(err)
	}

	arches := DefaultArchetypes()

	scenarios := []struct {
		name    string
		weights map[string]float64
	}{
		{"balanced", map[string]float64{
			"energy": 0.25, "latency": 0.25, "cost": 0.25, "privacy": 0.25,
		}},
		{"energy", map[string]float64{
			"energy": 0.7, "latency": 0.1, "cost": 0.1, "privacy": 0.1,
		}},
		{"latency", map[string]float64{
			"energy": 0.1, "latency": 0.7, "cost": 0.1, "privacy": 0.1,
		}},
		{"monetary", map[string]float64{
			"energy": 0.1, "latency": 0.1, "cost": 0.7, "privacy": 0.1,
		}},
		{"privacy", map[string]float64{
			"energy": 0.1, "latency": 0.1, "cost": 0.1, "privacy": 0.7,
		}},
	}

	sqlLimits := []int{3000, 6000, 9000, 12000, 15000, 18000, 21000, 24000, 27000, 30000}

	for _, scen := range scenarios {
		criteria := criteriaWithWeights(scen.weights)
		for _, limit := range sqlLimits {
			for _, cold := range []bool{false, true} {
				ctx := RequestContext{
					SQLLimit:          limit,
					HasSQLLimit:       true,
					DataProviderCount: 2,
					IsFirstRequest:    cold,
				}
				res, err := Topsis(arches, criteria, nil, ctx)
				if err != nil {
					t.Fatalf("topsis err scenario=%s limit=%d cold=%v: %v",
						scen.name, limit, cold, err)
				}
				state := "warm"
				if cold {
					state = "cold"
				}
				for _, r := range res.Ranking {
					_ = w.Write([]string{
						scen.name,
						state,
						strconv.Itoa(limit),
						r.Name,
						ftoa(r.Score),
					})
				}
			}
		}
	}
}

func TestExperiment_RankReversal(t *testing.T) {
	w, f := openCSV(t, "rank_reversal.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"run", "archetype", "rank", "score"}); err != nil {
		t.Fatal(err)
	}

	allArches := DefaultArchetypes()
	var baseOnly []Archetype
	for _, a := range allArches {
		if a.Name == "computeToData" || a.Name == "dataThroughTtp" {
			baseOnly = append(baseOnly, a)
		}
	}

	weights := map[string]float64{"energy": 0.4, "latency": 0.3, "cost": 0.2, "privacy": 0.1}
	criteria := criteriaWithWeights(weights)
	ctx := RequestContext{
		SQLLimit:          20000,
		HasSQLLimit:       true,
		DataProviderCount: 2,
		IsFirstRequest:    false,
	}

	writeRun := func(run string, ranking []ArchetypeResult) {
		for i, r := range ranking {
			_ = w.Write([]string{run, r.Name, strconv.Itoa(i + 1), ftoa(r.Score)})
		}
	}

	resFull, err := Topsis(allArches, criteria, nil, ctx)
	if err != nil {
		t.Fatalf("full: %v", err)
	}
	writeRun("full_catalogue", resFull.Ranking)

	resBase, err := Topsis(baseOnly, criteria, nil, ctx)
	if err != nil {
		t.Fatalf("base_only: %v", err)
	}
	writeRun("base_only", resBase.Ranking)
}


func TestExperiment_ArchetypeEnergyLatency(t *testing.T) {
	w, f := openCSV(t, "archetype_energy_latency.csv")
	defer f.Close()
	defer w.Flush()
	if err := w.Write([]string{"cache_state", "sql_limit", "archetype", "energy", "latency"}); err != nil {
		t.Fatal(err)
	}

	arches := DefaultArchetypes()
	sqlLimits := []int{3000, 6000, 9000, 12000, 15000, 18000, 21000, 24000, 27000, 30000}

	for _, expRows := range sqlLimits {
		for _, cold := range []bool{false, true} {
			ctx := RequestContext{
				SQLLimit:          expRows,
				HasSQLLimit:       true,
				DataProviderCount: 2,
				IsFirstRequest:    cold,
			}
			state := "warm"
			if cold {
				state = "cold"
			}
			for _, a := range arches {
				e := EstimateEnergy(a, ctx)
				l := EstimateLatency(a, ctx)
				_ = w.Write([]string{state, strconv.Itoa(expRows), a.Name, ftoa(e), ftoa(l)})
			}
		}
	}
}

package solver

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrNoSurvivors = errors.New("no archetypes satisfy the constraints")
)

func Topsis(archetypes []Archetype, criteria []Criterion, constraints []Constraint, ctx RequestContext) (SolverResult, error) {
	raw := evaluate(archetypes, criteria, ctx)

	survivors, dropped := filter(archetypes, constraints, raw)
	if len(survivors) == 0 {
		return SolverResult{
			Ranking:    nil,
			Normalised: map[string][]float64{},
			Dropped:    dropped,
		}, ErrNoSurvivors
	}

	normalised := normalise(survivors, criteria, raw)
	weighted := weight(survivors, criteria, normalised)
	scored := rank(survivors, criteria, weighted)
	ordered := sort(scored)

	return SolverResult{
		Ranking:    ordered,
		Normalised: normalised,
		Dropped:    dropped,
	}, nil
}

func evaluate(archetypes []Archetype, criteria []Criterion, ctx RequestContext) map[string]map[string]float64 {
	rawScores := make(map[string]map[string]float64)

	for _, a := range archetypes {
		rawScores[a.Name] = make(map[string]float64)

		for _, c := range criteria {
			rawScores[a.Name][c.Name] = c.Score(a, ctx)
		}
	}

	return rawScores
}

func filter(archetypes []Archetype, constraints []Constraint, raw map[string]map[string]float64) ([]Archetype, map[string]string) {
	survivors := make([]Archetype, 0)
	dropped := make(map[string]string)

	for _, a := range archetypes {
		scores := raw[a.Name]
		allowed := true

		for _, constraint := range constraints {
			value, exists := scores[constraint.Criterion]
			if !exists {
				dropped[a.Name] = "missing score for constraint: " + constraint.Criterion
				allowed = false
				break
			}

			if !passesConstraint(value, constraint) {
				dropped[a.Name] = fmt.Sprintf(
					"%s %.2f violates %s %.2f",
					constraint.Criterion,
					value,
					constraint.Operator,
					constraint.Threshold,
				)
				allowed = false
				break
			}
		}

		if allowed {
			survivors = append(survivors, a)
		}
	}

	return survivors, dropped
}

func normalise(archetypes []Archetype, criteria []Criterion, raw map[string]map[string]float64) map[string][]float64 {
	normalised := make(map[string][]float64)

	for _, a := range archetypes {
		row := make([]float64, len(criteria))

		for j, c := range criteria {
			row[j] = normaliseValue(raw[a.Name][c.Name], c.Lower, c.Upper, c.Minimize)
		}

		normalised[a.Name] = row
	}

	return normalised
}

func weight(archetypes []Archetype, criteria []Criterion, normalised map[string][]float64) map[string][]float64 {
	w := make([]float64, len(criteria))
	var total float64
	for _, c := range criteria {
		total += c.Weight
	}
	if total == 0 {
		for j := range criteria {
			w[j] = 1.0 / float64(len(criteria))
		}
	} else {
		for j, c := range criteria {
			w[j] = c.Weight / total
		}
	}

	weighted := make(map[string][]float64)
	for _, a := range archetypes {
		row := make([]float64, len(criteria))
		for j := range criteria {
			row[j] = normalised[a.Name][j] * w[j]
		}
		weighted[a.Name] = row
	}

	return weighted
}

func rank(archetypes []Archetype, criteria []Criterion, weighted map[string][]float64) []ArchetypeResult {
	ideal := make([]float64, len(criteria))
	antiIdeal := make([]float64, len(criteria))

	for j := range criteria {
		ideal[j] = math.Inf(-1)
		antiIdeal[j] = math.Inf(1)

		for _, a := range archetypes {
			v := weighted[a.Name][j]
			if v > ideal[j] {
				ideal[j] = v
			}
			if v < antiIdeal[j] {
				antiIdeal[j] = v
			}
		}
	}

	results := make([]ArchetypeResult, len(archetypes))

	for i, a := range archetypes {
		var dPos, dNeg float64

		for j := range criteria {
			dPos += sq(weighted[a.Name][j] - ideal[j])
			dNeg += sq(weighted[a.Name][j] - antiIdeal[j])
		}

		dPos = math.Sqrt(dPos)
		dNeg = math.Sqrt(dNeg)

		score := 0.0
		if dPos+dNeg > 0 {
			score = dNeg / (dPos + dNeg)
		}

		results[i] = ArchetypeResult{Name: a.Name, Score: score}
	}

	return results
}

func sort(results []ArchetypeResult) []ArchetypeResult {
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-1-i; j++ {
			if results[j].Score < results[j+1].Score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}
	return results
}

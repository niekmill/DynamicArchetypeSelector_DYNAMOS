package solver

func DefaultCriteria() []Criterion {
	return []Criterion{
		{
			Name:     "energy",
			Weight:   0.1,
			Lower:    0,
			Upper:    30,
			Minimize: true,
			Score: func(a Archetype, ctx RequestContext) float64 {
				return EstimateEnergy(a, ctx)
			},
		},
		{
			Name:     "latency",
			Weight:   0.1,
			Lower:    0,
			Upper:    8800,
			Minimize: true,
			Score: func(a Archetype, ctx RequestContext) float64 {
				return EstimateLatency(a, ctx)
			},
		},
		{
			Name:     "privacy",
			Weight:   0.1,
			Lower:    0,
			Upper:    30,
			Minimize: true,
			Score: func(a Archetype, ctx RequestContext) float64 {
				return privacyScore(a, ctx)
			},
		},
		{
			Name:     "cost",
			Weight:   0.7,
			Lower:    0,
			Upper:    3.0,
			Minimize: true,
			Score: func(a Archetype, ctx RequestContext) float64 {
				return EstimateCost(a, ctx)
			},
		},
	}
}

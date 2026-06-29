package solver

import (
	"fmt"
	"math"
)

func baseFamilyOnMiss(a Archetype, ctx RequestContext) (Archetype, bool) {
	if !ctx.IsFirstRequest {
		return Archetype{}, false
	}
	isCached, _ := a.Attributes["is_cached"].(bool)
	if !isCached {
		return Archetype{}, false
	}
	family, _ := a.Attributes["family"].(string)
	if family == "" {
		return Archetype{}, false
	}
	for _, candidate := range DefaultArchetypes() {
		if candidate.Name == family {
			return candidate, true
		}
	}
	return Archetype{}, false
}

func EstimateEnergy(a Archetype, ctx RequestContext) float64 {
	if base, ok := baseFamilyOnMiss(a, ctx); ok {
		return EstimateEnergy(base, ctx)
	}

	limit := ctx.SQLLimit

	if !ctx.HasSQLLimit {
		limit = 30000
	}

	transferWeight := 1.0
	processingWeight := 50.0

	baseHops := a.Attributes["transfer_hops"].(float64)
	processingPercentBase := a.Attributes["processing_percent_base"].(float64)
	processingPercentSlope := a.Attributes["processing_percent_slope"].(float64)

	processingLoad := processingPercentBase + processingPercentSlope*float64(limit)

	if processingLoad > 1 {
		processingLoad = 1
	}

	hops := baseHops + float64(ctx.OptionalServiceHops)

	dataSizeMB := float64(limit) * 135.0 / 1_000_000.0

	networkEnergy := transferWeight * hops * dataSizeMB
	processingEnergy := processingWeight * math.Pow(processingLoad, 2)

	return networkEnergy + processingEnergy
}

func EstimateLatency(a Archetype, ctx RequestContext) float64 {
	if base, ok := baseFamilyOnMiss(a, ctx); ok {
		return EstimateLatency(base, ctx)
	}

	limit := ctx.SQLLimit

	if !ctx.HasSQLLimit {
		limit = 30000
	}

	startup := a.Attributes["startup_ms"].(float64)
	procBase := a.Attributes["processing_base_ms"].(float64)
	procSlope := a.Attributes["processing_slope_ms_per_row"].(float64)
	transferBase := a.Attributes["transfer_base_ms"].(float64)
	transferSlope := a.Attributes["transfer_slope_ms_per_row"].(float64)

	processing := procBase + procSlope*float64(limit)
	transfer := transferBase + transferSlope*float64(limit)

	return startup + processing + transfer
}

func privacyScore(a Archetype, ctx RequestContext) float64 {
	baseVal, ok := a.Attributes["parties_exposed"].(float64)
	if !ok {
		baseVal = 1.0
	}

	extra := float64(ctx.DataProviderCount - 1)
	if extra < 0 {
		extra = 0
	}

	return baseVal + extra
}

func EstimateCost(a Archetype, ctx RequestContext) float64 {
	family, _ := a.Attributes["family"].(string)
	if family == "" {
		family = a.Name
	}
	switch family {
	case "computeToData":
		v, ok := a.Attributes["cost_per_provider_eur"].(float64)
		if !ok {
			return 0
		}
		return v * float64(ctx.DataProviderCount)
	case "dataThroughTtp":
		v, ok := a.Attributes["cost_per_request_eur"].(float64)
		if !ok {
			return 0
		}
		return v
	}
	return 0
}

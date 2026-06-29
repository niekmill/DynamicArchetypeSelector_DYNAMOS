package solver

func normaliseValue(raw, lower, upper float64, minimize bool) float64 {
	if upper == lower {
		return 0.5
	}
	if upper < lower {
		lower, upper = upper, lower
	}

	x := (raw - lower) / (upper - lower)

	if x < 0 {
		x = 0
	}
	if x > 1 {
		x = 1
	}

	if minimize {
		return 1 - x
	}
	return x
}

func sq(x float64) float64 {
	return x * x
}

func passesConstraint(value float64, constraint Constraint) bool {
	switch constraint.Operator {
	case LessThan:
		return value < constraint.Threshold
	case LessThanOrEqual:
		return value <= constraint.Threshold
	case GreaterThan:
		return value > constraint.Threshold
	case GreaterThanOrEqual:
		return value >= constraint.Threshold
	case Equal:
		return value == constraint.Threshold
	default:
		return false
	}
}

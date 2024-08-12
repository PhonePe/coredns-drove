package drovedns

func boolToDouble(varr bool) float64 {
	if varr {
		return 1.0
	}
	return 0.0
}

package main

func boolAsInt(v bool) int {
	if !v {
		return 0
	} else {
		return 1
	}
}

func cmpValue[T int | int32 | int64 | float32 | float64 | string](i, j T) int {
	switch {
	case i == j:
		return 0
	case i > j:
		return 1
	default:
		return -1
	}
}

type comparator func(i, j int) int

func seqCmp(c ...comparator) comparator {
	return func(i, j int) int {
		for _, c0 := range c {
			v := c0(i, j)
			if v != 0 {
				return v
			}
		}
		return 0
	}
}

func reverse(cmp comparator) comparator {
	return func(i, j int) int {
		return -cmp(i, j)
	}
}

package shard

// VUs distributes total virtual users across n workers.
// Remainder is given to the first rem workers (stable, deterministic).
func VUs(total, n int) []int {
	if n <= 0 || total <= 0 {
		return nil
	}
	out := make([]int, n)
	base := total / n
	rem := total % n
	for i := 0; i < n; i++ {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

// QPS distributes a global cap. Zero cap means unlimited on every worker (returns zeros).
func QPS(total, n int) []int {
	if total <= 0 || n <= 0 {
		out := make([]int, n)
		return out
	}
	return VUs(total, n)
}

func Sum(xs []int) int {
	s := 0
	for _, v := range xs {
		s += v
	}
	return s
}

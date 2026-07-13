package errx

const suggestThreshold = 2

func Nearest(input string, candidates []string) string {
	best := ""
	bestDist := suggestThreshold + 1
	for _, c := range candidates {
		if c == input {
			return c
		}
		d := editDistance(input, c)
		if d < bestDist {
			best, bestDist = c, d
		}
	}
	if bestDist <= suggestThreshold {
		return best
	}
	return ""
}

func editDistance(a, b string) int {
	ar, br := []rune(a), []rune(b)
	prev := make([]int, len(br)+1)
	curr := make([]int, len(br)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(ar); i++ {
		curr[0] = i
		for j := 1; j <= len(br); j++ {
			cost := 1
			if ar[i-1] == br[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(br)]
}

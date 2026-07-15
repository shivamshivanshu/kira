package errx

import "strings"

func Nearest(input string, candidates []string) string {
	needle := []rune(strings.ToLower(input))
	threshold := len(needle) / 3
	if threshold < 1 {
		threshold = 1
	}
	best := ""
	bestDist := threshold + 1
	for _, c := range candidates {
		hay := []rune(strings.ToLower(c))
		if abs(len(needle)-len(hay)) >= bestDist {
			continue
		}
		if d := editDistance(needle, hay); d < bestDist {
			best, bestDist = c, d
		}
	}
	if bestDist <= threshold {
		return best
	}
	return ""
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func editDistance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

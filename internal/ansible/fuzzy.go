package ansible

import "strings"

// Match is the result of a fuzzy subsequence match.
type Match struct {
	Score   float64
	Indexes []int // rune indexes in the target that matched
}

// Fuzzy does a subsequence match of query against text. Higher Score is better;
// contiguous runs and early matches are rewarded. ok=false when text does not
// contain query as a subsequence.
func Fuzzy(query, text string) (Match, bool) {
	if query == "" {
		return Match{}, true
	}
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(text))

	idxs := make([]int, 0, len(q))
	ti := 0
	for _, ch := range q {
		found := -1
		for j := ti; j < len(t); j++ {
			if t[j] == ch {
				found = j
				break
			}
		}
		if found < 0 {
			return Match{}, false
		}
		idxs = append(idxs, found)
		ti = found + 1
	}

	score := float64(-idxs[0])
	for k := 1; k < len(idxs); k++ {
		a, b := idxs[k-1], idxs[k]
		if b == a+1 {
			score += 6
		} else {
			score -= float64(b - a)
		}
	}
	score -= float64(len([]rune(text))) * 0.01
	return Match{Score: score, Indexes: idxs}, true
}

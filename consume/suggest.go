package consume

import (
	"fmt"
	"strings"
)

func levenshteinDistance(a, b string) int {
	n, m := len(a), len(b)
	if n == 0 {
		return m
	}
	if m == 0 {
		return n
	}

	dp := make([]int, m+1)
	for j := range dp {
		dp[j] = j
	}

	for i := 1; i <= n; i++ {
		prev := dp[0]
		dp[0] = i
		for j := 1; j <= m; j++ {
			temp := dp[j]
			if a[i-1] == b[j-1] {
				dp[j] = prev
			} else {
				dp[j] = 1 + min(prev, dp[j], dp[j-1])
			}
			prev = temp
		}
	}
	return dp[m]
}

const maxSuggestionDistance = 2
const maxSuggestions = 3

// suggestIdentifiers searches all scopes from innermost to outermost
// for identifiers within a small edit distance of the given name.
// Returns only matches at the minimum distance found, ordered by
// scope proximity (inner scopes first).
func suggestIdentifiers(name string, env *environment) []string {
	minDist := maxSuggestionDistance + 1
	var candidates []string

	for i := len(*env) - 1; i >= 0; i-- {
		scope := (*env)[i]
		for key := range *scope {
			dist := levenshteinDistance(name, key.Name)
			if dist > maxSuggestionDistance {
				continue
			}
			if dist < minDist {
				minDist = dist
				candidates = []string{key.Name}
			} else if dist == minDist {
				candidates = append(candidates, key.Name)
			}
		}
	}

	if len(candidates) > maxSuggestions {
		candidates = candidates[:maxSuggestions]
	}
	return candidates
}

func formatSuggestion(suggestions []string) string {
	if len(suggestions) == 0 {
		return ""
	}
	quoted := make([]string, len(suggestions))
	for i, s := range suggestions {
		quoted[i] = fmt.Sprintf("'%s'", s)
	}
	if len(quoted) == 1 {
		return fmt.Sprintf(", did you mean %s?", quoted[0])
	}
	return fmt.Sprintf(", did you mean %s or %s?",
		strings.Join(quoted[:len(quoted)-1], ", "),
		quoted[len(quoted)-1])
}

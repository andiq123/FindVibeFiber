package services

import (
	"cmp"
	"math"
	"slices"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type SearchRanker struct {
	weights domain.RankingWeights
}

func NewSearchRanker(weights domain.RankingWeights) *SearchRanker {
	return &SearchRanker{weights: weights}
}

type ScoredResult struct {
	Result     domain.ProviderResult
	FinalScore float64
}

func (rr *SearchRanker) RankResults(results []domain.ProviderResult, query string, priorities map[string]int) []ScoredResult {
	if len(results) == 0 {
		return nil
	}

	q := utils.NormalizeString(query)
	artistCount := make(map[string]int, len(results))
	titles := make([]string, len(results))
	artists := make([]string, len(results))
	for i := range results {
		titles[i] = utils.NormalizeString(results[i].Song.Title)
		artists[i] = utils.NormalizeString(results[i].Song.Artist)
		artistCount[artists[i]]++
	}

	scored := make([]ScoredResult, len(results))
	for i, r := range results {
		match := matchScore(titles[i], artists[i], q)
		prio := priorities[r.Provider]
		if prio <= 0 {
			prio = 5
		}

		score := float64(prio)/10*rr.weights.ProviderPriority +
			match*rr.weights.MatchScore +
			positionScore(r.ProviderRank)*rr.weights.Position +
			diversityBonus(artists[i], artistCount)*rr.weights.Diversity
		score *= remixPenalty(titles[i], q)
		if r.Song.Image != "" {
			score += 0.05
		}

		scored[i] = ScoredResult{Result: r, FinalScore: score}
	}

	slices.SortFunc(scored, func(a, b ScoredResult) int {
		if c := cmp.Compare(b.FinalScore, a.FinalScore); c != 0 {
			return c
		}
		return cmp.Compare(a.Result.ProviderRank, b.Result.ProviderRank)
	})
	return scored
}

func matchScore(title, artist, query string) float64 {
	combined := artist + " " + title
	switch {
	case combined == query:
		return 1.0
	case title == query:
		return 0.98
	case artist == query:
		return 0.95
	}

	qWords := strings.Fields(query)
	if len(qWords) == 0 {
		return 0.5
	}
	cWords := strings.Fields(combined)
	if allContained(qWords, cWords) {
		return 0.92
	}

	tRatio := float64(countOverlap(qWords, strings.Fields(title))) / float64(len(qWords))
	aRatio := float64(countOverlap(qWords, strings.Fields(artist))) / float64(len(qWords))
	word := tRatio*0.7 + aRatio*0.3

	bonus := 0.0
	switch {
	case strings.Contains(title, query):
		bonus = 0.15
	case strings.Contains(artist, query):
		bonus = 0.10
	}

	partial := 0.0
	if word > 0 {
		partial = word * 0.85
	}
	return math.Min(math.Max(partial, word+bonus), 0.91)
}

func positionScore(rank int) float64 {
	if rank <= 0 {
		rank = 1
	}
	return 1 / math.Pow(float64(rank), 0.15)
}

func diversityBonus(artist string, counts map[string]int) float64 {
	n := counts[artist]
	if n <= 1 {
		return 1
	}
	return 1 / (1 + math.Log(float64(n)))
}

var (
	remixHeavy  = []string{"remix", "bootleg", "mashup", "rework"}
	remixMedium = []string{"cover", "instrumental", "karaoke"}
	remixLight  = []string{"live", "acoustic", "edit", "version", "extended", "remaster", "dub"}
)

func remixPenalty(title, query string) float64 {
	qWords, tWords := strings.Fields(query), strings.Fields(title)
	if hasAny(qWords, remixHeavy) || hasAny(qWords, remixMedium) || hasAny(qWords, remixLight) ||
		strings.Contains(query, "radio edit") {
		return 1
	}
	switch {
	case hasAny(tWords, remixHeavy):
		return 0.5
	case hasAny(tWords, remixMedium):
		return 0.65
	case hasAny(tWords, remixLight) || strings.Contains(title, "radio edit"):
		return 0.8
	default:
		return 1
	}
}

func allContained(needles, haystack []string) bool {
	for _, n := range needles {
		if !slices.Contains(haystack, n) {
			return false
		}
	}
	return true
}

func countOverlap(a, b []string) int {
	n := 0
	for _, x := range a {
		if slices.Contains(b, x) {
			n++
		}
	}
	return n
}

func hasAny(words, keywords []string) bool {
	for _, w := range words {
		if slices.Contains(keywords, w) {
			return true
		}
	}
	return false
}

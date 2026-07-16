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
			score += 0.03
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
	titleWords := strings.Fields(title)
	artistWords := strings.Fields(artist)
	combinedWords := strings.Fields(combined)

	matched := countMatchedWords(qWords, combinedWords)
	full := len(qWords)

	// Every query word hit (fuzzy) — strongest signal for top match.
	if matched == full {
		score := 0.93
		if full > 1 && countMatchedWords(qWords, titleWords) > 0 && countMatchedWords(qWords, artistWords) > 0 {
			score = 0.97 // e.g. "bistro" title + "morgenshtern" artist
		}
		return score
	}

	ratio := float64(matched) / float64(full)
	score := ratio * 0.90

	titleHits := countMatchedWords(qWords, titleWords)
	artistHits := countMatchedWords(qWords, artistWords)
	if titleHits > 0 && artistHits > 0 {
		score += 0.08
	} else if titleHits > 0 {
		score += 0.03
	}

	switch {
	case strings.Contains(title, query):
		score += 0.10
	case strings.Contains(artist, query):
		score += 0.06
	}

	// ponytail: self-titled echo on multi-word search (Morgenstern/Morgenstern) — weak top match
	if full > 1 && title == artist && matched < full {
		score *= 0.55
	}

	return math.Min(score, 0.92)
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

// wordMatches: exact, substring, or long shared prefix (morgenstern/morgenshtern).
func wordMatches(a, b string) bool {
	if a == b {
		return true
	}
	if len(a) < 3 || len(b) < 3 {
		return false
	}
	if strings.Contains(a, b) || strings.Contains(b, a) {
		return true
	}
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	return prefix >= 6 && absInt(len(a)-len(b)) <= 2
}

func countMatchedWords(queryWords, fieldWords []string) int {
	n := 0
	for _, q := range queryWords {
		for _, f := range fieldWords {
			if wordMatches(q, f) {
				n++
				break
			}
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

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

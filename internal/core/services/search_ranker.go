package services

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type SearchRanker struct {
	weights domain.RankingWeights
}

func NewSearchRanker(weights domain.RankingWeights) *SearchRanker {
	return &SearchRanker{
		weights: weights,
	}
}

type ScoredResult struct {
	Result     domain.ProviderResult
	FinalScore float64
}

type normalizedSong struct {
	title  string
	artist string
}

func (rr *SearchRanker) RankResults(results []domain.ProviderResult, query string, maxProviderPriority int) []ScoredResult {
	if len(results) == 0 {
		return []ScoredResult{}
	}

	scored := make([]ScoredResult, len(results))
	artistCount := make(map[string]int, len(results)/2)
	normalizedQuery := utils.NormalizeString(query)

	normalizedSongs := make([]normalizedSong, len(results))
	for i := range results {
		normalizedSongs[i].artist = utils.NormalizeString(results[i].Song.Artist)
		normalizedSongs[i].title = utils.NormalizeString(results[i].Song.Title)
		artistCount[normalizedSongs[i].artist]++
	}

	for i, result := range results {
		providerScore := rr.calculateProviderScore(maxProviderPriority)
		matchScore := rr.calculateMatchScoreOptimized(
			normalizedSongs[i].title,
			normalizedSongs[i].artist,
			normalizedQuery,
		)
		positionScore := rr.calculatePositionScore(result.ProviderRank)
		diversityBonus := rr.calculateDiversityBonusOptimized(normalizedSongs[i].artist, artistCount)
		remixPenalty := rr.calculateRemixPenaltyOptimized(normalizedSongs[i].title, normalizedQuery)

		finalScore := (providerScore * rr.weights.ProviderPriority) +
			(matchScore * rr.weights.MatchScore) +
			(positionScore * rr.weights.Position) +
			(diversityBonus * rr.weights.Diversity)

		finalScore *= remixPenalty

		scored[i] = ScoredResult{
			Result:     result,
			FinalScore: finalScore,
		}
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FinalScore > scored[j].FinalScore
	})

	return scored
}

func (rr *SearchRanker) calculateProviderScore(maxPriority int) float64 {
	if maxPriority == 0 {
		return 1.0
	}
	return float64(maxPriority) / 10.0
}

func (rr *SearchRanker) calculateMatchScoreOptimized(normalizedTitle, normalizedArtist, normalizedQuery string) float64 {
	var combined strings.Builder
	combined.Grow(len(normalizedArtist) + len(normalizedTitle) + 1)
	combined.WriteString(normalizedArtist)
	combined.WriteByte(' ')
	combined.WriteString(normalizedTitle)
	combinedStr := combined.String()

	if combinedStr == normalizedQuery {
		return 1.0
	}

	if normalizedTitle == normalizedQuery {
		return 0.98
	}

	if normalizedArtist == normalizedQuery {
		return 0.9
	}

	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)
	artistWords := strings.Fields(normalizedArtist)
	combinedWords := strings.Fields(combinedStr)

	allWordsMatch := true
	for _, qw := range queryWords {
		if !slices.Contains(combinedWords, qw) {
			allWordsMatch = false
			break
		}
	}

	if allWordsMatch && len(queryWords) > 0 {
		return 0.95
	}

	titleMatchCount := countMatchingWords(queryWords, titleWords)
	artistMatchCount := countMatchingWords(queryWords, artistWords)
	totalWords := len(queryWords)

	if totalWords == 0 {
		return 0.5
	}

	titleMatchRatio := float64(titleMatchCount) / float64(totalWords)
	artistMatchRatio := float64(artistMatchCount) / float64(totalWords)

	combinedScore := (titleMatchRatio * 0.7) + (artistMatchRatio * 0.3)

	if strings.Contains(normalizedTitle, normalizedQuery) {
		combinedScore = math.Max(combinedScore, 0.85)
	}

	similarity := calculateSimilarity(normalizedQuery, combinedStr)
	fuzzyScore := similarity * 0.8

	return math.Max(combinedScore, fuzzyScore)
}

func (rr *SearchRanker) calculatePositionScore(position int) float64 {
	if position <= 0 {
		position = 1
	}
	return 1.0 / (1.0 + math.Log(float64(position)))
}

func (rr *SearchRanker) calculateDiversityBonusOptimized(normalizedArtist string, artistCount map[string]int) float64 {
	count := artistCount[normalizedArtist]

	if count <= 1 {
		return 1.0
	}

	return 1.0 / float64(count)
}

func (rr *SearchRanker) calculateRemixPenaltyOptimized(normalizedTitle, normalizedQuery string) float64 {
	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)

	remixKeywordsList := []string{"remix", "mix", "edit", "version", "cover", "live", "acoustic", "instrumental", "extended", "remaster", "rework", "bootleg", "mashup", "dub"}

	queryHasRemixKeyword := slices.ContainsFunc(queryWords, func(word string) bool {
		return slices.Contains(remixKeywordsList, word)
	}) || strings.Contains(normalizedQuery, "radio edit")

	if queryHasRemixKeyword {
		return 1.0
	}

	if slices.ContainsFunc(titleWords, func(word string) bool {
		return slices.Contains(remixKeywordsList, word)
	}) || strings.Contains(normalizedTitle, "radio edit") {
		return 0.5
	}

	return 1.0
}

func countMatchingWords(queryWords, targetWords []string) int {
	count := 0
	for _, qw := range queryWords {
		if slices.Contains(targetWords, qw) {
			count++
		}
	}
	return count
}

func calculateSimilarity(s1, s2 string) float64 {
	distance := levenshteinDistance(s1, s2)
	maxLen := math.Max(float64(len(s1)), float64(len(s2)))

	if maxLen == 0 {
		return 1.0
	}

	return 1.0 - (float64(distance) / maxLen)
}

func levenshteinDistance(s1, s2 string) int {
	len1 := len(s1)
	len2 := len(s2)

	if len1 == 0 {
		return len2
	}
	if len2 == 0 {
		return len1
	}

	if len1 > len2 {
		s1, s2 = s2, s1
		len1, len2 = len2, len1
	}

	prevRow := make([]int, len1+1)
	currRow := make([]int, len1+1)

	for i := range prevRow {
		prevRow[i] = i
	}

	for j := 1; j <= len2; j++ {
		currRow[0] = j

		for i := 1; i <= len1; i++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			currRow[i] = min(
				prevRow[i]+1,
				currRow[i-1]+1,
				prevRow[i-1]+cost,
			)
		}

		prevRow, currRow = currRow, prevRow
	}

	return prevRow[len1]
}

func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}

package services

import (
	"math"
	"sort"
	"strings"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

type ResultRanker struct {
	weights domain.RankingWeights
}

func NewResultRanker(weights domain.RankingWeights) *ResultRanker {
	return &ResultRanker{
		weights: weights,
	}
}

type ScoredResult struct {
	Result     domain.ProviderResult
	FinalScore float64
}

func (rr *ResultRanker) RankResults(results []domain.ProviderResult, query string, maxProviderPriority int) []ScoredResult {
	if len(results) == 0 {
		return []ScoredResult{}
	}

	scored := make([]ScoredResult, 0, len(results))
	artistCount := make(map[string]int)

	for _, result := range results {
		normalizedArtist := normalizeString(result.Song.Artist)
		artistCount[normalizedArtist]++
	}

	for _, result := range results {
		providerScore := rr.calculateProviderScore(result.Provider, maxProviderPriority)
		matchScore := rr.calculateMatchScore(result.Song, query)
		positionScore := rr.calculatePositionScore(result.ProviderRank)
		diversityBonus := rr.calculateDiversityBonus(result.Song.Artist, artistCount)
		remixPenalty := rr.calculateRemixPenalty(result.Song, query)

		finalScore := (providerScore * rr.weights.ProviderPriority) +
			(matchScore * rr.weights.MatchScore) +
			(positionScore * rr.weights.Position) +
			(diversityBonus * rr.weights.Diversity)

		finalScore *= remixPenalty

		scored = append(scored, ScoredResult{
			Result:     result,
			FinalScore: finalScore,
		})
	}

	sort.Slice(scored, func(i, j int) bool {
		return scored[i].FinalScore > scored[j].FinalScore
	})

	return scored
}

func (rr *ResultRanker) calculateProviderScore(provider string, maxPriority int) float64 {
	if maxPriority == 0 {
		return 1.0
	}
	return float64(maxPriority) / 10.0
}

func (rr *ResultRanker) calculateMatchScore(song domain.Song, query string) float64 {
	normalizedQuery := normalizeString(query)
	normalizedTitle := normalizeString(song.Title)
	normalizedArtist := normalizeString(song.Artist)
	combined := normalizedArtist + " " + normalizedTitle

	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)
	artistWords := strings.Fields(normalizedArtist)
	combinedWords := strings.Fields(combined)

	// Exact match on combined artist + title
	if combined == normalizedQuery {
		return 1.0
	}

	// Exact match on title only
	if normalizedTitle == normalizedQuery {
		return 0.98
	}

	// Exact match on artist only
	if normalizedArtist == normalizedQuery {
		return 0.9
	}

	// Check if all query words exist in combined (artist + title)
	allWordsMatch := true
	for _, qw := range queryWords {
		found := false
		for _, cw := range combinedWords {
			if qw == cw {
				found = true
				break
			}
		}
		if !found {
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

	similarity := calculateSimilarity(normalizedQuery, combined)
	fuzzyScore := similarity * 0.8

	return math.Max(combinedScore, fuzzyScore)
}

func (rr *ResultRanker) calculatePositionScore(position int) float64 {
	if position <= 0 {
		position = 1
	}
	return 1.0 / (1.0 + math.Log(float64(position)))
}

func (rr *ResultRanker) calculateDiversityBonus(artist string, artistCount map[string]int) float64 {
	normalizedArtist := normalizeString(artist)
	count := artistCount[normalizedArtist]

	if count <= 1 {
		return 1.0
	}

	return 1.0 / float64(count)
}

func (rr *ResultRanker) calculateRemixPenalty(song domain.Song, query string) float64 {
	normalizedTitle := normalizeString(song.Title)
	normalizedQuery := normalizeString(query)

	remixKeywords := []string{
		"remix", "mix", "edit", "version", "cover", "live",
		"acoustic", "instrumental", "radio edit", "extended",
		"remaster", "rework", "bootleg", "mashup", "dub",
	}

	// Check if query contains remix keywords - if so, user wants a remix
	for _, keyword := range remixKeywords {
		if strings.Contains(normalizedQuery, keyword) {
			return 1.0 // No penalty if user is searching for remixes
		}
	}

	// Check if title contains remix keywords but query doesn't
	for _, keyword := range remixKeywords {
		if strings.Contains(normalizedTitle, keyword) {
			return 0.5 // Heavy penalty for remixes when user didn't ask for them
		}
	}

	return 1.0 // No penalty for original songs
}

func normalizeString(s string) string {
	s = strings.ToLower(s)
	s = strings.TrimSpace(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func countMatchingWords(queryWords, targetWords []string) int {
	count := 0
	for _, qw := range queryWords {
		for _, tw := range targetWords {
			if qw == tw {
				count++
				break
			}
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

	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}

	return matrix[len1][len2]
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

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

	// Calculate provider score once since it's the same for all results
	providerScore := rr.calculateProviderScore(maxProviderPriority)

	for i, result := range results {
		matchScore := rr.calculateMatchScoreOptimized(
			normalizedSongs[i].title,
			normalizedSongs[i].artist,
			normalizedQuery,
		)
		positionScore := rr.calculatePositionScore(result.ProviderRank)
		diversityBonus := rr.calculateDiversityBonusOptimized(normalizedSongs[i].artist, artistCount)
		remixPenalty := rr.calculateRemixPenaltyOptimized(normalizedSongs[i].title, normalizedQuery)
		metadataBonus := rr.calculateMetadataScore(result.Song)

		finalScore := (providerScore * rr.weights.ProviderPriority) +
			(matchScore * rr.weights.MatchScore) +
			(positionScore * rr.weights.Position) +
			(diversityBonus * rr.weights.Diversity)

		finalScore *= remixPenalty
		finalScore += metadataBonus

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
		return 0.95
	}

	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)
	artistWords := strings.Fields(normalizedArtist)
	combinedWords := strings.Fields(combinedStr)

	if len(queryWords) == 0 {
		return 0.5
	}

	allWordsMatch := true
	for _, qw := range queryWords {
		if !slices.Contains(combinedWords, qw) {
			allWordsMatch = false
			break
		}
	}

	if allWordsMatch {
		return 0.92
	}

	titleMatchCount := countMatchingWords(queryWords, titleWords)
	artistMatchCount := countMatchingWords(queryWords, artistWords)
	totalWords := len(queryWords)

	titleMatchRatio := float64(titleMatchCount) / float64(totalWords)
	artistMatchRatio := float64(artistMatchCount) / float64(totalWords)

	// ponytail: word/substring only — Levenshtein was O(n²) per result for little gain
	wordMatchScore := (titleMatchRatio * 0.7) + (artistMatchRatio * 0.3)

	substringBonus := 0.0
	if strings.Contains(normalizedTitle, normalizedQuery) {
		substringBonus = 0.15
	} else if strings.Contains(normalizedArtist, normalizedQuery) {
		substringBonus = 0.10
	}

	partialMatchScore := 0.0
	if titleMatchCount > 0 || artistMatchCount > 0 {
		partialMatchScore = wordMatchScore * 0.85
	}

	return math.Min(math.Max(partialMatchScore, wordMatchScore+substringBonus), 0.91)
}

func (rr *SearchRanker) calculatePositionScore(position int) float64 {
	if position <= 0 {
		position = 1
	}

	return 1.0 / math.Pow(float64(position), 0.15)
}

func (rr *SearchRanker) calculateDiversityBonusOptimized(normalizedArtist string, artistCount map[string]int) float64 {
	count := artistCount[normalizedArtist]

	if count <= 1 {
		return 1.0
	}

	return 1.0 / (1.0 + math.Log(float64(count)))
}

func (rr *SearchRanker) calculateRemixPenaltyOptimized(normalizedTitle, normalizedQuery string) float64 {
	heavyPenaltyKeywords := []string{"remix", "bootleg", "mashup", "rework"}
	mediumPenaltyKeywords := []string{"cover", "instrumental", "karaoke"}
	lightPenaltyKeywords := []string{"live", "acoustic", "edit", "version", "extended", "remaster", "dub"}

	// Helper to check if any word in words matches any keyword in keywords
	containsKeyword := func(words []string, keywords []string) bool {
		for _, word := range words {
			if slices.Contains(keywords, word) {
				return true
			}
		}
		return false
	}

	queryWords := strings.Fields(normalizedQuery)
	titleWords := strings.Fields(normalizedTitle)

	// If query explicitly requests variant, don't penalize
	if containsKeyword(queryWords, heavyPenaltyKeywords) ||
		containsKeyword(queryWords, mediumPenaltyKeywords) ||
		containsKeyword(queryWords, lightPenaltyKeywords) ||
		strings.Contains(normalizedQuery, "radio edit") {
		return 1.0
	}

	// Check title for variant keywords
	if containsKeyword(titleWords, heavyPenaltyKeywords) {
		return 0.5
	}
	if containsKeyword(titleWords, mediumPenaltyKeywords) {
		return 0.65
	}
	if containsKeyword(titleWords, lightPenaltyKeywords) || strings.Contains(normalizedTitle, "radio edit") {
		return 0.8
	}

	return 1.0
}

func (rr *SearchRanker) calculateMetadataScore(song domain.Song) float64 {
	if song.Image != "" {
		return 0.05
	}
	return 0.0
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

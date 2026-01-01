package services

import (
	"testing"

	"github.com/andiq123/FindVibeFiber/internal/core/domain"
)

func TestResultRanker_RankResults(t *testing.T) {
	weights := domain.RankingWeights{
		ProviderPriority: 0.3,
		MatchScore:       0.4,
		Position:         0.2,
		Diversity:        0.1,
	}
	ranker := NewResultRanker(weights)

	tests := []struct {
		name        string
		results     []domain.ProviderResult
		query       string
		maxPriority int
		expected    []string
	}{
		{
			name: "Exact match should rank highest",
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Tower OST", "Artist A", "img1", "link1"),
					"Provider1", 0.5, 2,
				),
				*domain.NewProviderResult(
					*domain.NewSong("pizza tower ost", "Artist B", "img2", "link2"),
					"Provider1", 1.0, 1,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Song", "Artist C", "img3", "link3"),
					"Provider1", 0.3, 3,
				),
			},
			query:       "pizza tower ost",
			maxPriority: 8,
			expected:    []string{"pizza tower ost", "Pizza Tower OST", "Pizza Song"},
		},
		{
			name: "Position matters when match scores are similar",
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Song About Pizza", "Artist A", "img1", "link1"),
					"Provider1", 0.7, 5,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Pizza Music", "Artist B", "img2", "link2"),
					"Provider1", 0.7, 1,
				),
			},
			query:       "pizza",
			maxPriority: 8,
			expected:    []string{"Pizza Music", "Song About Pizza"},
		},
		{
			name: "Diversity bonus reduces repetitive artists",
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Song 1", "Same Artist", "img1", "link1"),
					"Provider1", 0.8, 1,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 2", "Same Artist", "img2", "link2"),
					"Provider1", 0.8, 2,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Song 3", "Different Artist", "img3", "link3"),
					"Provider1", 0.75, 3,
				),
			},
			query:       "song",
			maxPriority: 8,
			expected:    []string{"Song 1", "Song 3", "Song 2"},
		},
		{
			name: "Artist match scores highly",
			results: []domain.ProviderResult{
				*domain.NewProviderResult(
					*domain.NewSong("Random Song", "Coldplay", "img1", "link1"),
					"Provider1", 0.9, 1,
				),
				*domain.NewProviderResult(
					*domain.NewSong("Another Track", "The Beatles", "img2", "link2"),
					"Provider1", 0.5, 2,
				),
			},
			query:       "coldplay",
			maxPriority: 8,
			expected:    []string{"Random Song", "Another Track"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scored := ranker.RankResults(tt.results, tt.query, tt.maxPriority)

			if len(scored) != len(tt.expected) {
				t.Errorf("Expected %d results, got %d", len(tt.expected), len(scored))
				return
			}

			for i, expectedTitle := range tt.expected {
				actualTitle := scored[i].Result.Song.Title
				if actualTitle != expectedTitle {
					t.Errorf("Position %d: expected '%s', got '%s' (score: %.3f)",
						i, expectedTitle, actualTitle, scored[i].FinalScore)
				}
			}
		})
	}
}

func TestResultRanker_MatchScore(t *testing.T) {
	weights := domain.RankingWeights{
		ProviderPriority: 0.3,
		MatchScore:       0.4,
		Position:         0.2,
		Diversity:        0.1,
	}
	ranker := NewResultRanker(weights)

	tests := []struct {
		name           string
		song           domain.Song
		query          string
		expectedMin    float64
		expectedMax    float64
		shouldBeHigher bool
	}{
		{
			name:        "Exact title match",
			song:        *domain.NewSong("pizza tower", "artist", "", ""),
			query:       "pizza tower",
			expectedMin: 0.95,
			expectedMax: 1.0,
		},
		{
			name:        "Exact artist match",
			song:        *domain.NewSong("song title", "coldplay", "", ""),
			query:       "coldplay",
			expectedMin: 0.85,
			expectedMax: 0.95,
		},
		{
			name:        "Partial match",
			song:        *domain.NewSong("pizza tower ost", "artist", "", ""),
			query:       "pizza",
			expectedMin: 0.85,
			expectedMax: 1.0,
		},
		{
			name:        "No match",
			song:        *domain.NewSong("completely different", "artist", "", ""),
			query:       "pizza",
			expectedMin: 0.0,
			expectedMax: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ranker.calculateMatchScore(tt.song, tt.query)

			if score < tt.expectedMin || score > tt.expectedMax {
				t.Errorf("Match score %.3f not in expected range [%.3f, %.3f]",
					score, tt.expectedMin, tt.expectedMax)
			}
		})
	}
}

func TestLevenshteinDistance(t *testing.T) {
	tests := []struct {
		s1       string
		s2       string
		expected int
	}{
		{"", "", 0},
		{"hello", "hello", 0},
		{"hello", "helo", 1},
		{"kitten", "sitting", 3},
		{"pizza", "piza", 1},
		{"", "hello", 5},
		{"hello", "", 5},
	}

	for _, tt := range tests {
		t.Run(tt.s1+"_vs_"+tt.s2, func(t *testing.T) {
			result := levenshteinDistance(tt.s1, tt.s2)
			if result != tt.expected {
				t.Errorf("levenshteinDistance(%q, %q) = %d, want %d",
					tt.s1, tt.s2, result, tt.expected)
			}
		})
	}
}

func TestNormalizeString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  Pizza Tower  ", "pizza tower"},
		{"PIZZA TOWER", "pizza tower"},
		{"Pizza   Tower   OST", "pizza tower ost"},
		{"", ""},
		{"  ", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeString(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeString(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

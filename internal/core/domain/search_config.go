package domain

type SearchConfig struct {
	MaxResults     int
	RankingWeights RankingWeights
}

type RankingWeights struct {
	ProviderPriority float64
	MatchScore       float64
	Position         float64
	Diversity        float64
}

func DefaultSearchConfig() *SearchConfig {
	return &SearchConfig{
		MaxResults: 20,
		RankingWeights: RankingWeights{
			ProviderPriority: 0.08,
			MatchScore:       0.82, // query word coverage dominates top match
			Position:         0.05,
			Diversity:        0.05,
		},
	}
}

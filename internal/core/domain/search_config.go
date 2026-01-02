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
			ProviderPriority: 0.3,
			MatchScore:       0.4,
			Position:         0.2,
			Diversity:        0.1,
		},
	}
}

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
			ProviderPriority: 0.10,
			MatchScore:       0.70, // query accuracy first
			Position:         0.10,
			Diversity:        0.10,
		},
	}
}

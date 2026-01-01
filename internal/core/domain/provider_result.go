package domain

type ProviderResult struct {
	Song         Song
	Provider     string
	MatchScore   float64
	ProviderRank int
}

func NewProviderResult(song Song, provider string, matchScore float64, rank int) *ProviderResult {
	return &ProviderResult{
		Song:         song,
		Provider:     provider,
		MatchScore:   matchScore,
		ProviderRank: rank,
	}
}

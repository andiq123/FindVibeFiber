package domain

type ProviderResult struct {
	Song         Song
	Provider     string
	MatchScore   float64
	ProviderRank int
	Pagination   *PaginationInfo
}

type PaginationInfo struct {
	CurrentPage  int
	TotalResults int
	HasNextPage  bool
	HasPrevPage  bool
	TotalPages   int
}

func NewProviderResult(song Song, provider string, matchScore float64, rank int) *ProviderResult {
	return &ProviderResult{
		Song:         song,
		Provider:     provider,
		MatchScore:   matchScore,
		ProviderRank: rank,
	}
}

func NewProviderResultWithPagination(song Song, provider string, matchScore float64, rank int, pagination *PaginationInfo) *ProviderResult {
	return &ProviderResult{
		Song:         song,
		Provider:     provider,
		MatchScore:   matchScore,
		ProviderRank: rank,
		Pagination:   pagination,
	}
}

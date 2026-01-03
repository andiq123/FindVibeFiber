package domain

type ProviderResult struct {
	Song         Song
	Provider     string
	MatchScore   float64
	ProviderRank int
	Pagination   *PaginationInfo
}

type PaginationInfo struct {
	CurrentPage  int  `json:"currentPage"`
	TotalResults int  `json:"totalResults"`
	HasNextPage  bool `json:"hasNextPage"`
	HasPrevPage  bool `json:"hasPrevPage"`
	TotalPages   int  `json:"totalPages"`
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

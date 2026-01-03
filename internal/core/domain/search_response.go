package domain

type SearchResponse struct {
	Songs      []Song          `json:"songs"`
	Pagination *PaginationInfo `json:"pagination,omitempty"`
}

func NewSearchResponse(songs []Song, pagination *PaginationInfo) *SearchResponse {
	return &SearchResponse{
		Songs:      songs,
		Pagination: pagination,
	}
}

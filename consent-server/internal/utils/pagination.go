package utils

// PaginationParams holds pagination parameters
type PaginationParams struct {
	Limit  int
	Offset int
}

// PaginationMetadata holds pagination metadata for responses
type PaginationMetadata struct {
	Total      int  `json:"total"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	HasMore    bool `json:"hasMore"`
	TotalPages int  `json:"totalPages"`
}

// NewPaginationParams creates a new pagination params with defaults
func NewPaginationParams(limit, offset int) *PaginationParams {
	return &PaginationParams{
		Limit:  ValidateLimit(limit),
		Offset: ValidateOffset(offset),
	}
}

// CalculatePaginationMetadata calculates pagination metadata
func CalculatePaginationMetadata(total, limit, offset int) *PaginationMetadata {
	totalPages := (total + limit - 1) / limit
	if totalPages < 0 {
		totalPages = 0
	}

	hasMore := (offset + limit) < total

	return &PaginationMetadata{
		Total:      total,
		Limit:      limit,
		Offset:     offset,
		HasMore:    hasMore,
		TotalPages: totalPages,
	}
}

// GetPageNumber calculates the current page number (1-indexed)
func (p *PaginationParams) GetPageNumber() int {
	if p.Limit == 0 {
		return 1
	}
	return (p.Offset / p.Limit) + 1
}

// GetNextOffset calculates the offset for the next page
func (p *PaginationParams) GetNextOffset() int {
	return p.Offset + p.Limit
}

// GetPreviousOffset calculates the offset for the previous page
func (p *PaginationParams) GetPreviousOffset() int {
	offset := p.Offset - p.Limit
	if offset < 0 {
		return 0
	}
	return offset
}

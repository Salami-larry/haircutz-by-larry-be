package pagination

import (
	"fmt"
	"strconv"
)

const (
	DefaultPage     = 1
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type Params struct {
	Page     int
	PageSize int
}

type Metadata struct {
	TotalItems      int64 `json:"total_items"`
	CurrentItems    int   `json:"current_items"`
	CurrentPage     int   `json:"current_page"`
	LastPage        int   `json:"last_page"`
	NextPage        *int  `json:"next_page"`
	PreviousPage    *int  `json:"previous_page"`
	HasNextPage     bool  `json:"has_next_page"`
	HasPreviousPage bool  `json:"has_previous_page"`
}

type Paginated[T any] struct {
	Items    []T      `json:"items"`
	Metadata Metadata `json:"metadata"`
}

func ParseQuery(pageStr, pageSizeStr string) (Params, error) {
	page := DefaultPage
	pageSize := DefaultPageSize

	if pageStr != "" {
		p, err := strconv.Atoi(pageStr)
		if err != nil || p < 1 {
			return Params{}, fmt.Errorf("page must be a positive integer")
		}
		page = p
	}

	if pageSizeStr != "" {
		ps, err := strconv.Atoi(pageSizeStr)
		if err != nil || ps < 1 || ps > MaxPageSize {
			return Params{}, fmt.Errorf("page_size must be between 1 and %d", MaxPageSize)
		}
		pageSize = ps
	}

	return Params{Page: page, PageSize: pageSize}, nil
}

func (p Params) Skip() int64 {
	return int64((p.Page - 1) * p.PageSize)
}

func (p Params) Limit() int64 {
	return int64(p.PageSize)
}

func BuildMetadata(total int64, page, pageSize, currentCount int) Metadata {
	lastPage := LastPage(total, pageSize)

	hasNext := page < lastPage
	hasPrev := page > 1

	var nextPage, prevPage *int
	if hasNext {
		n := page + 1
		nextPage = &n
	}
	if hasPrev {
		p := page - 1
		prevPage = &p
	}

	return Metadata{
		TotalItems:      total,
		CurrentItems:    currentCount,
		CurrentPage:     page,
		LastPage:        lastPage,
		NextPage:        nextPage,
		PreviousPage:    prevPage,
		HasNextPage:     hasNext,
		HasPreviousPage: hasPrev,
	}
}

func LastPage(total int64, pageSize int) int {
	if pageSize < 1 {
		return 1
	}
	if total <= 0 {
		return 1
	}
	lp := int((total + int64(pageSize) - 1) / int64(pageSize))
	if lp < 1 {
		return 1
	}
	return lp
}

func NewPaginated[T any](items []T, total int64, page, pageSize int) Paginated[T] {
	if items == nil {
		items = []T{}
	}
	return Paginated[T]{
		Items:    items,
		Metadata: BuildMetadata(total, page, pageSize, len(items)),
	}
}

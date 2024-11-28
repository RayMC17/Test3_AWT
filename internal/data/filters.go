package data

import (
	"strings"
	"github.com/RayMC17/bookclub-api/internal/validator" // Update the path according to your module path
)

// Filters contains fields related to pagination and sorting.
type Filters struct {
	Page         int
	PageSize     int
	Sort         string
	SortSafelist []string
}

// ValidateFilters validates the filters used for pagination and sorting.
func ValidateFilters(v *validator.Validator, f *Filters) {
	// Check that the page and PageSize are positive values.
	v.Check(f.Page > 0, "page", "must be greater than zero")
	v.Check(f.PageSize > 0, "page_size", "must be greater than zero")
	v.Check(f.PageSize <= 100, "page_size", "must not be more than 100")

	// Check if the sort value is in the safelist.
	v.Check(validator.In(f.Sort, f.SortSafelist...), "sort", "invalid sort value")
}

// Limit returns the number of records to return based on PageSize.
func (f *Filters) Limit() int {
	return f.PageSize
}

// Offset calculates the number of records to skip based on Page and PageSize.
func (f *Filters) Offset() int {
	return (f.Page - 1) * f.PageSize
}

// SortColumn returns the column name for sorting if it exists in the safelist.
func (f *Filters) SortColumn() string {
	for _, safeValue := range f.SortSafelist {
		if f.Sort == safeValue {
			return strings.TrimPrefix(f.Sort, "-")
		}
	}
	// Default sorting column if none matches
	return "id"
}

// SortDirection returns "ASC" for ascending sort or "DESC" for descending sort.
func (f *Filters) SortDirection() string {
	if strings.HasPrefix(f.Sort, "-") {
		return "DESC"
	}
	return "ASC"
}

// Metadata holds information about pagination.
type Metadata struct {
	CurrentPage  int `json:"current_page"`
	PageSize     int `json:"page_size"`
	TotalRecords int `json:"total_records"`
	TotalPages   int `json:"total_pages"`
}

// CalculateMetadata calculates pagination metadata.
func CalculateMetadata(totalRecords, page, pageSize int) Metadata {
	if totalRecords == 0 {
		return Metadata{}
	}

	return Metadata{
		CurrentPage:  page,
		PageSize:     pageSize,
		TotalRecords: totalRecords,
		TotalPages:   (totalRecords + pageSize - 1) / pageSize,
	}
}

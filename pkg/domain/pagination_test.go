package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testItem struct {
	ID   int
	Name string
}

func TestNewPaginatedResult(t *testing.T) {
	tests := []struct {
		name            string
		page            int
		pageSize        int
		items           []testItem
		totalItems      int64
		expectedPages   int
		expectedHasNext bool
		expectedHasPrev bool
		withFilters     bool // Whether to add filter data to the PageRequest
	}{
		{
			name:            "Empty result",
			page:            1,
			pageSize:        10,
			items:           []testItem{},
			totalItems:      0,
			expectedPages:   0,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:            "Single page of results",
			page:            1,
			pageSize:        3,
			items:           []testItem{{ID: 1, Name: "Item 1"}, {ID: 2, Name: "Item 2"}, {ID: 3, Name: "Item 3"}},
			totalItems:      3,
			expectedPages:   1,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:            "Multiple pages with exact division",
			page:            2,
			pageSize:        2,
			items:           []testItem{{ID: 3, Name: "Item 3"}, {ID: 4, Name: "Item 4"}},
			totalItems:      6, // 6 items with 2 per page = 3 pages exactly
			expectedPages:   3,
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:            "Multiple pages with non-exact division",
			page:            2,
			pageSize:        2,
			items:           []testItem{{ID: 3, Name: "Item 3"}, {ID: 4, Name: "Item 4"}},
			totalItems:      5, // 5 items with 2 per page = 3 pages (2 full + 1 partial)
			expectedPages:   3,
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:            "First page of multi-page results",
			page:            1,
			pageSize:        2,
			items:           []testItem{{ID: 1, Name: "Item 1"}, {ID: 2, Name: "Item 2"}},
			totalItems:      5,
			expectedPages:   3,
			expectedHasNext: true,
			expectedHasPrev: false,
		},
		{
			name:            "Last page of multi-page results",
			page:            3,
			pageSize:        2,
			items:           []testItem{{ID: 5, Name: "Item 5"}},
			totalItems:      5,
			expectedPages:   3,
			expectedHasNext: false,
			expectedHasPrev: true,
		},
		{
			name:            "Middle page of multi-page results",
			page:            2,
			pageSize:        3,
			items:           []testItem{{ID: 4, Name: "Item 4"}, {ID: 5, Name: "Item 5"}, {ID: 6, Name: "Item 6"}},
			totalItems:      10,
			expectedPages:   4, // 10 items with 3 per page = 4 pages (3 full + 1 partial)
			expectedHasNext: true,
			expectedHasPrev: true,
		},
		{
			name:            "Page size greater than total items",
			page:            1,
			pageSize:        20,
			items:           []testItem{{ID: 1, Name: "Item 1"}, {ID: 2, Name: "Item 2"}, {ID: 3, Name: "Item 3"}},
			totalItems:      3,
			expectedPages:   1,
			expectedHasNext: false,
			expectedHasPrev: false,
		},
		{
			name:            "PageRequest with additional filters",
			page:            2,
			pageSize:        3,
			items:           []testItem{{ID: 4, Name: "Item 4"}, {ID: 5, Name: "Item 5"}, {ID: 6, Name: "Item 6"}},
			totalItems:      10,
			expectedPages:   4,
			expectedHasNext: true,
			expectedHasPrev: true,
			withFilters:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create PageRequest with the test case parameters
			pageRequest := &PageReq{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}

			// Add filters and other properties if needed
			if tt.withFilters {
				pageRequest.Filters = map[string][]string{
					"name": {"test"},
					"type": {"A", "B"},
				}
				pageRequest.Sort = true
				pageRequest.SortBy = "name"
				pageRequest.SortAsc = true
			}

			// Call the function under test
			result := NewPaginatedResult(tt.items, tt.totalItems, pageRequest)

			// Assert the results
			assert.NotNil(t, result)
			assert.Equal(t, tt.items, result.Items)
			assert.Equal(t, tt.totalItems, result.TotalItems)
			assert.Equal(t, tt.expectedPages, result.TotalPages)
			assert.Equal(t, tt.page, result.CurrentPage)
			assert.Equal(t, tt.expectedHasNext, result.HasNext)
			assert.Equal(t, tt.expectedHasPrev, result.HasPrev)
		})
	}
}

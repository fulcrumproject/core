package api

import (
	"net/http/httptest"
	"testing"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePageRequest(t *testing.T) {
	tests := []struct {
		name          string
		queryString   string
		expectedPage  int
		expectedSize  int
		expectedError bool
	}{
		{
			name:         "Default Values",
			queryString:  "",
			expectedPage: 1,
			expectedSize: 10,
		},
		{
			name:         "Custom Page and Size",
			queryString:  "?page=2&pageSize=20",
			expectedPage: 2,
			expectedSize: 20,
		},
		{
			name:         "Only Page",
			queryString:  "?page=3",
			expectedPage: 3,
			expectedSize: 10, // Default
		},
		{
			name:         "Only Size",
			queryString:  "?pageSize=30",
			expectedPage: 1, // Default
			expectedSize: 30,
		},
		{
			name:          "Invalid Page",
			queryString:   "?page=invalid",
			expectedError: true,
		},
		{
			name:          "Invalid Size",
			queryString:   "?pageSize=invalid",
			expectedError: true,
		},
		{
			name:          "Negative Page",
			queryString:   "?page=-1",
			expectedError: true,
		},
		{
			name:          "Zero Page",
			queryString:   "?page=0",
			expectedError: true,
		},
		{
			name:          "Negative Size",
			queryString:   "?pageSize=-10",
			expectedError: true,
		},
		{
			name:          "Zero Size",
			queryString:   "?pageSize=0",
			expectedError: true,
		},
		{
			name:          "Size Too Large",
			queryString:   "?pageSize=1001", // Max is 100 in the implementation
			expectedError: true,             // Now returns error for oversized values
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test"+tc.queryString, nil)
			pageReq, err := ParsePageRequest(req)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedPage, pageReq.Page)
				assert.Equal(t, tc.expectedSize, pageReq.PageSize)
			}
		})
	}
}

func TestNewPageResponse(t *testing.T) {
	// Use a simple struct for testing
	type TestItem struct {
		ID int
	}

	// Create test items
	item1 := TestItem{ID: 1}
	item2 := TestItem{ID: 2}
	item3 := TestItem{ID: 3}

	// Test conversion function - must take a pointer and return a pointer
	convertFn := func(item *TestItem) *int {
		val := item.ID * 10
		return &val
	}

	tests := []struct {
		name          string
		items         []TestItem
		totalItems    int64
		page          int
		pageSize      int
		expectedItems []int
	}{
		{
			name:          "Standard Page",
			items:         []TestItem{item1, item2, item3},
			totalItems:    10,
			page:          1,
			pageSize:      3,
			expectedItems: []int{10, 20, 30},
		},
		{
			name:          "Empty Items",
			items:         []TestItem{},
			totalItems:    0,
			page:          1,
			pageSize:      10,
			expectedItems: []int{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a domain page response
			domainResp := &domain.PageResponse[TestItem]{
				Items:       tc.items,
				TotalItems:  tc.totalItems,
				CurrentPage: tc.page,
				TotalPages:  (int(tc.totalItems) + tc.pageSize - 1) / tc.pageSize, // Ceiling division
				HasNext:     tc.page < ((int(tc.totalItems) + tc.pageSize - 1) / tc.pageSize),
				HasPrev:     tc.page > 1,
			}

			// Convert to API response
			apiResp := NewPageResponse(domainResp, convertFn)

			// Verify the response structure
			assert.Equal(t, len(tc.expectedItems), len(apiResp.Items))
			for i, item := range apiResp.Items {
				assert.Equal(t, tc.expectedItems[i], *item)
			}
			assert.Equal(t, tc.totalItems, apiResp.TotalItems)
			assert.Equal(t, tc.page, apiResp.CurrentPage)
		})
	}
}

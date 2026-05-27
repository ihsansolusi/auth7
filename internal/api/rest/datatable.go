package rest

// DataTableRequest is the standard cursor-pagination request body for admin list endpoints.
type DataTableRequest struct {
	ReqType    string         `json:"req_type"`
	PageSize   int            `json:"page_size"`
	TopData    map[string]any `json:"top_data"`
	BottomData map[string]any `json:"bottom_data"`
	SearchText *string        `json:"search_text"`
	SortColumn *struct {
		Name string `json:"name"`
		Dir  string `json:"dir"`
	} `json:"sort_column"`
}

// DataTableResponse is the standard cursor-pagination response for admin list endpoints.
type DataTableResponse struct {
	Data      any  `json:"data"`
	AllowNext bool `json:"allow_next"`
	AllowPrev bool `json:"allow_prev"`
}

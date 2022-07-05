package repository

type CommonListOptions struct {
	Page   int64  `json:"page,omitempty"`
	Size   int64  `json:"size,omitempty"`
	Search string `json:"search,omitempty"`

	// sort string, eg: "-name,-creationtime", "name,-creationtime"
	// the '-' prefix means descending,otherwise ascending
	Sort string `json:"sort,omitempty"`
}

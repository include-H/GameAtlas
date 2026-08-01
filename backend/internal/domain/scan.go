package domain

type ScanResult struct {
	Total     int          `json:"total"`
	Created   int          `json:"created"`
	Skipped   int          `json:"skipped"`
	Errors    int          `json:"errors"`
	Details   []ScanDetail `json:"details,omitempty"`
}

type ScanDetail struct {
	Title  string `json:"title"`
	Status string `json:"status"` // "created", "skipped", "error"
	Reason string `json:"reason,omitempty"`
}

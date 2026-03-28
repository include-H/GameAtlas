package domain

type DirectoryItem struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	IsDirectory bool   `json:"is_directory"`
	SizeBytes   *int64 `json:"size_bytes"`
}

type DirectoryListResponse struct {
	CurrentPath string          `json:"current_path"`
	ParentPath  *string         `json:"parent_path"`
	Items       []DirectoryItem `json:"items"`
}

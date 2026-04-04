package domain

type DirectoryItem struct {
	Name        string
	Path        string
	IsDirectory bool
	SizeBytes   *int64
}

type DirectoryListResponse struct {
	CurrentPath string
	ParentPath  *string
	Items       []DirectoryItem
}

package domain

// MetadataType enumerates the supported metadata entity kinds.
// Defined in the domain layer so that service and handler code can reference
// these constants without importing the repositories package.
type MetadataType int

const (
	MetadataDevelopers MetadataType = iota
	MetadataPublishers
	MetadataSeries
)

package domain

import "errors"

// Sentinel errors shared across layers.
// English messages are for logs and errors.Is matching only.
// User-visible messages are translated to Chinese by writeServiceError in handlers.
var (
	ErrNotFound        = errors.New("resource not found")
	ErrValidation      = errors.New("validation error")
	ErrUpstream        = errors.New("upstream request failed")
	ErrAuthDisabled    = errors.New("authentication is not configured")
	ErrForbiddenPath   = errors.New("file path is outside primary ROM root")
	ErrMissingFile     = errors.New("registered file is unavailable")
	ErrInvalidFile     = errors.New("registered path is not a file")
	ErrMissingConfig   = errors.New("PRIMARY_ROM_ROOT is not configured")
	ErrInvalidLaunchFile = errors.New("launch script only supports VHD or VHDX files")
	ErrMissingSMBConfig  = errors.New("SMB launch script configuration is incomplete")
)

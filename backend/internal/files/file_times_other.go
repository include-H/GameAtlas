//go:build !linux

package files

func ReadSourceCreatedAt(path string) (*string, error) {
	return nil, nil
}

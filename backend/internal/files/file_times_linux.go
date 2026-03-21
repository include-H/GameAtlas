//go:build linux

package files

import (
	"time"

	"golang.org/x/sys/unix"
)

func ReadSourceCreatedAt(path string) (*string, error) {
	var stat unix.Statx_t
	if err := unix.Statx(unix.AT_FDCWD, path, unix.AT_STATX_SYNC_AS_STAT, unix.STATX_BTIME, &stat); err != nil {
		return nil, err
	}

	if stat.Mask&unix.STATX_BTIME == 0 {
		return nil, nil
	}

	if stat.Btime.Sec == 0 && stat.Btime.Nsec == 0 {
		return nil, nil
	}

	value := time.Unix(int64(stat.Btime.Sec), int64(stat.Btime.Nsec)).UTC().Format(time.RFC3339)
	return &value, nil
}

//go:build !windows

package checkpoint

import "os"

func restorePathIsLinkLike(_ string, info os.FileInfo) (bool, error) {
	return info.Mode()&os.ModeSymlink != 0, nil
}

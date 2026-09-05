//go:build windows

package checkpoint

import (
	"os"

	"golang.org/x/sys/windows"
)

func restorePathIsLinkLike(path string, info os.FileInfo) (bool, error) {
	if info.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	pathPtr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return false, err
	}
	attrs, err := windows.GetFileAttributes(pathPtr)
	if err != nil {
		return false, err
	}
	return attrs&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0, nil
}

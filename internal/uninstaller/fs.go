package uninstaller

import (
	"os"

	"github.com/spf13/afero"
)

// dirSizeKB returns the on-disk size of a path in KiB via afero,
// so staging logic is testable against a mock filesystem.
func dirSizeKB(fs afero.Fs, path string) (int64, error) {
	info, err := fs.Stat(path)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return (info.Size() + 1023) / 1024, nil
	}
	var total int64
	walkErr := afero.Walk(fs, path, func(_ string, fi os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !fi.IsDir() {
			total += (fi.Size() + 1023) / 1024
		}
		return nil
	})
	if walkErr != nil {
		return 0, walkErr
	}
	return total, nil
}

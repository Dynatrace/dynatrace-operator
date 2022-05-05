package csigc

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Dynatrace/dynatrace-operator/src/installer/image"
	"github.com/spf13/afero"
)

var maxImageAge = time.Hour * 24 * 14 // 14 days

func (gc *CSIGarbageCollector) runImageGarbageCollection() {
	imageDirs, _ := getImageCacheDirs(gc.fs)
	if imageDirs == nil {
		return
	}

	imageCachesToDelete := collectUnusedImageCaches(gc.fs, imageDirs)

	deleteImageCaches(gc.fs, imageCachesToDelete)
}

func getImageCacheDirs(fs afero.Fs) ([]os.FileInfo, error) {
	imageDirs, err := afero.Afero{Fs: fs}.ReadDir(image.CacheDir)
	if os.IsNotExist(err) {
		log.Info("No image cache to clean up")
		return nil, nil
	}
	if err != nil {
		log.Error(err, "Failed to read image cache directory")
		return nil, err
	}
	return imageDirs, err
}

func collectUnusedImageCaches(fs afero.Fs, imageDirs []os.FileInfo) []string {
	var toDelete []string
	for _, imageDir := range imageDirs {
		if !imageDir.IsDir() {
			continue
		}
		modificationTime, err := getRelevantModificationTime(fs, imageDir.Name())
		if err != nil {
			log.Error(err, "failed to get modification time of image cache", "imageDir", imageDir.Name())
			continue
		}
		if time.Since(modificationTime) > maxImageAge {
			toDelete = append(toDelete, filepath.Join(image.CacheDir, imageDir.Name()))
		}
	}
	return toDelete
}

// getRelevantModificationTime returns the last modification time of an image in the cache
// an image is not a single file but a directory of several files.
// Most of which don't change when the image is tried to be pulled again.
// The index.json of the image is a file that is modified in reliable way.(after every pull)
func getRelevantModificationTime(fs afero.Fs, imageDir string) (time.Time, error) {
	var modificationTime time.Time
	indexPath := filepath.Join(image.CacheDir, imageDir, "index.json")
	indexFileInfo, err := fs.Stat(indexPath)
	if err != nil {
		return modificationTime, err
	}
	modificationTime = indexFileInfo.ModTime()
	return modificationTime, err
}

func deleteImageCaches(fs afero.Fs, imageCaches []string) {
	for _, dir := range imageCaches {
		log.Info("Deleting image cache", "dir", dir)
		err := fs.RemoveAll(dir)
		if err != nil {
			log.Error(err, "Failed to delete image cache", "dir", dir)
		}
	}
}

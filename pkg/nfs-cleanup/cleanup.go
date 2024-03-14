package nfscleanup

import (
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog/log"
)

func ListFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.Error().Msgf("Error accessing path %s: %v", path, err)
			return err
		}
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Error().Msgf("Error walking the path %s: %v", dir, err)
		return nil, err
	}
	log.Info().Msgf("Found %d files in %s", len(files), dir)
	return files, nil
}

func FindFileAge (path string) (fileAge time.Duration , err error) {
	file, err := os.Stat(path) 

	if err != nil {	
		log.Error().Msgf("Error accessing path %s: %v", path, err)	
		return time.Duration(0), err
	}

	age := time.Since(file.ModTime())
return age, err
}
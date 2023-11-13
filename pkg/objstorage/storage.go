package objstorage

import (
	"cloud.google.com/go/storage"
	"context"
	"github.com/rs/zerolog/log"
	"io"
	"os"
	"path/filepath"
)

func CopyToBucket(gcsClient *storage.Client, localSrcFile *os.File, bucketName string, objBasePath string) error {
	srcFilename, err := FilenameWithoutPath(localSrcFile)
	if err != nil {
		return err
	}
	ctx := context.Background()
	bucket := gcsClient.Bucket(bucketName)
	objName := filepath.Join(objBasePath, srcFilename)
	log.Info().Msgf("copying '%s' to '%s' in bucket '%s'", srcFilename, objName, bucketName)
	obj := bucket.Object(objName)
	objWriter := obj.NewWriter(ctx)
	written, err := io.Copy(objWriter, localSrcFile)
	if err != nil {
		return err
	}
	log.Info().Msgf("wrote %d bytes to '%s'", written, objName)
	return objWriter.Close()
}

func FilenameWithoutPath(f *os.File) (string, error) {
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	return info.Name(), nil
}

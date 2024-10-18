package storage

import (
	"os"
	"path/filepath"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func CreateDirectoryIfNotExists(path string) {
	if !Exists(path) {
		os.Mkdir(path, 0755)
	}
}

func SaveFileToS3Bucket(filePath string) (string, error) {
	mockedS3Location := "./S3/"
	CreateDirectoryIfNotExists(mockedS3Location)

	destination := mockedS3Location + filepath.Base(filePath)

	err := os.Rename(filePath, destination)
	if err != nil {
		return "", err
	}

	return destination, nil
}

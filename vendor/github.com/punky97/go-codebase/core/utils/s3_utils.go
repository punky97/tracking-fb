package utils

import (
	"fmt"
	s3cloudstorage "github.com/punky97/go-codebase/core/bkstorage/s3cloud"
	"github.com/punky97/go-codebase/core/logger"
	"io"
	"regexp"

	"github.com/spf13/viper"
)

var r = regexp.MustCompile("[^a-zA-Z0-9_\\.]+")

func UploadS3(io io.Reader, fileName string, path string, contentType string) (string, error) {
	accessKey := viper.GetString("amazon_s3.access_key")
	secretAccessKey := viper.GetString("amazon_s3.secret_access_key")
	region := viper.GetString("amazon_s3.region")
	bucket := viper.GetString("amazon_s3.bucket")
	token := viper.GetString("amazon_s3.token")
	filePath := viper.GetString("amazon_s3.files_path")

	fileName = r.ReplaceAllString(fileName, "")

	// Create an instance S3
	svc, err := s3cloudstorage.NewS3CloudStorage(accessKey, secretAccessKey, token, region)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Err when create an instance, detail: %v", err))
		return "", err
	}

	// File uploading
	err = svc.Upload(io, path+fileName, bucket, contentType)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when upload file, detail: %v", err))
		return "", err
	}

	return filePath + path + fileName, nil
}

//UploadS3WithOptionals -- Upload file to s3 with input optionals
/*
	For example:
	Add Content-Disposition Header by using optional WithContentDisposition(`attachment; filename="yourfilename"`)
	You can add multiple optionals as well.
	Some optionals are supported:
	WithPublicRead()
	WithPrivateAccess()
	WithBucketOwnerAccess()
	WithBucketOwnerRead()
	WithAuthenticatedRead()
	WithContentDisposition(contentDisposition string)
*/
func UploadS3WithOptionals(io io.Reader, fileName, path string, contentType string, options ...func(interface{})) (string, error) {
	accessKey := viper.GetString("amazon_s3.access_key")
	secretAccessKey := viper.GetString("amazon_s3.secret_access_key")
	region := viper.GetString("amazon_s3.region")
	bucket := viper.GetString("amazon_s3.bucket")
	token := viper.GetString("amazon_s3.token")
	filePath := viper.GetString("amazon_s3.files_path")

	fileName = r.ReplaceAllString(fileName, "")

	// Create an instance S3
	svc, err := s3cloudstorage.NewS3CloudStorage(accessKey, secretAccessKey, token, region)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Err when create an instance, detail: %v", err))
		return "", err
	}

	// File uploading
	err = svc.UploadWithOptions(io, path+fileName, bucket, contentType, options...)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when upload file, detail: %v", err))
		return "", err
	}

	return filePath + path + fileName, nil
}

func UploadPrivateS3(io io.Reader, fileName string, path string, contentType string) (string, error) {
	accessKey := viper.GetString("amazon_s3.access_key")
	secretAccessKey := viper.GetString("amazon_s3.secret_access_key")
	region := viper.GetString("amazon_s3.region")
	bucket := viper.GetString("amazon_s3.bucket")
	token := viper.GetString("amazon_s3.token")

	fileName = r.ReplaceAllString(fileName, "")

	// Create an instance S3
	svc, err := s3cloudstorage.NewS3CloudStorage(accessKey, secretAccessKey, token, region)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Err when create an instance, detail: %v", err))
		return "", err
	}

	// File uploading
	err = svc.UploadWithOptions(io, path+fileName, bucket, contentType, s3cloudstorage.WithPrivateAccess())
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when upload file, detail: %v", err))
		return "", err
	}

	return path + fileName, nil
}

func ReadS3(fileName string) (string, error) {
	accessKey := viper.GetString("amazon_s3.access_key")
	secretAccessKey := viper.GetString("amazon_s3.secret_access_key")
	region := viper.GetString("amazon_s3.region")
	bucket := viper.GetString("amazon_s3.bucket")
	token := viper.GetString("amazon_s3.token")

	// Create an instance S3
	svc, err := s3cloudstorage.NewS3CloudStorage(accessKey, secretAccessKey, token, region)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Err when create an instance, detail: %v", err))
		return "", err
	}

	// File downloading
	data, err := svc.Read(fileName, bucket)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when read file, detail: %v", err))
		return "", err
	}

	return data, nil
}

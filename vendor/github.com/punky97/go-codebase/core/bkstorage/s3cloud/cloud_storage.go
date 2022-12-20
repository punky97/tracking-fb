package s3cloudstorage

import (
	"io"
)

// CloudStorage --
type CloudStorage interface {
	ListBuckets()
	ListBucketItems(bucket string, prefix string)
	Upload(file io.Reader, key string, bucket string, contentType string) error
	UploadWithOptions(file io.Reader, key string, bucket string, contentType string, options ...func(interface{})) error
	Delete(bucket string, key string) error
}

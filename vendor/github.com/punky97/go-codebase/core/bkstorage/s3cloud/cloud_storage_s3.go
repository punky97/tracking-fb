package s3cloudstorage

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/punky97/go-codebase/core/logger"
	"io"
	"os"
	"time"

	"github.com/spf13/cast"
)

const DefaultLargeDownloaderOptionSizeInMB int64 = 256 // 256
var EmptyIterator = errors.New("iterator is already empty")

// S3CloudStorage --
type S3CloudStorage struct {
	Sess *session.Session
	S3   *s3.S3
}

type S3CloudStorageInterface interface {
	Upload(file io.Reader, filename string, bucket string, contentType string) error
	UploadWithMetadata(file io.Reader, filename string, bucket string, contentType string, metadata map[string]interface{}) error
	Read(filename, bucket string) (string, error)
	GetListBucketItems(bucket string, prefix string) ([]*s3.Object, error)
	Delete(bucket string, filepath string) error
	BatchDelete(bucket string, prefix string) error
}

// NewS3CloudStorage --
func NewS3CloudStorage(awsAccessKeyID, awsSecretAccessKey, awsSessionToken, awsRegion string) (*S3CloudStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, awsSessionToken),
	})

	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	storage := &S3CloudStorage{
		Sess: sess,
		S3:   svc,
	}

	return storage, nil
}

// NewS3CloudStorageWithEndpoint
func NewS3CloudStorageWithEndpoint(awsAccessKeyID, awsSecretAccessKey, awsSessionToken, awsRegion, awsEndpoint string) (*S3CloudStorage, error) {
	sess, err := session.NewSession(&aws.Config{
		Region:      aws.String(awsRegion),
		Endpoint:    aws.String(awsEndpoint),
		Credentials: credentials.NewStaticCredentials(awsAccessKeyID, awsSecretAccessKey, awsSessionToken),
	})

	if err != nil {
		return nil, err
	}

	svc := s3.New(sess)

	storage := &S3CloudStorage{
		Sess: sess,
		S3:   svc,
	}

	return storage, nil
}

func (s *S3CloudStorage) Read(key, bucket string) (string, error) {
	buff := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloader(s.Sess)
	_, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return "", err
	}

	return string(buff.Bytes()), nil
}

// ListBuckets --
func (s *S3CloudStorage) ListBuckets() {
	result, err := s.S3.ListBuckets(nil)

	if err != nil {
		fmt.Println(err)
	}

	fmt.Println("Buckets:")

	for _, b := range result.Buckets {
		fmt.Printf("* %s created on %s\n",
			aws.StringValue(b.Name), aws.TimeValue(b.CreationDate))
	}
}

// ListBucketItems --
func (s *S3CloudStorage) ListBucketItems(bucket string, prefix string) {
	resp, err := s.S3.ListObjects(&s3.ListObjectsInput{Bucket: aws.String(bucket), Prefix: aws.String(prefix)})

	if err != nil {
		logger.BkLog.Error(err)
		return
	}

	for _, item := range resp.Contents {
		fmt.Println("Name:         ", *item.Key)
		fmt.Println("Last modified:", *item.LastModified)
		fmt.Println("Size:         ", *item.Size)
		fmt.Println("Storage class:", *item.StorageClass)
		fmt.Println("")
	}
}

// ListBucketItems --
func (s *S3CloudStorage) GetListBucketItems(bucket string, prefix string) ([]*s3.Object, error) {
	var results []*s3.Object

	resp, err := s.S3.ListObjects(&s3.ListObjectsInput{Bucket: aws.String(bucket), Prefix: aws.String(prefix)})

	if err != nil {
		return results, err
	}

	return resp.Contents, nil
}

// Upload --
func (s *S3CloudStorage) Upload(file io.Reader, key string, bucket string, contentType string) error {
	uploader := s3manager.NewUploader(s.Sess)
	lastModified := time.Now().String()
	_, err := uploader.Upload(&s3manager.UploadInput{
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		Metadata: map[string]*string{
			"Last-Modified": &lastModified,
		},
	})

	if err != nil {
		return err
	}

	return nil
}

// Upload --
func (s *S3CloudStorage) UploadWithMetadata(file io.Reader, key string, bucket string, contentType string, metadata map[string]interface{}) error {
	uploader := s3manager.NewUploader(s.Sess)
	lastModified := time.Now().String()
	input := &s3manager.UploadInput{
		ACL:         aws.String(s3.ObjectCannedACLPublicRead),
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		Metadata: map[string]*string{
			"Last-Modified": &lastModified,
		},
	}

	for k, v := range metadata {
		input.Metadata[k] = aws.String(cast.ToString(v))
	}

	_, err := uploader.Upload(input)

	if err != nil {
		return err
	}

	return nil
}

// Delete --
func (s *S3CloudStorage) Delete(bucket string, key string) error {
	_, err := s.S3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	err = s.S3.WaitUntilObjectNotExists(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})

	if err != nil {
		return err
	}

	return nil
}

// BatchDelete -- batch delete by prefix - for example: delete by all keys start batch_001/ (keys: batch_001/test_01.txt, batch_001/test_02.txt) will be deleted. Delimiter (/) must be included
func (s *S3CloudStorage) BatchDelete(bucket string, prefix string) error {
	srv := s3.New(s.Sess)
	iter := s3manager.NewDeleteListIterator(srv, &s3.ListObjectsInput{
		Bucket: aws.String(bucket),
		Prefix: aws.String(prefix),
	})

	batcher := s3manager.NewBatchDeleteWithClient(srv)
	return batcher.Delete(aws.BackgroundContext(), iter)
}

// UploadWithOptions
func (s *S3CloudStorage) UploadWithOptions(file io.Reader, key string, bucket string, contentType string, options ...func(interface{})) error {
	lastModified := time.Now().String()
	defaultInput := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
		Metadata: map[string]*string{
			"Last-Modified": &lastModified,
		},
	}

	for _, opt := range options {
		opt(defaultInput)
	}

	uploader := s3manager.NewUploader(s.Sess)
	_, err := uploader.Upload(defaultInput)

	if err != nil {
		return err
	}

	return nil
}

// UploadFileWithOption --
func (s *S3CloudStorage) UploadFileWithOption(options s3manager.UploadInput) error {
	uploader := s3manager.NewUploader(s.Sess)
	_, err := uploader.Upload(&options)
	if err != nil {
		return err
	}

	return nil
}

// DownloadFile --
func (s *S3CloudStorage) DownloadFile(bucket string, key string) (*aws.WriteAtBuffer, error) {
	data := &aws.WriteAtBuffer{}
	downloader := s3manager.NewDownloader(s.Sess)
	_, err := downloader.Download(
		data,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)
	return data, err
}

// DownloadAndWriteStreamToFile --
func (s *S3CloudStorage) DownloadAndWriteStreamToFile(bucket string, key string, f *os.File) error {

	downloader := s3manager.NewDownloader(s.Sess, func(d *s3manager.Downloader) { d.PartSize = DefaultLargeDownloaderOptionSizeInMB * 1024 * 1024 })

	_, err := downloader.Download(
		f,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(key),
		},
	)
	return err
}

// MoveFileInsideBucket --
func (s *S3CloudStorage) MoveFileInsideBucket(bucket string, fromFilePath string, toFilePath string) error {
	fromFilePathBucket := fmt.Sprintf("%v/%v", bucket, fromFilePath)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(fromFilePathBucket),
		Key:        aws.String(toFilePath),
	}

	_, err := s.S3.CopyObject(input)
	if err != nil {
		logger.BkLog.Infof("Cannot copy object from %v to %v: %v", fromFilePath, toFilePath, err)
		return err
	}
	err = s.Delete(bucket, fromFilePath)
	if err != nil {
		return err
	}
	return nil
}

// GetObject
func (s *S3CloudStorage) GetObject(bucket string, key string) (*s3.GetObjectOutput, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	return s.S3.GetObject(input)
}

// CopyObject
func (s *S3CloudStorage) CopyObject(bucket string, fromFilePath string, toFilePath string, options ...func(interface{})) error {
	fromFilePathBucket := fmt.Sprintf("%v/%v", bucket, fromFilePath)
	input := &s3.CopyObjectInput{
		Bucket:     aws.String(bucket),
		CopySource: aws.String(fromFilePathBucket),
		Key:        aws.String(toFilePath),
	}

	for _, opt := range options {
		opt(input)
	}

	_, err := s.S3.CopyObject(input)
	if err != nil {
		logger.BkLog.Infof("Cannot copy object from %v to %v: %v", fromFilePath, toFilePath, err)
		return err
	}

	return nil
}

// CheckObjectExist
func (s *S3CloudStorage) CheckObjectExist(bucket string, key string) (*s3.HeadObjectOutput, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	return s.S3.HeadObject(input)
}

func CopyWithPublicRead() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3.CopyObjectInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLPublicRead)
		}
	}
}

func WithPublicRead() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLPublicRead)
		}
	}
}

func WithPrivateAccess() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLPrivate)
		}
	}
}

func WithBucketOwnerAccess() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLBucketOwnerFullControl)
		}
	}
}

func WithBucketOwnerRead() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLBucketOwnerRead)
		}
	}
}

func WithAuthenticatedRead() func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ACL = aws.String(s3.ObjectCannedACLAuthenticatedRead)
		}
	}
}

func WithContentDisposition(contentDisposition string) func(interface{}) {
	return func(opt interface{}) {
		if s3Opt, ok := opt.(*s3manager.UploadInput); ok {
			s3Opt.ContentDisposition = aws.String(contentDisposition)
		}
	}
}

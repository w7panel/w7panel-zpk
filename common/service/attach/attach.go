package attach

import (
	"crypto/md5"
	"fmt"
	"log"
	"mime/multipart"
	"os"
	"time"
)

const (
	STORAGE_TYPE_LOCAL = "local"
	STORAGE_TYPE_S3    = "s3"
)

var (
	clientMapping map[string]StorageClient
)

type StorageInput struct {
	Type      string
	Path      string
	SecretId  string
	SecretKey string
	Endpoint  string
	Region    string
	Bucket    string
	Debug     bool
}

type PresignUrl struct {
	Url      string
	Method   string
	UploadId string
}

type StorageClient interface {
	UploadByFile(remoteName string, file *os.File) error
	UploadByContent(remoteName string, content string) error
	UploadByFilePath(remoteName string, localFilePath string) error
	UploadByUploadFile(remoteName string, file multipart.File) error
	PresignUrl(remoteName string) (*PresignUrl, error)
	PresignUrlMultipart(remoteName string, uploadId string, partNumber int32) (*PresignUrl, error)
	MultipartCreateUploadId(remoteName string) (string, error)
	MultipartComplete(uploadId string) (string, error)
	GetString(remoteName string) (string, error)
	GetFile(remoteName string) (*os.File, error)
	GetSaveFile(remoteName string, localFilePath string) error
	Delete(remoteName string) error
	GetUrl(remoteName string) string
	GetPrivateUrl(remoteName string, expired time.Duration) (string, error)
}

func init() {
	clientMapping = make(map[string]StorageClient)
}

func NewStorageClient(input *StorageInput) StorageClient {
	var driver StorageClient
	if input.Type == STORAGE_TYPE_S3 {
		if input.SecretKey == "" || input.SecretId == "" || input.Region == "" || input.Endpoint == "" {
			log.Printf("Invalid storage s3 params \n")
			return nil
		}
		key := fmt.Sprintf("%x", md5.Sum([]byte(input.SecretId)))
		if _, ok := clientMapping[key]; ok {
			driver = clientMapping[key]
		} else {
			clientMapping[key] = newS3Client(input)
			driver = clientMapping[key]
		}
	} else if input.Type == STORAGE_TYPE_LOCAL {
		if input.Path == "" {
			log.Printf("Invalid storage local params \n")
			return nil
		}
		key := fmt.Sprintf("%x", md5.Sum([]byte(input.Path)))
		if _, ok := clientMapping[key]; ok {
			driver = clientMapping[key]
		} else {
			clientMapping[key] = newLocalClient(input)
			driver = clientMapping[key]
		}
	} else {
		panic("This storage type is not supported")
	}
	return driver
}

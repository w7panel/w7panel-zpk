package attach

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
	"github.com/w7panel/w7panel-zpk/common/function"
	"github.com/w7panel/w7panel-zpk/common/service"
)

func newLocalClient(input *StorageInput) *localClient {
	myLocalClient := &localClient{
		basePath:  strings.TrimRight(input.Path, "/") + "/",
		endpoint:  input.Endpoint,
		secretId:  input.SecretId,
		secretKey: input.SecretKey,
	}
	myLocalClient.uploadIdMapping = make(map[string]*uploadIdNode)
	return myLocalClient
}

type localClient struct {
	basePath        string
	uploadIdMapping map[string]*uploadIdNode
	endpoint        string
	secretId        string
	secretKey       string
}

func (self *localClient) UploadByFile(remoteName string, file *os.File) error {
	destFile, err := self.getFileHandler(remoteName)
	if err != nil {
		return err
	}
	defer destFile.Close()
	_, err = io.Copy(destFile, file)
	if err != nil {
		return err
	}
	return nil
}

func (self *localClient) UploadByContent(remoteName string, content string) error {
	err := os.MkdirAll(filepath.Dir(self.parseRemoteName(remoteName)), service.FileMode)

	if err != nil {
		return err
	}
	return os.WriteFile(self.parseRemoteName(remoteName), []byte(content), service.FileMode)
}

func (self *localClient) UploadByFilePath(remoteName string, localFilePath string) error {
	file, err := os.OpenFile(localFilePath, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	return self.UploadByFile(remoteName, file)
}

func (self *localClient) UploadByUploadFile(remoteName string, file multipart.File) error {
	destFile, err := self.getFileHandler(remoteName)
	defer destFile.Close()
	if err != nil {
		return err
	}
	_, err = io.Copy(destFile, file)
	if err != nil {
		return err
	}
	return nil
}

func (self *localClient) PresignUrl(remoteName string) (*PresignUrl, error) {
	path := fmt.Sprintf("%s%s", self.basePath, remoteName)
	os.MkdirAll(filepath.Dir(path), service.FileMode)

	return &PresignUrl{
		Url:    path,
		Method: "LOCAL",
	}, nil
}

func (self *localClient) PresignUrlMultipart(remoteName string, uploadId string, partNumber int32) (*PresignUrl, error) {
	if _, ok := self.uploadIdMapping[uploadId]; !ok {
		return nil, errors.New("The upload id not found or expired. Please create it first.")
	}

	path := fmt.Sprintf("%s/%s.%d.part", *self.uploadIdMapping[uploadId].uploadId, uploadId, partNumber)
	os.MkdirAll(filepath.Dir(path), service.FileMode)

	return &PresignUrl{
		Url:    path,
		Method: "LOCAL",
	}, nil
}

func (self *localClient) MultipartCreateUploadId(remoteName string) (string, error) {
	key := uuid.New().String()
	if _, ok := self.uploadIdMapping[key]; !ok {
		tempDir, _ := os.MkdirTemp("", "storage_part")
		self.uploadIdMapping[key] = &uploadIdNode{
			uploadId:  ptr.String(tempDir),
			key:       ptr.String(remoteName),
			abortDate: aws.Time(time.Now().Add(time.Hour)),
		}
	}
	return key, nil
}

func (self *localClient) MultipartComplete(uploadId string) (string, error) {
	if uploadId == "" || self.uploadIdMapping[uploadId] == nil {
		return "nil", errors.New("The upload id not found or expired. Please create it first.")
	}
	targetFile, err := self.getFileHandler(*self.uploadIdMapping[uploadId].key)
	if err != nil {
		return "", err
	}
	defer targetFile.Close()
	defer os.RemoveAll(*self.uploadIdMapping[uploadId].uploadId)
	defer delete(self.uploadIdMapping, uploadId)

	log.Printf("clear: %s", *self.uploadIdMapping[uploadId].uploadId)
	for i := 1; i <= 10000; i++ {
		chunk := fmt.Sprintf("%s/%s.%d.part", *self.uploadIdMapping[uploadId].uploadId, uploadId, i)
		content, err := os.ReadFile(chunk)
		if err != nil {
			if os.IsNotExist(err) {
				return targetFile.Name(), nil
			}
			return "", err
		}
		targetFile.Write(content)
	}
	return targetFile.Name(), nil
}

func (self *localClient) GetString(remoteName string) (string, error) {
	content, err := os.ReadFile(self.parseRemoteName(remoteName))
	if err != nil {
		return "", err
	}
	return string(content), nil
}

func (self *localClient) GetFile(remoteName string) (*os.File, error) {
	_, err := os.Stat(self.parseRemoteName(remoteName))
	if os.IsNotExist(err) {
		return nil, errors.New("file not found")
	}
	return os.OpenFile(self.parseRemoteName(remoteName), os.O_RDWR, 0666)
}

func (self *localClient) GetSaveFile(remoteName string, localFilePath string) error {
	localFile, err := os.OpenFile(localFilePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return err
	}
	sourceFile, err := os.OpenFile(self.parseRemoteName(remoteName), os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(localFile, sourceFile)
	return err
}

func (self *localClient) Delete(remoteName string) error {
	return os.Remove(self.parseRemoteName(remoteName))
}

func (self *localClient) GetUrl(remoteName string) string {
	return fmt.Sprintf("%s/%s", self.endpoint, strings.TrimLeft(remoteName, "/"))
}

func (self *localClient) GetPrivateUrl(remoteName string, expired time.Duration) (string, error) {
	//signer := v4.NewSigner(func(options *v4.SignerOptions) {
	//	options.Logger = logging.NewStandardLogger(os.Stdout)
	//})
	//fmt.Printf("%v \n", self.GetUrl(remoteName))
	//
	//request, _ := http.NewRequest("GET", self.GetUrl(remoteName), nil)
	//url, _, _ := signer.PresignHTTP(context.Background(), aws.Credentials{
	//	AccessKeyID:     self.secretId,
	//	SecretAccessKey: self.secretKey,
	//}, request, "", "", "", time.Now())
	return self.GetUrl(remoteName), nil
}

func (self *localClient) parseRemoteName(remoteName string) string {
	return self.basePath + strings.TrimLeft(function.EncodeURIComponent(remoteName, []byte{'/'}), "/")
}

func (self *localClient) getFileHandler(remoteName string) (*os.File, error) {
	filePath := self.parseRemoteName(remoteName)
	fmt.Printf("%v \n", filepath.Dir(filePath))
	err := os.MkdirAll(filepath.Dir(filePath), service.FileMode)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(self.parseRemoteName(remoteName), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return nil, err
	}
	return file, nil
}

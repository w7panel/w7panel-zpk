package attach

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go/logging"
	"github.com/aws/smithy-go/ptr"
	"github.com/w7panel/w7panel-zpk/common/function"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strings"
	"time"
)

type s3Client struct {
	client            *s3.Client
	presignClient     *s3.PresignClient
	bucket            *string
	url               string
	multipartUploadId map[string]*uploadIdNode
}

type uploadIdNode struct {
	uploadId  *string
	abortDate *time.Time
	key       *string
}

type cred struct {
	secretId  string
	secretKey string
}

func (self cred) Retrieve(ctx context.Context) (aws.Credentials, error) {
	return aws.Credentials{
		AccessKeyID:     self.secretId,
		SecretAccessKey: self.secretKey,
	}, nil
}

func newS3Client(params *StorageInput) *s3Client {
	myS3Client := &s3Client{}
	if params.Bucket != "" {
		myS3Client.url = fmt.Sprintf("%s/%s", params.Endpoint, params.Bucket)
	} else {
		myS3Client.url = fmt.Sprintf("%s", params.Endpoint)
	}
	awsConfig := aws.Config{
		Region: params.Region,
		Credentials: cred{
			secretId:  params.SecretId,
			secretKey: params.SecretKey,
		},
	}
	if params.Debug {
		awsConfig.Logger = logging.NewStandardLogger(os.Stdout)
		awsConfig.ClientLogMode = aws.LogRetries | aws.LogRequest | aws.LogResponse
	}
	myS3Client.client = s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.BaseEndpoint = aws.String(params.Endpoint)
		if params.Bucket != "" {
			options.UsePathStyle = true
		}
	})
	myS3Client.bucket = ptr.String(params.Bucket)
	myS3Client.multipartUploadId = make(map[string]*uploadIdNode)
	myS3Client.presignClient = s3.NewPresignClient(myS3Client.client, func(options *s3.PresignOptions) {
		options.Expires = time.Hour
	})
	go myS3Client.recycleUploadId()
	return myS3Client
}

func (self *s3Client) PresignUrl(remoteName string) (*PresignUrl, error) {
	url, err := self.presignClient.PresignPutObject(context.Background(), &s3.PutObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
	})
	if err != nil {
		return nil, err
	}
	return &PresignUrl{
		Url:    url.URL,
		Method: url.Method,
	}, err
}

func (self *s3Client) PresignUrlMultipart(remoteName string, uploadId string, partNumber int32) (*PresignUrl, error) {
	remoteName = self.parseRemoteName(remoteName)
	ctx := context.Background()

	if uploadId == "" || self.multipartUploadId[uploadId] == nil {
		return nil, errors.New("The upload id not found or expired. Please create it first.")
	}

	url, err := self.presignClient.PresignUploadPart(ctx, &s3.UploadPartInput{
		Bucket:     self.bucket,
		Key:        self.multipartUploadId[uploadId].key,
		PartNumber: ptr.Int32(partNumber),
		UploadId:   self.multipartUploadId[uploadId].uploadId,
	})
	if err != nil {
		return nil, err
	}
	if remoteName != *self.multipartUploadId[uploadId].key {
		return nil, errors.New("filename and multipart filename are different")
	}

	return &PresignUrl{
		Url:      url.URL,
		Method:   url.Method,
		UploadId: uploadId,
	}, err
}

func (self *s3Client) MultipartCreateUploadId(remoteName string) (string, error) {
	multipart, err := self.client.CreateMultipartUpload(context.Background(), &s3.CreateMultipartUploadInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
	})
	if err != nil {
		return "", err
	}
	uploadId := *multipart.UploadId
	self.multipartUploadId[uploadId] = &uploadIdNode{
		uploadId:  multipart.UploadId,
		abortDate: aws.Time(time.Now().Add(time.Hour)),
		key:       multipart.Key,
	}
	return *multipart.UploadId, nil
}

func (self *s3Client) MultipartComplete(uploadId string) (string, error) {
	if uploadId == "" || self.multipartUploadId[uploadId] == nil {
		return "nil", errors.New("The upload id not found or expired. Please create it first.")
	}
	ctx := context.Background()
	partList, err := self.client.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   self.bucket,
		Key:      self.multipartUploadId[uploadId].key,
		UploadId: self.multipartUploadId[uploadId].uploadId,
	})
	if partList.Parts == nil {
		return "", errors.New("The part data is empty, please upload it first")
	}
	var completePartList []types.CompletedPart
	for _, part := range partList.Parts {
		completePartList = append(completePartList, types.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}
	result, err := self.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   self.bucket,
		Key:      self.multipartUploadId[uploadId].key,
		UploadId: self.multipartUploadId[uploadId].uploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completePartList,
		},
	})
	if err != nil {
		return "", err
	}
	return *result.Location, nil
}

func (self *s3Client) MultipartList(remoteName string) (result []*uploadIdNode, err error) {
	ctx := context.Background()
	partList, err := self.client.ListMultipartUploads(ctx, &s3.ListMultipartUploadsInput{
		Bucket: self.bucket,
		Prefix: aws.String(self.parseRemoteName(remoteName)),
	})
	if err != nil || partList.Uploads == nil {
		return nil, err
	}
	for _, upload := range partList.Uploads {
		result = append(result, &uploadIdNode{
			uploadId: upload.UploadId,
			key:      upload.Key,
		})
	}
	return result, nil
}

func (self *s3Client) UploadByFile(remoteName string, file *os.File) error {
	_, err := self.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
		Body:   file,
	})
	if err != nil {
		return err
	}
	return nil
}

func (self *s3Client) UploadByContent(remoteName string, content string) error {
	reader := strings.NewReader(content)
	_, err := self.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
		Body:   reader,
	})
	if err != nil {
		return err
	}
	return nil
}

func (self *s3Client) UploadByFilePath(remoteName string, localFilePath string) error {
	temp, _ := os.OpenFile(localFilePath, os.O_RDONLY, 755)
	return self.UploadByFile(remoteName, temp)
}

func (self *s3Client) UploadByUploadFile(remoteName string, file multipart.File) error {
	_, err := self.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
		Body:   file,
	})
	if err != nil {
		return err
	}
	return nil
}

func (self *s3Client) GetString(remoteName string) (string, error) {
	response, err := self.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
	})
	if err != nil {
		return "", err
	}
	content, _ := io.ReadAll(response.Body)
	defer response.Body.Close()
	return string(content), nil
}

func (self *s3Client) GetFile(remoteName string) (*os.File, error) {
	localFile, err := os.CreateTemp("", *self.bucket)
	if err != nil {
		return nil, err
	}
	defer localFile.Close()
	err = self.GetSaveFile(remoteName, localFile.Name())
	temp, err := os.OpenFile(localFile.Name(), os.O_RDONLY, 755)
	return temp, err
}

func (self *s3Client) GetSaveFile(remoteName string, localFilePath string) error {
	localFile, err := os.OpenFile(localFilePath, os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	response, err := self.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: self.bucket,
		Key:    aws.String(self.parseRemoteName(remoteName)),
	})
	if err != nil {
		return err
	}
	_, err = io.Copy(localFile, response.Body)
	response.Body.Close()
	localFile.Close()
	return nil
}

func (self *s3Client) Delete(remoteName string) error {
	_, err := self.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Key:    aws.String(self.parseRemoteName(remoteName)),
		Bucket: self.bucket,
	})
	return err
}

func (self *s3Client) GetUrl(remoteName string) string {
	return fmt.Sprintf("%s/%s", self.url, self.parseRemoteName(remoteName))
}

func (self *s3Client) GetPrivateUrl(remoteName string, expired time.Duration) (string, error) {
	url, err := self.presignClient.PresignGetObject(context.Background(), &s3.GetObjectInput{
		Bucket:          self.bucket,
		Key:             aws.String(self.parseRemoteName(remoteName)),
		ResponseExpires: aws.Time(time.Now().Add(expired)),
	})
	if err != nil {
		return "", err
	}
	return url.URL, nil
}

func (self *s3Client) parseRemoteName(remoteName string) string {
	return strings.TrimLeft(function.EncodeURIComponent(remoteName, []byte{'/'}), "/")
}

func (self *s3Client) recycleUploadId() {
	for true {
		// 检测回收站连接
		for key, value := range self.multipartUploadId {
			if time.Now().Unix() > value.abortDate.Unix() {
				delete(self.multipartUploadId, key)
				_, err := self.client.AbortMultipartUpload(context.Background(), &s3.AbortMultipartUploadInput{
					UploadId: value.uploadId,
					Key:      value.key,
					Bucket:   self.bucket,
				})
				if err != nil {
					log.Printf("清理分片上传：%s, %s", key, err.Error())
				} else {
					log.Printf("清理分片上传：%s", key)
				}
			}
		}
		time.Sleep(time.Second)
	}
}

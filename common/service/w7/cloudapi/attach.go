package cloudapi

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/go-resty/resty/v2"
	"github.com/w7corp/sdk-open-cloud-go/service"
	"github.com/w7panel/w7panel-zpk/common/service/w7/base"
)

type Attach struct {
	HttpClient *resty.Client
	BaseUrl    string
}

type Ticket struct {
	JsTicket   string `json:"js_ticket"`
	ExpireTime uint   `json:"expire_time"`
	Host       string `json:"host"`
}

type AttachInfo struct {
	Md5  string `json:"md5"`
	Path string `json:"path"`
}

type UploadAttachResp struct {
	Attach AttachInfo `json:"attach"`
}

func (s Attach) GetJsTicketByHost(host string) (*Ticket, *service.ErrApiResult) {
	apiResult := &Ticket{}
	errResult := &service.ErrApiResult{}

	_, err := s.HttpClient.R().
		SetFormData(map[string]string{
			"host": host,
		}).
		SetResult(apiResult).
		SetError(errResult).
		Post("/oauth/access-token/js-ticket-api-v1")

	if err != nil {
		return nil, service.NewErrApiResult(err)
	}
	if errResult.IsError() {
		return nil, errResult
	}

	apiResult.Host = host

	return apiResult, service.NewErrApiResult(nil)
}

func (s Attach) GetAttachDownloadUrl(attachMd5 string) (string, *service.ErrApiResult) {
	type AttachResp struct {
		Url string `json:"data"`
	}
	apiResult := &AttachResp{}
	errResult := &service.ErrApiResult{}

	_, err := s.HttpClient.R().
		SetFormData(map[string]string{
			"md5": attachMd5,
		}).
		SetHeader("x-requested-with", "XMLHttpRequest").
		SetResult(apiResult).
		SetError(errResult).
		Post("/attach/zip/download")

	if err != nil {
		return "", service.NewErrApiResult(err)
	}
	if errResult.IsError() {
		return "", errResult
	}

	return apiResult.Url, service.NewErrApiResult(nil)
}

func (s Attach) GetAttachContent(attachMd5 string, path string) (string, *service.ErrApiResult) {
	type ContentResp struct {
		Content  string `json:"content"`
		FileName string `json:"file_name"`
	}
	contentResult := &ContentResp{}
	errResult := &service.ErrApiResult{}

	resp, err := s.HttpClient.R().
		SetFormData(map[string]string{
			"md5":       attachMd5,
			"file_name": path,
		}).
		SetError(errResult).
		Post("/attach/zip/get-content")

	if err != nil {
		return "", service.NewErrApiResult(err)
	}
	if errResult.IsError() {
		return "", errResult
	}

	err = json.Unmarshal(resp.Body(), &contentResult)
	if err != nil {
		return "", service.NewErrApiResult(err)
	}

	return contentResult.Content, service.NewErrApiResult(nil)
}

func (s Attach) UploadImg(ticket *Ticket, img *os.File, filename string) (*UploadAttachResp, error) {
	mime, err := mimetype.DetectReader(img)
	if err != nil {
		return nil, err
	}
	// 重置文件指针到文件开头
	_, err = img.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	body := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(body)

	quoteEscaper := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			quoteEscaper.Replace("file"), quoteEscaper.Replace(filename)))
	h.Set("Content-Type", mime.String())
	// 添加文件到表单
	fileWriter, err := multipartWriter.CreatePart(h)
	if err != nil {
		return nil, fmt.Errorf("创建表单文件失败: %v", err)
	}
	if _, err := io.Copy(fileWriter, img); err != nil {
		return nil, fmt.Errorf("复制文件内容失败: %v", err)
	}
	field, err := multipartWriter.CreateFormField("js_ticket")
	if err != nil {
		return nil, err
	}
	_, err = field.Write([]byte(ticket.JsTicket))
	if err != nil {
		return nil, err
	}
	field, err = multipartWriter.CreateFormField("host")
	if err != nil {
		return nil, err
	}
	_, err = field.Write([]byte(ticket.Host))
	if err != nil {
		return nil, err
	}
	field, err = multipartWriter.CreateFormField("filename")
	if err != nil {
		return nil, err
	}
	_, err = field.Write([]byte(filename))
	if err != nil {
		return nil, err
	}
	multipartWriter.Close()

	// 创建 HTTP 请求
	request, err := http.NewRequestWithContext(context.Background(), "POST", s.BaseUrl+"/attach/upload/image", body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	if base.DefaultUserAgent != "" {
		request.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	// 设置 Content-Type
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// 发送请求
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取服务器响应失败: %v", err)
	}
	statusCode := response.StatusCode
	if statusCode != 200 && statusCode != 201 {
		var apiError base.ApiError
		err = json.Unmarshal(responseBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var uploadResp *UploadAttachResp
	err = json.Unmarshal(responseBody, &uploadResp)
	if err != nil {
		return nil, err
	}

	return uploadResp, nil
}

func (s Attach) UploadZip(ticket *Ticket, file *os.File, filename string) (*UploadAttachResp, error) {
	// 计算文件MD5
	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return nil, err
	}
	fileMd5 := hex.EncodeToString(hash.Sum(nil))

	// 重置文件指针
	_, err := file.Seek(0, io.SeekStart)
	if err != nil {
		return nil, err
	}

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// 分块配置
	chunkSize := int64(1050306)
	totalChunks := int64(math.Ceil(float64(fileSize) / float64(chunkSize)))

	// 基础参数
	baseParams := map[string][]byte{
		"filename":    []byte(filename),
		"totalChunks": []byte(strconv.Itoa(int(totalChunks))),
		"md5":         []byte(fileMd5),
		"type":        []byte("2"),
		"js_ticket":   []byte(ticket.JsTicket),
		"host":        []byte(ticket.Host),
	}

	// 分块上传
	for partNumber := int64(1); partNumber <= totalChunks; partNumber++ {
		// 准备当前块的buffer
		currentChunkSize := chunkSize
		if partNumber == totalChunks {
			currentChunkSize = fileSize - (chunkSize * (totalChunks - 1))
		}
		buffer := make([]byte, currentChunkSize)

		// 读取数据块
		bytesRead, err := io.ReadFull(file, buffer)
		if err != nil && err != io.ErrUnexpectedEOF {
			return nil, err
		}

		// 设置分块参数
		chunkParams := cloneMap(baseParams)
		chunkParams["file"] = buffer[:bytesRead]
		chunkParams["chunkNumber"] = []byte(strconv.Itoa(int(partNumber)))

		// 上传分块
		result, err := s.uploadAttach(chunkParams, filename)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}

	// 发送完成请求
	finishParams := cloneMap(baseParams)
	finishParams["finish"] = []byte("1")
	return s.uploadAttach(finishParams, filename)
}

func (s Attach) uploadAttach(params map[string][]byte, filename string) (*UploadAttachResp, error) {
	body := &bytes.Buffer{}
	multipartWriter := multipart.NewWriter(body)

	file, exists := params["file"]
	if exists {
		quoteEscaper := strings.NewReplacer("\\", "\\\\", `"`, "\\\"")
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				quoteEscaper.Replace("file"), quoteEscaper.Replace(filename)))
		// 添加文件到表单
		fileWriter, err := multipartWriter.CreatePart(h)
		if err != nil {
			return nil, fmt.Errorf("创建表单文件失败: %v", err)
		}
		_, err = fileWriter.Write(file)
		if err != nil {
			return nil, err
		}
		delete(params, "file")
	}

	// 添加文件到表单
	for key, val := range params {
		field, err := multipartWriter.CreateFormField(key)
		if err != nil {
			return nil, err
		}
		_, err = field.Write(val)
		if err != nil {
			return nil, err
		}
	}
	multipartWriter.Close()

	// 创建 HTTP 请求
	request, err := http.NewRequestWithContext(context.Background(), "POST", s.BaseUrl+"/attach/upload/chunk", body)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %v", err)
	}
	if base.DefaultUserAgent != "" {
		request.Header.Set("User-Agent", base.DefaultUserAgent)
	}
	// 设置 Content-Type
	request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

	// 发送请求
	client := &http.Client{
		Timeout: 60 * time.Second,
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("读取服务器响应失败: %v", err)
	}
	statusCode := response.StatusCode
	if statusCode != 200 && statusCode != 201 {
		var apiError base.ApiError
		err = json.Unmarshal(responseBody, &apiError)
		if err != nil {
			return nil, err
		}
		return nil, apiError
	}

	var uploadResp *UploadAttachResp
	err = json.Unmarshal(responseBody, &uploadResp)
	if err != nil {
		return nil, err
	}
	if uploadResp.Attach.Md5 == "" {
		return nil, nil
	}

	return uploadResp, nil
}

func cloneMap(original map[string][]byte) map[string][]byte {
	clone := make(map[string][]byte)
	for k, v := range original {
		clone[k] = v
	}
	return clone
}

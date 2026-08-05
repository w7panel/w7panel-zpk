package controller

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	path2 "path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/core/err_handler"
)

var uploadTempDir = map[string]string{}

type Attach struct {
	Abstract
}

func (c Attach) Upload(ctx *gin.Context) {
	type ParamsValidate struct {
		Filename    string `form:"filename" binding:"required,endswith=.zip|endswith=.pdf|endswith=.tgz"`
		TotalChunks int32  `form:"totalChunks" binding:"required,number,gt=0"`
		Finish      int    `form:"finish" binding:"omitempty,eq=1|eq=0"`
		ChunkNumber int32  `form:"chunkNumber" binding:"required_if=Finish 0,omitempty,number,gt=0"`
		UploadId    string `form:"upload_id" binding:"omitempty"`
		FileMd5     string `form:"md5" binding:"required"`
	}

	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	storageLocalClient := logic.GetLocalClient()

	pathInfo := function.GetPathInfo(params.Filename)
	saveFileName := fmt.Sprintf("/Storage/%s/%s%s", time.Now().Format("200601"), params.FileMd5, pathInfo.Extension)

	if params.Finish != 1 {
		if params.ChunkNumber < 1 || params.ChunkNumber > params.TotalChunks {
			c.JsonResponseWithServerError(ctx, errors.New("分卷数据错误"))
			return
		}
		_, fileHeader, err := ctx.Request.FormFile("file")
		if err != nil {
			c.JsonResponseWithServerError(ctx, errors.New("请上传文件"))
			return
		}

		// 此处应该按正常流程，先获取UploadId再上传
		// 为了兼容现有程序，如果没有传 UploadId 时，按 remoteName 给生成一个
		if params.UploadId == "" && uploadTempDir[params.Filename] == "" {
			uploadTempDir[params.Filename], _ = storageLocalClient.MultipartCreateUploadId(saveFileName)
		}

		uploadPartSavePath, _ := storageLocalClient.PresignUrlMultipart(params.Filename, uploadTempDir[params.Filename], params.ChunkNumber)
		ctx.SaveUploadedFile(fileHeader, uploadPartSavePath.Url)
		c.JsonSuccessResponse(ctx)
		return
	} else {
		if params.TotalChunks == 1 {
			_, fileHeader, err := ctx.Request.FormFile("file")
			if err_handler.Found(err) {
				c.JsonResponseWithServerError(ctx, errors.New("请上传文件"))
				return
			}
			uploadSavePath, _ := logic.GetLocalClient().PresignUrl(saveFileName)
			ctx.SaveUploadedFile(fileHeader, uploadSavePath.Url)

			saveFileName = "file://" + saveFileName

			c.JsonResponseWithoutError(ctx, gin.H{
				"url": saveFileName,
			})
			return
		} else {
			_, err := storageLocalClient.MultipartComplete(uploadTempDir[params.Filename])
			defer delete(uploadTempDir, params.Filename)
			if err_handler.Found(err) {
				c.JsonResponseWithServerError(ctx, err)
				return
			}

			saveFileName = "file://" + saveFileName
		}
		c.JsonResponseWithoutError(ctx, gin.H{
			"url": saveFileName,
		})
		return
	}
}

func (c Attach) Download(ctx *gin.Context) {
	type ParamsValidate struct {
		Token string `uri:"id" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	if val, ok := depot.DownloadMapping.Load(params.Token); ok {
		depot.DownloadMapping.Delete(params.Token)
		path := val.(string)
		file, err := logic.GetLocalClient().GetFile(path)
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		defer file.Close()

		fileInfo, _ := file.Stat()
		ctx.Header("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))
		ctx.Header("Content-Type", "application/zip")
		ctx.Header("Content-Disposition", "attachment; filename="+params.Token+path2.Ext(path))

		ctx.File(file.Name())

		return
	}

	c.JsonResponseWithServerError(ctx, errors.New("请先获取仓库信息"))
}

func (c Attach) GetBackendZipFileList(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Version   string `form:"version" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	formula, err := depot.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("请先添加仓库"))
		return
	}

	list, err := depot.GetBackendZipFileList(formula)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"list": list,
	})
}

func (c Attach) GetBackendZipFileContent(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Path      string `form:"path" binding:"required"`
		Version   string `form:"version" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	formula, err := depot.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("请先添加仓库"))
		return
	}
	content, err := depot.GetBackendZipFileContent(formula, params.Path)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"content": string(content),
	})
}

func (c Attach) GetFrontendZipFileContent(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `uri:"identifie" binding:"required"`
		Version   string `uri:"version" binding:"required"`
		Path      string `uri:"path" binding:"required"`
		Ticket    string `form:"ticket"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	formula, err := depot.GetFormula(params.Identifie, params.Version, nil)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	if formula.GoodsId > 0 && formula.InstallServiceFee > 0 {
		ticketInfo, err := logic.Ticket{}.ParseTicket(params.Ticket)
		slog.Info("ticket解析", "ticket", params.Ticket, "info", ticketInfo, "err", err)
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		if ticketInfo.FormulaId != formula.ID {
			c.JsonResponseWithServerError(ctx, errors.New("ticket错误"))
			return
		}
	}

	content, err := depot.GetFrontendZipFileContent(formula, params.Path)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	ext := filepath.Ext(params.Path)
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	ctx.Data(http.StatusOK, mimeType, content)
}

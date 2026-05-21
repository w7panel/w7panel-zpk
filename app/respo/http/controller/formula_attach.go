package controller

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
	"github.com/w7panel/w7panel-zpk/common/dao"
	"github.com/w7panel/w7panel-zpk/common/entity"
	"github.com/w7panel/w7panel-zpk/common/function"
	logic2 "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"sigs.k8s.io/yaml"
)

type FormulaAttach struct {
	Abstract
}

var formulaAttachDangerousBlockTagPattern = regexp.MustCompile(`(?is)<\s*(script|iframe|object|embed|applet|style|form)\b[^>]*>.*?<\s*/\s*(script|iframe|object|embed|applet|style|form)\s*>`)
var formulaAttachDangerousSingleTagPattern = regexp.MustCompile(`(?is)<\s*(script|iframe|object|embed|applet|meta|link|style|base|form)\b[^>]*?/?>`)
var formulaAttachAnchorHrefPattern = regexp.MustCompile(`(?i)(<\s*a\b[^>]*?)\s+href\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
var formulaAttachEventAttrPattern = regexp.MustCompile(`(?i)\s+on[a-z0-9_-]+\s*=\s*(".*?"|'.*?'|[^\s>]+)`)
var formulaAttachJSProtocolPattern = regexp.MustCompile(`(?i)(href|src|xlink:href)\s*=\s*(['"]?)\s*(javascript:|vbscript:|data:text/html)`)

func sanitizeFormulaAttachContent(content string) string {
	content = formulaAttachDangerousBlockTagPattern.ReplaceAllString(content, "")
	content = formulaAttachDangerousSingleTagPattern.ReplaceAllString(content, "")
	content = formulaAttachAnchorHrefPattern.ReplaceAllString(content, `$1`)
	content = formulaAttachEventAttrPattern.ReplaceAllString(content, "")
	content = formulaAttachJSProtocolPattern.ReplaceAllString(content, `${1}=$2#blocked:`)
	return content
}

func (c FormulaAttach) SaveFile(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Filename  string `form:"filename" binding:"required"`
		Content   string `form:"content" binding:"omitempty"`
		Version   string `form:"version"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	formula, err := depot.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	if params.Filename == "manifest.yaml" {
		manifest := &logic.Manifest{}
		err = yaml.Unmarshal([]byte(params.Content), manifest)
		if err != nil {
			c.JsonResponseWithError(ctx, errors.New("manifest 解析失败"+err.Error()), 500)
			return
		}
		if manifest.Source.Url != "" && !strings.HasSuffix(manifest.Source.Url, ".git") && !strings.HasSuffix(manifest.Source.Url, ".zip") {
			c.JsonResponseWithError(ctx, errors.New("zip_url 必须以 .git 或是 .zip 结尾"), 500)
			return
		}
		if manifest.Application.Identifie == "" {
			manifest.Application.Identifie = manifest.Platform.BaseInfo.Identifie
		}
		if manifest.Application.Identifie != formula.Name {
			c.JsonResponseWithError(ctx, errors.New("manifest 中标识与仓库不一致"), 500)
			return
		}

		if formula.Manifest.Source.Url != manifest.Source.Url {
			if strings.HasPrefix(formula.Manifest.Source.Url, "file://") {
				file, err := logic.GetLocalClient().GetFile(formula.ZipPath)
				if err == nil {
					os.Remove(file.Name())
				}
			}
		}

		_, err = dao.Q.Formula.Where(dao.Formula.ID.Eq(formula.ID)).Updates(entity.Formula{
			Title: manifest.Application.Name,
		})
		if err != nil {
			c.JsonResponseWithError(ctx, errors.New("更新 title 失败"), 500)
			return
		}
	} else {
		params.Content = sanitizeFormulaAttachContent(params.Content)
	}

	err = depot.SaveFile(formula, params.Filename, params.Content)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c FormulaAttach) Files(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
		Version   string `form:"version"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	depot := c.getDepot()
	formula, err := depot.GetFormula(params.Identifie, params.Version, logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	fileList, err := depot.GetFileList(formula)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	delete(fileList, "manifest.yaml")

	returnFileList := map[string]string{}
	for index, val := range fileList {
		if strings.HasSuffix(index, "manifest.yaml") {
			tmpManifest := &logic.Manifest{}
			err = yaml.Unmarshal([]byte(val), tmpManifest)
			if err != nil {
				continue
			}
			tmpManifestV2 := logic.ProcessManifestIdentify(*tmpManifest)
			tmpManifestV2 = logic.GetManifestV2(tmpManifestV2)
			tmpContent, _ := yaml.Marshal(tmpManifestV2)
			responseManifestMap := map[string]interface{}{}
			_ = yaml.Unmarshal(tmpContent, &responseManifestMap)
			if platform, ok := responseManifestMap["platform"].(map[string]interface{}); ok {
				delete(platform, "container")
			}
			tmpContent, _ = yaml.Marshal(responseManifestMap)
			returnFileList[strings.ReplaceAll(index, "_", "-")] = string(tmpContent)
		}
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"list": returnFileList,
	})
}

func (c FormulaAttach) EditIcon(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `form:"identifie" binding:"required"`
	}

	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	uploadFile, fileHeader, err := ctx.Request.FormFile("file")
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("请上传文件"), 500)
		return
	}
	if fileHeader.Size > 1024*1024 {
		c.JsonResponseWithError(ctx, errors.New("图标大小不能超过1m"), 500)
		return
	}

	depotLogin := c.getDepot()
	formula, err := depotLogin.GetFormula(params.Identifie, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, errors.New("请先添加仓库"), 500)
		return
	}
	iconPath := formula.GetIconRelativePath()
	content, _ := io.ReadAll(uploadFile)
	localStore := logic.GetLocalClient()

	err = localStore.UploadByContent(iconPath, string(content))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	url, err := localStore.GetPrivateUrl(iconPath, time.Hour*24)
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	domain := facade.GetConfig().GetString("setting.depot.external_domain")
	c.JsonResponseWithoutError(ctx, gin.H{
		"url": fmt.Sprintf("https://%s%s", domain, "/zpk"+url),
	})
}

func (c FormulaAttach) GetIcon(ctx *gin.Context) {
	remoteName := ctx.Param("path")
	remoteName = strings.ReplaceAll(remoteName, ".icon.jpg", "")
	formula, err := c.getDepot().GetFormula(remoteName, "", logic2.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}
	if formula.RemoteFormulaInfoUrl != "" {
		iconSavePath := filepath.Join(c.getDepot().GetBasePath(), formula.GetIconRelativePath())
		function.CreateDirIfNotExist(filepath.Dir(iconSavePath), os.ModePerm)
		err = w7.ZpkSdk.DownloadRemoteFormulaIcon(formula.RemoteFormulaInfoUrl, iconSavePath)
		if err != nil {
			slog.Error("download remote formula icon error", "formula", formula.Name, "err", err)
		}
	}

	iconFile, err := logic.GetLocalClient().GetFile(formula.GetIconRelativePath())
	if err == nil {
		ctx.Header("Content-Type", "application/png")
		ctx.File(iconFile.Name())
	} else {
		ctx.Header("Content-Type", "application/png")
		ctx.File(facade.GetConfig().GetString("setting.depot.default_icon_path"))
	}
}

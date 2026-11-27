package controller

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/entity"
	commonlogic "github.com/w7panel/w7panel-zpk/common/logic"
	"github.com/w7panel/w7panel-zpk/common/service/w7"
	permissionsdk "github.com/w7panel/w7panel-zpk/common/service/w7/permission"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"net/http"
	"strings"
)

func checkFormulaPullPermission(ctx *gin.Context, name string) bool {
	repository := commonlogic.GetFormulaOciName(name)
	// Only a source ZPK login establishes a local user; target panel IDs are never used.
	user, _ := ctx.Get("user")
	localUser, _ := user.(*entity.RegistryUser)
	if localUser != nil && (commonlogic.User{}).IsAdminUser(localUser) {
		return true
	}
	userID := int32(0)
	if localUser != nil {
		userID = localUser.ID
	}
	token := ""
	if raw := ctx.GetHeader("Authorization"); strings.HasPrefix(raw, "Bearer ") {
		token = strings.TrimPrefix(raw, "Bearer ")
	}
	allowed, err := w7.PermissionSdk.Check(ctx.Request.Context(), userID, permissionsdk.Resource{Type: "repository", Value: repository}, "pull", token)
	if err != nil {
		ctx.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"code": 503, "error": err.Error()})
		return false
	}
	if allowed {
		return true
	}
	domain := facade.GetConfig().GetString("setting.depot.external_domain")
	challenge := fmt.Sprintf(`Bearer realm="https://%s/zpk/v2/registry/auth",service="docker-registry",scope="repository:%s:pull"`, domain, repository)
	ctx.Header("WWW-Authenticate", challenge)
	ctx.Header("Cache-Control", "no-store")
	ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "error": "需要源仓库 pull 权限", "errors": []gin.H{{"code": "UNAUTHORIZED", "message": "authentication required", "detail": []gin.H{{"Type": "repository", "Name": repository, "Action": "pull"}}}}})
	return false
}

type FormulaPermission struct{ Abstract }

func (c FormulaPermission) Get(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie string `json:"identifie" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	formula, err := c.getDepot().GetFormula(params.Identifie, "", commonlogic.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}
	name := commonlogic.GetFormulaOciName(formula.Name)
	repository, err := w7.PermissionSdk.GetResource(ctx.Request.Context(), permissionsdk.Resource{Type: "repository", Value: name})
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	visible := repository.VisibleType
	if visible == 3 {
		_, namespace := commonlogic.ParseRepositoryNameAndNamespace(name)
		space, err := w7.PermissionSdk.GetNamespace(ctx.Request.Context(), namespace)
		if err != nil {
			c.JsonResponseWithServerError(ctx, err)
			return
		}
		visible = space.VisibleType
	}
	if visible != 1 {
		visible = 4
	}
	ctx.Header("Cache-Control", "no-store")
	c.JsonResponseWithoutError(ctx, gin.H{"visible_type": visible})
}
func (c FormulaPermission) Set(ctx *gin.Context) {
	type ParamsValidate struct {
		Identifie   string `json:"identifie" binding:"required"`
		VisibleType int32  `json:"visible_type" binding:"required,oneof=1 4"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}
	formula, err := c.getDepot().GetFormula(params.Identifie, "", commonlogic.User{}.GetUser(ctx))
	if err != nil {
		c.JsonResponseWithServerError(ctx, errors.New("制品不存在"))
		return
	}
	if err := w7.PermissionSdk.SetResource(ctx.Request.Context(), permissionsdk.Resource{Type: "repository", Value: commonlogic.GetFormulaOciName(formula.Name)}, formula.UserId, params.VisibleType); err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	c.JsonSuccessResponse(ctx)
}

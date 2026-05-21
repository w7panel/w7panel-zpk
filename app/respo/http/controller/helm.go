package controller

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/respo/logic"
)

type Helm struct {
	Abstract
}

func (c Helm) GetHelmRepositoryCharts(ctx *gin.Context) {
	type ParamsValidate struct {
		RepositoryUrl string `form:"repository_url" json:"repository_url" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	index, err := logic.HelmRepository{}.GetRepositoryEntityIndex(params.RepositoryUrl)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}

	charts := make([]string, len(index.Entries))
	entryIndex := 0
	for name, _ := range index.Entries {
		charts[entryIndex] = name
		entryIndex++
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{"charts": charts})
}

func (c Helm) GetHelmRepositoryChartVersions(ctx *gin.Context) {
	type ParamsValidate struct {
		RepositoryUrl string `form:"repository_url" json:"repository_url" binding:"required"`
		Chart         string `form:"chart" json:"chart" binding:"required"`
	}
	params := ParamsValidate{}
	if !c.Validate(ctx, &params) {
		return
	}

	index, err := logic.HelmRepository{}.GetRepositoryEntityIndex(params.RepositoryUrl)
	if err != nil {
		c.JsonResponseWithServerError(ctx, err)
		return
	}
	entity, exists := index.Entries[params.Chart]
	if !exists {
		c.JsonResponseWithServerError(ctx, errors.New("chart not found"))
		return
	}

	versions := make([]string, entity.Len())
	for i, item := range entity {
		versions[i] = item.Metadata.Version
	}

	c.JsonResponseWithoutError(ctx, map[string]interface{}{"versions": versions})
}

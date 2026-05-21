package controller

import (
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/system/logic"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type RegistryStorage struct {
	controller.Abstract
}

type RegistryStorageUpdateParams struct {
	S3RegionEndpoint string `form:"s3_endpoint" json:"s3_endpoint" binding:"required"`
	S3AccessKey      string `form:"s3_access_key" json:"s3_access_key" binding:"required"`
	S3SecretKey      string `form:"s3_secret_key" json:"s3_secret_key" binding:"required"`
	S3Bucket         string `form:"s3_bucket" json:"s3_bucket" binding:"required"`
	S3Region         string `form:"s3_region" json:"s3_region" binding:"required"`
	S3RootDirectory  string `form:"s3_root_directory" json:"s3_root_directory"`
	S3Secure         bool   `json:"s3_secure"`
}

func (c RegistryStorage) GetConfig(ctx *gin.Context) {
	manager := logic.NewRegistryStorageManager()
	info, err := manager.GetConfig(logic.RegistryStorageTarget{
		K8sConfig:      "",
		K8sNamespace:   "default",
		DeploymentName: "w7-zpkv2-registry",
		ConfigMapName:  "w7-zpkv2-registry-config",
		CronJobName:    "w7-zpkv2-registry",
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonResponseWithoutError(ctx, gin.H{
		"info": info,
	})
}

func (c RegistryStorage) UpdateConfig(ctx *gin.Context) {
	params := RegistryStorageUpdateParams{}
	if !c.Validate(ctx, &params) {
		return
	}
	endpoint := strings.TrimSpace(params.S3RegionEndpoint)

	manager := logic.NewRegistryStorageManager()
	err := manager.UpdateConfig(logic.RegistryStorageUpdateInput{
		RegistryStorageTarget: logic.RegistryStorageTarget{
			K8sConfig:      "",
			K8sNamespace:   "default",
			DeploymentName: "w7-zpkv2-registry",
			ConfigMapName:  "w7-zpkv2-registry-config",
			CronJobName:    "w7-zpkv2-registry",
		},
		StorageType: logic.RegistryStorageTypeS3,
		S3: logic.RegistryS3StorageConfig{
			AccessKey:      strings.TrimSpace(params.S3AccessKey),
			SecretKey:      strings.TrimSpace(params.S3SecretKey),
			Bucket:         strings.TrimSpace(params.S3Bucket),
			Region:         strings.TrimSpace(params.S3Region),
			RegionEndpoint: endpoint,
			RootDirectory:  strings.TrimSpace(params.S3RootDirectory),
			Secure:         resolveS3Secure(endpoint),
			V4Auth:         true,
			Encrypt:        false,
			SkipVerify:     false,
		},
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func (c RegistryStorage) TestS3Connection(ctx *gin.Context) {
	params := RegistryStorageUpdateParams{}
	if !c.Validate(ctx, &params) {
		return
	}
	endpoint := strings.TrimSpace(params.S3RegionEndpoint)

	manager := logic.NewRegistryStorageManager()
	err := manager.TestS3Connection(logic.RegistryS3StorageConfig{
		AccessKey:      strings.TrimSpace(params.S3AccessKey),
		SecretKey:      strings.TrimSpace(params.S3SecretKey),
		Bucket:         strings.TrimSpace(params.S3Bucket),
		Region:         strings.TrimSpace(params.S3Region),
		RegionEndpoint: endpoint,
		RootDirectory:  strings.TrimSpace(params.S3RootDirectory),
		Secure:         resolveS3Secure(endpoint),
		V4Auth:         true,
		Encrypt:        false,
		SkipVerify:     false,
	})
	if err != nil {
		c.JsonResponseWithError(ctx, err, 500)
		return
	}

	c.JsonSuccessResponse(ctx)
}

func resolveS3Secure(endpoint string) bool {
	parsed, err := url.Parse(strings.TrimSpace(endpoint))
	if err == nil {
		switch strings.ToLower(parsed.Scheme) {
		case "http":
			return false
		case "https":
			return true
		}
	}
	return true
}

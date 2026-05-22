package controller

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/common/types/registry"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
)

type WebHook struct {
	controller.Abstract
}

func (c WebHook) Hook(ctx *gin.Context) {
	if !verifyToken(ctx) {
		c.JsonResponseWithServerError(ctx, errors.New("invalid signature"))
		return
	}

	var event registry.RegistryEvent
	if err := json.NewDecoder(ctx.Request.Body).Decode(&event); err != nil {
		c.JsonResponseWithError(ctx, errors.New("invalid JSON"), http.StatusBadRequest)
		return
	}

	slog.Info("registry webhook", "event", event)

	// 4. 处理事件
	for _, e := range event.Events {
		switch e.Action {
		case "push":
			facade.GetEvent().Publish(registry.RegistryRepositoryPushedEvent, registry.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "delete":
			facade.GetEvent().Publish(registry.RegistryRepositoryDeletedEvent, registry.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "pull":
			facade.GetEvent().Publish(registry.RegistryRepositoryPulledEvent, registry.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "mount":
			facade.GetEvent().Publish(registry.RegistryRepositoryMountedEvent, registry.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		default:
			slog.Error("invalid action", "type", e.Action)
		}
	}

	ctx.Writer.WriteHeader(http.StatusAccepted)

}

func verifyToken(ctx *gin.Context) bool {
	secretToken := facade.GetConfig().GetString("setting.registry.webhook_token")
	if secretToken == "" {
		return false
	}

	providedSig := ctx.Request.Header.Get("X-Webhook-Token")
	if providedSig == "" {
		return false
	}

	return secretToken == providedSig
}

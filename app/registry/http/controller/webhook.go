package controller

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/app/registry/types"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"github.com/we7coreteam/w7-rangine-go/v2/src/http/controller"
	"log/slog"
	"net/http"
)

type WebHook struct {
	controller.Abstract
}

func (c WebHook) Hook(ctx *gin.Context) {
	if !verifyToken(ctx) {
		c.JsonResponseWithServerError(ctx, errors.New("invalid signature"))
		return
	}

	var event types.RegistryEvent
	if err := json.NewDecoder(ctx.Request.Body).Decode(&event); err != nil {
		c.JsonResponseWithError(ctx, errors.New("invalid JSON"), http.StatusBadRequest)
		return
	}

	slog.Info("registry webhook", "event", event)

	// 4. 处理事件
	for _, e := range event.Events {
		switch e.Action {
		case "push":
			facade.GetEvent().Publish(types.RegistryRepositoryPushedEvent, types.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "delete":
			facade.GetEvent().Publish(types.RegistryRepositoryDeletedEvent, types.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "pull":
			facade.GetEvent().Publish(types.RegistryRepositoryPulledEvent, types.RegistryRepositoryWebHookPayLoad{
				Event: e,
			})
		case "mount":
			facade.GetEvent().Publish(types.RegistryRepositoryMountedEvent, types.RegistryRepositoryWebHookPayLoad{
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

package logic

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/w7panel/w7panel-zpk/modules/document/common/dao"
	"github.com/w7panel/w7panel-zpk/modules/document/common/entity"
	"github.com/w7panel/w7panel-zpk/modules/document/common/service/zpk"
	"github.com/we7coreteam/w7-rangine-go/v2/pkg/support/facade"
	"gorm.io/gorm"
)

const TokenHeader = "X-Document-Token"

type Session struct{}

func (Session) SaveUserInfo(ctx *gin.Context, user zpk.User) (string, error) {
	content, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	tokenTTLSeconds := facade.GetConfig().GetInt("auth.token_ttl_seconds")
	if tokenTTLSeconds <= 0 {
		return "", errors.New("文档 token 有效期未配置")
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	token := hex.EncodeToString(random)
	digest := sha256.Sum256([]byte(token))
	err = dao.Q.DocumentSession.WithContext(ctx.Request.Context()).Create(&entity.DocumentSession{
		ID: hex.EncodeToString(digest[:]), UserInfo: string(content),
		ExpiresAt: time.Now().Add(time.Duration(tokenTTLSeconds) * time.Second),
	})
	if err != nil {
		return "", err
	}
	return token, nil
}

func (Session) GetUserInfo(ctx *gin.Context) (zpk.User, error) {
	token := strings.TrimSpace(ctx.GetHeader(TokenHeader))
	if len(token) != 64 {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	digest := sha256.Sum256([]byte(token))
	query := dao.Q.DocumentSession
	current, err := query.WithContext(ctx.Request.Context()).Where(query.ID.Eq(hex.EncodeToString(digest[:])), query.ExpiresAt.Gt(time.Now())).First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	if err != nil {
		return zpk.User{}, err
	}
	var user zpk.User
	if err := json.Unmarshal([]byte(current.UserInfo), &user); err != nil {
		return zpk.User{}, err
	}
	if user.ID <= 0 {
		return zpk.User{}, zpk.ErrUnauthenticated
	}
	return user, nil
}

package handler

import (
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/kekim-go/Author/app/ctx"
	"github.com/kekim-go/Author/constant"
	"github.com/kekim-go/Author/model"
	grpc_author "github.com/kekim-go/Protobuf/gen/proto/author"
	"github.com/sirupsen/logrus"
)

type AppTokenHandler struct {
	Ctx *ctx.Context
}

func NewAppTokenHandler(ctx *ctx.Context) *AppTokenHandler {
	return &AppTokenHandler{
		Ctx: ctx,
	}
}

func (h *AppTokenHandler) CheckAppToken(token *model.Token, operation *model.Operation) grpc_author.ApiAuthRes_Code {
	h.Ctx.Logger.Debug(fmt.Sprintf("token: %+v, operation: %+v", token, operation))

	// App 조회
	appKey := operation.App.KeyName()
	appId, err := h.Ctx.RedisDB.Get(appKey, "uint")
	if err != nil && err == redis.Nil {
		err = operation.App.FindApp(h.Ctx.Orm)
		if err != nil {
			return grpc_author.ApiAuthRes_UNREGISTERED_SERVICE
		}
		h.Ctx.Logger.WithField("DB", fmt.Sprintf("%+v", operation.App)).Debug("Find App")
		h.Ctx.RedisDB.Set(appKey, operation.App.Id)
	} else {
		h.Ctx.Logger.WithField("Redis", appId).Debug("Find App")
		operation.App.Id = appId.(uint)
	}
	operation.AppId = operation.App.Id

	// Operation 조회
	opKey := operation.KeyName()
	opId, err := h.Ctx.RedisDB.Get(opKey, "uint")
	if err != nil && err == redis.Nil {
		err = operation.FindOperation(h.Ctx.Orm)
		if err != nil {
			return grpc_author.ApiAuthRes_UNREGISTERED_SERVICE
		}
		h.Ctx.Logger.WithField("DB", fmt.Sprintf("%+v", operation)).Debug("Find Operation")
		operation.SetRedis(h.Ctx.RedisDB)
	} else {
		h.Ctx.Logger.WithField("Redis", opId).Debug("Find Operation")
		operation.Id = opId.(uint)
	}

	// Token 조회
	tokenKey := token.KeyName()
	tokenId, err := h.Ctx.RedisDB.Get(tokenKey, "uint")
	if err != nil && err == redis.Nil {
		err = token.FindByToken(h.Ctx.Orm)
		if err = token.FindByToken(h.Ctx.Orm); err != nil {
			return grpc_author.ApiAuthRes_UNAUTHORIZED
		}
		h.Ctx.Logger.WithField("DB", fmt.Sprintf("%+v", token)).Debug("Find Token")
		h.Ctx.RedisDB.Set(tokenKey, token.Id)
	} else {
		h.Ctx.Logger.WithField("Redis", tokenId).Debug("Find Token")
		token.Id = tokenId.(uint)
	}

	// App-Token 조회
	appToken := model.AppToken{TokenId: token.Id, AppId: operation.AppId}
	appTokenKey := appToken.KeyName()
	appTokenId, err := h.Ctx.RedisDB.Get(appTokenKey, "uint")
	if err != nil && err == redis.Nil {
		err = appToken.FindByAppAndToken(h.Ctx.Orm)
		if err != nil {
			return grpc_author.ApiAuthRes_UNAUTHORIZED
		}
		h.Ctx.Logger.WithField("DB", fmt.Sprintf("%+v", appToken)).Debug("Find AppToken")
		// App-Token 정보는 24시간 유지
		h.Ctx.RedisDB.SetWithExpiration(appTokenKey, appToken.Id, 60*60*24)
	} else {
		h.Ctx.Logger.WithField("Redis", tokenId).Debug("Find AppToken")
		appToken.Id = appTokenId.(uint)
	}

	// App-Traffic 조회
	var finded = false
	var trafficMap = map[string]uint{}
	for _, unit := range constant.GetTrafficUnits() {
		t := model.Traffic{Unit: unit, AppId: operation.AppId}
		key := t.KeyName()
		trafficVal, err := h.Ctx.RedisDB.Get(key, "uint")
		if err == nil {
			finded = true
			trafficMap[key] = trafficVal.(uint)
		}
	}
	if !finded {
		// Traffic 조회 및 Cache
		traffics, err := model.FindTrafficsByApp(h.Ctx.Orm, operation.AppId)
		if err != nil {
			h.Ctx.Logger.WithField("DB", appToken.Id).Info("Count not found AppToken Traffic Info")
			return grpc_author.ApiAuthRes_UNKNOWN
		}

		for _, traffic := range traffics {
			key := traffic.KeyName()
			h.Ctx.RedisDB.Set(key, traffic.Val)
			trafficMap[key] = traffic.Val
		}
	}

	// 사용자 트래픽 조회
	var isValid = true
	for _, unit := range constant.GetTrafficUnits() {
		var tokenTraffic uint
		t := model.Traffic{Unit: unit, AppId: operation.AppId}
		appTrafficKey := t.KeyName()

		if maxTraffic, ok := trafficMap[appTrafficKey]; ok {
			h.Ctx.Logger.WithFields(logrus.Fields{
				"AppTrafficKey": appTrafficKey,
				"MaxTraffic":    maxTraffic,
			}).Debug("AppTrafficKey Check")

			tokenTrafficKey := fmt.Sprintf("%s%d:%d:%s", constant.KeyTrafficPrefix, token.Id, operation.AppId, unit)
			tokenTrafficDetailKey := fmt.Sprintf("%s%d:%d:%d:%s", constant.KeyTrafficDetailPrefix, token.Id, operation.AppId, operation.Id, unit)

			h.Ctx.Logger.WithField("TokenTrafficKey", tokenTrafficKey).Debug("TokenTrafficKey Check")
			tokenTrafficVal, err := h.Ctx.RedisDB.Get(tokenTrafficKey, "uint")
			if err != nil && err == redis.Nil {
				tokenTraffic = uint(0)
				h.Ctx.RedisDB.SAdd(constant.KEY_TRAFFIC_SET+unit, tokenTrafficKey)
			} else {
				h.Ctx.RedisDB.SAdd(constant.KeyTrafficDetailSet+unit, tokenTrafficDetailKey)
				tokenTraffic = tokenTrafficVal.(uint)
			}

			h.Ctx.Logger.WithFields(logrus.Fields{
				"AppTrafficKey":   appTrafficKey,
				"TokenTrafficKey": tokenTrafficKey,
				"MaxTraffic":      maxTraffic,
				"TokenTraffic":    tokenTraffic,
			}).Debug("Token Traffic Check")

			if tokenTraffic < maxTraffic {
				h.Ctx.RedisDB.Incr(tokenTrafficKey)
				h.Ctx.RedisDB.Incr(tokenTrafficDetailKey)
			} else {
				isValid = false
			}
		}
	}

	if isValid {
		return grpc_author.ApiAuthRes_VALID
	}

	return grpc_author.ApiAuthRes_LIMIT_EXCEEDED
}

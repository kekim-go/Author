package server

import (
	"context"

	"github.com/kekim-go/Author/handler"
	"github.com/kekim-go/Author/model"
	grpc_author "github.com/kekim-go/Protobuf/gen/proto/author"
)

type apiAuthServer struct {
	handler *handler.AppTokenHandler
}

func newApiAuthServer(handler *handler.AppTokenHandler) grpc_author.ApiAuthServiceServer {
	return &apiAuthServer{
		handler: handler,
	}
}

func (a *apiAuthServer) Auth(ctx context.Context, req *grpc_author.ApiAuthReq) (*grpc_author.ApiAuthRes, error) {
	token := model.Token{Token: req.Token, IsDel: false}
	operation := model.Operation{
		EndPoint: req.OperationUrl, IsDel: false,
		App: model.App{NameSpace: req.NameSpace, IsDel: false},
	}

	authCode := a.handler.CheckAppToken(&token, &operation)

	res := &grpc_author.ApiAuthRes{
		Code: authCode,
	}

	return res, nil
}

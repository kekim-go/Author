package handler

import "github.com/kekim-go/Author/app/ctx"

type UserHandler struct {
	Ctx *ctx.Context
}

func NewUserHandler(ctx *ctx.Context) *UserHandler {
	return &UserHandler{
		Ctx: ctx,
	}
}

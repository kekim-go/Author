package handler

import (
	"github.com/kekim-go/Author/app/ctx"
)

type AuthHandler struct {
	Ctx *ctx.Context
}

func NewAuthHandler(ctx *ctx.Context) *AuthHandler {
	return &AuthHandler{Ctx: ctx}
}

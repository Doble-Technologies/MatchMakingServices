package sse

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type JSendFailResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data"`
}

type JSendSuccessResponse[T any] struct {
	Status string `json:"status"`
	Data   T      `json:"data,omitempty"`
}

func BadRequestResponse(c *gin.Context, error error) {
	c.JSON(
		http.StatusBadRequest,
		JSendFailResponse[string]{
			Status: "fail",
			Data:   error.Error(),
		},
	)

	return
}

func CreatedResponse[T interface{}](c *gin.Context, i *T) {
	c.JSON(
		http.StatusCreated,
		JSendSuccessResponse[T]{
			Status: "success",
			Data:   *i,
		},
	)

	return
}

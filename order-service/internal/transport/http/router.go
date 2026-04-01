package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, handler *OrderHandler) {
	r.POST("/orders", handler.CreateOrder)
	r.GET("/orders/:id", handler.GetOrder)
	r.PATCH("/orders/:id/cancel", handler.CancelOrder)
}

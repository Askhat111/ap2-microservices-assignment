package http

import "github.com/gin-gonic/gin"

func RegisterRoutes(r *gin.Engine, handler *PaymentHandler) {
	r.POST("/payments", handler.ProcessPayment)
	r.GET("/payments/:order_id", handler.GetPayment)
}

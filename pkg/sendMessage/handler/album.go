package send_handler

import (
	"net/http"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	send_service "github.com/EvolutionAPI/evolution-go/pkg/sendMessage/service"
	"github.com/gin-gonic/gin"
)

func (s *sendHandler) SendAlbum(ctx *gin.Context) {
	instance, ok := ctx.MustGet("instance").(*instance_model.Instance)
	if !ok {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "instance not found"})
		return
	}
	var data send_service.AlbumStruct
	if err := ctx.ShouldBindBodyWithJSON(&data); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := data.Validate(); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	message, err := s.sendMessageService.SendAlbum(&data, instance)
	if err != nil {
		// Delivery can have reached WhatsApp before a send error is returned.
		ctx.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "code": "ambiguous_delivery"})
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "success", "data": message})
}

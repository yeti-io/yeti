package deduplication

import (
	"yeti/internal/config_handler"
	"yeti/internal/logger"
	"yeti/pkg/models"
)

type Handler = config_handler.Handler

func NewHandler(service *Service, log logger.Logger) *Handler {
	return config_handler.NewHandlerWithUpdater(
		models.EventTypeDedupConfigUpdated,
		models.ServiceTypeDeduplication,
		service,
		log,
	)
}

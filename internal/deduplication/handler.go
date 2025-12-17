package deduplication

import (
	"yeti/internal/config_handler"
	"yeti/internal/logger"
	"yeti/pkg/models"
)

// Handler is an alias for config_handler.Handler to maintain backward compatibility
type Handler = config_handler.Handler

// NewHandler creates a new config event handler for deduplication service
func NewHandler(service *Service, log logger.Logger) *Handler {
	return config_handler.NewHandlerWithUpdater(
		models.EventTypeDedupConfigUpdated,
		models.ServiceTypeDeduplication,
		service,
		log,
	)
}

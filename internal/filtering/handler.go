package filtering

import (
	"yeti/internal/config_handler"
	"yeti/internal/logger"
	"yeti/pkg/models"
)

type Handler = config_handler.Handler

func NewHandler(service *Service, log logger.Logger) *Handler {
	return config_handler.NewHandlerWithReloader(
		models.EventTypeFilteringRuleUpdated,
		models.ServiceTypeFiltering,
		service,
		log,
	)
}

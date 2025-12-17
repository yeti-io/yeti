package enrichment

import (
	"yeti/internal/config_handler"
	"yeti/internal/logger"
	"yeti/pkg/models"
)

type Handler = config_handler.Handler

func NewHandler(service Service, log logger.Logger) *Handler {
	return config_handler.NewHandlerWithReloader(
		models.EventTypeEnrichmentRuleUpdated,
		models.ServiceTypeEnrichment,
		service,
		log,
	)
}

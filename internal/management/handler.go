package management

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"yeti/internal/constants"
	"yeti/internal/logger"
	"yeti/pkg/errors"
)

type BaseHandler struct {
	Service Service
	Logger  logger.Logger
}

func (h *BaseHandler) HandleError(c *gin.Context, err error) {
	h.Logger.ErrorwCtx(c.Request.Context(), "Request error", "error", err, "path", c.Request.URL.Path)

	status := errors.ToHTTPStatus(err)
	response := errors.ToErrorResponse(err)

	c.JSON(status, response)
}

type Handler struct {
	BaseHandler
}

func NewHandler(service Service, log logger.Logger) *Handler {
	return &Handler{
		BaseHandler: BaseHandler{
			Service: service,
			Logger:  log,
		},
	}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		rules := v1.Group("/rules/filtering")
		{
			rules.GET("", h.ListRules)
			rules.POST("", h.CreateRule)
			rules.GET("/:id", h.GetRule)
			rules.PUT("/:id", h.UpdateRule)
			rules.DELETE("/:id", h.DeleteRule)
			rules.GET("/:id/versions", h.GetRuleVersions)
			rules.GET("/:id/audit", h.GetRuleAuditLogs)
		}

		audit := v1.Group("/audit")
		{
			audit.GET("/logs", h.GetAuditLogs)
		}
	}
}

// ListRules godoc
// @Summary      List all filtering rules
// @Description  Get a list of all filtering rules
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Success      200  {array}    FilteringRule
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/filtering [get]
func (h *Handler) ListRules(c *gin.Context) {
	rules, err := h.Service.ListFilteringRules(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, rules)
}

// CreateRule godoc
// @Summary      Create a new filtering rule
// @Description  Create a new filtering rule with the provided data
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        rule  body       CreateFilteringRuleRequest  true  "Filtering rule data"
// @Success      201   {object}   FilteringRule
// @Failure      400   {object}  errors.ErrorResponse
// @Failure      409   {object}  errors.ErrorResponse
// @Failure      500   {object}  errors.ErrorResponse
// @Router       /rules/filtering [post]
func (h *Handler) CreateRule(c *gin.Context) {
	var req CreateFilteringRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ToErrorResponse(errors.ErrValidation.WithCause(err)))
		return
	}

	rule, err := h.Service.CreateFilteringRule(c.Request.Context(), req)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetRule godoc
// @Summary      Get a filtering rule by ID
// @Description  Get a specific filtering rule by its ID
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Rule ID"
// @Success      200  {object}   FilteringRule
// @Failure      404  {object}  errors.ErrorResponse
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/filtering/{id} [get]
func (h *Handler) GetRule(c *gin.Context) {
	id := c.Param("id")
	rule, err := h.Service.GetFilteringRule(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateRule godoc
// @Summary      Update a filtering rule
// @Description  Update an existing filtering rule by ID
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        id    path      string                      true  "Rule ID"
// @Param        rule  body       UpdateFilteringRuleRequest  true  "Updated rule data"
// @Success      200   {object}   FilteringRule
// @Failure      400   {object}  errors.ErrorResponse
// @Failure      404   {object}  errors.ErrorResponse
// @Failure      500   {object}  errors.ErrorResponse
// @Router       /rules/filtering/{id} [put]
func (h *Handler) UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var req UpdateFilteringRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ToErrorResponse(errors.ErrValidation.WithCause(err)))
		return
	}

	rule, err := h.Service.UpdateFilteringRule(c.Request.Context(), id, req)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteRule godoc
// @Summary      Delete a filtering rule
// @Description  Delete a filtering rule by ID
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Rule ID"
// @Success      204  "No Content"
// @Failure      404  {object}  errors.ErrorResponse
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/filtering/{id} [delete]
func (h *Handler) DeleteRule(c *gin.Context) {
	id := c.Param("id")
	err := h.Service.DeleteFilteringRule(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// GetRuleVersions godoc
// @Summary      Get rule version history
// @Description  Get version history for a specific filtering rule
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Rule ID"
// @Success      200  {array}   RuleVersion
// @Failure      404  {object}  errors.ErrorResponse
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/filtering/{id}/versions [get]
func (h *Handler) GetRuleVersions(c *gin.Context) {
	id := c.Param("id")
	versions, err := h.Service.GetRuleVersions(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, versions)
}

// GetRuleAuditLogs godoc
// @Summary      Get audit logs for a rule
// @Description  Get audit logs for a specific filtering rule
// @Tags         filtering-rules
// @Accept       json
// @Produce      json
// @Param        id     path      string  true   "Rule ID"
// @Param        limit  query     int     false  "Maximum number of logs to return (1-1000)" default(100)
// @Success      200    {array}   AuditLog
// @Failure      404    {object}  errors.ErrorResponse
// @Failure      500    {object}  errors.ErrorResponse
// @Router       /rules/filtering/{id}/audit [get]
func (h *Handler) GetRuleAuditLogs(c *gin.Context) {
	id := c.Param("id")
	limit := parseLimit(c.Query("limit"))

	logs, err := h.Service.GetAuditLogs(c.Request.Context(), &id, "filtering", limit)
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

// GetAuditLogs godoc
// @Summary      Get audit logs
// @Description  Get audit logs with optional filtering by rule ID and rule type
// @Tags         audit
// @Accept       json
// @Produce      json
// @Param        rule_id    query     string  false  "Filter by rule ID"
// @Param        rule_type  query     string  false  "Filter by rule type (filtering, enrichment, deduplication)"
// @Param        limit      query     int     false  "Maximum number of logs to return (1-1000)" default(100)
// @Success      200        {array}   AuditLog
// @Failure      500        {object}  errors.ErrorResponse
// @Router       /audit/logs [get]
func (h *Handler) GetAuditLogs(c *gin.Context) {
	ruleID := c.Query("rule_id")
	ruleType := c.Query("rule_type")
	limit := parseLimit(c.Query("limit"))

	var ruleIDPtr *string
	if ruleID != "" {
		ruleIDPtr = &ruleID
	}

	logs, err := h.Service.GetAuditLogs(c.Request.Context(), ruleIDPtr, ruleType, limit)
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, logs)
}

func parseLimit(limitStr string) int {
	if limitStr == "" {
		return constants.DefaultLimit
	}
	parsed, err := strconv.Atoi(limitStr)
	if err != nil || parsed <= 0 || parsed > constants.MaxLimit {
		return constants.DefaultLimit
	}
	return parsed
}

type EnrichmentHandler struct {
	BaseHandler
}

func NewEnrichmentHandler(service Service, log logger.Logger) *EnrichmentHandler {
	return &EnrichmentHandler{
		BaseHandler: BaseHandler{
			Service: service,
			Logger:  log,
		},
	}
}

func (h *EnrichmentHandler) RegisterEnrichmentRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		rules := v1.Group("/rules/enrichment")
		{
			rules.GET("", h.ListEnrichmentRules)
			rules.POST("", h.CreateEnrichmentRule)
			rules.GET("/:id", h.GetEnrichmentRule)
			rules.PUT("/:id", h.UpdateEnrichmentRule)
			rules.DELETE("/:id", h.DeleteEnrichmentRule)
		}
	}
}

// ListEnrichmentRules godoc
// @Summary      List all enrichment rules
// @Description  Get a list of all enrichment rules
// @Tags         enrichment-rules
// @Accept       json
// @Produce      json
// @Success      200  {array}    EnrichmentRule
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/enrichment [get]
func (h *EnrichmentHandler) ListEnrichmentRules(c *gin.Context) {
	rules, err := h.Service.ListEnrichmentRules(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, rules)
}

// CreateEnrichmentRule godoc
// @Summary      Create a new enrichment rule
// @Description  Create a new enrichment rule with the provided data
// @Tags         enrichment-rules
// @Accept       json
// @Produce      json
// @Param        rule  body       CreateEnrichmentRuleRequest  true  "Enrichment rule data"
// @Success      201   {object}   EnrichmentRule
// @Failure      400   {object}  errors.ErrorResponse
// @Failure      409   {object}  errors.ErrorResponse
// @Failure      500   {object}  errors.ErrorResponse
// @Router       /rules/enrichment [post]
func (h *EnrichmentHandler) CreateEnrichmentRule(c *gin.Context) {
	var req CreateEnrichmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ToErrorResponse(errors.ErrValidation.WithCause(err)))
		return
	}

	rule, err := h.Service.CreateEnrichmentRule(c.Request.Context(), req)
	if err != nil {
		if errors.IsValidation(err) {
			response := errors.ToErrorResponse(err)
			if err.Error() != "" {
				response["message"] = err.Error()
			}
			c.JSON(http.StatusBadRequest, response)
			return
		}
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusCreated, rule)
}

// GetEnrichmentRule godoc
// @Summary      Get an enrichment rule by ID
// @Description  Get a specific enrichment rule by its ID
// @Tags         enrichment-rules
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Rule ID"
// @Success      200  {object}   EnrichmentRule
// @Failure      404  {object}  errors.ErrorResponse
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/enrichment/{id} [get]
func (h *EnrichmentHandler) GetEnrichmentRule(c *gin.Context) {
	id := c.Param("id")
	rule, err := h.Service.GetEnrichmentRule(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

// UpdateEnrichmentRule godoc
// @Summary      Update an enrichment rule
// @Description  Update an existing enrichment rule by ID
// @Tags         enrichment-rules
// @Accept       json
// @Produce      json
// @Param        id    path      string                        true  "Rule ID"
// @Param        rule  body       UpdateEnrichmentRuleRequest    true  "Updated rule data"
// @Success      200   {object}   EnrichmentRule
// @Failure      400   {object}  errors.ErrorResponse
// @Failure      404   {object}  errors.ErrorResponse
// @Failure      500   {object}  errors.ErrorResponse
// @Router       /rules/enrichment/{id} [put]
func (h *EnrichmentHandler) UpdateEnrichmentRule(c *gin.Context) {
	id := c.Param("id")
	var req UpdateEnrichmentRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ToErrorResponse(errors.ErrValidation.WithCause(err)))
		return
	}

	rule, err := h.Service.UpdateEnrichmentRule(c.Request.Context(), id, req)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, rule)
}

// DeleteEnrichmentRule godoc
// @Summary      Delete an enrichment rule
// @Description  Delete an enrichment rule by ID
// @Tags         enrichment-rules
// @Accept       json
// @Produce      json
// @Param        id   path      string  true  "Rule ID"
// @Success      204  "No Content"
// @Failure      404  {object}  errors.ErrorResponse
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /rules/enrichment/{id} [delete]
func (h *EnrichmentHandler) DeleteEnrichmentRule(c *gin.Context) {
	id := c.Param("id")
	err := h.Service.DeleteEnrichmentRule(c.Request.Context(), id)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

type DeduplicationHandler struct {
	BaseHandler
}

func NewDeduplicationHandler(service Service, log logger.Logger) *DeduplicationHandler {
	return &DeduplicationHandler{
		BaseHandler: BaseHandler{
			Service: service,
			Logger:  log,
		},
	}
}

func (h *DeduplicationHandler) RegisterDeduplicationRoutes(router *gin.Engine) {
	v1 := router.Group("/api/v1")
	{
		config := v1.Group("/config/deduplication")
		{
			config.GET("", h.GetDeduplicationConfig)
			config.PUT("", h.UpdateDeduplicationConfig)
		}
	}
}

// GetDeduplicationConfig godoc
// @Summary      Get deduplication configuration
// @Description  Get the current deduplication service configuration
// @Tags         deduplication
// @Accept       json
// @Produce      json
// @Success      200  {object}   DeduplicationConfig
// @Failure      500  {object}  errors.ErrorResponse
// @Router       /config/deduplication [get]
func (h *DeduplicationHandler) GetDeduplicationConfig(c *gin.Context) {
	config, err := h.Service.GetDeduplicationConfig(c.Request.Context())
	if err != nil {
		h.HandleError(c, err)
		return
	}
	c.JSON(http.StatusOK, config)
}

// UpdateDeduplicationConfig godoc
// @Summary      Update deduplication configuration
// @Description  Update the deduplication service configuration
// @Tags         deduplication
// @Accept       json
// @Produce      json
// @Param        config  body       UpdateDeduplicationConfigRequest  true  "Updated configuration"
// @Success      200     {object}   DeduplicationConfig
// @Failure      400     {object}  errors.ErrorResponse
// @Failure      500     {object}  errors.ErrorResponse
// @Router       /config/deduplication [put]
func (h *DeduplicationHandler) UpdateDeduplicationConfig(c *gin.Context) {
	var req UpdateDeduplicationConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, errors.ToErrorResponse(errors.ErrValidation.WithCause(err)))
		return
	}

	config, err := h.Service.UpdateDeduplicationConfig(c.Request.Context(), req)
	if err != nil {
		h.HandleError(c, err)
		return
	}

	c.JSON(http.StatusOK, config)
}

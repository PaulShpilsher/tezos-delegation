package api

import (
	"net/http"
	"strconv"
	"tezos-delegation/internal/apperrors"
	"tezos-delegation/internal/model"
	"tezos-delegation/internal/ports"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
	cacheTTL        = 30 * time.Second // Cache responses for 30 seconds
)

// DelegationHandler implements DelegationHandlerPort
type DelegationHandler struct {
	Service ports.DelegationServicePort
	Logger  zerolog.Logger
}

func NewDelegationHandler(service ports.DelegationServicePort, logger zerolog.Logger) *DelegationHandler {
	return &DelegationHandler{
		Service: service,
		Logger:  logger.With().Str("component", "DelegationHttpHandler").Logger(),
	}
}

// respondWithError sends a consistent error response with proper status code
func respondWithError(ctx iris.Context, status int, message string) {
	ctx.StatusCode(status)
	ctx.JSON(iris.Map{"error": message})
}

// logAndRespondWithError logs detailed error information but returns sanitized response
func (h *DelegationHandler) logAndRespondWithError(ctx iris.Context, status int, userMessage, logMessage string, err error) {
	// Log detailed error for debugging
	h.Logger.Error().Err(err).Str("user_message", userMessage).Msg(logMessage)

	// Return sanitized message to user
	respondWithError(ctx, status, userMessage)
}

// toDelegationDto converts a model.Delegation to DelegationDto
func toDelegationDto(d model.Delegation) DelegationDto {
	return DelegationDto{
		Timestamp: d.Timestamp.UTC().Format(time.RFC3339),
		Amount:    strconv.FormatInt(d.Amount, 10),
		Delegator: d.Delegator,
		Level:     strconv.FormatInt(d.Level, 10),
	}
}

// validatePaginationParams validates and returns page and pageSize parameters
func (h *DelegationHandler) validatePaginationParams(ctx iris.Context) (int, int, bool) {
	// Parse page parameter
	page := 1
	if ctx.URLParamExists("page") {
		pageStr := ctx.URLParam("page")
		// Validate string length to prevent resource exhaustion
		if len(pageStr) > 10 {
			h.Logger.Warn().Str("page", pageStr).Msg("Page parameter too long")
			respondWithError(ctx, http.StatusBadRequest, "Invalid page parameter: too long")
			return 0, 0, false
		}

		p, err := ctx.URLParamInt("page")
		if err != nil || p < 1 {
			h.Logger.Warn().Str("page", pageStr).Msg("Invalid page parameter")
			respondWithError(ctx, http.StatusBadRequest, "Invalid page parameter: must be a positive integer")
			return 0, 0, false
		}
		page = p
	}

	// Parse pageSize parameter
	pageSize := defaultPageSize
	if ctx.URLParamExists("pageSize") {
		pageSizeStr := ctx.URLParam("pageSize")
		// Validate string length to prevent resource exhaustion
		if len(pageSizeStr) > 10 {
			h.Logger.Warn().Str("pageSize", pageSizeStr).Msg("PageSize parameter too long")
			respondWithError(ctx, http.StatusBadRequest, "Invalid pageSize parameter: too long")
			return 0, 0, false
		}

		ps, err := ctx.URLParamInt("pageSize")
		if err != nil || ps < 1 || ps > maxPageSize {
			h.Logger.Warn().Str("pageSize", pageSizeStr).Msg("Invalid pageSize parameter")
			respondWithError(ctx, http.StatusBadRequest, "Invalid pageSize parameter: must be between 1 and 1000")
			return 0, 0, false
		}
		pageSize = ps
	}

	return page, pageSize, true
}

// validateYearParam validates and returns the year parameter if provided
func (h *DelegationHandler) validateYearParam(ctx iris.Context) (*int, bool) {
	yearStr := ctx.URLParam("year")
	if yearStr == "" {
		return nil, true
	}

	// Validate string length to prevent resource exhaustion
	if len(yearStr) > 10 {
		h.Logger.Warn().Str("year", yearStr).Msg("Year parameter too long")
		respondWithError(ctx, http.StatusBadRequest, "Invalid year parameter: too long")
		return nil, false
	}

	yearInt, err := strconv.Atoi(yearStr)
	if err != nil || yearInt < 2018 {
		h.Logger.Warn().Str("year", yearStr).Msg("Invalid year parameter")
		respondWithError(ctx, http.StatusBadRequest, "Invalid year parameter: must be a valid year from 2018 onwards")
		return nil, false
	}

	return &yearInt, true
}

// GetDelegations handles GET /xtz/delegations
// @Summary Get delegations with pagination and optional year filter
// @Description Retrieves a paginated list of Tezos delegations with optional year filtering
// @Tags delegations
// @Produce json
// @Param page query int false "Page number (default: 1)" minimum(1)
// @Param pageSize query int false "Number of items per page (default: 50, max: 1000)" minimum(1) maximum(1000)
// @Param year query int false "Filter by year (optional) minimum(2018)"
// @Success 200 {object} GetDelegationsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /xtz/delegations [get]
func (h *DelegationHandler) GetDelegations(ctx iris.Context) {
	// Validate pagination parameters
	page, pageSize, ok := h.validatePaginationParams(ctx)
	if !ok {
		return
	}

	// Validate year parameter
	yearPtr, ok := h.validateYearParam(ctx)
	if !ok {
		return
	}

	// Get delegations from service
	delegations, err := h.Service.GetDelegations(ctx.Request().Context(), page, pageSize, yearPtr)
	if err != nil {
		// Determine appropriate HTTP status code and message based on error type
		var statusCode int
		var userMessage string
		var logMessage string

		// Check if it's a validation error from the service
		if apperrors.IsValidationError(err) {
			statusCode = http.StatusBadRequest
			userMessage = "Invalid request parameters"
			logMessage = "Validation error in GetDelegations"
		} else if apperrors.IsDatabaseError(err) {
			statusCode = http.StatusInternalServerError
			userMessage = "Database error"
			logMessage = "Database error in GetDelegations"
		} else {
			statusCode = http.StatusInternalServerError
			userMessage = "Internal server error"
			logMessage = "Unexpected error in GetDelegations"
		}

		h.logAndRespondWithError(ctx, statusCode, userMessage, logMessage, err)
		return
	}

	// Convert to DTOs
	dtos := make([]DelegationDto, len(delegations))
	for i, d := range delegations {
		dtos[i] = toDelegationDto(d)
	}

	// Return response
	ctx.StatusCode(http.StatusOK)
	ctx.JSON(GetDelegationsResponse{Data: dtos})
}

package api

import (
	"net/http"
	"strconv"
	"tezos-delegation/internal/apperrors"
	"tezos-delegation/internal/model"
	"tezos-delegation/internal/services"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// DelegationHandlerPort defines the contract for delegation HTTP handlers
type DelegationHandlerPort interface {
	GetDelegations(ctx iris.Context)
}

type DelegationHandler struct {
	Service services.DelegationServicePort
	Logger  zerolog.Logger
}

func NewDelegationHandler(service services.DelegationServicePort, logger zerolog.Logger) *DelegationHandler {
	return &DelegationHandler{
		Service: service,
		Logger:  logger.With().Str("component", "delegation_handler").Logger(),
	}
}

// respondWithError sends a consistent error response with proper status code
func respondWithError(ctx iris.Context, status int, message string) {
	ctx.StatusCode(status)
	ctx.JSON(iris.Map{"error": message})
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
		p, err := ctx.URLParamInt("page")
		if err != nil || p < 1 {
			h.Logger.Warn().Str("page", ctx.URLParam("page")).Msg("Invalid page parameter")
			respondWithError(ctx, http.StatusBadRequest, "Invalid page parameter: must be a positive integer")
			return 0, 0, false
		}
		page = p
	}

	// Parse pageSize parameter
	pageSize := defaultPageSize
	if ctx.URLParamExists("pageSize") {
		ps, err := ctx.URLParamInt("pageSize")
		if err != nil || ps < 1 || ps > maxPageSize {
			h.Logger.Warn().Str("pageSize", ctx.URLParam("pageSize")).Msg("Invalid pageSize parameter")
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
		// Log the error with context
		h.Logger.Error().Err(err).Int("page", page).Int("pageSize", pageSize).Interface("year", yearPtr).Msg("Service error in GetDelegations")

		// Determine appropriate HTTP status code based on error type
		var statusCode int
		var message string

		// Check if it's a validation error from the service
		if apperrors.IsValidationError(err) {
			statusCode = http.StatusBadRequest
			message = "Invalid request parameters"
		} else if apperrors.IsDatabaseError(err) {
			statusCode = http.StatusInternalServerError
			message = "Database error"
		} else {
			statusCode = http.StatusInternalServerError
			message = "Internal server error"
		}

		respondWithError(ctx, statusCode, message)
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

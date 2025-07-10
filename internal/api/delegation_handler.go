package api

import (
	"net/http"
	"strconv"
	"tezos-delegation/internal/services"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/rs/zerolog"
)

const (
	defaultPageSize = 50
	maxPageSize     = 1000
)

// DelegationHandlerInterface defines the contract for delegation HTTP handlers
type DelegationHandlerInterface interface {
	GetDelegations(ctx iris.Context)
}

type DelegationHandler struct {
	Service *services.DelegationService
	Logger  zerolog.Logger
}

func NewDelegationHandler(service *services.DelegationService, logger zerolog.Logger) *DelegationHandler {
	return &DelegationHandler{Service: service, Logger: logger}
}

func respondWithError(ctx iris.Context, status int, message string) {
	ctx.StatusCode(status)
	ctx.JSON(iris.Map{"error": message})
}

func (h *DelegationHandler) GetDelegations(ctx iris.Context) {
	// Parse 'page' query param (default 1)
	page := 1
	if ctx.URLParamExists("page") {
		p, err := ctx.URLParamInt("page")
		if err != nil || p < 1 {
			h.Logger.Warn().Str("page", ctx.URLParam("page")).Msg("Invalid page param")
			respondWithError(ctx, http.StatusBadRequest, "invalid page parameter")
			return
		}
		page = p
	}

	yearStr := ctx.URLParam("year")
	var yearPtr *int
	if yearStr != "" {
		yearInt, err := strconv.Atoi(yearStr)
		if err != nil || yearInt < 0 {
			h.Logger.Warn().Str("year", yearStr).Msg("Invalid year param")
			respondWithError(ctx, http.StatusBadRequest, "invalid year parameter")
			return
		}
		yearPtr = &yearInt
	}

	delegations, err := h.Service.GetDelegations(ctx.Request().Context(), page, defaultPageSize, yearPtr)
	if err != nil {
		h.Logger.Error().Err(err).Msg("Service error in GetDelegations")
		respondWithError(ctx, http.StatusInternalServerError, "internal server error")
		return
	}

	result := make([]DelegationDto, 0, len(delegations))
	for _, d := range delegations {
		result = append(result, DelegationDto{
			Timestamp: d.Timestamp.UTC().Format(time.RFC3339),
			Amount:    strconv.FormatInt(d.Amount, 10),
			Delegator: d.Delegator,
			Level:     strconv.FormatInt(d.Level, 10),
		})
	}
	ctx.JSON(GetDelegationsResponse{Data: result})
	ctx.StatusCode(http.StatusOK)
}

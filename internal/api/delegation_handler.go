package api

import (
	"net/http"
	"strconv"
	"tezoz-delegation/internal/services"
	"time"

	"github.com/kataras/iris/v12"
)

type DelegationHandler struct {
	Service *services.DelegationService
}

func NewDelegationHandler(service *services.DelegationService) *DelegationHandler {
	return &DelegationHandler{Service: service}
}

// TODO: possibly make this configurable
const defaultPageSize = 50

func (h *DelegationHandler) GetDelegations(ctx iris.Context) {
	// Parse 'page' query param (default 1)
	page := 1
	if ctx.URLParamExists("page") {
		p, err := ctx.URLParamInt("page")
		if err != nil || p < 1 {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.JSON(iris.Map{"error": "invalid page parameter"})
			return
		}
		page = p
	}

	// Parse 'page_size' query param (default defaultPageSize, max 1000)
	pageSize := defaultPageSize

	// Future enhancement: add page_size as an optional query parameter
	// commented out code as an example below
	// if ctx.URLParamExists("page_size") {
	// 	ps, err := ctx.URLParamInt("page_size")
	// 	if err != nil || ps < 1 {
	// 		ctx.StatusCode(http.StatusBadRequest)
	// 		ctx.JSON(iris.Map{"error": "invalid page_size parameter"})
	// 		return
	// 	}
	// 	if ps > 1000 {
	// 		ps = 1000 // enforce max page size
	// 	}
	// 	pageSize = ps
	// }

	// Parse 'year' query param (optional)
	yearStr := ctx.URLParam("year")
	var yearPtr *int
	if yearStr != "" {
		yearInt, err := strconv.Atoi(yearStr)
		if err != nil || yearInt < 0 {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.JSON(iris.Map{"error": "invalid year parameter"})
			return
		}
		yearPtr = &yearInt
	}

	delegations, err := h.Service.GetDelegations(ctx.Request().Context(), page, pageSize, yearPtr)
	if err != nil {
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": err.Error()})
		return
	}
	result := make([]DelegationDto, 0, len(delegations))
	for _, d := range delegations {
		result = append(result, DelegationDto{
			Timestamp: d.Timestamp.UTC().Format(time.RFC3339),
			Amount:    d.Amount,
			Delegator: d.Delegator,
			Level:     strconv.FormatInt(d.Level, 10),
		})
	}
	ctx.JSON(GetDelegationsResponse{Data: result})
}

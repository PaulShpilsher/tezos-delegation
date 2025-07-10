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

func (h *DelegationHandler) GetDelegations(ctx iris.Context) {
	page, _ := strconv.Atoi(ctx.URLParamDefault("page", "1"))
	year := ctx.URLParam("year")
	delegations, err := h.Service.GetDelegations(ctx.Request().Context(), page, year)
	if err != nil {
		if err.Error() == "parsing time \"\": month out of range" || err.Error() == "invalid year format" {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.JSON(iris.Map{"error": "invalid year format"})
			return
		}
		ctx.StatusCode(http.StatusInternalServerError)
		ctx.JSON(iris.Map{"error": err.Error()})
		return
	}
	data := make([]iris.Map, 0, len(delegations))
	for _, d := range delegations {
		data = append(data, iris.Map{
			"timestamp": d.Timestamp.UTC().Format(time.RFC3339),
			"amount":    d.Amount,
			"delegator": d.Delegator,
			"level":     strconv.Itoa(int(d.Level)),
		})
	}
	ctx.JSON(iris.Map{"data": data})
}

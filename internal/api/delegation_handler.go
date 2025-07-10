package api

import (
	"net/http"
	"strconv"
	"time"

	"tezoz-delegation/internal/db"

	"github.com/kataras/iris/v12"
)

type DelegationHandler struct {
	Repo *db.DelegationRepository
}

func NewDelegationHandler(repo *db.DelegationRepository) *DelegationHandler {
	return &DelegationHandler{Repo: repo}
}

func (h *DelegationHandler) GetDelegations(ctx iris.Context) {
	// Pagination
	page, _ := strconv.Atoi(ctx.URLParamDefault("page", "1"))
	if page < 1 {
		page = 1
	}
	limit := 50
	offset := (page - 1) * limit

	// Year filter
	year := ctx.URLParam("year")
	var from, to time.Time
	var err error
	if year != "" {
		from, err = time.Parse("2006", year)
		if err != nil {
			ctx.StatusCode(http.StatusBadRequest)
			ctx.JSON(iris.Map{"error": "invalid year format"})
			return
		}
		to = from.AddDate(1, 0, 0)
	}

	delegations, err := h.Repo.ListDelegations(limit, offset, from, to)
	if err != nil {
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

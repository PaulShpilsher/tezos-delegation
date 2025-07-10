package api

import (
	"github.com/kataras/iris/v12"
)

type RouterDeps struct {
	DelegationHandler *DelegationHandler
}

func RegisterRoutes(app *iris.Application, deps RouterDeps) {
	app.Get("/xtz/delegations", deps.DelegationHandler.GetDelegations)
}

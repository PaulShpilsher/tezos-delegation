package api

import (
	"github.com/kataras/iris/v12"
)

func RegisterRoutes(app *iris.Application, delegationHandler *DelegationHandler) {
	app.Get("/xtz/delegations", delegationHandler.GetDelegations)
}

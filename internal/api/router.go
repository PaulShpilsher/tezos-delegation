package api

import (
	"github.com/kataras/iris/v12"
)

// securityHeadersMiddleware adds security headers to responses
func securityHeadersMiddleware() iris.Handler {
	return func(ctx iris.Context) {
		ctx.Header("X-Content-Type-Options", "nosniff")
		ctx.Header("X-Frame-Options", "DENY")
		ctx.Header("X-XSS-Protection", "1; mode=block")
		ctx.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		ctx.Header("Content-Security-Policy", "default-src 'self'")
		ctx.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		ctx.Next()
	}
}

func RegisterRoutes(app *iris.Application, delegationHandler *DelegationHandler) {

	app.Use(securityHeadersMiddleware())

	// TODO: Rate limiter

	app.Get("/xtz/delegations", delegationHandler.GetDelegations)
}

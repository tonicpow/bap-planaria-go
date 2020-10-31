package api

import (
	apirouter "github.com/mrz1836/go-api-router"
)

// RegisterRoutes register all the package specific routes
func RegisterRoutes(router *apirouter.Router) {

	// Find
	router.HTTPRouter.GET("/find/:collection", router.Request(bitQuery))
}

package api

import (
	apirouter "github.com/mrz1836/go-api-router"
)

// RegisterRoutes register all the package specific routes
func RegisterRoutes(router *apirouter.Router) {

	// Use the authentication middleware wrapper
	s := apirouter.NewStack()
	// Authenticated requests
	router.HTTPRouter.GET("/find/:collection", router.Request(s.Wrap(bitquery))) // Get an app
	// Update an existing app
}

package router

import (
	"github.com/julienschmidt/httprouter"
	apirouter "github.com/mrz1836/go-api-router"
	"github.com/tonicpow/bap-planaria-go/api"
)

// Handlers isolated the handlers / router for API or Link Service (helps with testing)
func Handlers() *httprouter.Router {

	// Create a new router
	r := apirouter.New()

	// This is used for the "Origin" to be returned as the origin
	r.CrossOriginAllowOriginAll = true

	// Set headers to expose via CORs
	r.AccessControlExposeHeaders = "Authorization"

	// Register all actions
	api.RegisterRoutes(r)

	// Return the router
	return r.HTTPRouter.Router
}

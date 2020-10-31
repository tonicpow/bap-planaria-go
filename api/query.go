package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"time"

	"github.com/julienschmidt/httprouter"
	apirouter "github.com/mrz1836/go-api-router"
	"github.com/tonicpow/bap-planaria-go/database"
	"go.mongodb.org/mongo-driver/bson"
)

func bitQuery(w http.ResponseWriter, req *http.Request, _ httprouter.Params) {

	// Parse the params
	params := apirouter.GetParams(req)
	collection := params.GetString("collection")
	limit := params.GetInt("limit")
	skip := params.GetInt("skip")
	find := params.GetString("query")

	// decode b64 string
	decoded, err := base64.StdEncoding.DecodeString(find)
	if err != nil {
		apirouter.ReturnResponse(w, req, http.StatusBadRequest, err.Error())
		return
	}

	q := bson.M{}
	if err = json.Unmarshal(decoded, &q); err != nil {
		apirouter.ReturnResponse(w, req, http.StatusBadRequest, err.Error())
		return
	}

	// Create a DB connection
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer func() {
		cancel()
	}()
	var conn *database.Connection
	if conn, err = database.Connect(ctx); err != nil {
		apirouter.ReturnResponse(w, req, http.StatusExpectationFailed, err.Error())
		return
	}
	defer func() {
		_ = conn.Disconnect(ctx)
	}()

	// Get matching documents
	var records []bson.M
	if records, err = conn.GetStateDocs(collection, int64(limit), int64(skip), q); err != nil {
		apirouter.ReturnResponse(w, req, http.StatusExpectationFailed, err.Error())
		return
	}

	// Return the records
	apirouter.ReturnResponse(w, req, http.StatusOK, records)
}

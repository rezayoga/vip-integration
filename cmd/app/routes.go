package main

import (
	"github.com/NYTimes/gziphandler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"
)

func (app *application) routes() http.Handler {
	// create a router mux
	mux := chi.NewRouter()
	//var api = "/api"
	mux.Use(middleware.RequestID)
	mux.Use(middleware.RealIP)
	mux.Use(middleware.Recoverer)
	mux.Use(middleware.Logger)
	mux.Use(gziphandler.GzipHandler)
	mux.NotFound(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse := JSONResponse{
			Error:   true,
			Message: "route does not exist",
			Data:    nil,
		}

		err := app.writeJSON(w, http.StatusNotFound, jsonResponse)
		if err != nil {
			return
		}
	})

	mux.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		jsonResponse := JSONResponse{
			Error:   true,
			Message: "method not allowed",
			Data:    nil,
		}

		err := app.writeJSON(w, http.StatusMethodNotAllowed, jsonResponse)
		if err != nil {
			return
		}
	})

	mux.Get("/", app.home)
	mux.Post("/invoice/draft-submission", app.DraftSubmissionHandler)
	mux.Post("/invoice/upload", app.UploadInvoiceHandler)
	mux.Get("/invoice", app.GetInvoiceHandler)

	return mux
}

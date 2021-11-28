package main

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Message string `json:"message"`
}

func InternalError(w http.ResponseWriter) {
	e := Error{"Internal Server Error"}
	bts, _ := json.Marshal(&e)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	http.Error(w, string(bts), http.StatusInternalServerError)
}

func BadRequest(message string, w http.ResponseWriter) {
	e := Error{"Bad request: " + message}

	bts, _ := json.Marshal(&e)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	http.Error(w, string(bts), http.StatusBadRequest)
}

func NotFound(message string, w http.ResponseWriter) {
	e := Error{"Resource not found: " + message}

	bts, _ := json.Marshal(&e)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	http.Error(w, string(bts), http.StatusNotFound)
}

func MiscError(message string, code int, w http.ResponseWriter) {
	e := Error{message}

	bts, _ := json.Marshal(&e)

	header := w.Header()
	header.Set("Content-Type", "application/json")
	http.Error(w, string(bts), code)
}

func ErrorSwitch(err error, code int, w http.ResponseWriter) {
	switch code {
	case http.StatusInternalServerError:
		InternalError(w)
	case http.StatusBadRequest:
		BadRequest(err.Error(), w)
	case http.StatusNotFound:
		NotFound(err.Error(), w)
	default:
		MiscError(err.Error(), code, w)
	}
}

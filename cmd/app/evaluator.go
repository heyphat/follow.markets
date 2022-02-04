package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"follow.markets/internal/pkg/strategy"
	"github.com/gorilla/mux"
)

func dropSignals(w http.ResponseWriter, req *http.Request) {
	str, ok := mux.Vars(req)["names"]
	if !ok {
		BadRequest("missing signal names", w)
		//w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, s := range strings.Split(str, ",") {
		if err := market.DropSignal(s); err != nil {
			logger.Error.Println(err)
			InternalError(w)
		}
	}
	w.WriteHeader(http.StatusOK)
}

func addSignal(w http.ResponseWriter, req *http.Request) {
	bts, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	signal, err := strategy.NewSignalFromBytes(bts)
	if err != nil {
		logger.Error.Println(err)
		BadRequest(err.Error(), w)
		return
	}
	str, ok := mux.Vars(req)["patterns"]
	if !ok {
		logger.Error.Println(err)
		BadRequest("missing patterns", w)
		return
	}
	patterns, err := url.PathUnescape(str)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	if err := market.AddSignal(strings.Split(patterns, ","), signal); err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func listSignals(w http.ResponseWriter, req *http.Request) {
	opts := req.URL.Query()
	var signals strategy.Signals
	if str, ok := opts["names"]; ok && len(str) > 0 {
		signals = market.GetSingals(strings.Split(str[0], ","))
	} else {
		signals = market.GetSingals([]string{})
	}
	bts, err := json.Marshal(signals)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	header := w.Header()
	header.Set("Content-Length", strconv.Itoa(len(bts)))
	w.WriteHeader(http.StatusOK)
	w.Write(bts)
}

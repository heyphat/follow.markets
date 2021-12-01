package main

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"follow.market/internal/pkg/strategy"
	"github.com/gorilla/mux"
)

func dropSignals(w http.ResponseWriter, req *http.Request) {
	str, ok := mux.Vars(req)["signals"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
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
		InternalError(w)
		return
	}
	str, ok := mux.Vars(req)["patterns"]
	if !ok {
		logger.Error.Println(err)
		w.WriteHeader(http.StatusBadRequest)
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

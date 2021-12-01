package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"

	tax "follow.market/internal/pkg/techanex"
)

func watchlist(w http.ResponseWriter, req *http.Request) {
	type watchlist struct {
		List []string `json:"watchlist"`
	}
	wl := watchlist{List: market.Watchlist()}
	bts, err := json.Marshal(wl)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	fmt.Println(string(bts))
	header := w.Header()
	header.Set("Content-Length", strconv.Itoa(len(bts)))
	w.WriteHeader(http.StatusOK)
	w.Write(bts)
}

func watch(w http.ResponseWriter, req *http.Request) {
	str, ok := mux.Vars(req)["ticker"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}
	for _, t := range strings.Split(str, ",") {
		if err := market.Watch(t); err != nil {
			logger.Error.Println(err)
			InternalError(w)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func last(w http.ResponseWriter, req *http.Request) {
	str, ok := mux.Vars(req)["ticker"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	strs := strings.Split(str, ",")
	if len(strs) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if !market.IsWatchingOn(strs[0]) {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	clast := market.LastCandles(strs[0])
	ilast := market.LastIndicators(strs[0])
	type candles struct {
		Candles    tax.CandlesJSON    `json:"candles"`
		Indicators tax.IndicatorsJSON `json:"indicators"`
	}
	bts, err := json.Marshal(candles{Candles: clast, Indicators: ilast})
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

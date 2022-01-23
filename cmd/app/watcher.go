package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	tax "follow.markets/internal/pkg/techanex"
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
	header := w.Header()
	header.Set("Content-Length", strconv.Itoa(len(bts)))
	w.WriteHeader(http.StatusOK)
	w.Write(bts)
}

func watch(w http.ResponseWriter, req *http.Request) {
	tickers, ok := parseVars(mux.Vars(req), "ticker")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
	}
	mk, ok := parseOptions(req.URL.Query(), "market")
	if !ok {
		mk = []string{"CASH"}
	}
	for _, t := range tickers {
		if err := market.Watch(t, mk[0]); err != nil {
			logger.Error.Println(err)
			InternalError(w)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func last(w http.ResponseWriter, req *http.Request) {
	strs, ok := parseVars(mux.Vars(req), "ticker")
	if !ok {
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

func synced(w http.ResponseWriter, req *http.Request) {
	strs, ok := parseVars(mux.Vars(req), "ticker")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ticker := strs[0]
	strs, ok = parseVars(mux.Vars(req), "frame")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	fmt.Println(strs)
	frame, err := strconv.Atoi(strs[0])
	if err != nil {
		logger.Error.Println(err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	isSynced := market.IsSynced(ticker, time.Duration(frame)*time.Minute)
	type out struct {
		Result bool `json:"is_synced"`
	}
	bts, err := json.Marshal(out{Result: isSynced})
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

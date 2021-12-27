package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"follow.market/internal/pkg/strategy"
	"github.com/gorilla/mux"
)

func test(w http.ResponseWriter, req *http.Request) {
	strs, ok := parseVars(mux.Vars(req), "ticker")
	if !ok {
		BadRequest("missing ticker", w)
		return
	}
	ticker := strs[0]
	bts, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	signal, err := strategy.NewSignalFromBytes(bts)
	if err != nil {
		BadRequest(err.Error(), w)
		return
	}
	opts := req.URL.Query()
	balance := 10000
	if rs, ok := parseOptions(opts, "balance"); ok && len(rs) > 0 {
		if bl, err := strconv.Atoi(rs[0]); err != nil {
			balance = bl
		}
	}
	end := time.Now()
	start := end.AddDate(0, -1, 0)
	if rs, ok := parseOptions(opts, "start"); ok && len(rs) > 0 {
		if st, err := strconv.Atoi(rs[0]); err == nil {
			logger.Error.Println(err)
		} else {
			start = time.Unix(int64(st), 0)
		}
	}
	if rs, ok := parseOptions(opts, "end"); ok && len(rs) > 0 {
		if ed, err := strconv.Atoi(rs[0]); err == nil {
			logger.Error.Println(err)
		} else {
			end = time.Unix(int64(ed), 0)
		}
	}
	profitMargin := 0.1
	lossTolerance := 0.05
	if rs, ok := parseOptions(opts, "profit_margin"); ok && len(rs) > 0 {
		if pm, err := strconv.ParseFloat(rs[0], 32); err == nil {
			logger.Error.Println(err)
		} else {
			profitMargin = pm
		}
	}
	if rs, ok := parseOptions(opts, "loss_tolerance"); ok && len(rs) > 0 {
		if lt, err := strconv.ParseFloat(rs[0], 32); err == nil {
			logger.Error.Println(err)
		} else {
			lossTolerance = lt
		}
	}

	stg := strategy.Strategy{
		EntryRule:      strategy.NewRule(*signal),
		ExitRule:       nil,
		RiskRewardRule: strategy.NewRiskRewardRule(-lossTolerance, profitMargin),
	}
	rs, err := market.Test(ticker, float64(balance), &stg, start, end)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	fmt.Println(rs)
	w.WriteHeader(http.StatusOK)
}

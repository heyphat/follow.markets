package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"follow.markets/internal/pkg/strategy"
	"follow.markets/pkg/util"
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
	balance := configs.Market.Tester.InitBalance
	if rs, ok := parseOptions(opts, "balance"); ok && len(rs) > 0 {
		if bl, err := strconv.Atoi(rs[0]); err != nil {
			balance = float64(bl)
		}
	}
	var start, end *time.Time
	nw := time.Now()
	end = &nw
	if rs, ok := parseOptions(opts, "start"); ok && len(rs) > 0 {
		if st, err := strconv.Atoi(rs[0]); err != nil {
			logger.Error.Println(err)
		} else {
			start = &[]time.Time{time.Unix(int64(st), 0)}[0]
		}
	}
	if rs, ok := parseOptions(opts, "end"); ok && len(rs) > 0 {
		if ed, err := strconv.Atoi(rs[0]); err == nil {
			logger.Error.Println(err)
		} else {
			end = &[]time.Time{time.Unix(int64(ed), 0)}[0]
		}
	}
	profitMargin := configs.Market.Tester.ProfitMargin
	lossTolerance := configs.Market.Tester.LossTolerance
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
	savePath, err := util.ConcatPath(configs.Market.Tester.SavePath, ticker+"-"+signal.Name+"-"+time.Now().Format("2006-01-02T15:04:05"))
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	go func() {
		rs, err := market.Test(ticker, balance, &stg, start, end, savePath)
		if err != nil {
			logger.Error.Println(err)
			InternalError(w)
			return
		}
		fmt.Println(rs)
	}()
	w.WriteHeader(http.StatusOK)
}

package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"

	"follow.markets/pkg/config"
)

func balances(w http.ResponseWriter, req *http.Request) {
	type balances struct {
		Balances map[string]string `json:"balances"`
	}
	mk, ok := parseOptions(req.URL.Query(), "market")
	if !ok {
		mk = []string{"CASH"}
	}
	bs := balances{}
	var err error
	if bs.Balances, err = market.Balances(mk[0]); err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	bts, err := json.Marshal(bs)
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

func updateConfigs(w http.ResponseWriter, req *http.Request) {
	bts, err := ioutil.ReadAll(req.Body)
	if err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	c := &config.Configs{}
	if err = json.Unmarshal(bts, c); err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	if err := market.UpdateConfigs(c); err != nil {
		logger.Error.Println(err)
		InternalError(w)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func getConfigs(w http.ResponseWriter, req *http.Request) {
	c := market.GetConfigs()
	bts, err := json.Marshal(c.Market.Trader)
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

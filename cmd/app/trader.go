package main

import (
	"encoding/json"
	"net/http"
	"strconv"
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

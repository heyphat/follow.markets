package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
)

func addChatIDs(w http.ResponseWriter, req *http.Request) {
	str, ok := mux.Vars(req)["chat_ids"]
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var cids []int64
	for _, s := range strings.Split(str, ",") {
		if cid, err := strconv.Atoi(s); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		} else {
			cids = append(cids, int64(cid))
		}
	}
	market.AddChatIDs(cids)
	w.WriteHeader(http.StatusOK)
}

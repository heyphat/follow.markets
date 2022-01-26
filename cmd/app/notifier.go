package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

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

func getNotifications(w http.ResponseWriter, req *http.Request) {
	notis := market.GetNotifications()
	if len(notis) == 0 {
		w.WriteHeader(http.StatusOK)
		return
	}
	type Noti struct {
		Ticker   string    `json:"ticker"`
		Strategy string    `json:"strategy"`
		LastSent time.Time `json:"last_sent"`
	}
	out := struct {
		Notis []Noti `json:"notis"`
	}{}
	for k, v := range notis {
		names := strings.Split(k, "-")
		if len(names) < 2 {
			continue
		}
		out.Notis = append(out.Notis, Noti{
			Ticker:   names[0],
			Strategy: names[1],
			LastSent: v,
		})
	}
	sort.Slice(out.Notis, func(i, j int) bool {
		return out.Notis[i].LastSent.After(out.Notis[j].LastSent)
	})
	bts, err := json.Marshal(out)
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

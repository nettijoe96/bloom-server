package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nettijoe96/bloom"
)

var (
	msgs []string // TODO: replace with database
)

func main() {
	populateMsgs()

	r := mux.NewRouter()
	r.HandleFunc("/bloom-request", handleBloom)

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
	}
	http.Handle("/", r)

	log.Fatal(srv.ListenAndServe())
}

func populateMsgs() {
	msgs = make([]string, 0)
	msgs = append(msgs, "test1")
	msgs = append(msgs, "test2")
}

func handleBloom(w http.ResponseWriter, r *http.Request) {

	// used for request
	type bloomReq struct {
		Bloom string `json:"bloom"`
	}

	// used for response
	type msgsResp struct {
		Msgs []string `json:"msgs"`
	}

	var ctx context.Context
	var cancel context.CancelFunc
	// allows for timeout as query param: ?timeout=5s
	timeout, err := time.ParseDuration(r.FormValue("timeout"))
	if err == nil {
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		ctx, cancel = context.WithCancel(context.Background())
	}
	defer cancel()

	// unmarshall json into struct
	dec := json.NewDecoder(r.Body)
	var req bloomReq
	err = dec.Decode(&req)
	if err != nil {
		// 400 error code because couldn't unmarshal into struct
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	// convert hex string
	bs, err := hex.DecodeString(req.Bloom)
	if err != nil {
		// 400 error code because couldn't unmarshal into struct
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	b := bloom.FromBytes(bs)
	msgCh := make(chan []string)
	// go routine that checks against bloom filter
	go func(msgCh chan []string) {
		msgsNeeded := make([]string, 0)
		for _, msg := range msgs {
			if exists, _ := b.ExistsStr(msg); exists {
				msgsNeeded = append(msgsNeeded, msg)
			}
		}
		msgCh <- msgsNeeded
	}(msgCh)

	select {
	case <-ctx.Done():
		// reaches this point if uses cancels request before or user provides timeout as query param and timeout is exceeded
		http.Error(w, "Connection Timed Out", http.StatusRequestTimeout)
	case msgsNeeded := <-msgCh: // wait until all messages are processed to constuct response
		resp := msgsResp{
			Msgs: msgsNeeded,
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(respBytes)
	}
}

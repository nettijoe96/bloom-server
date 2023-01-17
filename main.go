package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/nettijoe96/bloom"
)

var (
	b *bloom.Bloom
)

func main() {
	// TODO: possibly move somewhere better
	b = bloom.NewBloom()

	r := mux.NewRouter()
	r.HandleFunc("/notify", handleNotify)

	srv := &http.Server{
		Handler:      r,
		Addr:         "127.0.0.1:8080",
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
	}
	http.Handle("/", r)

	log.Fatal(srv.ListenAndServe())
}

func handleNotify(w http.ResponseWriter, r *http.Request) {

	// used for request and response
	type notify struct {
		MsgHashes []string `json:"msgHashes"`
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
	var notifyRequest notify
	err = dec.Decode(&notifyRequest)
	if err != nil {
		// 400 error code because couldn't unmarshal into struct
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	msgCh := make(chan []string)
	// go routine that checks against bloom filter
	go func(msgCh chan []string) {
		msgsNeeded := make([]string, 0)
		for _, h := range notifyRequest.MsgHashes {
			if exists, _ := b.ExistsStr(h); !exists {
				msgsNeeded = append(msgsNeeded, h)
			}
		}
		msgCh <- msgsNeeded

	}(msgCh)

	var notifyResp notify
	select {
	case <-ctx.Done():
		// reaches this point if uses cancels request before or user provides timeout as query param and timeout is exceeded
		http.Error(w, "Connection Timed Out", http.StatusRequestTimeout)
	case msgsNeeded := <-msgCh:
		// wait until all messages are processed to constuct response
		notifyResp.MsgHashes = msgsNeeded
		resp, err := json.Marshal(notifyResp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(resp)
	}

}

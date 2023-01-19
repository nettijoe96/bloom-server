package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-openapi/runtime/middleware"
	"github.com/gorilla/mux"
	"github.com/nettijoe96/bloom"
)

var (
	msgs []string
	msgm map[string]bool
)

func main() {
	msgs = make([]string, 0)
	msgm = make(map[string]bool)

	r := mux.NewRouter()
	r.HandleFunc("/bloom-request", handleBloom)
	r.HandleFunc("/publish", handlePublish)
	r.Handle("/swagger.yaml", http.FileServer(http.Dir("./")))
	opts := middleware.SwaggerUIOpts{SpecURL: "/swagger.yaml"}
	sh := middleware.SwaggerUI(opts, nil)
	r.Handle("/docs", sh)

	srv := &http.Server{
		Handler:      r,
		Addr:         "0.0.0.0:8080",
		WriteTimeout: 2 * time.Second,
		ReadTimeout:  2 * time.Second,
	}
	http.Handle("/", r)

	log.Fatal(srv.ListenAndServe())
}

func handleBloom(w http.ResponseWriter, r *http.Request) {

	type bloomEncoding struct {
		Filter string `json:"filter"`
		K      int    `json:"k"`
	}

	// used for request
	type bloomReq struct {
		Bloom bloomEncoding `json:"bloom"`
	}

	// used for response
	type msgsResp struct {
		Messages []string `json:"messages"`
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
	bs, err := hex.DecodeString(req.Bloom.Filter)
	if err != nil {
		// 400 error code because couldn't unmarshal into struct
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	b, err := bloom.FromBytes(bs, req.Bloom.K)
	if err != nil {
		// 400 error code because issue with user inputted bloom filter
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
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
		// reaches this point if user cancels request before or user provides timeout as query param and timeout is exceeded
		w.WriteHeader(http.StatusRequestTimeout)
	case msgsNeeded := <-msgCh: // wait until all messages are processed to constuct response
		resp := msgsResp{
			Messages: msgsNeeded,
		}
		respBytes, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(respBytes)
	}
}

func handlePublish(w http.ResponseWriter, r *http.Request) {

	// used for response
	type msgsReq struct {
		Messages []string `json:"messages"`
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
	var req msgsReq
	err = dec.Decode(&req)
	if err != nil {
		// 400 error code because couldn't unmarshal into struct
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	doneCh := make(chan interface{})
	// go routine that goes through each message and adds it if it doesn't already exist
	go func(doneCh chan interface{}) {
		for _, msg := range req.Messages {
			if exists, ok := msgm[msg]; !ok || !exists {
				msgs = append(msgs, msg)
				msgm[msg] = true
			}
		}
		close(doneCh)
	}(doneCh)

	select {
	case <-ctx.Done():
		// reaches this point if user cancels request before or user provides timeout as query param and timeout is exceeded
		w.WriteHeader(http.StatusRequestTimeout)
	case <-doneCh: // wait until all messages are processed to constuct response
		// no content in response
		w.WriteHeader(http.StatusNoContent)
	}
}

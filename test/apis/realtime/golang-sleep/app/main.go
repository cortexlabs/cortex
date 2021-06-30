package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

func main() {
	var (
		port string
	)

	flag.StringVar(&port, "port", "8080", "port")
	flag.Parse()

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "ok")
	}

	mainHandler := func(w http.ResponseWriter, req *http.Request) {
		sleepTime := req.URL.Query().Get("sleep")
		if sleepTime == "" {
			sleepTime = "1s"
		}
		t, err := time.ParseDuration(sleepTime)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		time.Sleep(t)

		io.WriteString(w, "ok")
	}

	http.HandleFunc("/healthz", helloHandler)
	http.HandleFunc("/", mainHandler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

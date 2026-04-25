// Command echo-upstream is a tiny HTTP server that logs each request's Host
// and Authorization header to stdout and echoes every request header back in
// the response body. Used as a stand-in upstream when smoke-testing the
// proxy locally.
//
// Usage:
//
//	go run ./hack/echo-upstream           # listens on :8081
//	go run ./hack/echo-upstream -addr=:9000
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

func main() {
	addr := flag.String("addr", ":8081", "host:port to listen on")
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("%s %s Host=%s Authorization=%q\n",
			r.Method, r.URL.RequestURI(), r.Host, r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_ = r.Header.Write(w)
	})

	log.Printf("echo-upstream listening on %s", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal(err)
	}
}

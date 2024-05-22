package http

import (
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func ExampleWSServe() {
	r := mux.NewRouter()
	r.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Publishing Custom Message")
	})

	r.HandleFunc("/subscribe", WSServe(&JSONHandler{}, nil))
	srv := http.Server{Handler: r}
	log.Fatal(srv.ListenAndServe())
}

func ExampleWSHandleConn() {
}

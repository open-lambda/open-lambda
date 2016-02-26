package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

type Timestamp struct {
	Time string `json:"startup"`
}

func StartTimer(w http.ResponseWriter, r *http.Request) {
	t := Timestamp{
		Time: fmt.Sprint(time.Now().UTC()),
	}

	b, err := json.Marshal(t)
	if err != nil {
		http.Error(w, "failed to get time", http.StatusInternalServerError)
	} else {
		reader := bytes.NewReader(b)
		reader.WriteTo(w)
	}
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		handler.ServeHTTP(w, r)
	})
}

func main() {
	http.HandleFunc("/runLambda/pausable-start-timer", StartTimer)
	log.Fatal(http.ListenAndServe(":8080", Log(http.DefaultServeMux)))
}

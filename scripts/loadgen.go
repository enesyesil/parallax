package main

import (
	"bytes"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"time"
)

func main() {
	lb := "http://localhost:8080"
	concurrency := 16
	dur := 10 * time.Second
	log.Println("loadgen:", concurrency, "threads for", dur)

	stop := time.Now().Add(dur)
	errc := make(chan error, concurrency)

	for i := 0; i < concurrency; i++ {
		go func() {
			for time.Now().Before(stop) {
				body, _ := json.Marshal(map[string]any{
					"cost":    40 + rand.Intn(60),
					"payload": "x",
				})
				_, err := http.Post(lb+"/submit", "application/json", bytes.NewReader(body))
				if err != nil {
					errc <- err
				}
			}
			errc <- nil
		}()
	}

	for i := 0; i < concurrency; i++ {
		<-errc
	}
	log.Println("done")
}

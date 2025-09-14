package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)


type workIn struct {
	CostMS  int    `json:"cost"`
	Payload string `json:"payload"`
}

func burn(ms int) {
    if ms <= 0 {
        ms = 50
    }
    deadline := time.Now().Add(time.Duration(ms) * time.Millisecond)

    
    buf := make([]byte, 4096)
    h := sha256.New()

    for time.Now().Before(deadline) {
        
        if _, err := io.ReadFull(rand.Reader, buf); err != nil {
            
            panic(fmt.Errorf("crypto/rand read failed: %w", err))
        }
        
        if _, err := h.Write(buf); err != nil {
            panic(err)
        }
    }

    // Consume the digest so the compiler can’t dead-code eliminate the loop
    _ = h.Sum(nil)[0]
}


func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	lbURL := getenv("LB_URL", "http://localhost:8080")
	id := getenv("WORKER_ID", fmt.Sprintf("w-%d", time.Now().UnixNano()))
	port := getenv("WORKER_PORT", "9000")

	// heartbeat register
	go func() {
		for {
			body, _ := json.Marshal(map[string]any{
				"id":   id,
				"addr": "http://localhost:" + port,
			})
			http.Post(lbURL+"/register", "application/json", bytes.NewReader(body))
			time.Sleep(2 * time.Second)
		}
	}()

	slowExtra := 0

	if v := os.Getenv("SLOW_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 { slowExtra = n }
	}




	http.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
		var in workIn
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":"bad json"}`)
			return
		}
		start := time.Now()

		total := in.CostMS + slowExtra
		burn(total)

		
		io.WriteString(w, fmt.Sprintf(`{"ok":true,"took_ms":%d}`, time.Since(start).Milliseconds()))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ok":true}`)
	})

	log.Println("worker", id, "listening on :"+port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

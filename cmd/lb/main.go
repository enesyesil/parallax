package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/enesyesil/parallax/internal/core"
	"github.com/enesyesil/parallax/internal/metrics"
	"github.com/enesyesil/parallax/internal/scheduler"
)

type LB struct {
	reg    *core.Registry
	rr     *scheduler.RRScheduler
	ll     *scheduler.LeastLoad
	p2c    *scheduler.P2C
	policy atomic.Value // one of them "rr" | "ll" | "p2c"
	statsByPolicy map[string]*core.Stats

	stats *core.Stats

	batcher *core.Batcher
}

func (lb *LB) currentPolicy() string {
	if v := lb.policy.Load(); v != nil {
		return v.(string)
	}
	return string(core.PolicyRR)
}

func (lb *LB) choose() *core.WorkerInfo {
	ws := lb.reg.Snapshot()
	switch lb.currentPolicy() {
	case string(core.PolicyLeastLoad):
		return lb.ll.Choose(ws)
	case string(core.PolicyP2C):
		return lb.p2c.Choose(ws)
	default:
		return lb.rr.Choose(ws)
	}
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	port := os.Getenv("LB_PORT")
	if port == "" {
		port = "8080"
	}

	
	registry := prometheus.NewRegistry()
	for _, c := range metrics.Collectors() {
		registry.MustRegister(c)
	}

	lb := &LB{
		reg: core.NewRegistry(),
		rr:  &scheduler.RRScheduler{},
		ll:  &scheduler.LeastLoad{},
		p2c: &scheduler.P2C{},  
	}
	lb.policy.Store(string(core.PolicyRR))

	lb.stats = core.NewStats(10 * time.Second)

	enableBatch := os.Getenv("ENABLE_BATCH") == "1"


	win := 20 * time.Millisecond

	if v := os.Getenv("BATCH_WINDOW_MS"); v != "" {
		if d, err := time.ParseDuration(v + "ms"); err == nil {
			win = d
		}
	}

	maxBatch := 8
	if v := os.Getenv("MAX_BATCH_SIZE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxBatch = n
		}
	}

	
	maxCost := 480
	if v := os.Getenv("MAX_BATCH_COST"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxCost = n
		}
	}


	if enableBatch {
		
		lb.batcher = core.NewBatcher(win, maxBatch, maxCost, func(batch []core.MicroReq) (int, error) {
			
			dst := lb.choose()
			if dst == nil {
				return 0, fmt.Errorf("no workers")
			}
			

			lb.reg.MarkStart(dst.ID)
	


			items := make([]map[string]any, 0, len(batch))


			sumCost := 0
			for _, br := range batch {
				items = append(items, map[string]any{
					"cost":    br.Cost,
					"payload": br.Payload,
				})
				sumCost += br.Cost
			}
	
			body, _ := json.Marshal(map[string]any{"items": items})
			start := time.Now()
			resp, err := http.Post(dst.Addr+"/work_batch", "application/json", bytes.NewReader(body))
			elapsed := time.Since(start)
	
			
			lb.reg.MarkFinish(dst.ID, int(elapsed.Milliseconds()))
	
			if err != nil {
				return 0, err
			}
			defer resp.Body.Close()
	
			
			var out struct {
				OK     bool `json:"ok"`
				Count  int  `json:"count"`
				TookMS int  `json:"took_ms"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&out)
	
			
			metrics.ReqTotal.WithLabelValues("ok").Add(float64(len(batch)))
			metrics.Latency.Observe(elapsed.Seconds())
	
			
			pol := lb.currentPolicy()
			if s := lb.statsByPolicy[pol]; s != nil {
				now := time.Now()
				ls := elapsed.Seconds()
				for range batch {
					s.Add(now, ls)
				}
			}
			// reply took_ms to each item’s waiter
			return int(elapsed.Milliseconds()), nil
		})
		go lb.batcher.Run()
	}


	
	http.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var wi core.WorkerInfo
		if err := json.NewDecoder(r.Body).Decode(&wi); err != nil || wi.ID == "" || wi.Addr == "" {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":"bad worker info"}`)
			return
		}
		lb.reg.Upsert(wi)
		io.WriteString(w, `{"ok":true}`)
	})

	
	http.HandleFunc("/mode", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(405)
			return
		}
		var in struct{ Policy string `json:"policy"` }
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			w.WriteHeader(400)
			return
		}
		switch in.Policy {
		case "rr", "ll", "p2c":
			lb.policy.Store(in.Policy)
			io.WriteString(w, `{"ok":true}`)
		default:
			w.WriteHeader(400)
			io.WriteString(w, `{"error":"unknown policy"}`)
		}
	})

	
	http.HandleFunc("/submit", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		var in struct {
			Cost    int    `json:"cost"`
			Payload string `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			w.WriteHeader(400)
			io.WriteString(w, `{"error":"bad json"}`)
			return
		}

		if lb.batcher != nil {
			
			respCh := make(chan core.MicroResp, 1)
			lb.batcher.Enqueue(core.MicroReq{
				Cost:    in.Cost,
				Payload: in.Payload,
				RespCh:  respCh,
			})
			resp := <-respCh
			if !resp.OK {
				metrics.ReqTotal.WithLabelValues("err").Inc()
				w.WriteHeader(503)
				io.WriteString(w, `{"error":"batch dispatch failed"}`)
				return
			}
			
			io.WriteString(w, fmt.Sprintf(`{"ok":true,"took_ms":%d}`, resp.TookMS))
			return
		}

		dst := lb.choose()
		if dst == nil {
			metrics.ReqTotal.WithLabelValues("drop").Inc()
			w.WriteHeader(503)
			io.WriteString(w, `{"error":"no workers"}`)
			return
		}

		lb.reg.MarkStart(dst.ID)

		body, _ := json.Marshal(map[string]any{"cost": in.Cost, "payload": in.Payload})
		resp, err := http.Post(dst.Addr+"/work", "application/json", bytes.NewReader(body))
		elapsed := time.Since(start)



		
		lb.reg.MarkFinish(dst.ID, int(elapsed.Milliseconds()))

		metrics.ReqTotal.WithLabelValues("ok").Inc()
		metrics.Latency.Observe(elapsed.Seconds())

		if err != nil {
			metrics.ReqTotal.WithLabelValues("err").Inc()
			w.WriteHeader(502)
			io.WriteString(w, `{"error":"worker unreachable"}`)
			return
		}


		defer resp.Body.Close()
		io.Copy(w, resp.Body)

		lb.stats.Add(time.Now(), elapsed.Seconds())

		metrics.ReqTotal.WithLabelValues("ok").Inc()
		metrics.Latency.Observe(elapsed.Seconds())
	})

	

	http.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
        now := time.Now()
        rps, p50, p95, count := lb.stats.Snapshot(now)
        ws := lb.reg.Snapshot()

        type wstat struct {
            ID       string  `json:"id"`
            Addr     string  `json:"addr"`
            InFlight int     `json:"inflight"`
            LoadEMA  float64 `json:"load_ema_ms"`
        }
        resp := struct {
            Policy   string  `json:"policy"`
            WindowS  int     `json:"window_seconds"`
            Samples  int     `json:"samples"`
            RPS      float64 `json:"rps"`
            P50Ms    float64 `json:"p50_ms"`
            P95Ms    float64 `json:"p95_ms"`
            Workers  []wstat `json:"workers"`
        }{
            Policy:  lb.currentPolicy(),
            WindowS: 10,
            Samples: count,
            RPS:     rps,
            P50Ms:   p50 * 1000.0,
            P95Ms:   p95 * 1000.0,
        }

        for _, wkr := range ws {
            resp.Workers = append(resp.Workers, wstat{
                ID:       wkr.ID,
                Addr:     wkr.Addr,
                InFlight: wkr.InFlight,
                LoadEMA:  wkr.LoadEMA,
            })
        }

        w.Header().Set("content-type", "application/json")
        json.NewEncoder(w).Encode(resp)
    })

	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))

	log.Println("LB listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

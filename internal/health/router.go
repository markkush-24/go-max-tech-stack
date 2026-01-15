package health

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

type response struct {
	Status  string            `json:"status"`
	TS      string            `json:"ts"`
	Details map[string]string `json:"details,omitempty"`
}

func LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		_ = json.NewEncoder(w).Encode(response{
			Status: "ok",
			TS:     time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
}

func ReadinessHandler(rd *Readiness, timeout time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		ts := time.Now().UTC().Format(time.RFC3339Nano)

		// lifecycle fail-fast
		if !rd.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(response{
				Status: "not_ready",
				TS:     ts,
				Details: map[string]string{
					"lifecycle": "not_ready",
				},
			})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		details := make(map[string]string)

		for _, c := range rd.checks {
			if err := c.Fn(ctx); err != nil {
				details[c.Name] = err.Error()
			}
			// если дедлайн истёк — нет смысла продолжать
			if ctx.Err() != nil {
				details["timeout"] = ctx.Err().Error()
				break
			}
		}

		if len(details) > 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = json.NewEncoder(w).Encode(response{
				Status:  "not_ready",
				TS:      ts,
				Details: details,
			})
			return
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(response{
			Status: "ok",
			TS:     ts,
		})
	}
}

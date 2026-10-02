package transport

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/manny-ovena/microendpoint-stack/services/endpoints/inventory.adjust/internal"
)

func NewHTTPHandler(svc *internal.Service) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/inventory/adjust", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req internal.AdjustRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		resp, err := svc.Adjust(req)
		if errors.Is(err, internal.ErrNotImplemented) {
			http.Error(w, err.Error(), http.StatusNotImplemented)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		body, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write(body); err != nil {
			log.Printf("write inventory adjustment response: %v", err)
		}
	})

	return mux
}

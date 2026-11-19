package http

import (
	"encoding/json"
	"net/http"
)

// writeJSON คือ helper เขียน response กลับเป็น JSON พร้อมตั้ง header/status code ให้ครบ
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

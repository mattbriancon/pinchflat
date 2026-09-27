package web

import "net/http"

// HealthControllerCheck: json(conn, %{status: "ok"}).
func (s *Server) HealthControllerCheck(w http.ResponseWriter, r *http.Request) {
	s.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

package web

// Port of lib/pinchflat_web/controllers/health_controller.ex.

import "net/http"

// HealthControllerCheck: json(conn, %{status: "ok"}).
func (s *Server) HealthControllerCheck(w http.ResponseWriter, r *http.Request) {
	s.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

package web_test

import (
	"testing"

	"github.com/mattbriancon/pinchflat/internal/web/webtest"
)

func TestHealthController_Check(t *testing.T) {
	t.Run("GET /healthcheck returns ok", func(t *testing.T) {
		c := webtest.New(t)

		res := c.Get("/healthcheck")

		data := res.JSON(t, 200)
		if status, ok := data["status"]; !ok || status != "ok" {
			t.Errorf("expected status 'ok', got %v", status)
		}
	})
}

package handle_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo"
	"github.com/stretchr/testify/assert"

	"github.com/coreycole/go_md/webserver/handle"
)

func TestHealth(t *testing.T) {
	// Setup
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Assertions
	if assert.NoError(t, handle.Health(c)) {
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

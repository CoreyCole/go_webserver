package handle

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

type HealthRes struct {
	Status string `json:"status" form:"status"`
}

func Health(c echo.Context) error {
	res := new(HealthRes)
	res.Status = "ok"
	return c.JSON(http.StatusOK, res)
}

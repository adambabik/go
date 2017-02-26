package middleware

import (
	"github.com/labstack/echo"
	uuid "github.com/satori/go.uuid"
)

// RequestID sets a request id into the request and response headers.
// It also sets is as a c.Logger() prefix.
func RequestID(headerName string) echo.MiddlewareFunc {
	return func(h echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			reqID := uuid.NewV4().String()[0:8]
			if incomingReqID := c.Request().Header.Get(headerName); incomingReqID != "" {
				reqID = incomingReqID
			} else {
				// Set header in request to be available for the logger middleware.
				c.Request().Header.Set(headerName, reqID)
			}

			c.Logger().SetPrefix(reqID)
			c.Logger().Debugf("Set requestid=%s\n", reqID)

			c.Response().Header().Set(headerName, reqID)

			return h(c)
		}
	}
}

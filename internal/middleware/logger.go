package middleware

import (
	"log/slog"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// RequestLogger logs one line per request through slog.
//
// This deliberately uses Echo's own logger rather than wrapping a net/http
// logger with echo.WrapMiddleware. Echo converts a handler's returned error
// into a response *after* any wrapped net/http middleware has already
// returned, so a wrapped logger observes a response that was never written and
// reports status 000 with 0 bytes — every handled 404 logged as a non-answer.
// HandleError below runs the HTTP error handler first, so the status logged is
// the status sent.
func RequestLogger() echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		HandleError:      true,
		LogStatus:        true,
		LogMethod:        true,
		LogURI:           true,
		LogLatency:       true,
		LogResponseSize:  true,
		LogRequestID:     true,
		LogRemoteIP:      true,
		LogError:         true,
		LogProtocol:      true,
		LogUserAgent:     false,
		LogContentLength: false,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency.Round(time.Microsecond)),
				slog.Int64("bytes", v.ResponseSize),
				slog.String("request_id", v.RequestID),
				slog.String("remote_ip", v.RemoteIP),
			}

			// Level follows the status, not the presence of an error value.
			// Echo represents a handled 404 as an error, and logging every
			// mistyped URL at ERROR makes the level meaningless.
			level := slog.LevelInfo
			switch {
			case v.Status >= 500:
				level = slog.LevelError
			case v.Status >= 400:
				level = slog.LevelWarn
			}
			if v.Error != nil {
				attrs = append(attrs, slog.String("error", v.Error.Error()))
			}

			slog.LogAttrs(c.Request().Context(), level, "request", attrs...)
			return nil
		},
	})
}

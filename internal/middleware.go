package server

import (
	"log/slog"
	"net/http"
)

type loggingWriter struct {
	http.ResponseWriter
	status int
}

func (w *loggingWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *loggingWriter) Write(b []byte) (int, error) {
	return w.ResponseWriter.Write(b)
}

func loggingAccess(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			lw := &loggingWriter{ResponseWriter: w}
			next.ServeHTTP(lw, r)

			reqLogger := logger.With(
				slog.Group("http",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.String("remote_addr", r.RemoteAddr),
					slog.Int("status", lw.status),
				))

			reqLogger.Info("request")
		})
	}
}

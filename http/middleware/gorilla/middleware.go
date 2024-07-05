package gorilla

import (
	"net/http"

	"github.com/gorilla/mux"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{w, http.StatusOK}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func TraceMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		route := mux.CurrentRoute(r)
		path, _ := route.GetPathTemplate()

		ctx := otel.GetTextMapPropagator().Extract(r.Context(), propagation.HeaderCarrier(r.Header))
		ctx, span := otel.Tracer("").Start(ctx, r.Method+" "+path, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		newRequest := r.WithContext(ctx)
		newResponseWriter := newResponseWriter(w)

		next.ServeHTTP(newResponseWriter, newRequest)

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", r.Method+" "+r.URL.Path),
			attribute.String("http.method", r.Method),
			attribute.String("http.url", path),
			attribute.String("http.raw.query", r.URL.RawQuery),
			attribute.String("http.route", path),
			attribute.String("http.target", path),
			attribute.String("http.useragent", r.UserAgent()),
			attribute.String("http.host", r.Host),
			attribute.Int("http.status_code", newResponseWriter.statusCode),
		)
	})
}

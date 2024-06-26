package fiber

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type response struct {
	Status int `json:"status,omitempty"`
}

func (r *response) Error() string {
	return "error"
}

func TraceMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routePattern := ""
		for _, route := range c.App().GetRoutes() {
			if fiber.RoutePatternMatch(c.Path(), route.Path) {
				routePattern = route.Path
				break
			}
		}

		ctx, span := otel.Tracer("").Start(c.Context(), c.Method()+" "+routePattern, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		c.SetUserContext(ctx)

		statusCode := 0
		err := c.Next()
		resp, ok := err.(*response)
		if ok {
			statusCode = resp.Status
		}

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", c.Method()+" "+c.Path()),
			attribute.String("http.method", c.Method()),
			attribute.String("http.url", routePattern),
			attribute.String("http.raw.query", string(c.Context().URI().QueryString())),
			attribute.String("http.route", routePattern),
			attribute.String("http.target", routePattern),
			attribute.String("http.useragent", string(c.Context().UserAgent())),
			attribute.String("http.host", string(c.Context().Host())),
			attribute.Int("http.status_code", statusCode),
		)

		return err
	}
}

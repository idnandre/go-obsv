package fiber

import (
	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

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

		return c.Next()
	}
}

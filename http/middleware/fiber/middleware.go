package fiber

import (
	"encoding/json"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type responseStatus struct {
	Status int `json:"status"`
}

func (r *responseStatus) Error() string {
	return "error"
}

type responseCode struct {
	Code int `json:"code"`
}

func (r *responseCode) Error() string {
	return "error"
}

func TraceMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		routePattern := ""
		currentPath := string(c.Context().Path())
		for _, route := range c.App().GetRoutes() {
			if fiber.RoutePatternMatch(currentPath, route.Path) {
				routePattern = route.Path
				break
			}
		}

		ctx, span := otel.Tracer("").Start(c.Context(), string(c.Context().Method())+" "+routePattern, trace.WithSpanKind(trace.SpanKindServer))
		defer span.End()

		c.SetUserContext(ctx)

		statusCode := 0
		err := c.Next()

		if len(c.Context().Response.Body()) > 0 {
			statusCode = c.Context().Response.StatusCode()
		} else {
			resp, ok := err.(interface{})
			if ok {
				respJSON, _ := json.Marshal(resp)
				rspStatus := &responseStatus{}
				rspCode := &responseCode{}
				json.Unmarshal(respJSON, &rspStatus)
				json.Unmarshal(respJSON, &rspCode)

				if rspStatus.Status > 0 {
					statusCode = rspStatus.Status
				} else if rspCode.Code > 0 {
					statusCode = rspCode.Code
				}
			}
		}

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", string(c.Context().Method())+" "+string(c.Context().Path())),
			attribute.String("http.method", string(c.Context().Method())),
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

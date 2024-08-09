package middlewarev2

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/idnandre/gobsv/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type handlerFunc func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error)

func TraceMiddleware(f handlerFunc) handlerFunc {
	return func(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayProxyResponse, error) {
		routPattern := event.RouteKey

		newCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.MapCarrier(event.Headers))
		newCtx, span := otel.Tracer("").Start(newCtx, event.RequestContext.HTTP.Method+" "+routPattern, trace.WithSpanKind(trace.SpanKindServer))
		defer lambda.ForceFlush(newCtx)
		defer span.End()

		response, err := f(newCtx, event)

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", event.RequestContext.HTTP.Method+" "+event.RawPath),
			attribute.String("http.method", event.RequestContext.HTTP.Method),
			attribute.String("http.url", routPattern),
			attribute.String("http.raw.query", event.RawQueryString),
			attribute.String("http.route", routPattern),
			attribute.String("http.target", routPattern),
			attribute.String("http.useragent", event.RequestContext.HTTP.UserAgent),
			attribute.Int("http.status_code", response.StatusCode),
		)

		return response, err

	}
}

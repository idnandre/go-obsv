package middleware

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/idnandre/gobsv/lambda"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

type handlerFunc func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)

func TraceMiddleware(f handlerFunc) handlerFunc {
	return func(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		newCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(event.MultiValueHeaders))
		newCtx, span := otel.Tracer("").Start(newCtx, event.HTTPMethod+" "+event.Path, trace.WithSpanKind(trace.SpanKindServer))
		defer lambda.ForceFlush(newCtx)
		defer span.End()

		response, err := f(newCtx, event)

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", event.HTTPMethod+" "+event.Path),
			attribute.String("http.method", event.HTTPMethod),
			attribute.String("http.url", event.Path),
			attribute.String("http.route", event.Path),
			attribute.String("http.target", event.Path),
			attribute.String("http.useragent", event.RequestContext.Identity.UserAgent),
			attribute.Int("http.status_code", response.StatusCode),
		)

		return response, err

	}
}

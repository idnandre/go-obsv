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
		routPattern := event.Resource

		newCtx := otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(event.MultiValueHeaders))
		newCtx, span := otel.Tracer("").Start(newCtx, event.HTTPMethod+" "+routPattern, trace.WithSpanKind(trace.SpanKindServer))
		defer lambda.ForceFlush(newCtx)
		defer span.End()

		response, err := f(newCtx, event)

		queryStrings := ""
		for key, values := range event.MultiValueQueryStringParameters {
			for _, value := range values {
				queryStrings += key + "=" + value + "&"
			}
		}

		span.SetAttributes(
			attribute.String("span.kind", "server"),
			attribute.String("resource.name", event.HTTPMethod+" "+event.Path),
			attribute.String("http.method", event.HTTPMethod),
			attribute.String("http.url", routPattern),
			attribute.String("http.raw.query", queryStrings),
			attribute.String("http.route", routPattern),
			attribute.String("http.target", routPattern),
			attribute.String("http.useragent", event.RequestContext.Identity.UserAgent),
			attribute.Int("http.status_code", response.StatusCode),
		)

		return response, err

	}
}

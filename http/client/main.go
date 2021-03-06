package main

import (
	"context"
	"io/ioutil"
	"log"
	"time"

	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/semconv"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/trace/jaeger"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// initTracer creates a new trace provider instance and registers it as global trace provider.
func initTracer() func() {
	// Create and install Jaeger export pipeline
	flush, err := jaeger.InstallNewPipeline(
		jaeger.WithCollectorEndpoint("http://localhost:14268/api/traces"),
		jaeger.WithProcess(jaeger.Process{
			ServiceName: "http-client",
			Tags: []attribute.KeyValue{
				attribute.String("version", "1.0"),
			},
		}),
		jaeger.WithSDK(&sdktrace.Config{DefaultSampler: sdktrace.AlwaysSample()}),
	)
	if err != nil {
		log.Fatal(err)
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	return func() {
		flush()
	}
}

func main() {
	fn := initTracer()
	defer fn()

	client := http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}

	ctx := baggage.ContextWithValues(context.Background(),
		attribute.String("username", "donuts"),
	)

	tr := otel.Tracer("example/client")

	ctx, span := tr.Start(ctx, "client http hello demo", trace.WithAttributes(semconv.PeerServiceKey.String("ExampleService")))
	defer span.End()

	req, _ := http.NewRequestWithContext(ctx, "GET", "http://localhost:7777/hello", nil)

	res, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	_, err = ioutil.ReadAll(res.Body)
	_ = res.Body.Close()

	label1 := attribute.KeyValue{
		Key:   attribute.Key("request_id"),
		Value: attribute.StringValue("abc"),
	}
	span.SetAttributes(label1)
	span.SetStatus(codes.Ok, "OK")

	time.Sleep(time.Second * 2)
}

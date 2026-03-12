package otel

import (
	"context"
	"io"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// InitTracerProvider 初始化 TracerProvider，并返回 shutdown 回调。
func InitTracerProvider(serviceName string, out io.Writer) (*sdktrace.TracerProvider, func(context.Context) error, error) {
	exp, err := stdouttrace.New(stdouttrace.WithWriter(out))
	if err != nil {
		return nil, nil, err
	}

	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(semconv.ServiceName(serviceName)),
	)
	if err != nil {
		return nil, nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp, tp.Shutdown, nil
}

package main

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"log"
	"os"
	"sync/atomic"
	"unsafe"
)

// SetLogger overrides the globalLogger with l.
//
// To see Info messages use a logger with `l.V(1).Enabled() == true`
// To see Debug messages use a logger with `l.V(5).Enabled() == true`.
func SetLogger(l logr.Logger) {
	atomic.StorePointer(&globalLogger, unsafe.Pointer(&l))
}

func getLogger() logr.Logger {
	return *(*logr.Logger)(atomic.LoadPointer(&globalLogger))
}

var globalLogger unsafe.Pointer

// Info prints messages about the general state of the API or SDK.
// This should usually be less then 5 messages a minute.
func Info(msg string, keysAndValues ...interface{}) {
	getLogger().V(1).Info(msg, keysAndValues...)
}

// Error prints messages about exceptional states of the API or SDK.
func Error(err error, msg string, keysAndValues ...interface{}) {
	getLogger().Error(err, msg, keysAndValues...)
}

// Debug prints messages about all internal changes in the API or SDK.
func Debug(msg string, keysAndValues ...interface{}) {
	getLogger().V(5).Info(msg, keysAndValues...)
}

func initLogging(v int) func() {
	ctx := context.Background()
	logger := stdr.New(log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile))
	stdr.SetVerbosity(v)
	SetLogger(logger)
	shutdown, err := installExportPipeline(logger)
	if err != nil {
		logger.Error(err, "failed to install export pipeline")
		log.Fatal(err)
	}
	return func() {
		if err := shutdown(ctx); err != nil {
			log.Fatal(err)
		}
	}

}

func installExportPipeline(logger logr.Logger) (func(context.Context) error, error) {

	otel.SetLogger(logger)
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, fmt.Errorf("creating stdout exporter: %w", err)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(newResource()),
	)
	otel.SetTracerProvider(tracerProvider)
	otel.SetLogger(logger)
	return tracerProvider.Shutdown, nil
}

// newResource returns a resource describing this application.
func newResource() *resource.Resource {
	r, _ := resource.Merge(
		resource.Default(),
		resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("http_proxy"),
			semconv.ServiceVersionKey.String("v0.0.0"),
			attribute.String("environment", os.Getenv("ENVIRONMENT"))),
	)

	return r
}

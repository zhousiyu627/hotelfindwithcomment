// Defines a New function that creates a Jaeger tracer using the specified
// service name and host address. The tracer is configured with a constant
// sampler and a reporter, and the function returns the created tracer or
// an error if any occurred during the creation process.
package trace

import (
	"fmt"
	"time"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go/config"
)

// New creates a new Jaeger tracer
// Defined with two parameters: serviceName (the name of the service)
// and host (the host address of the Jaeger agent).
func New(serviceName, host string) (opentracing.Tracer, error) {
	cfg := config.Configuration{
		// All traces are sampled
		Sampler: &config.SamplerConfig{
			Type:  "const",
			Param: 1,
		},
		Reporter: &config.ReporterConfig{
			LogSpans:            false,
			BufferFlushInterval: 1 * time.Second,
			LocalAgentHostPort:  host,
		},
	}

	// Create a new Jaeger tracer based on the provided configuration
	tracer, _, err := cfg.New(serviceName)
	if err != nil {
		return nil, fmt.Errorf("new tracer error: %v", err)
	}
	return tracer, nil
}

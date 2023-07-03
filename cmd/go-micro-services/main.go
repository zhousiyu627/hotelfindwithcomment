// Entry point of a Go language program, responsible for running different microservices
// This is a microservice orchestrator that selects and runs different microservices based
// on command-line parameters, and communicates through gRPC. It also utilizes Jaeger for
// distributed tracing.
package main

// Imports necessary packages including flag, fmt, log, os, as well as some custom packages
import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	frontendsrv "github.com/harlow/go-micro-services/internal/services/frontend"
	geosrv "github.com/harlow/go-micro-services/internal/services/geo"
	profilesrv "github.com/harlow/go-micro-services/internal/services/profile"
	ratesrv "github.com/harlow/go-micro-services/internal/services/rate"
	searchsrv "github.com/harlow/go-micro-services/internal/services/search"
	"github.com/harlow/go-micro-services/internal/trace"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

// Defines a server interface representing the interface of a microservice
type server interface {
	Run(int) error
}

// The main function serves as the entry point of the program. It first
// defines command-line parameters such as port, jaegeraddr, profileaddr,
// etc., and uses flag.Parse() to parse these parameters.
func main() {
	var (
		port        = flag.Int("port", 8080, "The service port")
		jaegeraddr  = flag.String("jaeger", "jaeger:6831", "Jaeger address")
		profileaddr = flag.String("profileaddr", "profile:8080", "Profile service addr")
		geoaddr     = flag.String("geoaddr", "geo:8080", "Geo server addr")
		rateaddr    = flag.String("rateaddr", "rate:8080", "Rate server addr")
		searchaddr  = flag.String("searchaddr", "search:8080", "Search service addr")
	)
	flag.Parse()

	// It calls the trace.New() function to create a Jaeger tracing instance,
	// providing the service name and Jaeger address. If the creation fails,
	// the program logs the error and terminates.
	t, err := trace.New("search", *jaegeraddr)
	if err != nil {
		log.Fatalf("trace new error: %v", err)
	}

	var srv server
	var cmd = os.Args[1]

	// Selects and creates instances of different microservices
	// based on the command-line parameters. Depending on the
	// value of cmd, it initializes the corresponding microservice.
	switch cmd {
	case "geo":
		srv = geosrv.New(t)
	case "rate":
		srv = ratesrv.New(t)
	case "profile":
		srv = profilesrv.New(t)
	case "search":
		srv = searchsrv.New(
			t,
			dial(*geoaddr, t),
			dial(*rateaddr, t),
		)
	case "frontend":
		srv = frontendsrv.New(
			t,
			dial(*searchaddr, t),
			dial(*profileaddr, t),
		)
	default:
		log.Fatalf("unknown cmd: %s", cmd)
	}

	if err := srv.Run(*port); err != nil {
		log.Fatalf("run %s error: %v", cmd, err)
	}
}

// When selecting a microservice, it uses the dial() function
// to create a gRPC client connection. The dial() function takes
// the service address and tracing instance as parameters, creates
// an insecure connection using grpc.WithInsecure(), and adds the
// OpenTracing interceptor.
func dial(addr string, t opentracing.Tracer) *grpc.ClientConn {
	opts := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otgrpc.OpenTracingClientInterceptor(t)),
	}

	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		panic(fmt.Sprintf("ERROR: dial error: %v", err))
	}

	return conn
}

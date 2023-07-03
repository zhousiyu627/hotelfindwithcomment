package geo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/hailocab/go-geoindex"
	"github.com/harlow/go-micro-services/data"
	geo "github.com/harlow/go-micro-services/internal/services/geo/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

const (
	maxSearchRadius  = 10
	maxSearchResults = 5
)

// point represents a hotels's geo location on map
type point struct {
	Pid  string  `json:"hotelId"`
	Plat float64 `json:"lat"`
	Plon float64 `json:"lon"`
}

// Implement Point interface
func (p *point) Lat() float64 { return p.Plat }
func (p *point) Lon() float64 { return p.Plon }
func (p *point) Id() string   { return p.Pid }

// The New function creates a new Geo server instance. It takes an
// opentracing.Tracer as a parameter and initializes the server with a
// new geospatial index (geoidx) created using the newGeoIndex function.
func New(tr opentracing.Tracer) *Geo {
	return &Geo{
		tracer: tr,
		geoidx: newGeoIndex("data/geo.json"),
	}
}

// Server implements the geo service
// storing the geospatial index
// tracing requests
type Geo struct {
	geoidx *geoindex.ClusteringIndex
	tracer opentracing.Tracer
}

// Run starts the server
// Creates a new gRPC server instance, sets the 'unary interceptor'
// for tracing using the 'opentracing' package, registers the Geo
// server implementation with the gRPC server, and starts listening
// for incoming connections on the specified port.
func (s *Geo) Run(port int) error {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(s.tracer),
		),
	)
	geo.RegisterGeoServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	return srv.Serve(lis)
}

// Nearby returns all hotels within a given distance.
// It takes a context and a geo.Request as input and returns a geo.Result and an error.
// It calls the getNearbyPoints method to retrieve the nearby points (hotels) based on
// the provided latitude and longitude. It populates the HotelIds field of the geo.Result
// with the IDs of the nearby hotels and returns the result.
func (s *Geo) Nearby(ctx context.Context, req *geo.Request) (*geo.Result, error) {
	var (
		points = s.getNearbyPoints(ctx, float64(req.Lat), float64(req.Lon))
		res    = &geo.Result{}
	)

	for _, p := range points {
		res.HotelIds = append(res.HotelIds, p.Id())
	}

	return res, nil
}

//	It creates a geoindex.GeoPoint with the given coordinates and calls the KNearest method of
//
// the geospatial index (geoidx) to find the nearest points. It specifies the maximum number
// of search results, the search radius, and a filter function.
func (s *Geo) getNearbyPoints(ctx context.Context, lat, lon float64) []geoindex.Point {
	center := &geoindex.GeoPoint{
		Pid:  "",
		Plat: lat,
		Plon: lon,
	}

	return s.geoidx.KNearest(
		center,
		maxSearchResults,
		geoindex.Km(maxSearchRadius), func(p geoindex.Point) bool {
			return true
		},
	)
}

// newGeoIndex returns a geo index with points loaded
// The newGeoIndex function creates a new geospatial index (geoindex.ClusteringIndex) and
// populates it with points (hotels) loaded from a JSON file. It reads the file using
// data.MustAsset from the go-micro-services/data package, unmarshals the JSON data into
// a slice of point structs, and adds each point to the index using the Add method.
func newGeoIndex(path string) *geoindex.ClusteringIndex {
	var (
		file   = data.MustAsset(path)
		points []*point
	)

	// load geo points from json file
	if err := json.Unmarshal(file, &points); err != nil {
		log.Fatalf("Failed to load hotels: %v", err)
	}

	// add points to index
	index := geoindex.NewClusteringIndex()
	for _, point := range points {
		index.Add(point)
	}

	return index
}

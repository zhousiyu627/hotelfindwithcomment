package profile

import (
	"encoding/json"
	"fmt"
	"log"
	"net"

	"github.com/grpc-ecosystem/grpc-opentracing/go/otgrpc"
	"github.com/harlow/go-micro-services/data"
	profile "github.com/harlow/go-micro-services/internal/services/profile/proto"
	opentracing "github.com/opentracing/opentracing-go"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

// New returns a new server
// It initializes the server with a new tracer and loads the hotel profiles from the JSON file "data/hotels.json"
func New(tr opentracing.Tracer) *Profile {
	return &Profile{
		tracer:   tr,
		profiles: loadProfiles("data/hotels.json"),
	}
}

// Profile implements the profile service
type Profile struct {
	profiles map[string]*profile.Hotel
	tracer   opentracing.Tracer
}

// Run starts the server
func (s *Profile) Run(port int) error {
	srv := grpc.NewServer(
		grpc.UnaryInterceptor(
			otgrpc.OpenTracingServerInterceptor(s.tracer),
		),
	)
	profile.RegisterProfileServer(srv, s)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	return srv.Serve(lis)
}

// GetProfiles returns hotel profiles for requested IDs
// It takes a context and a profile.Request as input and returns a profile.Result and an error.
// It iterates over the hotel IDs in the request and retrieves the corresponding hotel profile
// using the getProfile method. It then adds the profile to the response.
func (s *Profile) GetProfiles(ctx context.Context, req *profile.Request) (*profile.Result, error) {
	res := new(profile.Result)
	for _, id := range req.HotelIds {
		res.Hotels = append(res.Hotels, s.getProfile(id))
	}
	return res, nil
}

func (s *Profile) getProfile(id string) *profile.Hotel {
	return s.profiles[id]
}

// loadProfiles loads hotel profiles from a JSON file.
// Loads the hotel profiles from a JSON file. It reads the file contents
// using data.MustAsset, unmarshals the JSON data into a slice of profile.Hotel
// structs, and builds a map of hotel IDs to profiles.
func loadProfiles(path string) map[string]*profile.Hotel {
	var (
		file   = data.MustAsset(path)
		hotels []*profile.Hotel
	)

	if err := json.Unmarshal(file, &hotels); err != nil {
		log.Fatalf("Failed to load json: %v", err)
	}

	profiles := make(map[string]*profile.Hotel)
	for _, hotel := range hotels {
		profiles[hotel.Id] = hotel
	}
	return profiles
}

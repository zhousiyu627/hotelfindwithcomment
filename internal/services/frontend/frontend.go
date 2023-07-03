package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"

	profile "github.com/harlow/go-micro-services/internal/services/profile/proto"
	search "github.com/harlow/go-micro-services/internal/services/search/proto"
	"github.com/harlow/go-micro-services/internal/trace"
	opentracing "github.com/opentracing/opentracing-go"
	"google.golang.org/grpc"
)

// New returns a new server
// It takes an opentracing.Tracer and two *grpc.ClientConn objects (for search and
// profile services) as parameters and initializes the searchClient, profileClient,
// and tracer fields of the Frontend struct.
func New(t opentracing.Tracer, searchconn, profileconn *grpc.ClientConn) *Frontend {
	return &Frontend{
		searchClient:  search.NewSearchClient(searchconn),
		profileClient: profile.NewProfileClient(profileconn),
		tracer:        t,
	}
}

// Frontend implements frontend service
type Frontend struct {
	searchClient  search.SearchClient
	profileClient profile.ProfileClient
	tracer        opentracing.Tracer
}

// Run the server. It takes a port integer as a parameter and starts the server to
// listen on that port. It creates a new trace.ServeMux using trace.NewServeMux
// (which is a custom implementation for tracing), and then registers two handlers:
// one for serving static files from the "public" directory and another for handling
// requests to the "/hotels" endpoint. Finally, it starts the HTTP server using
// http.ListenAndServe.
func (s *Frontend) Run(port int) error {
	mux := trace.NewServeMux(s.tracer)
	mux.Handle("/", http.FileServer(http.Dir("public")))
	mux.Handle("/hotels", http.HandlerFunc(s.searchHandler))

	return http.ListenAndServe(fmt.Sprintf(":%d", port), mux)
}

// HTTP handler that processes requests to the "/hotels" endpoint. It handles incoming
// requests, performs search and profile operations using the searchClient and
// profileClient, and returns a JSON-encoded response.
func (s *Frontend) searchHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	ctx := r.Context()

	// in/out dates from query params
	// The function retrieves the values of the "inDate" and "outDate" query parameters
	// from the request URL. If either of these parameters is missing or empty, it returns
	// a "Bad Request" error with an appropriate error message.
	inDate, outDate := r.URL.Query().Get("inDate"), r.URL.Query().Get("outDate")
	if inDate == "" || outDate == "" {
		http.Error(w, "Please specify inDate/outDate params", http.StatusBadRequest)
		return
	}

	// search for best hotels
	// The function performs a search for the best hotels by calling the Nearby method
	// of the searchClient (which is a gRPC client for the search service). It passes
	// the context, latitude, longitude, inDate, and outDate as parameters. If an error
	// occurs during the search, it returns a "Internal Server Error" response with the
	// error message.
	searchResp, err := s.searchClient.Nearby(ctx, &search.NearbyRequest{
		Lat:     37.7879,
		Lon:     -122.4075,
		InDate:  inDate,
		OutDate: outDate,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// grab locale from query params or default to en
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "en"
	}

	// hotel profiles
	// The function retrieves the hotel profiles by calling the GetProfiles method
	// of the profileClient (which is a gRPC client for the profile service). It
	// passes the context, the hotel IDs obtained from the search response, and the
	// locale as parameters.
	profileResp, err := s.profileClient.GetProfiles(ctx, &profile.Request{
		HotelIds: searchResp.HotelIds,
		Locale:   locale,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(geoJSONResponse(profileResp.Hotels))
}

// return a geoJSON response that allows google map to plot points directly on map
// https://developers.google.com/maps/documentation/javascript/datalayer#sample_geojson
func geoJSONResponse(hs []*profile.Hotel) map[string]interface{} {
	fs := []interface{}{}

	for _, h := range hs {
		fs = append(fs, map[string]interface{}{
			"type": "Feature",
			"id":   h.Id,
			"properties": map[string]string{
				"name":         h.Name,
				"phone_number": h.PhoneNumber,
			},
			"geometry": map[string]interface{}{
				"type": "Point",
				"coordinates": []float32{
					h.Address.Lon,
					h.Address.Lat,
				},
			},
		})
	}

	return map[string]interface{}{
		"type":     "FeatureCollection",
		"features": fs,
	}
}

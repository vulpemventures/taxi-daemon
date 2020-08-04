package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/apex/gateway"
	"github.com/rs/cors"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/taxiserver"
	t "github.com/vulpemventures/taxi-protobuf/rpc/taxi"
)

func main() {

	server := &taxiserver.Server{} // implements Taxi interface
	twirpServer := t.NewTaxiServer(server, nil)

	// Make a CORS wrapper:
	corsWrapper := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"POST"},
		AllowedHeaders: []string{"Content-Type"},
	})

	// Do the wrapping, and serve it:
	handler := corsWrapper.Handler(twirpServer)

	if config.GetBool(config.LambdaContextKey) {
		log.Fatal(gateway.ListenAndServe("", handler))
		return
	}

	log.Printf("Taxi Listening on %+v", config.GetInt(config.ListeningPortKey))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%+v", config.GetInt(config.ListeningPortKey)), handler))

}

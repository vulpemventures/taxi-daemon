package main

import (
	"log"
	"net/http"

	"github.com/vulpemventures/liquid-taxi/internal/taxiserver"
	liquid_taxi "github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

func main() {

	server := &taxiserver.Server{} // implements Haberdasher interface
	handler := liquid_taxi.NewTaxiServer(server, nil)
	log.Fatal(http.ListenAndServe(":8080", handler))
}

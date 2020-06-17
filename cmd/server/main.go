package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/apex/gateway"
	"github.com/vulpemventures/liquid-taxi/config"
	"github.com/vulpemventures/liquid-taxi/internal/taxiserver"
	t "github.com/vulpemventures/liquid-taxi/rpc/taxi"
)

func main() {

	server := &taxiserver.Server{} // implements Taxi interface
	handler := t.NewTaxiServer(server, nil)

	if config.GetBool(config.LambdaContextKey) {
		log.Fatal(gateway.ListenAndServe(":8080", handler))
		return
	}

	log.Printf("Taxi Listening on %+v", config.GetInt(config.ListeningPortKey))
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%+v", config.GetInt(config.ListeningPortKey)), handler))

}

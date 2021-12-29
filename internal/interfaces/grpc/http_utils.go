package grpc

import (
	"context"
	"crypto/tls"
	"net/http"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/go-acme/lego/v3/challenge/tlsalpn01"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"golang.org/x/net/http2"
	"google.golang.org/grpc"
)

func credsFromCertMagic(email, domain string) (*tls.Config, error) {
	magic := certmagic.NewDefault()
	myACME := certmagic.NewACMEManager(magic, certmagic.ACMEManager{
		Agreed: true,
		Email:  email,
	})
	magic.Issuer = myACME

	// we run this server only for handling HTTP ACME challange
	tempServer := &http.Server{
		Addr: ":80",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if myACME.HandleHTTPChallenge(w, r) {
				return
			}
		}),
	}
	go tempServer.ListenAndServe()
	defer tempServer.Close()

	if err := magic.ManageAsync(
		context.Background(), []string{domain},
	); err != nil {
		return nil, err
	}

	config := &tls.Config{
		NextProtos: []string{"http/1.1", http2.NextProtoTLS, "h2-14"}, // h2-14 is just for compatibility. will be eventually removed.
	}
	config.GetCertificate = magic.GetCertificate
	config.NextProtos = append(config.NextProtos, tlsalpn01.ACMETLS1Protocol)

	return config, nil
}

func router(
	grpcServer *grpc.Server,
	grpcWebServer *grpcweb.WrappedGrpcServer,
	grpcGateway http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
			grpcGateway.ServeHTTP(w, r)
			return
		}
		if isGrpcWebRequest(r) {
			grpcWebServer.ServeHTTP(w, r)
			return
		}
		grpcServer.ServeHTTP(w, r)
	})
}

func isGrpcWebRequest(req *http.Request) bool {
	return isValidGrpcWebOptionRequest(req) || isValidGrpcWebRequest(req)
}

func isValidGrpcWebRequest(req *http.Request) bool {
	return req.Method == http.MethodPost && isValidGrpcContentTypeHeader(req.Header.Get("content-type"))
}

func isValidGrpcContentTypeHeader(contentType string) bool {
	return strings.HasPrefix(contentType, "application/grpc-web-text") ||
		strings.HasPrefix(contentType, "application/grpc-web")
}

func isValidGrpcWebOptionRequest(req *http.Request) bool {
	accessControlHeader := req.Header.Get("Access-Control-Request-Headers")
	return req.Method == http.MethodOptions &&
		strings.Contains(accessControlHeader, "x-grpc-web") &&
		strings.Contains(accessControlHeader, "content-type")
}

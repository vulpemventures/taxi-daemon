package grpc

import (
	"net/http"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
)

func certmagicConfig(email, domain string) (*certmagic.Config, error) {
	magic := certmagic.NewDefault()
	myACME := certmagic.NewACMEManager(magic, certmagic.ACMEManager{
		Agreed: true,
		Email:  email,
	})
	magic.Issuer = myACME

	if err := magic.ManageSync([]string{domain}); err != nil {
		return nil, err
	}

	return magic, nil
}

func router(
	grpcServer *grpc.Server,
	grpcWebServer *grpcweb.WrappedGrpcServer,
	grpcGateway http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isHttpRequest(r) {
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

func isHttpRequest(req *http.Request) bool {
	return strings.ToLower(req.Method) == "get" ||
		strings.Contains(req.Header.Get("Content-Type"), "application/json")
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

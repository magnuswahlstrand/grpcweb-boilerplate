// Copyright 2017 Johan Brandhorst. All Rights Reserved.
// See LICENSE for licensing terms.

package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"

	"github.com/kyeett/grpcweb-boilerplate/backend"
	"github.com/kyeett/grpcweb-boilerplate/proto/server"
)

var logger *logrus.Logger

func init() {
	logger = logrus.StandardLogger()
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
		DisableSorting:  true,
	})
	// Should only be done from init functions
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(logger.Out, logger.Out, logger.Out))
}

func main() {
	gs := grpc.NewServer()
	server.RegisterBackendServer(gs, &backend.Backend{})

	wrappedServer := grpcweb.WrapServer(gs,
		grpcweb.WithWebsockets(true),
		grpcweb.WithCorsForRegisteredEndpointsOnly(false),
		grpcweb.WithWebsocketOriginFunc(func(req *http.Request) bool { return true }),
		grpcweb.WithOriginFunc(func(origin string) bool { return true }),
	)

	handler := func(resp http.ResponseWriter, req *http.Request) {

		// log.Println(req)
		log.Println(gs.GetServiceInfo())
		log.Println("Trolo:", req.ProtoMajor,
			wrappedServer.IsAcceptableGrpcCorsRequest(req),
			websocket.IsWebSocketUpgrade(req),
			strings.Contains(req.Header.Get("Content-Type"), "application/grpc"))
		log.Println(req.URL)
		log.Println(req.ProtoMajor == 2, strings.Contains(req.Header.Get("Content-Type"), "application/grpc"),
			websocket.IsWebSocketUpgrade(req))

		log.Println()

		if req.Method == "OPTIONS" {
			allowCors(resp, req)
			return
		}

		// Redirect gRPC and gRPC-Web requests to the gRPC-Web Websocket Proxy server

		// log.Println("Handle!", req)
		// Redirect gRPC and gRPC-Web requests to the gRPC-Web Websocket Proxy server
		log.Println(req.Header)

		wrappedServer.ServeHTTP(resp, req)
		if strings.Contains(req.Header.Get("Content-Type"), "application/grpc") || websocket.IsWebSocketUpgrade(req) {
			log.Println("In here!")
		} else {
			log.Println("Serve files!", req)
			// Serve the GopherJS client
			// folderReader(gzipped.FileServer(bundle.Assets)).ServeHTTP(resp, req)
		}
	}

	addr := "localhost:10000"
	httpsSrv := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(handler),
		// Some security settings
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
		TLSConfig: &tls.Config{
			PreferServerCipherSuites: true,
			CurvePreferences: []tls.CurveID{
				tls.CurveP256,
				tls.X25519,
			},
		},
	}

	logger.Info("Serving on https://" + addr)
	logger.Fatal(httpsSrv.ListenAndServe())
	// logger.Fatal(httpsSrv.ListenAndServeTLS("./cert.pem", "./key.pem"))
}

func folderReader(fn http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		if strings.HasSuffix(req.URL.Path, "/") {
			// Use contents of index.html for directory, if present.
			req.URL.Path = path.Join(req.URL.Path, "index.html")
		}
		fn.ServeHTTP(w, req)
	}
}

func allowCors(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
	w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, x-grpc-web")
}

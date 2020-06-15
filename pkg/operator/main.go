/*
Copyright 2020 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/cortexlabs/cortex/pkg/operator/pb"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"context"

	"github.com/cortexlabs/cortex/pkg/lib/exit"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/endpoints"
	"github.com/cortexlabs/cortex/pkg/operator/operator"

	"github.com/gorilla/mux"
	"google.golang.org/grpc"
)

// option define server launch option
type option struct {
	// Enable security
	TlsCertFile string
	TlsKeyFile  string
	// ClientCAFile
	ClientCAFile string
	// GRPCAddress is the IP address and port for the grpc server to serve on,
	// default is ""
	GRPCAddress string
	// HTTPAddress is the IP address and port for the http server to serve on,
	// defaulting to 0.0.0.0:8888
	HTTPAddress string
	// grpc server
	grpcServer   *grpc.Server

	// http server
	httpServer *http.Server

}

func NewOption() *option {
	return &option{}
}

func (op *option) Validate() error {
	if len(op.TlsCertFile) * len(op.TlsKeyFile) != 0 {
		return fmt.Errorf("cert/key file must both be set")
	}
	return nil
}

func main() {
	o := NewOption()
	flag.StringVar(&o.TlsCertFile, "tls-cert-file", "", ""+
		"File containing the default x509 Certificate for HTTPS. (CA cert, if any, concatenated "+
		"after server cert). ")
	flag.StringVar(&o.TlsKeyFile, "tls-private-key-file", "", "File containing the default x509 private key matching --tls-cert-file.")
	flag.StringVar(&o.ClientCAFile, "tls-ca-file", "", "clientCAFile is the path to a PEM-encoded certificate bundle."+
		"If set, any request presenting a client certificate signed by one of the authorities in the bundle is authenticated with a username corresponding to the CommonName, "+
		"and groups corresponding to the Organization in the client certificate.")
	// service address config
	flag.StringVar(&o.GRPCAddress, "grpc-addr", "", "the IP address and port for the grpc server")
	flag.StringVar(&o.HTTPAddress, "http-addr", "0.0.0.0:8888", "the IP address and port for the http server")
	if err := o.Validate(); err != nil {
		exit.Error(err)
	}
	var tlsConfig *tls.Config
	// build tlsConfig
	if o.TlsCertFile != "" && o.TlsKeyFile != "" {
		crt, err := tls.LoadX509KeyPair(o.TlsCertFile, o.TlsKeyFile)
		if err != nil {
			exit.Error(err)
		}
		tlsConfig = &tls.Config{Certificates: []tls.Certificate{crt}}

		if o.ClientCAFile != "" {
			cas := x509.NewCertPool()
			ca, err := ioutil.ReadFile(o.ClientCAFile)
			if err != nil {
				exit.Error(err)
			}
			cas.AppendCertsFromPEM(ca)
			tlsConfig.ClientCAs = cas
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
	}
	// exit with SIGINT and SIGTERM
	ctx := context.Background()
	ctx = withSignals(ctx, os.Interrupt, syscall.SIGTERM)

	if err := config.Init(); err != nil {
		exit.Error(err)
	}

	if err := operator.Init(); err != nil {
		exit.Error(err)
	}

	if o.GRPCAddress != "" {
		ln, err := net.Listen("tcp", o.GRPCAddress)
		if err != nil {
			exit.Error(err)
		}
		if tlsConfig != nil {
			opts := []grpc.ServerOption{grpc.Creds(credentials.NewTLS(tlsConfig))}
			o.grpcServer = grpc.NewServer(opts...)
		} else {
			o.grpcServer = grpc.NewServer()
		}
		go func(){
			log.Print("grpc server starting\n")
			pb.RegisterEndPointServiceServer(o.grpcServer, endpoints.NewEndpoint())
			if err := o.grpcServer.Serve(ln); err != nil {
				log.Fatalf("grpc server failed with %v", err)
			}
		}()
	}

	if o.HTTPAddress != "" {
		ln, err := net.Listen("tcp", o.HTTPAddress)
		if err != nil {
			exit.Error(err)
		}
		o.httpServer.Handler = installRouter()
		if tlsConfig != nil {
			o.httpServer.TLSConfig = tlsConfig
			go func(){
				log.Print("https server starting\n")
				if err := o.httpServer.ServeTLS(ln, "", ""); err != nil {
					log.Fatalf("https server failed with %v", err)
				}
			}()
		} else {
			log.Print("http server starting\n")
			if err := o.httpServer.Serve(ln); err != nil {
				log.Fatalf("http server failed with %v", err)
			}
		}
	}

	<-ctx.Done()
}

func installRouter() http.Handler {
	router := mux.NewRouter()

	routerWithoutAuth := router.NewRoute().Subrouter()
	routerWithoutAuth.Use(endpoints.PanicMiddleware)
	routerWithoutAuth.HandleFunc("/verifycortex", endpoints.VerifyCortex).Methods("GET")

	routerWithAuth := router.NewRoute().Subrouter()

	routerWithAuth.Use(endpoints.PanicMiddleware)
	routerWithAuth.Use(endpoints.ClientIDMiddleware)
	routerWithAuth.Use(endpoints.APIVersionCheckMiddleware)
	routerWithAuth.Use(endpoints.AuthMiddleware)

	routerWithAuth.HandleFunc("/info", endpoints.Info).Methods("GET")
	routerWithAuth.HandleFunc("/deploy", endpoints.Deploy).Methods("POST")
	routerWithAuth.HandleFunc("/refresh/{apiName}", endpoints.Refresh).Methods("POST")
	routerWithAuth.HandleFunc("/delete/{apiName}", endpoints.Delete).Methods("DELETE")
	routerWithAuth.HandleFunc("/get", endpoints.GetAPIs).Methods("GET")
	routerWithAuth.HandleFunc("/get/{apiName}", endpoints.GetAPI).Methods("GET")
	routerWithAuth.HandleFunc("/logs/{apiName}", endpoints.ReadLogs)
	return router
}

// withSignals returns a context that is canceled with any signal in sigs.
func withSignals(ctx context.Context, sigs ...os.Signal) context.Context {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, sigs...)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		defer cancel()
		select {
		case <-ctx.Done():
			return
		case <-sigCh:
			return
		}
	}()
	return ctx
}
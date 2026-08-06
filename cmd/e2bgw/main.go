// Copyright 2026 The xuanji authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// e2bgw is the E2B-protocol gateway: it translates the E2B REST surface
// (POST /sandboxes, pause/resume, ...) into ateapi Control gRPC calls, so
// unmodified E2B SDKs can drive substrate actors. Data-plane endpoints
// (/execute, /files) are answered with a redirect to the actor's atenet
// domain (wired up in migration milestone M4).
//
// Part of the xuanji branch additions (see XUANJI.md); acceptance assets in
// contrib/e2b-e2e/.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/agent-substrate/substrate/cmd/e2bgw/internal/server"
	"github.com/agent-substrate/substrate/internal/ateapiauth"
	"github.com/agent-substrate/substrate/pkg/proto/ateapipb"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

var (
	listenAddress = pflag.String("listen-address", "0.0.0.0:8080", "Address for the E2B REST API")

	ateAPIConnSpec   = pflag.String("ateapi-conn-spec", "dns:///api.ate-system.svc:443", "ateapi gRPC target")
	ateapiCA         = pflag.String("ateapi-ca-file", "", "PEM CA bundle for the ateapi server certificate")
	ateapiServerName = pflag.String("ateapi-server-name", "", "Expected ateapi TLS server name (SNI)")
	ateapiTokenFile  = pflag.String("ateapi-token-file", "", "Bearer token file for ateapi (Kubernetes SA token)")
	ateapiClientCert = pflag.String("ateapi-client-cred-bundle", "", "PEM credential bundle for ateapi client cert auth")

	jwtSecretFile     = pflag.String("jwt-secret-file", "", "File holding the HS256 secret that signs E2B API keys (JWTs)")
	templateNamespace = pflag.String("template-namespace", "ate-wasm", "Kubernetes namespace holding the ActorTemplates that E2B templateIDs refer to")
	actorDomain       = pflag.String("actor-domain", "", "atenet actor domain suffix (e.g. actors.resources.example.com); used for data-plane redirects")
)

func main() {
	pflag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	if err := run(context.Background()); err != nil {
		slog.Error("e2bgw exiting", slog.Any("err", err))
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	secret, err := os.ReadFile(*jwtSecretFile)
	if err != nil {
		return fmt.Errorf("reading --jwt-secret-file: %w", err)
	}

	dialOpts, err := ateapiauth.DialOptions(ateapiauth.ClientConfig{
		CAFile:           *ateapiCA,
		ServerName:       *ateapiServerName,
		TokenFile:        *ateapiTokenFile,
		ClientCredBundle: *ateapiClientCert,
	})
	if err != nil {
		return fmt.Errorf("building ateapi dial options: %w", err)
	}
	conn, err := grpc.NewClient(*ateAPIConnSpec, dialOpts...)
	if err != nil {
		return fmt.Errorf("creating ateapi grpc client: %w", err)
	}
	defer conn.Close()

	srv := server.New(server.Config{
		Control:           ateapipb.NewControlClient(conn),
		JWTSecret:         secret,
		TemplateNamespace: *templateNamespace,
		ActorDomain:       *actorDomain,
	})

	lis, err := net.Listen("tcp", *listenAddress)
	if err != nil {
		return fmt.Errorf("listening on %s: %w", *listenAddress, err)
	}
	slog.Info("e2bgw serving", slog.String("address", *listenAddress))
	httpServer := &http.Server{Handler: srv, BaseContext: func(net.Listener) context.Context { return ctx }}
	return httpServer.Serve(lis)
}

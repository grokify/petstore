// Command petstore-server serves the Petstore API from memory, optionally
// as an AAuth resource.
package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/grokify/petstore/server"
)

func main() {
	if err := newRoot().Execute(); err != nil {
		os.Exit(1)
	}
}

type options struct {
	listen string
	auth   string
	url    string
	trust  []string
	caFile string
}

func newRoot() *cobra.Command {
	o := &options{}
	cmd := &cobra.Command{
		Use:   "petstore-server",
		Short: "Serve the Petstore API, optionally as an AAuth resource",
		Long: `Serve every operation in the Petstore OpenAPI definition from memory,
seeded with pets.

With --auth aauth the server is an AAuth resource: each operation requires
the access its x-aauth annotation names (the agent's identity, the person,
or an auth token for a scope), and the server accepts person and auth
tokens only from the person servers given with --trust.

Run behind the aauth-go sandbox:

  petstore-server --listen 127.0.0.1:8080 --auth aauth \
    --url https://petstore.aauth.test --trust https://ps.aauth.test \
    --ca-file ~/.aauth-sandbox/ca.pem
  aauth-sandbox serve --upstream petstore=http://127.0.0.1:8080`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			h, err := o.handler()
			if err != nil {
				return err
			}
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop()
			slog.Info("petstore listening", "addr", o.listen, "auth", o.auth, "spec", "/openapi.json")
			if err := server.Serve(ctx, o.listen, h); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.listen, "listen", ":8080", "address to listen on (plain HTTP; put TLS in front, as the sandbox does)")
	f.StringVar(&o.auth, "auth", "none", "none serves every operation to anyone; aauth makes the server an AAuth resource")
	f.StringVar(&o.url, "url", "", "the resource's public URL and AAuth identifier, such as https://petstore.aauth.test")
	f.StringSliceVar(&o.trust, "trust", nil, "person server whose tokens to accept (repeatable; required with --auth aauth)")
	f.StringVar(&o.caFile, "ca-file", "", "PEM file of an extra root CA for fetching issuers' keys, such as the sandbox CA")
	return cmd
}

func (o *options) handler() (http.Handler, error) {
	cfg := server.Config{PublicURL: o.url}
	switch o.auth {
	case "none":
	case "aauth":
		gc := server.GuardConfig{URL: o.url, Trust: o.trust}
		if o.caFile != "" {
			pem, err := os.ReadFile(o.caFile)
			if err != nil {
				return nil, fmt.Errorf("--ca-file: %w", err)
			}
			gc.RootCAsPEM = pem
		}
		g, err := server.NewGuard(gc)
		if err != nil {
			return nil, err
		}
		cfg.Guard = g
	default:
		return nil, fmt.Errorf("--auth must be none or aauth, not %q", o.auth)
	}
	return server.New(cfg)
}

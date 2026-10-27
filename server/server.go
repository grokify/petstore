// Package server serves the Petstore API from its OpenAPI definition,
// with an in-memory store. With a guard it is an AAuth resource: each
// operation requires the access level its x-aauth annotation names.
package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	aauth "github.com/aauth-dev/aauth-go"

	"github.com/grokify/petstore"
)

// Config configures a Petstore server.
type Config struct {
	// Spec is the OpenAPI document (default petstore.OpenAPI).
	Spec []byte
	// Store holds the data (default NewSeededStore()).
	Store *Store
	// Guard, when set, makes the server an AAuth resource; see NewGuard.
	// Nil serves every operation to anyone.
	Guard *aauth.ResourceGuard
	// PublicURL is the origin the served spec names, such as
	// https://petstore.aauth.test. Empty uses the request's own.
	PublicURL string
}

// New returns the server's handler. It fails when an operation in the
// spec has no handler.
func New(cfg Config) (http.Handler, error) {
	if cfg.Spec == nil {
		cfg.Spec = petstore.OpenAPI
	}
	if cfg.Store == nil {
		cfg.Store = NewSeededStore()
	}
	spec, err := ParseSpec(cfg.Spec)
	if err != nil {
		return nil, err
	}
	table := (&handlers{store: cfg.Store}).table()
	mux := http.NewServeMux()
	var missing []string
	for _, op := range spec.Operations {
		h, ok := table[op.ID]
		if !ok {
			missing = append(missing, op.ID)
			continue
		}
		var handler http.Handler = h
		if cfg.Guard != nil {
			handler = cfg.Guard.Require(op.Access, h)
		}
		mux.Handle(op.Method+" "+spec.BasePath+op.Path, handler)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("petstore: no handler for %s", strings.Join(missing, ", "))
	}
	for path, asJSON := range map[string]bool{"/openapi.yaml": false, "/openapi.json": true} {
		mux.HandleFunc("GET "+path, serveSpec(spec, cfg.PublicURL, asJSON))
	}
	var wellKnown http.Handler
	if cfg.Guard != nil {
		wellKnown = cfg.Guard.WellKnown()
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if wellKnown != nil && (strings.HasPrefix(r.URL.Path, "/.well-known/") || r.URL.Path == "/jwks.json") {
			wellKnown.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusNotFound, "no operation at "+r.Method+" "+r.URL.Path)
	})
	return mux, nil
}

func serveSpec(spec *Spec, publicURL string, asJSON bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := publicURL
		if origin == "" {
			origin = requestOrigin(r)
		}
		b, err := spec.Document(origin, asJSON)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if asJSON {
			w.Header().Set("Content-Type", "application/json")
		} else {
			w.Header().Set("Content-Type", "application/yaml")
		}
		// A failed write is a closed connection the client already sees.
		_, _ = w.Write(b) //nolint:errcheck // see above
	}
}

func requestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// GuardConfig configures NewGuard.
type GuardConfig struct {
	// URL is the resource's identifier, such as https://petstore.aauth.test.
	URL string
	// Trust lists the person servers whose person and auth tokens the
	// resource accepts.
	Trust []string
	// HTTPClient fetches issuers' keys (default: one trusting the system
	// roots and RootCAs).
	HTTPClient *http.Client
	// RootCAsPEM, when set, holds certificates trusted as well as the
	// system roots, such as the aauth-go sandbox CA.
	RootCAsPEM []byte
	// Spec supplies the scope descriptions (default petstore.OpenAPI).
	Spec []byte
}

// NewGuard returns a ResourceGuard for the Petstore: a fresh signing key
// for resource tokens (served at /jwks.json), the spec's scope
// descriptions in its metadata, and the trusted person servers.
func NewGuard(cfg GuardConfig) (*aauth.ResourceGuard, error) {
	if cfg.URL == "" {
		return nil, errors.New("petstore: the AAuth resource identifier (URL) is required")
	}
	if len(cfg.Trust) == 0 {
		return nil, errors.New("petstore: trust at least one person server")
	}
	if cfg.Spec == nil {
		cfg.Spec = petstore.OpenAPI
	}
	spec, err := ParseSpec(cfg.Spec)
	if err != nil {
		return nil, err
	}
	hc := cfg.HTTPClient
	if hc == nil {
		hc, err = clientTrusting(cfg.RootCAsPEM)
		if err != nil {
			return nil, err
		}
	}
	key, err := aauth.GenerateKey(aauth.AlgEd25519)
	if err != nil {
		return nil, err
	}
	return aauth.NewResourceGuard(aauth.ResourceGuardConfig{
		Resource:    strings.TrimSuffix(cfg.URL, "/"),
		Key:         key,
		Verify:      aauth.TokenVerifyOptions{Resolver: aauth.NewJWKSResolver(hc)},
		TrustIssuer: aauth.TrustIssuers(cfg.Trust...),
		Metadata: aauth.ResourceMetadata{
			ServerMetadata:    aauth.ServerMetadata{Name: "Petstore"},
			ScopeDescriptions: spec.Scopes,
		},
	})
}

func clientTrusting(extraPEM []byte) (*http.Client, error) {
	tr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return nil, errors.New("petstore: http.DefaultTransport is not an *http.Transport")
	}
	tr = tr.Clone()
	if len(extraPEM) > 0 {
		roots, err := x509.SystemCertPool()
		if err != nil || roots == nil {
			roots = x509.NewCertPool() // minimal containers have no system pool
		}
		if !roots.AppendCertsFromPEM(extraPEM) {
			return nil, errors.New("petstore: no certificates in the extra root CA PEM")
		}
		tr.TLSClientConfig = &tls.Config{RootCAs: roots, MinVersion: tls.VersionTLS12}
	}
	return &http.Client{Timeout: 30 * time.Second, Transport: tr}, nil
}

// Serve runs h on addr until ctx ends.
func Serve(ctx context.Context, addr string, h http.Handler) error {
	srv := &http.Server{Addr: addr, Handler: h, ReadHeaderTimeout: 10 * time.Second}
	errs := make(chan error, 1)
	go func() { errs <- srv.ListenAndServe() }()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdown)
	}
}

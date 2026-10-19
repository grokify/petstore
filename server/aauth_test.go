package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	aauth "github.com/aauth-dev/aauth-go"
	"github.com/aauth-dev/aauth-go/agentprovider"
	"github.com/aauth-dev/aauth-go/sandbox"
)

const petstoreURL = "https://petstore.aauth.test"

// aauthWorld runs the Petstore as an AAuth resource behind the aauth-go
// sandbox, at https://petstore.aauth.test, and returns the sandbox and an
// HTTP client for an agent enrolled at its agent provider.
func aauthWorld(t *testing.T) (*sandbox.Sandbox, *http.Client) {
	t.Helper()
	var h http.Handler
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { h.ServeHTTP(w, r) }))
	t.Cleanup(upstream.Close)

	sb, err := sandbox.New(sandbox.Config{StateDir: t.TempDir(), Upstreams: map[string]string{"petstore": upstream.URL}},
		slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	sb.SetAddr(ln.Addr().String())
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sb.Serve(ctx, ln) }()
	t.Cleanup(func() {
		cancel()
		if err := <-done; err != nil {
			t.Error(err)
		}
	})
	cfg := sb.Config()

	g, err := NewGuard(GuardConfig{URL: petstoreURL, Trust: []string{cfg.URL(sandbox.RolePS)}, HTTPClient: sb.Client("petstore")})
	if err != nil {
		t.Fatal(err)
	}
	if h, err = New(Config{Guard: g, PublicURL: petstoreURL}); err != nil {
		t.Fatal(err)
	}

	hc := sb.Client("agent")
	key, err := aauth.GenerateKey(aauth.AlgEd25519)
	if err != nil {
		t.Fatal(err)
	}
	ap := &agentprovider.Client{BaseURL: cfg.URL(sandbox.RoleAP), Paths: agentprovider.Paths{Token: sandbox.PathEnroll}, HTTPClient: hc}
	agent, err := ap.Enroll(ctx, key, nil)
	if err != nil {
		t.Fatal(err)
	}
	ps := aauth.NewPSClient(agent.PS, agent)
	ps.HTTPClient = hc
	tr := aauth.NewTransport(agent, ps)
	tr.Base = hc.Transport
	tr.ResourceVerify = aauth.TokenVerifyOptions{Resolver: aauth.NewJWKSResolver(hc)}
	tr.Capabilities = []string{aauth.CapabilityInteraction}
	return sb, &http.Client{Transport: tr}
}

func agentCall(t *testing.T, hc *http.Client, method, path string, in, out any) int {
	t.Helper()
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(b)
	}
	req, err := http.NewRequest(method, petstoreURL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := hc.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			t.Error(err)
		}
	}()
	if out != nil && res.StatusCode == http.StatusOK {
		if err := json.NewDecoder(res.Body).Decode(out); err != nil {
			t.Fatal(err)
		}
	}
	return res.StatusCode
}

func TestAAuthPetstore(t *testing.T) {
	sb, hc := aauthWorld(t)

	// Agent identity: browse the catalog.
	var pets []Pet
	expect(t, "find available", agentCall(t, hc, "GET", "/v2/pet/findByStatus?status=available", nil, &pets), 200)
	if len(pets) != 4 {
		t.Fatalf("%d available pets", len(pets))
	}
	// Consent for write:orders: place an order.
	var o Order
	expect(t, "order Fido", agentCall(t, hc, "POST", "/v2/store/order", Order{PetID: 1}, &o), 200)
	// The person's own order, with the auth token already held.
	expect(t, "get the order", agentCall(t, hc, "GET", "/v2/store/order/1", nil, &o), 200)
	// Consent for write:pets as well: add a pet.
	var p Pet
	expect(t, "add a pet", agentCall(t, hc, "POST", "/v2/pet", Pet{Name: "Nemo", PhotoURLs: []string{}}, &p), 200)
	// write:orders is kept: cancel without asking again.
	expect(t, "cancel", agentCall(t, hc, "DELETE", "/v2/store/order/1", nil, nil), 200)

	var consents int
	for _, e := range sb.Trace().Exchanges() {
		if e.To == "ps" && e.Response.Status == http.StatusAccepted {
			consents++
		}
	}
	if consents != 2 {
		t.Errorf("%d consent prompts, want 2 (write:orders, then write:pets)", consents)
	}
}

func TestAAuthPetstoreChallengesAndMetadata(t *testing.T) {
	sb, _ := aauthWorld(t)
	plain := sb.Client("client")
	res, err := plain.Get(petstoreURL + "/v2/pet/1")
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if r, err := aauth.ParseRequirement(res.Header.Get(aauth.HeaderRequirement)); err != nil || res.StatusCode != 401 || r.Requirement != aauth.RequirementAgentToken {
		t.Fatalf("unsigned: %d %q", res.StatusCode, res.Header.Get(aauth.HeaderRequirement))
	}
	var md aauth.ResourceMetadata
	if err := aauth.FetchMetadata(context.Background(), plain, petstoreURL, aauth.WellKnownResource, &md); err != nil {
		t.Fatal(err)
	}
	if md.Issuer != petstoreURL || md.ScopeDescriptions["write:orders"] == "" || md.JWKSURI != petstoreURL+"/jwks.json" {
		t.Fatalf("metadata %+v", md)
	}
	res, err = plain.Get(petstoreURL + "/openapi.json")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Servers []struct{ URL string } `json:"servers"`
	}
	err = json.NewDecoder(res.Body).Decode(&doc)
	if cerr := res.Body.Close(); err == nil {
		err = cerr
	}
	if err != nil || len(doc.Servers) != 1 || doc.Servers[0].URL != petstoreURL+"/v2" {
		t.Fatalf("spec servers %+v, %v", doc.Servers, err)
	}
}

func TestNewGuardErrors(t *testing.T) {
	if _, err := NewGuard(GuardConfig{Trust: []string{"https://ps.example"}}); err == nil {
		t.Error("a guard without a URL was created")
	}
	if _, err := NewGuard(GuardConfig{URL: petstoreURL}); err == nil {
		t.Error("a guard trusting no person server was created")
	}
	if _, err := NewGuard(GuardConfig{URL: petstoreURL, Trust: []string{"https://ps.example"}, RootCAsPEM: []byte("not a certificate")}); err == nil {
		t.Error("a CA file without certificates was accepted")
	}
}

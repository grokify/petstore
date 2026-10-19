package server

import (
	"strings"
	"testing"

	aauth "github.com/aauth-dev/aauth-go"

	"github.com/grokify/petstore"
)

func TestEmbeddedSpec(t *testing.T) {
	s, err := ParseSpec(petstore.OpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	if s.BasePath != "/v2" || len(s.Operations) != 20 {
		t.Fatalf("base %q, %d operations", s.BasePath, len(s.Operations))
	}
	levels := map[string]aauth.Access{}
	for _, op := range s.Operations {
		levels[op.ID] = op.Access
	}
	for id, want := range map[string]aauth.Access{
		"findPetsByStatus": {Level: aauth.AccessAgent},
		"getOrderById":     {Level: aauth.AccessPerson},
		"placeOrder":       {Level: aauth.AccessAuth, Scope: "write:orders"},
		"deletePet":        {Level: aauth.AccessAuth, Scope: "write:pets"},
	} {
		if levels[id] != want {
			t.Errorf("%s: %+v, want %+v", id, levels[id], want)
		}
	}
	if s.Scopes["write:pets"] == "" || s.Scopes["write:orders"] == "" {
		t.Errorf("scopes = %v", s.Scopes)
	}
	table := (&handlers{}).table()
	for _, op := range s.Operations {
		if table[op.ID] == nil {
			t.Errorf("no handler for %s", op.ID)
		}
	}
}

func TestSpecErrors(t *testing.T) {
	const head = "servers:\n  - url: http://x/v2\nx-aauth:\n  scopes:\n    'write:pets': change pets\npaths:\n  /pet:\n    post:\n"
	for name, tc := range map[string]struct{ op, want string }{
		"no annotation":  {"      operationId: addPet\n", "no x-aauth"},
		"no operationId": {"      x-aauth:\n        access: agent\n", "no operationId"},
		"bad level":      {"      operationId: addPet\n      x-aauth:\n        access: admin\n", "not agent, person or auth"},
		"auth no scope":  {"      operationId: addPet\n      x-aauth:\n        access: auth\n", "needs a scope"},
		"scope on agent": {"      operationId: addPet\n      x-aauth:\n        access: agent\n        scope: 'write:pets'\n", "only for access auth"},
		"undescribed":    {"      operationId: addPet\n      x-aauth:\n        access: auth\n        scope: 'write:cats'\n", "not described"},
	} {
		_, err := ParseSpec([]byte(head + tc.op))
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v, want %q", name, err, tc.want)
		}
	}
	if _, err := New(Config{Spec: []byte(head + "      operationId: feedPet\n      x-aauth:\n        access: agent\n")}); err == nil || !strings.Contains(err.Error(), "no handler for feedPet") {
		t.Errorf("unknown operation: %v", err)
	}
}

func TestSpecDocument(t *testing.T) {
	s, err := ParseSpec(petstore.OpenAPI)
	if err != nil {
		t.Fatal(err)
	}
	j, err := s.Document("https://petstore.aauth.test/", true)
	if err != nil || !strings.Contains(string(j), `"url": "https://petstore.aauth.test/v2"`) {
		t.Fatalf("json document: %v\n%.300s", err, j)
	}
	y, err := s.Document("https://petstore.aauth.test", false)
	if err != nil || !strings.Contains(string(y), "url: https://petstore.aauth.test/v2") || !strings.Contains(string(y), "x-aauth:") {
		t.Fatalf("yaml document: %v", err)
	}
}

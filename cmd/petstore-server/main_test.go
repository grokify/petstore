package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandlerOptions(t *testing.T) {
	if _, err := (&options{auth: "none"}).handler(); err != nil {
		t.Fatal(err)
	}
	for name, o := range map[string]options{
		"unknown auth": {auth: "oauth"},
		"no url":       {auth: "aauth", trust: []string{"https://ps.aauth.test"}},
		"no trust":     {auth: "aauth", url: "https://petstore.aauth.test"},
		"missing ca":   {auth: "aauth", url: "https://petstore.aauth.test", trust: []string{"https://ps.aauth.test"}, caFile: filepath.Join(t.TempDir(), "none.pem")},
	} {
		if _, err := o.handler(); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	ca := filepath.Join(t.TempDir(), "empty.pem")
	if err := os.WriteFile(ca, []byte("empty"), 0o600); err != nil {
		t.Fatal(err)
	}
	o := options{auth: "aauth", url: "https://petstore.aauth.test", trust: []string{"https://ps.aauth.test"}, caFile: ca}
	if _, err := o.handler(); err == nil || !strings.Contains(err.Error(), "no certificates") {
		t.Errorf("a CA file without certificates: %v", err)
	}
	o.caFile = ""
	if _, err := o.handler(); err != nil {
		t.Errorf("aauth mode: %v", err)
	}
}

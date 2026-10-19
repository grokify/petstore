package server

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"strings"

	aauth "github.com/aauth-dev/aauth-go"
	"go.yaml.in/yaml/v3"
)

// Operation is one operation of the spec.
type Operation struct {
	ID     string       // operationId
	Method string       // GET, POST, …
	Path   string       // the spec's path template, such as /pet/{petId}
	Access aauth.Access // from x-aauth
}

// Spec is what the server reads from an OpenAPI document.
type Spec struct {
	// BasePath is the path of the first server URL, such as /v2.
	BasePath string
	// Operations are in path, then method, order.
	Operations []Operation
	// Scopes describes each AAuth scope, from the top-level x-aauth.
	Scopes map[string]string

	doc map[string]any
}

var methods = []string{"get", "put", "post", "delete", "patch", "head", "options", "trace"}

type xAAuth struct {
	Access string `yaml:"access"`
	Scope  string `yaml:"scope"`
}

// ParseSpec reads an OpenAPI document. Every operation needs an
// operationId and an x-aauth access level.
func ParseSpec(b []byte) (*Spec, error) {
	var raw struct {
		Servers []struct {
			URL string `yaml:"url"`
		} `yaml:"servers"`
		Paths  map[string]map[string]yaml.Node `yaml:"paths"`
		XAAuth struct {
			Scopes map[string]string `yaml:"scopes"`
		} `yaml:"x-aauth"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, fmt.Errorf("petstore: parsing the spec: %w", err)
	}
	s := &Spec{Scopes: raw.XAAuth.Scopes}
	if len(raw.Servers) > 0 {
		u, err := url.Parse(raw.Servers[0].URL)
		if err != nil {
			return nil, fmt.Errorf("petstore: server URL: %w", err)
		}
		s.BasePath = strings.TrimSuffix(u.Path, "/")
	}
	paths := make([]string, 0, len(raw.Paths))
	for p := range raw.Paths {
		paths = append(paths, p)
	}
	slices.Sort(paths)
	var errs []string
	for _, p := range paths {
		for _, m := range methods {
			node, ok := raw.Paths[p][m]
			if !ok {
				continue
			}
			var op struct {
				OperationID string  `yaml:"operationId"`
				XAAuth      *xAAuth `yaml:"x-aauth"`
			}
			if err := node.Decode(&op); err != nil {
				return nil, fmt.Errorf("petstore: %s %s: %w", strings.ToUpper(m), p, err)
			}
			where := strings.ToUpper(m) + " " + p
			if op.OperationID == "" {
				errs = append(errs, where+": no operationId")
				continue
			}
			access, err := op.XAAuth.access(s.Scopes)
			if err != nil {
				errs = append(errs, fmt.Sprintf("%s (%s): %v", where, op.OperationID, err))
				continue
			}
			s.Operations = append(s.Operations, Operation{ID: op.OperationID, Method: strings.ToUpper(m), Path: p, Access: access})
		}
	}
	if len(errs) > 0 {
		return nil, fmt.Errorf("petstore: the spec is incomplete:\n  %s", strings.Join(errs, "\n  "))
	}
	if err := yaml.Unmarshal(b, &s.doc); err != nil {
		return nil, fmt.Errorf("petstore: parsing the spec: %w", err)
	}
	return s, nil
}

func (x *xAAuth) access(scopes map[string]string) (aauth.Access, error) {
	if x == nil {
		return aauth.Access{}, fmt.Errorf("no x-aauth access level")
	}
	level, err := aauth.ParseAccessLevel(x.Access)
	if err != nil {
		return aauth.Access{}, err
	}
	a := aauth.Access{Level: level, Scope: x.Scope}
	switch {
	case level == aauth.AccessAuth && x.Scope == "":
		return a, fmt.Errorf("x-aauth access auth needs a scope")
	case level != aauth.AccessAuth && x.Scope != "":
		return a, fmt.Errorf("x-aauth scope is only for access auth")
	}
	for _, sc := range strings.Fields(x.Scope) {
		if _, ok := scopes[sc]; !ok {
			return a, fmt.Errorf("scope %s is not described in the top-level x-aauth scopes", sc)
		}
	}
	return a, nil
}

// Document returns the spec with servers set to origin plus the base
// path, as YAML, or as JSON when asJSON is set.
func (s *Spec) Document(origin string, asJSON bool) ([]byte, error) {
	doc := make(map[string]any, len(s.doc))
	for k, v := range s.doc {
		doc[k] = v
	}
	doc["servers"] = []map[string]string{{"url": strings.TrimSuffix(origin, "/") + s.BasePath}}
	if asJSON {
		return json.MarshalIndent(doc, "", "  ")
	}
	return yaml.Marshal(doc)
}

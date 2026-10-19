package server

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type client struct {
	t   *testing.T
	url string
}

func plainServer(t *testing.T) *client {
	t.Helper()
	h, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &client{t: t, url: srv.URL}
}

// do sends a request and decodes a JSON reply into out (when not nil).
func (c *client) do(method, path, contentType string, body io.Reader, out any) int {
	c.t.Helper()
	req, err := http.NewRequest(method, c.url+path, body)
	if err != nil {
		c.t.Fatal(err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			c.t.Error(err)
		}
	}()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		c.t.Fatal(err)
	}
	if out != nil && res.StatusCode < 300 {
		if err := json.Unmarshal(b, out); err != nil {
			c.t.Fatalf("%s %s: %v: %s", method, path, err, b)
		}
	}
	return res.StatusCode
}

func (c *client) json(method, path string, in, out any) int {
	c.t.Helper()
	var body io.Reader
	if in != nil {
		b, err := json.Marshal(in)
		if err != nil {
			c.t.Fatal(err)
		}
		body = bytes.NewReader(b)
	}
	return c.do(method, path, "application/json", body, out)
}

func expect(t *testing.T, what string, got, want int) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: status %d, want %d", what, got, want)
	}
}

func TestPets(t *testing.T) {
	c := plainServer(t)
	var pets []Pet
	expect(t, "find available", c.json("GET", "/v2/pet/findByStatus?status=available", nil, &pets), 200)
	if len(pets) != 4 || pets[0].Name != "Fido" {
		t.Fatalf("available pets = %+v", pets)
	}
	expect(t, "find two statuses", c.json("GET", "/v2/pet/findByStatus?status=pending,sold", nil, &pets), 200)
	if len(pets) != 2 {
		t.Fatalf("pending and sold = %d", len(pets))
	}
	expect(t, "bad status", c.json("GET", "/v2/pet/findByStatus?status=lost", nil, nil), 400)
	expect(t, "find by tag", c.json("GET", "/v2/pet/findByTags?tags=calm", nil, &pets), 200)
	if len(pets) != 3 {
		t.Fatalf("calm pets = %d", len(pets))
	}
	expect(t, "no tags", c.json("GET", "/v2/pet/findByTags", nil, nil), 400)

	var p Pet
	expect(t, "add", c.json("POST", "/v2/pet", Pet{Name: "Nemo", PhotoURLs: []string{}}, &p), 200)
	if p.ID != 7 || p.Status != StatusAvailable {
		t.Fatalf("added %+v", p)
	}
	expect(t, "add without photos", c.json("POST", "/v2/pet", Pet{Name: "x"}, nil), 405)
	p.Name = "Dory"
	expect(t, "update", c.json("PUT", "/v2/pet", p, &p), 200)
	expect(t, "update unknown", c.json("PUT", "/v2/pet", Pet{ID: 99, Name: "x", PhotoURLs: []string{}}, nil), 404)
	expect(t, "update no id", c.json("PUT", "/v2/pet", Pet{Name: "x", PhotoURLs: []string{}}, nil), 400)
	expect(t, "update with form", c.do("POST", "/v2/pet/7", "application/x-www-form-urlencoded", strings.NewReader(url.Values{"status": {"sold"}}.Encode()), &p), 200)
	if p.Name != "Dory" || p.Status != StatusSold {
		t.Fatalf("after form %+v", p)
	}
	expect(t, "get", c.json("GET", "/v2/pet/7", nil, &p), 200)
	expect(t, "get bad id", c.json("GET", "/v2/pet/x", nil, nil), 400)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	if err := mw.WriteField("additionalMetadata", "front"); err != nil {
		t.Fatal(err)
	}
	fw, err := mw.CreateFormFile("file", "dory.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fw.Write([]byte("jpeg")); err != nil {
		t.Fatal(err)
	}
	if err := mw.Close(); err != nil {
		t.Fatal(err)
	}
	var ar APIResponse
	expect(t, "upload", c.do("POST", "/v2/pet/7/uploadImage", mw.FormDataContentType(), &buf, &ar), 200)
	if !strings.Contains(ar.Message, "dory.jpg, 4 bytes") || !strings.HasPrefix(ar.Message, "additionalMetadata: front") {
		t.Fatalf("upload: %+v", ar)
	}

	expect(t, "delete", c.json("DELETE", "/v2/pet/7", nil, nil), 200)
	expect(t, "deleted", c.json("GET", "/v2/pet/7", nil, nil), 404)
}

func TestOrders(t *testing.T) {
	c := plainServer(t)
	var o Order
	expect(t, "order Fido", c.json("POST", "/v2/store/order", Order{PetID: 1}, &o), 200)
	if o.ID != 1 || o.Status != "placed" || o.Quantity != 1 {
		t.Fatalf("order %+v", o)
	}
	expect(t, "order Fido again", c.json("POST", "/v2/store/order", Order{PetID: 1}, nil), 400)
	expect(t, "order nobody", c.json("POST", "/v2/store/order", Order{PetID: 99}, nil), 400)
	inv := map[string]int{}
	expect(t, "inventory", c.json("GET", "/v2/store/inventory", nil, &inv), 200)
	if inv["available"] != 3 || inv["pending"] != 2 {
		t.Fatalf("inventory %v", inv)
	}
	expect(t, "get order", c.json("GET", "/v2/store/order/1", nil, &o), 200)
	expect(t, "cancel", c.json("DELETE", "/v2/store/order/1", nil, nil), 200)
	expect(t, "cancelled", c.json("GET", "/v2/store/order/1", nil, nil), 404)
	var p Pet
	expect(t, "Fido again", c.json("GET", "/v2/pet/1", nil, &p), 200)
	if p.Status != StatusAvailable {
		t.Fatalf("Fido is %s after the cancellation", p.Status)
	}
}

func TestUsers(t *testing.T) {
	c := plainServer(t)
	var u User
	expect(t, "create", c.json("POST", "/v2/user", User{Username: "alice", Password: "s3cret", Email: "alice@example.com"}, &u), 200)
	if u.ID != 1 || u.Password != "" {
		t.Fatalf("created %+v", u)
	}
	var us []User
	expect(t, "create many", c.json("POST", "/v2/user/createWithArray", []User{{Username: "bob"}, {Username: "carol"}}, &us), 200)
	expect(t, "create with list", c.json("POST", "/v2/user/createWithList", []User{{Username: "dave"}}, &us), 200)
	expect(t, "create without name", c.json("POST", "/v2/user", User{}, nil), 400)
	var session string
	expect(t, "login", c.json("GET", "/v2/user/login?username=alice&password=s3cret", nil, &session), 200)
	if !strings.HasPrefix(session, "logged in user session:") {
		t.Fatalf("session %q", session)
	}
	expect(t, "wrong password", c.json("GET", "/v2/user/login?username=alice&password=x", nil, nil), 400)
	expect(t, "logout", c.json("GET", "/v2/user/logout", nil, nil), 200)
	expect(t, "update", c.json("PUT", "/v2/user/alice", User{FirstName: "Alice"}, &u), 200)
	expect(t, "get", c.json("GET", "/v2/user/alice", nil, &u), 200)
	if u.FirstName != "Alice" || u.ID != 1 || u.Password != "" {
		t.Fatalf("user %+v", u)
	}
	expect(t, "delete", c.json("DELETE", "/v2/user/alice", nil, nil), 200)
	expect(t, "deleted", c.json("GET", "/v2/user/alice", nil, nil), 404)
	expect(t, "update unknown", c.json("PUT", "/v2/user/alice", User{}, nil), 404)
}

func TestServedSpec(t *testing.T) {
	c := plainServer(t)
	var doc map[string]any
	expect(t, "spec", c.json("GET", "/openapi.json", nil, &doc), 200)
	servers, _ := doc["servers"].([]any)
	if len(servers) != 1 || servers[0].(map[string]any)["url"] != c.url+"/v2" {
		t.Fatalf("servers = %v", doc["servers"])
	}
	expect(t, "no metadata without AAuth", c.json("GET", "/.well-known/aauth-resource.json", nil, nil), 404)
	expect(t, "unknown path", c.json("GET", "/v3/pet", nil, nil), 404)
}

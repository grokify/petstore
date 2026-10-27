package server

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	aauth "github.com/aauth-dev/aauth-go"
)

// maxBody bounds request bodies; uploads may be larger.
const (
	maxBody   = 1 << 20
	maxUpload = 10 << 20
)

// APIResponse is the spec's ApiResponse, used for messages and errors.
type APIResponse struct {
	Code    int32  `json:"code"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

type handlers struct{ store *Store }

// table maps each operationId to its handler.
func (h *handlers) table() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"addPet":                    h.addPet,
		"updatePet":                 h.updatePet,
		"findPetsByStatus":          h.findPetsByStatus,
		"findPetsByTags":            h.findPetsByTags,
		"getPetById":                h.getPetByID,
		"updatePetWithForm":         h.updatePetWithForm,
		"deletePet":                 h.deletePet,
		"uploadFile":                h.uploadFile,
		"getInventory":              h.getInventory,
		"placeOrder":                h.placeOrder,
		"getOrderById":              h.getOrderByID,
		"deleteOrder":               h.deleteOrder,
		"createUser":                h.createUser,
		"createUsersWithArrayInput": h.createUsers,
		"createUsersWithListInput":  h.createUsers,
		"loginUser":                 h.loginUser,
		"logoutUser":                h.logoutUser,
		"getUserByName":             h.getUserByName,
		"updateUser":                h.updateUser,
		"deleteUser":                h.deleteUser,
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	b, err := json.Marshal(v)
	if err != nil {
		http.Error(w, `{"code":500,"type":"error","message":"encoding the response"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// The status line is sent; a failed write is a closed connection the
	// client already sees.
	_, _ = w.Write(append(b, '\n')) //nolint:errcheck // see above
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Code: int32(status), Type: "error", Message: msg}) // #nosec G115 -- HTTP status codes fit in int32
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBody))
	if err := dec.Decode(v); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue(name), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid ID supplied")
		return 0, false
	}
	return id, true
}

// owner is the person a request acts for: "<iss> <sub>" from a person or
// auth token, or "" when the server runs without AAuth.
func owner(r *http.Request) string {
	if c, ok := aauth.CallerFromContext(r.Context()); ok && c.Issuer != "" {
		return c.Issuer + " " + c.Subject
	}
	return ""
}

// csv splits a form-style array parameter: ?status=a,b or ?status=a&status=b.
func csv(r *http.Request, name string) []string {
	var out []string
	for _, v := range r.URL.Query()[name] {
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, s)
			}
		}
	}
	return out
}

func validPet(p Pet) string {
	switch {
	case p.Name == "":
		return "name is required"
	case p.PhotoURLs == nil:
		return "photoUrls is required"
	case p.Status != "" && !slices.Contains(petStatuses, p.Status):
		return "status must be available, pending or sold"
	}
	return ""
}

func (h *handlers) addPet(w http.ResponseWriter, r *http.Request) {
	var p Pet
	if !decode(w, r, &p) {
		return
	}
	if msg := validPet(p); msg != "" {
		writeError(w, http.StatusMethodNotAllowed, "Invalid input: "+msg)
		return
	}
	writeJSON(w, http.StatusOK, h.store.AddPet(p))
}

func (h *handlers) updatePet(w http.ResponseWriter, r *http.Request) {
	var p Pet
	if !decode(w, r, &p) {
		return
	}
	if p.ID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid ID supplied")
		return
	}
	if msg := validPet(p); msg != "" {
		writeError(w, http.StatusMethodNotAllowed, "Validation exception: "+msg)
		return
	}
	if p.Status == "" {
		p.Status = StatusAvailable
	}
	got, err := h.store.UpdatePet(p.ID, func(old *Pet) { *old = p })
	if err != nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}
	writeJSON(w, http.StatusOK, got)
}

func (h *handlers) findPetsByStatus(w http.ResponseWriter, r *http.Request) {
	statuses := csv(r, "status")
	if len(statuses) == 0 {
		statuses = []string{StatusAvailable}
	}
	for _, s := range statuses {
		if !slices.Contains(petStatuses, s) {
			writeError(w, http.StatusBadRequest, "Invalid status value "+s)
			return
		}
	}
	writeJSON(w, http.StatusOK, h.store.Pets(func(p Pet) bool { return slices.Contains(statuses, p.Status) }))
}

func (h *handlers) findPetsByTags(w http.ResponseWriter, r *http.Request) {
	tags := csv(r, "tags")
	if len(tags) == 0 {
		writeError(w, http.StatusBadRequest, "Invalid tag value: tags is required")
		return
	}
	writeJSON(w, http.StatusOK, h.store.Pets(func(p Pet) bool {
		return slices.ContainsFunc(p.Tags, func(t Tag) bool { return slices.Contains(tags, t.Name) })
	}))
}

func (h *handlers) getPetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "petId")
	if !ok {
		return
	}
	p, err := h.store.Pet(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handlers) updatePetWithForm(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "petId")
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		writeError(w, http.StatusMethodNotAllowed, "Invalid input: "+err.Error())
		return
	}
	name, status := r.PostFormValue("name"), r.PostFormValue("status")
	if status != "" && !slices.Contains(petStatuses, status) {
		writeError(w, http.StatusMethodNotAllowed, "Invalid input: status must be available, pending or sold")
		return
	}
	p, err := h.store.UpdatePet(id, func(p *Pet) {
		if name != "" {
			p.Name = name
		}
		if status != "" {
			p.Status = status
		}
	})
	if err != nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handlers) deletePet(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "petId")
	if !ok {
		return
	}
	if err := h.store.DeletePet(id); err != nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Code: 200, Type: "unknown", Message: strconv.FormatInt(id, 10)})
}

func (h *handlers) uploadFile(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "petId")
	if !ok {
		return
	}
	if _, err := h.store.Pet(id); err != nil {
		writeError(w, http.StatusNotFound, "Pet not found")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUpload)
	if err := r.ParseMultipartForm(maxUpload); err != nil { //nolint:gosec // G120: body limited by http.MaxBytesReader above; gosec does not recognize it for ParseMultipartForm
		writeError(w, http.StatusBadRequest, "a multipart/form-data body is required: "+err.Error())
		return
	}
	msg := ""
	if meta := r.FormValue("additionalMetadata"); meta != "" {
		msg = "additionalMetadata: " + meta + "\n"
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	n, err := io.Copy(io.Discard, f)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, "reading the file: "+err.Error())
		return
	}
	// The image is not kept; the sample server only reports it.
	writeJSON(w, http.StatusOK, APIResponse{Code: 200, Type: "unknown", Message: fmt.Sprintf("%sFile uploaded to ./%s, %d bytes", msg, hdr.Filename, n)})
}

func (h *handlers) getInventory(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.store.Inventory())
}

func (h *handlers) placeOrder(w http.ResponseWriter, r *http.Request) {
	var o Order
	if !decode(w, r, &o) {
		return
	}
	if o.PetID <= 0 {
		writeError(w, http.StatusBadRequest, "Invalid Order: petId is required")
		return
	}
	if o.Quantity <= 0 {
		o.Quantity = 1
	}
	got, err := h.store.PlaceOrder(o, owner(r))
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusBadRequest, "Invalid Order: no pet with that id")
	case errors.Is(err, ErrUnavailable):
		writeError(w, http.StatusBadRequest, "Invalid Order: the pet is not available")
	case err != nil:
		writeError(w, http.StatusInternalServerError, err.Error())
	default:
		writeJSON(w, http.StatusOK, got)
	}
}

func (h *handlers) getOrderByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "orderId")
	if !ok {
		return
	}
	o, err := h.store.Order(id, owner(r))
	if err != nil {
		writeError(w, http.StatusNotFound, "Order not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (h *handlers) deleteOrder(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "orderId")
	if !ok {
		return
	}
	if err := h.store.DeleteOrder(id, owner(r)); err != nil {
		writeError(w, http.StatusNotFound, "Order not found")
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Code: 200, Type: "unknown", Message: strconv.FormatInt(id, 10)})
}

// public returns u without its password.
func public(u User) User {
	u.Password = ""
	return u
}

func (h *handlers) createUser(w http.ResponseWriter, r *http.Request) {
	var u User
	if !decode(w, r, &u) {
		return
	}
	if u.Username == "" {
		writeError(w, http.StatusBadRequest, "username is required")
		return
	}
	writeJSON(w, http.StatusOK, public(h.store.PutUser(u)))
}

func (h *handlers) createUsers(w http.ResponseWriter, r *http.Request) {
	var us []User
	if !decode(w, r, &us) {
		return
	}
	for _, u := range us {
		if u.Username == "" {
			writeError(w, http.StatusBadRequest, "every user needs a username")
			return
		}
	}
	out := make([]User, 0, len(us))
	for _, u := range us {
		out = append(out, public(h.store.PutUser(u)))
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *handlers) loginUser(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	u, err := h.store.User(q.Get("username"))
	if err != nil || subtle.ConstantTimeCompare([]byte(u.Password), []byte(q.Get("password"))) != 1 {
		writeError(w, http.StatusBadRequest, "Invalid username/password supplied")
		return
	}
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		writeError(w, http.StatusInternalServerError, "generating a session")
		return
	}
	w.Header().Set("X-Rate-Limit", "5000")
	w.Header().Set("X-Expires-After", time.Now().Add(time.Hour).UTC().Format(time.RFC3339))
	writeJSON(w, http.StatusOK, "logged in user session:"+hex.EncodeToString(b))
}

func (h *handlers) logoutUser(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, APIResponse{Code: 200, Type: "unknown", Message: "ok"})
}

func (h *handlers) getUserByName(w http.ResponseWriter, r *http.Request) {
	u, err := h.store.User(r.PathValue("username"))
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	writeJSON(w, http.StatusOK, public(u))
}

func (h *handlers) updateUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("username")
	if _, err := h.store.User(name); err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	var u User
	if !decode(w, r, &u) {
		return
	}
	u.Username = name
	writeJSON(w, http.StatusOK, public(h.store.PutUser(u)))
}

func (h *handlers) deleteUser(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("username")
	if err := h.store.DeleteUser(name); err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}
	writeJSON(w, http.StatusOK, APIResponse{Code: 200, Type: "unknown", Message: name})
}

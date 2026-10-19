package server

import (
	"errors"
	"maps"
	"slices"
	"sync"
	"time"
)

// Pet is a pet in the catalog.
type Pet struct {
	ID        int64     `json:"id"`
	Category  *Category `json:"category,omitempty"`
	Name      string    `json:"name"`
	PhotoURLs []string  `json:"photoUrls"`
	Tags      []Tag     `json:"tags,omitempty"`
	Status    string    `json:"status,omitempty"`
}

// Category groups pets.
type Category struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Tag labels a pet.
type Tag struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Order is an order for a pet.
type Order struct {
	ID       int64      `json:"id"`
	PetID    int64      `json:"petId"`
	Quantity int32      `json:"quantity"`
	ShipDate *time.Time `json:"shipDate,omitempty"`
	Status   string     `json:"status"`
	Complete bool       `json:"complete"`

	owner string // the person who placed it: "<iss> <sub>", or "" without AAuth
}

// User is a store user. The password is accepted and never returned.
type User struct {
	ID         int64  `json:"id"`
	Username   string `json:"username"`
	FirstName  string `json:"firstName,omitempty"`
	LastName   string `json:"lastName,omitempty"`
	Email      string `json:"email,omitempty"`
	Password   string `json:"password,omitempty"`
	Phone      string `json:"phone,omitempty"`
	UserStatus int32  `json:"userStatus,omitempty"`
}

// Pet statuses.
const (
	StatusAvailable = "available"
	StatusPending   = "pending"
	StatusSold      = "sold"
)

var petStatuses = []string{StatusAvailable, StatusPending, StatusSold}

// Store errors.
var (
	ErrNotFound    = errors.New("not found")
	ErrUnavailable = errors.New("the pet is not available")
)

// Store holds the catalog, orders and users in memory.
type Store struct {
	mu        sync.Mutex
	pets      map[int64]Pet
	orders    map[int64]Order
	users     map[string]User
	nextPet   int64
	nextOrder int64
	nextUser  int64
}

// NewStore returns an empty store.
func NewStore() *Store {
	return &Store{pets: map[int64]Pet{}, orders: map[int64]Order{}, users: map[string]User{}, nextPet: 1, nextOrder: 1, nextUser: 1}
}

// NewSeededStore returns a store with a few pets of each status.
func NewSeededStore() *Store {
	s := NewStore()
	dogs, cats, fish := &Category{ID: 1, Name: "Dogs"}, &Category{ID: 2, Name: "Cats"}, &Category{ID: 3, Name: "Fish"}
	friendly, calm, playful := Tag{ID: 1, Name: "friendly"}, Tag{ID: 2, Name: "calm"}, Tag{ID: 3, Name: "playful"}
	for _, p := range []Pet{
		{Name: "Fido", Category: dogs, Tags: []Tag{friendly, playful}, Status: StatusAvailable},
		{Name: "Rex", Category: dogs, Tags: []Tag{friendly}, Status: StatusAvailable},
		{Name: "Whiskers", Category: cats, Tags: []Tag{calm}, Status: StatusAvailable},
		{Name: "Luna", Category: cats, Tags: []Tag{playful}, Status: StatusPending},
		{Name: "Bubbles", Category: fish, Tags: []Tag{calm}, Status: StatusAvailable},
		{Name: "Max", Category: dogs, Tags: []Tag{calm}, Status: StatusSold},
	} {
		p.PhotoURLs = []string{"https://example.com/pets/" + p.Name + ".jpg"}
		s.AddPet(p)
	}
	return s
}

// AddPet adds p, assigning an ID when p has none or its ID is taken, and
// returns it as stored.
func (s *Store) AddPet(p Pet) Pet {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, taken := s.pets[p.ID]; p.ID <= 0 || taken {
		p.ID = s.nextPet
	}
	if p.ID >= s.nextPet {
		s.nextPet = p.ID + 1
	}
	if p.Status == "" {
		p.Status = StatusAvailable
	}
	s.pets[p.ID] = p
	return p
}

// Pet returns the pet with id.
func (s *Store) Pet(id int64) (Pet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pets[id]
	if !ok {
		return Pet{}, ErrNotFound
	}
	return p, nil
}

// UpdatePet changes the stored pet with id by applying fn to it.
func (s *Store) UpdatePet(id int64, fn func(*Pet)) (Pet, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pets[id]
	if !ok {
		return Pet{}, ErrNotFound
	}
	fn(&p)
	p.ID = id
	s.pets[id] = p
	return p, nil
}

// DeletePet removes the pet with id.
func (s *Store) DeletePet(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.pets[id]; !ok {
		return ErrNotFound
	}
	delete(s.pets, id)
	return nil
}

// Pets returns the pets that match, in ID order.
func (s *Store) Pets(match func(Pet) bool) []Pet {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Pet{}
	for _, id := range slices.Sorted(maps.Keys(s.pets)) {
		if p := s.pets[id]; match(p) {
			out = append(out, p)
		}
	}
	return out
}

// Inventory counts pets by status.
func (s *Store) Inventory() map[string]int32 {
	s.mu.Lock()
	defer s.mu.Unlock()
	inv := map[string]int32{}
	for _, st := range petStatuses {
		inv[st] = 0
	}
	for _, p := range s.pets {
		inv[p.Status]++
	}
	return inv
}

// PlaceOrder records o for owner and marks an available pet pending.
func (s *Store) PlaceOrder(o Order, owner string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pets[o.PetID]
	if !ok {
		return Order{}, ErrNotFound
	}
	if p.Status != StatusAvailable {
		return Order{}, ErrUnavailable
	}
	p.Status = StatusPending
	s.pets[p.ID] = p
	o.ID, o.owner, o.Status, o.Complete = s.nextOrder, owner, "placed", false
	s.nextOrder++
	s.orders[o.ID] = o
	return o, nil
}

// Order returns owner's order with id. Another person's order is not
// found.
func (s *Store) Order(id int64, owner string) (Order, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok || o.owner != owner {
		return Order{}, ErrNotFound
	}
	return o, nil
}

// DeleteOrder cancels owner's order with id, making a pending pet
// available again.
func (s *Store) DeleteOrder(id int64, owner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.orders[id]
	if !ok || o.owner != owner {
		return ErrNotFound
	}
	delete(s.orders, id)
	if p, ok := s.pets[o.PetID]; ok && p.Status == StatusPending {
		p.Status = StatusAvailable
		s.pets[p.ID] = p
	}
	return nil
}

// PutUser creates or replaces the user u.Username, assigning an ID to a
// new user.
func (s *Store) PutUser(u User) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	if old, ok := s.users[u.Username]; ok {
		u.ID = old.ID
	} else {
		u.ID = s.nextUser
		s.nextUser++
	}
	s.users[u.Username] = u
	return u
}

// User returns the user with username.
func (s *Store) User(username string) (User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[username]
	if !ok {
		return User{}, ErrNotFound
	}
	return u, nil
}

// DeleteUser removes the user with username.
func (s *Store) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[username]; !ok {
		return ErrNotFound
	}
	delete(s.users, username)
	return nil
}

package server

import (
	"errors"
	"testing"
)

func TestOrdersBelongToTheirPerson(t *testing.T) {
	s := NewSeededStore()
	alice, bob := "https://ps.example alice", "https://ps.example bob"
	o, err := s.PlaceOrder(Order{PetID: 1}, alice)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Order(o.ID, bob); !errors.Is(err, ErrNotFound) {
		t.Errorf("bob read alice's order: %v", err)
	}
	if err := s.DeleteOrder(o.ID, bob); !errors.Is(err, ErrNotFound) {
		t.Errorf("bob cancelled alice's order: %v", err)
	}
	if _, err := s.Order(o.ID, alice); err != nil {
		t.Errorf("alice cannot read her order: %v", err)
	}
	if _, err := s.PlaceOrder(Order{PetID: 6}, alice); !errors.Is(err, ErrUnavailable) {
		t.Errorf("ordering a sold pet: %v", err)
	}
}

func TestAddPetIDs(t *testing.T) {
	s := NewStore()
	if p := s.AddPet(Pet{ID: 10, Name: "a"}); p.ID != 10 {
		t.Fatalf("id %d", p.ID)
	}
	if p := s.AddPet(Pet{ID: 10, Name: "b"}); p.ID != 11 {
		t.Fatalf("a taken id became %d", p.ID)
	}
	if err := s.DeletePet(99); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

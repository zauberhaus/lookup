package main

import (
	"fmt"
	"log"

	"github.com/zauberhaus/lookup"
)

type Address struct {
	City string
	Zip  int
}

type User struct {
	Name    string
	Address *Address
	Tags    []string
	Meta    map[string]any
}

func main() {
	user := &User{
		Name: "Alice",
		Tags: []string{"admin", "active"},
		Meta: map[string]any{
			"login_count": 5,
		},
	}

	// 1. Get a field value
	name, err := lookup.Get(user, "Name")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Name: %v\n", name)

	// 2. Get a value from a slice by index
	tag, err := lookup.Get(user, "Tags[0]")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("First Tag: %v\n", tag)

	// 3. Set a field value
	_, err = lookup.Set(user, "Name", "Bob")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("New Name: %v\n", user.Name)

	// 4. Check if a nested path exists (Address is nil here)
	exists, err := lookup.Exists(user, "Address.City")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Address.City exists: %v\n", exists)
}

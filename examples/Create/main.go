package main

import (
	"fmt"
	"log"

	"github.com/zauberhaus/lookup"
)

type Nested struct {
	Deep *Deep
}

type Deep struct {
	Value string
}

type Root struct {
	Nested *Nested
}

func main() {
	root := &Root{}

	// Create ensures that the path exists, initializing nil pointers along the way.
	// Here it initializes root.Nested and root.Nested.Deep.
	_, err := lookup.Create(root, "Nested.Deep")
	if err != nil {
		log.Fatal(err)
	}

	// Now we can set the value on the initialized structure.
	_, err = lookup.Set(root, "Nested.Deep.Value", "Success")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Value: %s\n", root.Nested.Deep.Value)
}

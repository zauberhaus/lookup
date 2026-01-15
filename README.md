# Lookup

`lookup` is a Go library that provides reflection-based access to nested fields within structs, maps, slices, and arrays using string paths. It allows for dynamic retrieval, modification, and creation of data structures.

## Features

*   **Get**: Retrieve values from deeply nested structures.
*   **Set**: Update values in nested structures using a path string.
*   **Exists**: Check if a specific path is populated (not nil).
*   **Create**: Traverse a path, initializing nil maps, slices, or pointers along the way.
*   **Flexible Syntax**: Supports dot notation for fields and bracket notation for indexes/keys.
*   **Type Conversion**: Convert strings to Go types including complex structures.

## Installation

```bash
go get github.com/your-username/lookup
```

> **Note**: This package depends on `github.com/zauberhaus/reflect_utils`.

## Usage

### Data Structure Example

```go
type Address struct {
	City string
	Zip  int
}

type User struct {
	Name    string
	Address *Address
	Tags    []string
	Meta    map[string]interface{}
}
```

### Get

Retrieve a value from a struct.

```go
user := User{
	Name: "Alice",
	Tags: []string{"admin", "active"},
}

name, err := lookup.Get(user, "Name")
// name: "Alice"

tag, err := lookup.Get(user, "Tags[0]")
// tag: "admin"
```

### Set

Set a value at a specific path. Note that `Set` requires a pointer to the struct to modify it.

```go
user := &User{}

// Sets the Name field
_, err := lookup.Set(user, "Name", "Bob")

// Automatically expands slices if index is reachable
_, err = lookup.Set(user, "Tags[0]", "guest")
```

### Create

`Create` ensures that the path exists, initializing nil pointers, maps, or slices with default values if necessary.

```go
user := &User{}

// Initializes the Address pointer if it is nil
_, err := lookup.Create(user, "Address")

// Initializes the Meta map if nil and sets the key
_, err = lookup.Set(user, "Meta[\"login_count\"]", 1)
```

### Path Syntax

*   **Struct Fields**: `Field.SubField` (e.g., `User.Address.City`)
*   **Arrays/Slices**: `List[index]` (e.g., `Tags[0]`)
*   **Maps**: `Map["key"]` (e.g., `Meta["version"]`) - supports double quotes, single quotes, or backticks.

### Type Conversion

Convert a string to a specific Go type. This supports basic types, pointers, slices, maps, and structs (via JSON).

```go
// Parse an integer
val, err := lookup.Parse("123", reflect.TypeOf(0))
// val: 123

// Parse a slice
val, err = lookup.Parse("1,2,3", reflect.TypeOf([]int{}))
// val: []int{1, 2, 3}

// Parse a map
val, err = lookup.Parse("a=1,b=2", reflect.TypeOf(map[string]int{}))
// val: map[string]int{"a": 1, "b": 2}
```

### Custom Parsing

Register custom parsers for specific types.

```go
type MyType struct {
	Data string
}

hook := lookup.NewParserHook(reflect.TypeOf(MyType{}), func(input string) (any, error) {
	return MyType{Data: input}, nil
})

val, err := lookup.Parse("some data", reflect.TypeOf(MyType{}), hook)
```
# di

A dependency injection module based on reflection.

## Installation

```sh
go get -u github.com/dozm/di
```

## Quick start

```go
package main

import (
    "fmt"
    "github.com/dozm/di"
)

func main() {
    // Create a ContainerBuilder
    b := di.Builder()
    
    // Register some services with generic helper function.
    di.AddSingleton[string](b, func() string { return "hello" })
    di.AddTransient[int](b, func() int { return 1 })
    di.AddScoped[int](b, func() int { return 2 })

    // Build the container
    c := b.Build()

    // Usually, you should not resolve a service directly from the root scope.
    // So, get the di.ScopeFactory (it's a built-in service) to create a scope.
    // Typically, in web application we create a scope for per HTTP request.
    scopeFactory := di.Get[di.ScopeFactory](c)
    scope := scopeFactory.CreateScope()
    c = scope.Container()

    // Get a service from the container
    s := di.Get[string](c)
    fmt.Println(s)

    // Get all of the services with the type int as a slice.
    intSlice := di.Get[[]int](c)
    fmt.Println(intSlice)
}
```

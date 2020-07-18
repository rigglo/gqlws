# gqlws

[![go.dev reference](https://img.shields.io/badge/go.dev-reference-007d9c?logo=go&logoColor=white)](https://pkg.go.dev/github.com/rigglo/gqlws)
[![Coverage Status](https://coveralls.io/repos/github/rigglo/gqlws/badge.svg?branch=master)](https://coveralls.io/github/rigglo/gqlws?branch=master)

A GraphQL Subscriptions handler over WebSockets

An example using the `rigglo/gql` package

```golang
package main

import (
 "net/http"

 "github.com/rigglo/gql"
 "github.com/rigglo/gql/pkg/handler"
 "github.com/rigglo/gqlws"
)

func main() {
 exec := gql.NewExecutor(gql.ExecutorConfig{
  EnableGoroutines: false,
  Schema:           schema,
    })

 h := handler.New(handler.Config{
  Executor:   exec,
  Playground: true,
    })

 wsh := gqlws.New(
  gqlws.Config{
   Subscriber: exec.Subscribe,
  },
  h,
 )

 http.Handle("/graphql", wsh)

 if err := http.ListenAndServe(":9999", nil); err != nil {
  log.Println(err)
 }
}
```

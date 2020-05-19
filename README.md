# gqlws

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

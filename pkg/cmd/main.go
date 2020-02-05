package main

import (
	"fmt"
	"os"

	"github.com/joshblakeley/tyk-mixer-adapter/pkg/adapter"
	)

func main() {
	addr := ""
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}

	s, err := adapter.NewTykGrpcAdapter(addr)
	if err != nil {
		fmt.Printf("unable to start server: %v", err)
		os.Exit(-1)
	}

	shutdown := make(chan error, 1)
	go func() {
		s.Run(shutdown)
	}()
	_ = <-shutdown
}

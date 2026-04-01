package main

import (
	"fmt"
	"os"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8150"
	}
	fmt.Printf("webdata-service starting on :%s\n", port)
	// TODO: bootstrap DI, router, server (Fase 3+)
}

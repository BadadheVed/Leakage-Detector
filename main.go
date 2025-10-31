package main

import (
	"fmt"

	"log"

	"github.com/BadadheVed/leakage-detector/route"
	"github.com/BadadheVed/leakage-detector/setup"
	"github.com/gin-gonic/gin"
)

func main() {
	setup := setup.Setup()
	r := gin.Default()
	route.RegisterRoutes(r, setup)
	port := "8080"
	log.Printf("[server] running on http://localhost:%s", port)

	if err := r.Run(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	apiserver "github.com/mdsavian/budget-tracker-api/api-server"
)

func main() {
	godotenv.Load(".env")

	store, err := NewPostgresStore()
	if err != nil {
		log.Fatal(err)
	}

	if err := store.Init(); err != nil {
		log.Fatal(err)
	}

	portString := os.Getenv("PORT")
	if portString == "" {
		log.Fatal("PORT is not found in the environment")
	}

	server := apiserver.NewServer(fmt.Sprintf(":%s", portString), store)
	server.Start()
}

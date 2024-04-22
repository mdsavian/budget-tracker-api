package main

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	apiserver "github.com/mdsavian/budget-tracker-api/internal/api-server"
	storage "github.com/mdsavian/budget-tracker-api/internal/storage"
)

func main() {
	godotenv.Load(".env")

	store, err := storage.NewPostgresStore()
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

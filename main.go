package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/mdsavian/budget-tracker-api/cmd"
	apiserver "github.com/mdsavian/budget-tracker-api/internal/api-server"
	storage "github.com/mdsavian/budget-tracker-api/internal/storage"
)

func main() {

	env := os.Getenv("ENV")
	if env == "" {
		env = "dev"
	}
	godotenv.Load(".env." + env + ".local")

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
	if len(os.Args) > 2 {
		importData := os.Args[1]
		if ok, _ := strconv.ParseBool(importData); ok && os.Args[2] != "" {
			path := os.Args[2]
			cmd.ImportData(path, store)

		}
	} else {
		server := apiserver.NewServer(fmt.Sprintf(":%s", portString), store)
		server.Start()
	}
}

package initializer

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func LoadEnvs() {
	err := godotenv.Load()
	if err != nil {
		key := "DB_URL"
		_, exists := os.LookupEnv(key)
		if !exists {
			log.Fatal("Error loading .env file")

		}
	}

}

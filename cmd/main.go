package main

import (
	"errors"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"battlebarge/db"
	"battlebarge/middleware"
	"battlebarge/routes"
)

func main() {
	//load environment: .env is optional (production sets real env variables),
	//and variables that are already set take precedence over the file
	loadEnvFile()

	//initialize firebase
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if err := db.ConnectFirebase(projectID); err != nil {
		panic(err)
	}

	//connect postgres
	if err := db.ConnectPostgres(); err != nil {
		panic(err)
	}

	//router setup
	r := gin.Default()

	//allow browser frontends on these origins (comma-separated)
	r.Use(middleware.CORS(middleware.ParseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))))

	//get routes and controllers
	routes.GetAuthControllers(r)
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)
	routes.GetUnitControllers(r)
	routes.GetCampaignControllers(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		panic(err)
	}

}

// Arguments: None
//
// Returns: None
//
// Loads a .env file from the working directory or its parent (so it works from
// the repo root or from cmd/). A missing file is fine; a file that exists but
// cannot be read or parsed stops startup.
func loadEnvFile() {
	for _, path := range []string{".env", "../.env"} {
		err := godotenv.Load(path)
		if err == nil {
			log.Printf("loaded environment from %s", path)
			return
		}
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}
}

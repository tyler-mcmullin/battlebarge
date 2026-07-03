package main

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"battlebarge/db"
	"battlebarge/routes"
)

func main() {
	//load environment
	err := godotenv.Load("../.env")
	if err != nil {
		panic(err)
	}

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

	//get routes and controllers
	routes.GetAuthControllers(r)
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)

	err = r.Run(":8080")
	if err != nil {
		panic(err)
	}

}

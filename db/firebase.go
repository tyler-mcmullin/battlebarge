package db

import (
	"context"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

var AuthClient *auth.Client

func ConnectFirebase(projectID string) error {
	ctx := context.Background()

	app, err := firebase.NewApp(ctx, &firebase.Config{
		ProjectID: projectID,
	})
	if err != nil {
		return err
	}

	AuthClient, err = app.Auth(ctx)
	if err != nil {
		return err
	}

	return nil
}

package db

import (
	"context"

	firebaseAdmin "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

var AuthClient *auth.Client

// Arguments: Firebase project ID
// Returns: None
// Connects to firebase 
func ConnectFirebase(projectID string) error {
	ctx := context.Background()

	app, err := firebaseAdmin.NewApp(ctx, &firebaseAdmin.Config{
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

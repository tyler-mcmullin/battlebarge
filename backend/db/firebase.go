package db

import (
	"context"

	firebaseAdmin "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
)

var AuthClient *auth.Client

// ConnectFirebase connects to Firebase and stores the Auth client in AuthClient
//
// Arguments: projectID (string) - Firebase project ID
//
// Returns: error - non-nil if the Firebase app or its Auth client cannot be created
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

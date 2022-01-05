package main

import (
	"context"
	"fmt"
	"os"

	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/gcp"
)

// Fetches a key-value secret (kv-v2) after authenticating to Vault
// via GCP IAM, one of two auth methods used to authenticate with
// GCP (the other is GCE auth).
func getSecretWithGCPAuthIAM() (string, error) {
	config := vault.DefaultConfig() // modify for more granular configuration

	client, err := vault.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("unable to initialize Vault client: %w", err)
	}

	// For IAM-style auth, the environment variable GOOGLE_APPLICATION_CREDENTIALS
	// must be set with the path to a valid credentials JSON file, otherwise
	// Vault will fall back to Google's default instance credentials.
	// Learn about authenticating to GCS with service account credentials at https://cloud.google.com/docs/authentication/production
	if pathToCreds := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); pathToCreds == "" {
		fmt.Printf("WARNING: Environment variable GOOGLE_APPLICATION_CREDENTIALS was not set. IAM client for JWT signing and Vault server IAM client will both fall back to default instance credentials.\n")
	}

	svcAccountEmail := fmt.Sprintf("%s@%s.iam.gserviceaccount.com", os.Getenv("GCP_SERVICE_ACCOUNT_NAME"), os.Getenv("GOOGLE_CLOUD_PROJECT"))

	// We pass the auth.WithIAMAuth option to use the IAM-style authentication
	// of the GCP auth backend. Otherwise, we default to using GCE-style
	// authentication, which gets its credentials from the metadata server.
	gcpAuth, err := auth.NewGCPAuth(
		"dev-role-iam",
		auth.WithIAMAuth(svcAccountEmail),
	)
	if err != nil {
		return "", fmt.Errorf("unable to initialize GCP auth method: %w", err)
	}

	authInfo, err := client.Auth().Login(context.TODO(), gcpAuth)
	if err != nil {
		return "", fmt.Errorf("unable to login to GCP auth method: %w", err)
	}
	if authInfo == nil {
		return "", fmt.Errorf("login response did not return client token")
	}

	// get secret
	secret, err := client.Logical().Read("kv-v2/data/creds")
	if err != nil {
		return "", fmt.Errorf("unable to read secret: %w", err)
	}

	data, ok := secret.Data["data"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("data type assertion failed: %T %#v", secret.Data["data"], secret.Data["data"])
	}

	// data map can contain more than one key-value pair,
	// in this case we're just grabbing one of them
	key := "password"
	value, ok := data[key].(string)
	if !ok {
		return "", fmt.Errorf("value type assertion failed: %T %#v", data[key], data[key])
	}

	return value, nil
}
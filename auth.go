package main

import (
	"context"
	"fmt"
	"os"

	"golang.org/x/oauth2/google"
)

const (
	SCOPES = "https://www.googleapis.com/auth/cloud-platform"
)

type ServiceAccount struct {
	Type                    string `json:"type"`
	ProjectID               string `json:"project_id"`
	PrivateKeyID            string `json:"private_key_id"`
	PrivateKey              string `json:"private_key"`
	ClientEmail             string `json:"client_email"`
	ClientID                string `json:"client_id"`
	AuthURI                 string `json:"auth_uri"`
	TokenURI                string `json:"token_uri"`
	AuthProviderX509CertURL string `json:"auth_provider_x509_cert_url"`
	ClientX509CertURL       string `json:"client_x509_cert_url"`
}

func GetAccessToken(ctx context.Context) (string, error) {

	// Read the service account JSON file
	data, err := os.ReadFile("./firebaseConfig.json")
	if err != nil {
		return "", fmt.Errorf("error reading service account file: %v", err)
	}

	// Configure the JWT client
	conf, err := google.JWTConfigFromJSON(data, SCOPES)
	if err != nil {
		return "", fmt.Errorf("error configuring JWT: %v", err)
	}

	// Get the token
	token, err := conf.TokenSource(ctx).Token()
	if err != nil {
		return "", fmt.Errorf("error getting token: %v", err)
	}

	return token.AccessToken, nil
}

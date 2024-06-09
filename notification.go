package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/firestore"
)

type Notification struct {
	SessionID        string    `firestore:"sessionId"`
	UserToken        string    `firestore:"interpreterToken"`
	Title            string    `firestore:"title"`
	Body             string    `firestore:"body"`
	SessionTimestamp time.Time `firestore:"sessionTimestamp"`
	Read             bool      `firestore:"read"`
}

func addNotificationToUser(client *firestore.Client, ctx context.Context, userId string, notification Notification) error {
	userRef := client.Collection("users").Doc(userId)

	_, err := userRef.Update(ctx, []firestore.Update{
		{
			Path:  "notifications",
			Value: firestore.ArrayUnion(notification),
		},
	})

	if err != nil {
		log.Printf("Error updating user notifications: %v", err)
		return err
	}

	log.Println("Notification added successfully")
	return nil
}

func SendNotification(userId string, expoPushToken string, title string, content string, sessionId string, ctx context.Context, firestoreClient *firestore.Client) error {
	message := map[string]interface{}{
		"to":    expoPushToken,
		"sound": "default",
		"title": title,
		"body":  content,
		"data":  map[string]string{"sessionId": sessionId},
	}

	body, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://exp.host/--/api/v2/push/send", bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to send push notification: %s", resp.Status)
	}

	// Example of writing data to Firestore
	notification := Notification{
		SessionID:        sessionId,
		UserToken:        expoPushToken,
		Title:            title,
		Body:             content,
		SessionTimestamp: time.Now(),
		Read:             false,
	}

	err = addNotificationToUser(firestoreClient, ctx, userId, notification)
	if err != nil {
		log.Fatalf("Error writing to Firestore: %v", err)
	}

	return nil
}

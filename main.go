package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/robfig/cron/v3"
	"google.golang.org/api/option"
)

type NotificationPayload struct {
	UserId           string `json:"userId"`
	UserToken        string `json:"userToken"`
	SessionTimestamp string `json:"sessionTimestamp"` // ISO 8601 format
	SessionID        string `json:"sessionId"`
}

func main() {

	router := mux.NewRouter()

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // Adjust this to your needs
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	c := cron.New(cron.WithLocation(time.UTC))
	db, err := InitializeDb()
	if err != nil {
		log.Fatalf("Database initialization error: %v\n", err)
	}
	defer db.Close()

	ctx := context.Background()

	firebaseConfigBase64 := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	if firebaseConfigBase64 == "" {
		log.Fatalf("GOOGLE_APPLICATION_CREDENTIALS environment variable not set")
	}

	firebaseConfigJson, err := base64.StdEncoding.DecodeString(firebaseConfigBase64)
	if err != nil {
		log.Fatalf("Error decoding Firebase config: %v", err)
	}

	sa := option.WithCredentialsJSON(firebaseConfigJson)

	app, err := firebase.NewApp(ctx, nil, sa)
	if err != nil {
		log.Fatalf("Firebase initialization error: %v\n", err)
	}

	client, err := app.Firestore(ctx)
	if err != nil {
		log.Fatalln(err)
	}
	defer client.Close()

	router.HandleFunc("/generate-jwt", handleJwt).Methods("POST")
	router.HandleFunc("/schedule-notification", handleNotification(db, c, ctx, client)).Methods("POST")

	http.ListenAndServe(":8080", cors(router))
}

func handleJwt(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserId      string `json:"userId"`
		SessionName string `json:"sessionName"`
		RoleType    string `json:"roleType"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	token := generateJwt(req.UserId, req.SessionName, req.RoleType)
	if token == "" {
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func generateJwt(userId string, sessionName string, roleType string) string {
	appKey := os.Getenv("ZOOM_APP_KEY")
	appSecret := os.Getenv("ZOOM_APP_SECRET")

	role, err := strconv.Atoi(roleType)
	if err != nil {
		log.Fatalf("Error converting roleType to integer: %v", err)
		return ""
	}

	claims := jwt.MapClaims{
		"app_key":                appKey,
		"version":                1,
		"user_identity":          userId,
		"iat":                    time.Now().Unix(),
		"exp":                    time.Now().Add(23 * time.Hour).Unix(),
		"tpc":                    sessionName,
		"role_type":              role,
		"cloud_recording_option": 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(appSecret))
	if err != nil {
		log.Fatalf("Error in generating token: %v", err)
		return ""
	}

	return signedToken
}

func handleNotification(db *sql.DB, c *cron.Cron, ctx context.Context, client *firestore.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var payload NotificationPayload

		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sessionDate, err := time.Parse(time.RFC3339, payload.SessionTimestamp)
		if err != nil {
			http.Error(w, "Invalid session time format", http.StatusBadRequest)
			return
		}

		reminderDate := sessionDate.Add(-30 * time.Minute)

		// Schedule a notification 30 minutes before the session
		reminderDateCron := createCronExpression(reminderDate)
		fmt.Println(reminderDateCron)
		reminderJobID, err := c.AddFunc(reminderDateCron, func() {
			fmt.Println("send")
			err := SendNotification(payload.UserId, payload.UserToken, "Session Reminder", "Your session starts in 30 minutes.", payload.SessionID, ctx, client)
			if err != nil {
				log.Printf("Error sending reminder notification: %v", err)
			}
		})
		if err != nil {
			fmt.Println(err)
			http.Error(w, "Failed to schedule reminder notification", http.StatusInternalServerError)
			return
		}

		// Schedule a notification at the time of the session
		sessionDateCron := createCronExpression(sessionDate)
		fmt.Println(sessionDateCron)
		startJobID, err := c.AddFunc(sessionDateCron, func() {
			fmt.Println("send")
			err := SendNotification(payload.UserId, payload.UserToken, "Session Starting", "Your session is starting now.", payload.SessionID, ctx, client)
			if err != nil {
				log.Printf("Error sending start notification: %v", err)
			}
		})
		if err != nil {
			http.Error(w, "Failed to schedule session start notification", http.StatusInternalServerError)
			return
		}

		// Store the jobs in SQLite
		insertQuery := `
	INSERT INTO jobs (sessionId, reminderJobID, startJobID, userToken, sessionTimestamp, reminderTimestamp)
	VALUES (?, ?, ?, ?, ?, ?)
	`
		_, err = db.Exec(insertQuery, payload.SessionID, reminderJobID, startJobID, payload.UserToken, sessionDate.Format(time.RFC3339), reminderDate.Format(time.RFC3339))
		if err != nil {
			log.Printf("Error storing job in SQLite: %v", err)
			http.Error(w, "Failed to store job in database", http.StatusInternalServerError)
			return
		}

		c.Start()
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "Notifications scheduled"})
	}
}

func createCronExpression(t time.Time) string {
	return fmt.Sprintf("%d %d %d %d *", t.Minute(), t.Hour(), t.Day(), t.Month())
}

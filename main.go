package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

func main() {

	router := mux.NewRouter()

	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}), // Adjust this to your needs
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	router.HandleFunc("/generate-jwt", handleJwt).Methods("POST")

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

	fmt.Println(appKey)
	fmt.Println(appSecret)

	role, err := strconv.Atoi(roleType)
	if err != nil {
		log.Fatalf("Error converting roleType to integer: %v", err)
		return ""
	}

	claims := jwt.MapClaims{
		"app_Key":                appKey,
		"version":                1,
		"user_identity":          userId,
		"iat":                    time.Now().Unix(),
		"exp":                    time.Now().Add(23 * time.Hour).Unix(),
		"tpc":                    sessionName,
		"role":                   role,
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

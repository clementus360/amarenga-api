package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/generate-jwt", handleJwt).Methods("POST")

	http.ListenAndServe(":8080", router)
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
	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

func generateJwt(userId string, sessionName string, roleType string) string {

	err := godotenv.Load() // Load .env file
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	zoomAppKey := os.Getenv("ZOOM_APP_KEY")
	zoomAppSecret := os.Getenv("ZOOM_APP_SECRET")

	role, err := strconv.Atoi(roleType)
	if err != nil {
		log.Fatalf("Error converting roleType to integer: %v", err)
		return "" // or handle the error differently depending on your error handling strategy
	}

	claims := jwt.MapClaims{
		"app_key":                zoomAppKey,
		"version":                1,
		"user_identity":          userId,
		"iat":                    time.Now().Unix(),
		"exp":                    time.Now().Add(23 * time.Hour).Unix(),
		"tpc":                    sessionName,
		"role_type":              role,
		"cloud_recording_option": 1,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(zoomAppSecret))
	if err != nil {
		log.Fatalf("Error in generating token: %v", err)
		return ""
	}

	return signedToken
}

package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/classlink"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/google"
)

var jwtKey = []byte(os.Getenv("STRATA_JWT_KEY"))

type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func init() {
	// Set up authentication providers
	goth.UseProviders(
		google.New(os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"), "http://localhost:8080/auth/google/callback"),
		classlink.New(os.Getenv("CLASSLINK_CLIENT_ID"), os.Getenv("CLASSLINK_CLIENT_SECRET"), "http://localhost:8080/auth/classlink/callback"),
		github.New(os.Getenv("GITHUB_CLIENT_ID"), os.Getenv("GITHUB_CLIENT_SECRET"), "http://localhost:8080/auth/github/callback"),
	)
}

// authenticate handles the authentication process (OAuth)
func authenticate(w http.ResponseWriter, r *http.Request) {
	provider := r.URL.Query().Get("provider")
	if provider == "" {
		http.Error(w, "Provider not specified", http.StatusBadRequest)
		return
	}

	// Use Goth to handle OAuth with the specified provider
	gothic.BeginAuthHandler(w, r)
}

// callbackHandler processes the OAuth callback and issues a JWT
func callbackHandler(w http.ResponseWriter, r *http.Request) {
	user, err := gothic.CompleteUserAuth(w, r)
	if err != nil {
		fmt.Fprintln(w, err)
		return
	}

	// Create the JWT token
	expirationTime := time.Now().Add(5 * time.Hour)
	claims := &Claims{
		Username: user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		http.Error(w, "Error generating JWT", http.StatusInternalServerError)
		return
	}

	// Send the token as a response (could also be stored in a cookie)
	http.SetCookie(w, &http.Cookie{
		Name:    "token",
		Value:   tokenString,
		Expires: expirationTime,
	})

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// dashboardHandler displays the dashboard if authenticated
func dashboardHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("token")
	if err != nil {
		if err == http.ErrNoCookie {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		http.Error(w, "Failed to retrieve JWT", http.StatusUnauthorized)
		return
	}

	// Parse the JWT token
	tokenStr := cookie.Value
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})

	if err != nil || !token.Valid {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Display dashboard
	renderTemplate(w, "dashboard.html", claims)
}

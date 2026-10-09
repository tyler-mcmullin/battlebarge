package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/time/rate"

	"battlebarge/db"
	"battlebarge/middleware"
	"battlebarge/routes"
)

func main() {
	//load environment: .env is optional (production sets real env variables),
	//and variables that are already set take precedence over the file
	loadEnvFile()

	//initialize firebase
	projectID := os.Getenv("FIREBASE_PROJECT_ID")
	if err := db.ConnectFirebase(projectID); err != nil {
		panic(err)
	}

	//connect postgres
	if err := db.ConnectPostgres(); err != nil {
		panic(err)
	}

	//signed-in users must verify their email before using the API. Only turn
	//this off for local development.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("REQUIRE_EMAIL_VERIFICATION")), "false") {
		middleware.SetRequireVerifiedEmail(false)
		log.Println("WARNING: email verification is DISABLED (REQUIRE_EMAIL_VERIFICATION=false); do not run production like this")
	}

	//router setup
	r := gin.Default()

	//client IPs (used for rate limiting) are taken from the connection itself
	//unless TRUSTED_PROXIES lists the proxies/load balancers allowed to set
	//X-Forwarded-For. Without this, clients could fake their IP.
	if err := r.SetTrustedProxies(splitList(os.Getenv("TRUSTED_PROXIES"))); err != nil {
		panic(err)
	}

	//allow browser frontends on these origins (comma-separated)
	r.Use(middleware.CORS(middleware.ParseOrigins(os.Getenv("CORS_ALLOWED_ORIGINS"))))

	//cap request bodies at 64 KiB and each client IP at 20 requests/second (burst 60)
	r.Use(middleware.MaxBodySize(64 << 10))
	r.Use(middleware.RateLimit(rate.Limit(20), 60))

	//get routes and controllers
	routes.GetAuthControllers(r)
	routes.GetUserControllers(r)
	routes.GetWarbandControllers(r)
	routes.GetUnitControllers(r)
	routes.GetCampaignControllers(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	//timeouts stop slow or stalled clients from holding connections open
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() { serverErr <- srv.ListenAndServe() }()

	//on SIGINT/SIGTERM finish in-flight requests before exiting
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		panic(err)
	case <-ctx.Done():
		log.Println("shutting down")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	db.PGClient.Close()
}

// Arguments: raw (string) - comma-separated values
//
// Returns: []string - the trimmed, non-empty values (nil if there are none, which
// gin's SetTrustedProxies treats the same as an empty list: trust no proxies)
//
// Splits a comma-separated environment variable such as TRUSTED_PROXIES
func splitList(raw string) []string {
	var out []string
	for v := range strings.SplitSeq(raw, ",") {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// Arguments: None
//
// Returns: None
//
// Loads backend/.env from the working directory or its parent, so it is found
// when running from backend/ or from backend/cmd/. It deliberately does not
// look any higher, so a file at the repo root (for example the frontend's)
// is never picked up by the backend. A missing file is fine; a file that
// exists but cannot be read or parsed stops startup.
func loadEnvFile() {
	for _, path := range []string{".env", "../.env"} {
		err := godotenv.Load(path)
		if err == nil {
			log.Printf("loaded environment from %s", path)
			return
		}
		if !errors.Is(err, os.ErrNotExist) {
			panic(err)
		}
	}
}

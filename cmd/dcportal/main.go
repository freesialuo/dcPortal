package main

import (
	"context"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"dcportal/internal/config"
	"dcportal/internal/discord"
	"dcportal/internal/handler"
	"dcportal/internal/middleware"
	"dcportal/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/config.yaml", "path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	// Ensure database directory exists
	dbDir := filepath.Dir(cfg.Database.Path)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}

	// Initialize store
	st, err := store.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("init store: %v", err)
	}
	defer st.Close()

	// Parse templates — each page gets its own template set (layout + page)
	// to avoid {{define "content"}} conflicts between pages.
	layoutContent, err := os.ReadFile("web/templates/layout.html")
	if err != nil {
		log.Fatalf("read layout: %v", err)
	}

	parsePage := func(pagePath string) *template.Template {
		tmpl := template.Must(template.New("layout.html").Parse(string(layoutContent)))
		template.Must(tmpl.ParseFiles(pagePath))
		return tmpl
	}

	adminTmpl := parsePage("web/templates/admin.html")
	loginTmpl := parsePage("web/templates/login.html")
	portalTmpl := parsePage("web/templates/portal.html")
	resultTmpl := parsePage("web/templates/result.html")

	// Initialize Discord client
	dc := discord.NewClient()

	// Set up router
	mux := http.NewServeMux()

	// Static files
	mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	// Home login routes
	authHandler := handler.NewAuthHandler(loginTmpl, cfg.Admin.Token)
	authHandler.RegisterRoutes(mux)

	// Protected portal routes (with Discord OAuth2 flow)
	portalHandler := handler.NewPortalHandler(st, portalTmpl, resultTmpl, dc)
	portalMux := http.NewServeMux()
	portalHandler.RegisterRoutes(portalMux)

	// Admin routes (protected)
	adminHandler := handler.NewAdminHandler(st, adminTmpl)
	adminMux := http.NewServeMux()
	adminHandler.RegisterRoutes(adminMux)

	// Wrap web routes with token auth middleware (redirect to home login page).
	authMiddleware := middleware.AdminAuthWithRedirect(cfg.Admin.Token, "/")
	mux.Handle("GET /portal", authMiddleware(portalMux))
	mux.Handle("GET /install/", authMiddleware(portalMux))
	mux.Handle("GET /callback", authMiddleware(portalMux))
	mux.Handle("GET /admin", authMiddleware(adminMux))
	mux.Handle("POST /admin/", authMiddleware(adminMux))

	// Create server
	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("DCPortal starting on %s (port %d)", cfg.Server.BaseURL, cfg.Server.Port)
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

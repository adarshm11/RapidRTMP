package main

import (
	"log"

	"rapidrtmp/config"
	"rapidrtmp/httpServer"
	"rapidrtmp/internal/auth"
	"rapidrtmp/internal/rtmp"
	"rapidrtmp/internal/storage"
	"rapidrtmp/internal/streammanager"
)

func main() {
	log.Println("Starting RapidRTMP Server...")

	// Load configuration
	cfg := config.Load()
	log.Printf("HTTP Server: %s", cfg.HTTPAddr)
	log.Printf("RTMP Server: %s (not yet implemented)", cfg.RTMPAddr)
	log.Printf("Storage Directory: %s", cfg.StorageDir)

	// Initialize storage
	localStorage, err := storage.NewLocalStorage(cfg.StorageDir)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	log.Printf("Storage initialized: %s", localStorage.GetFullPath(""))

	// Initialize managers
	streamManager := streammanager.New()
	authManager := auth.New()
	log.Println("Stream manager and auth manager initialized")

	// Initialize HTTP server
	httpSrv := httpServer.New(streamManager, authManager, cfg.RTMPIngestAddr)
	log.Printf("HTTP server ready to start on %s", cfg.HTTPAddr)

	// Initialize RTMP ingest server
	rtmpSrv := rtmp.New(cfg.RTMPAddr, streamManager, authManager)
	go func() {
		log.Printf("Starting RTMP ingest server on %s...", cfg.RTMPAddr)
		if err := rtmpSrv.ListenAndServe(); err != nil {
			log.Fatalf("RTMP server failed: %v", err)
		}
	}()

	log.Println("RapidRTMP server started successfully")
	log.Println("---")
	log.Println("API Endpoints:")
	log.Println("  GET  /api/ping")
	log.Println("  POST /api/v1/publish")
	log.Println("  GET  /api/v1/streams")
	log.Println("  GET  /api/v1/streams/:streamKey")
	log.Println("  POST /api/v1/streams/:streamKey/stop")
	log.Println("---")

	// Start HTTP server (blocking)
	if err := httpSrv.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

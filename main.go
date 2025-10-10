package main

import (
	"context"
	"log"

	"rapidrtmp/config"
	"rapidrtmp/httpServer"
	"rapidrtmp/internal/auth"
	"rapidrtmp/internal/metrics"
	"rapidrtmp/internal/rtmp"
	"rapidrtmp/internal/segmenter"
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
	var storageBackend storage.Storage
	
	if cfg.StorageType == "gcs" {
		// Initialize GCS storage
		if cfg.GCSProjectID == "" || cfg.GCSBucketName == "" {
			log.Fatal("GCS_PROJECT_ID and GCS_BUCKET_NAME must be set when STORAGE_TYPE=gcs")
		}
		
		ctx := context.Background()
		gcsStorage, err := storage.NewGCSStorage(ctx, cfg.GCSProjectID, cfg.GCSBucketName, cfg.GCSBaseDir)
		if err != nil {
			log.Fatalf("Failed to initialize GCS storage: %v", err)
		}
		storageBackend = gcsStorage
		log.Printf("Storage initialized: GCS bucket=%s, project=%s, baseDir=%s", 
			cfg.GCSBucketName, cfg.GCSProjectID, cfg.GCSBaseDir)
	} else {
		// Initialize local storage (default)
		localStorage, err := storage.NewLocalStorage(cfg.StorageDir)
		if err != nil {
			log.Fatalf("Failed to initialize local storage: %v", err)
		}
		storageBackend = localStorage
		log.Printf("Storage initialized: Local directory=%s", cfg.StorageDir)
	}

	// Initialize metrics
	m := metrics.New()
	log.Println("Prometheus metrics initialized")

	// Initialize managers
	streamManager := streammanager.New()
	authManager := auth.New()
	log.Println("Stream manager and auth manager initialized")

	// Initialize segmenter
	seg := segmenter.New(storageBackend, streamManager)
	log.Println("HLS segmenter initialized")

	// Initialize HTTP server
	httpSrv := httpServer.New(streamManager, authManager, seg, m, cfg.RTMPIngestAddr)
	log.Printf("HTTP server ready to start on %s", cfg.HTTPAddr)

	// Initialize RTMP ingest server
	rtmpSrv := rtmp.New(cfg.RTMPAddr, streamManager, authManager, seg)
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

package ctq

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/cmd184psu/alfredo"
)

const (
	defaultDBPath   = "/var/queue/state.sqlite"
	ctq_version_fmt = "CTQ (c) C Delezenski <cmd184psu@gmail.com> - %s\n"
)

func RunServices(asCoordinator bool) {
	alfredo.SetVerbose(true)
	var (
		dbPath   string
		httpAddr string
		workerID string
	)

	flag.StringVar(&dbPath, "db", defaultDBPath, "Path to SQLite database")
	flag.StringVar(&httpAddr, "http", DefaultCoordinatorURL, "HTTP address for coordinator (coordinator mode only)")
	if asCoordinator {
		flag.StringVar(&workerID, "worker-id", "", "Worker ID (worker mode only, defaults to hostname)")
	}
	flag.Parse()

	// if mode == "" {
	// 	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
	// 	flag.PrintDefaults()
	// 	os.Exit(1)
	// }

	// Ensure database directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	// Initialize database
	db, err := InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	//don't need this
	//defer db.Close()

	log.Printf(ctq_version_fmt, alfredo.BuildVersion())

	if asCoordinator {
		coordinator := NewCoordinator(db, httpAddr)
		if err := coordinator.Start(); err != nil {
			log.Fatalf("Coordinator error: %v", err)
		}
	} else {
		// Default worker ID to hostname
		if workerID == "" {
			hostname, err := os.Hostname()
			if err != nil {
				log.Fatalf("Failed to get hostname: %v", err)
			}
			workerID = hostname
		}

		worker := NewWorker(db, workerID)
		if err := worker.Start(); err != nil {
			log.Fatalf("Worker error: %v", err)
		}
	}
}

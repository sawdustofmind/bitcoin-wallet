package main

import (
	"log"
	"os"

	"github.com/sawdustofmind/bitcoin-wallet/backend/api"
	"github.com/sawdustofmind/bitcoin-wallet/backend/config"
	"github.com/sawdustofmind/bitcoin-wallet/backend/db"
	"github.com/sawdustofmind/bitcoin-wallet/backend/wallet"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.Connect(cfg.DB)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	w, err := wallet.New(cfg.Bitcoin, cfg.XPUB, database)
	if err != nil {
		log.Fatalf("Failed to initialize wallet: %v", err)
	}

	// Start wallet background tasks (e.g. scanning)
	go w.Start()

	r := gin.Default()

	// Manual CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	api.RegisterRoutes(r, w)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}

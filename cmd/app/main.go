package main

import (
	"fmt"
	"github.com/go-redis/redis/v8"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"log"
	"net/http"
	"time"
	"vip-integration/config"
)

type application struct {
}

var rdb *redis.Client
var ctx = context.Background()

func main() {
	// set application config
	var app application

	// Initialize Redis client
	rdb = redis.NewClient(&redis.Options{
		Addr:     "10.1.94.46:6379",                                                                  // Update with your Redis server address
		Password: "vDBAAwyqiEz4/9OupVhu59hrRk3VT4SqsrMDVKqp8E3NTv3vUJCSX4lIs8t9tcGT7rV666OFPEFZGKFw", // No password set
		DB:       9,                                                                                  // Use default DB
	})

	h2s := &http2.Server{}
	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", config.ApplicationPort),
		Handler:      h2c.NewHandler(app.routes(), h2s),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Print("Listening vip-integration [h2c] on", server.Addr)
	log.Fatal(server.ListenAndServe())
}

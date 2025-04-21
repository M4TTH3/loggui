package main

import (
	"flag"
	"github.com/m4tth3/loggui/server"
	"log"
)

// Provide a compilable version of the server client
func main() {
	username := flag.String("username", "", "Non-empty username for the server")
	password := flag.String("password", "", "Non-empty password for the server")

	if *username == "" || *password == "" {
		flag.Usage()
		return
	}

	flag.Parse()

	srv := server.NewServer(*username, *password)

	log.Fatal(srv.ListenAndServe(":8080"))
}

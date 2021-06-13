package main

import (
	"flag"
	"log"
	"os"

	"github.com/sgielen/rufs/security"
	"github.com/sgielen/rufs/version"
)

var (
	circle  = flag.String("circle", "", "Hostname of the circle's discovery server")
	certdir = flag.String("certdir", "", "Where CA certs are read from (see create_ca_pair)")
)

func main() {
	flag.Parse()

	log.Printf("starting rufs %s", version.GetVersion())

	if *circle == "" || *certdir == "" {
		log.Fatal("Flags --circle and --certdir are required")
	}

	if err := os.MkdirAll(*certdir, 0755); err != nil {
		log.Fatalf("Failed to create %s: %v", *certdir, err)
	}

	if err := security.NewCA(*certdir, *circle); err != nil {
		log.Fatalf("Failed to create CA key pair: %v", err)
	}
}

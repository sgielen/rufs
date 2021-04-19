package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/sgielen/rufs/client/connectivity"
	"github.com/sgielen/rufs/config"
	"github.com/sgielen/rufs/content"
	"github.com/sgielen/rufs/security"
)

var (
	discoveryPort = flag.Int("discovery-port", 12000, "Port of the discovery server")
	flag_endp     = flag.String("endpoints", "", "Override our RuFS endpoints (comma-separated IPs or IP:port, autodetected if empty)")
	port          = flag.Int("port", 12010, "content server listen port")
	allowUsers    = flag.String("allow_users", "", "Which local users to allow access to the fuse mount, comma separated")
	mountpoint    = flag.String("mountpoint", "", "Where to mount everyone's stuff")
)

func splitMaybeEmpty(str, sep string) []string {
	if str == "" {
		return nil
	}
	return strings.Split(str, sep)
}

func main() {
	flag.Parse()
	ctx := context.Background()

	if *mountpoint == "" {
		log.Fatalf("--mountpoint must not be empty (see -help)")
	}
	config.MustLoadConfig()

	circles := map[string]*security.KeyPair{}
	var kps []*security.KeyPair
	for _, c := range config.GetCircles() {
		kp, err := config.LoadCerts(c.Name)
		if err != nil {
			log.Fatalf("Failed to read certificates: %v", err)
		}
		circles[c.Name] = kp
		kps = append(kps, kp)
	}

	content, err := content.New(fmt.Sprintf(":%d", *port), kps)
	if err != nil {
		log.Fatalf("failed to create content server: %v", err)
	}
	go content.Run()

	for c, kp := range circles {
		if err := connectivity.ConnectToCircle(ctx, net.JoinHostPort(c, fmt.Sprint(*discoveryPort)), splitMaybeEmpty(*flag_endp, ","), *port, kp); err != nil {
			log.Fatalf("Failed to connect to circle %q: %v", c, err)
		}
	}

	fuse, err := NewFuseMount(*mountpoint, *allowUsers)
	if err != nil {
		log.Fatalf("failed to mount fuse: %v", err)
	}
	err = fuse.Run(ctx)
	if err != nil {
		log.Fatalf("failed to run fuse: %v", err)
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"

	pb "github.com/sgielen/rufs/proto"
	"github.com/sgielen/rufs/security"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	port = flag.Int("port", 12000, "gRPC port")
)

func main() {
	flag.Parse()

	ca, err := security.LoadCAKeyPair("/tmp/rufs/")
	if err != nil {
		log.Fatalf("Failed to load CA key pair: %v", err)
	}

	d := &discovery{
		ca:      ca,
		clients: map[string]*pb.Peer{},
		streams: map[string]pb.DiscoveryService_ConnectServer{},
	}
	d.cond = sync.NewCond(&d.mtx)
	s := grpc.NewServer(grpc.Creds(credentials.NewTLS(ca.TLSConfigForDiscovery())))
	pb.RegisterDiscoveryServiceServer(s, d)
	reflection.Register(s)
	sock, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}
	log.Printf("listening on port %d.", *port)
	if err := s.Serve(sock); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

type discovery struct {
	pb.UnimplementedDiscoveryServiceServer

	ca *security.CAKeyPair

	mtx     sync.Mutex
	clients map[string]*pb.Peer
	streams map[string]pb.DiscoveryService_ConnectServer
	cond    *sync.Cond
}

func (d *discovery) Connect(req *pb.ConnectRequest, stream pb.DiscoveryService_ConnectServer) error {
	ri, ok := credentials.RequestInfoFromContext(stream.Context())
	if !ok {
		// This should never happen.
		return status.Error(codes.Internal, "no RequestInfo attached to context; TLS issue?")
	}
	ti, ok := ri.AuthInfo.(credentials.TLSInfo)
	if !ok {
		return status.Error(codes.PermissionDenied, "couldn't get TLSInfo; TLS issue?")
	}
	if len(ti.State.PeerCertificates) == 0 {
		return status.Error(codes.PermissionDenied, "no client certificate given")
	}
	if ti.State.PeerCertificates[0].Subject.CommonName != req.GetUsername() {
		// TODO(quis): Don't let people pass in their username but just grab it from the certificate.
		return status.Error(codes.PermissionDenied, "given username mismatches certificate")
	}
	d.mtx.Lock()
	defer d.mtx.Unlock()
	d.clients[req.GetUsername()] = &pb.Peer{
		Username:  req.GetUsername(),
		Endpoints: req.GetEndpoints(),
	}
	d.streams[req.GetUsername()] = stream
	d.cond.Broadcast()

	for {
		msg := &pb.ConnectResponse{}
		for _, p := range d.clients {
			msg.Peers = append(msg.Peers, p)
		}
		d.mtx.Unlock()
		if err := stream.Send(msg); err != nil {
			d.mtx.Lock()
			delete(d.clients, req.GetUsername())
			delete(d.streams, req.GetUsername())
			return err
		}
		d.mtx.Lock()

		d.cond.Wait()
		if stream != d.streams[req.GetUsername()] {
			return errors.New("connection from another process for your username")
		}
	}
}

func (d *discovery) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	token := d.ca.CreateToken(req.GetUsername())
	if token != req.GetToken() {
		return nil, status.Error(codes.PermissionDenied, "token is incorrect")
	}
	cert, err := d.ca.Sign(req.GetPublicKey(), req.GetUsername())
	if err != nil {
		return nil, err
	}
	return &pb.RegisterResponse{
		Certificate: cert,
	}, nil
}

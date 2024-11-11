package main

import (
	pb "broadcast_study/pkg/grpc"
	"broadcast_study/server"
	"fmt"
	"log"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/reflection"
)

func main() {
	// 証明書と秘密鍵のパス
	certFile := "/broadcast-grpc-server/certs/cert.pem"
	keyFile := "/broadcast-grpc-server/certs/key.pem"
	log.Printf("certFile: %s, keyFile: %s", certFile, keyFile)

	// TLS認証情報の読み込み
	creds, err := credentials.NewServerTLSFromFile(certFile, keyFile)
	if err != nil {
		log.Fatalf("failed to load TLS keys: %v", err)
	}
	log.Printf(creds.Info().SecurityProtocol)

	port := 8081
	log.Printf("Server starting on port %d", port)
	log.Println(("gRPCのサーバを起動します"))
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	// TLS対応のgRPCサーバーを起動
	s := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterMatchRoomServer(s, server.NewServer())
	reflection.Register(s)

	log.Printf("Server listening at %v", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
package main

import (
	pb "broadcast_study/pkg/grpc"
	"log"
	"net"

	"github.com/google/uuid"
	"google.golang.org/grpc"

	"sync"
)

type waitingPlayer struct {
	playerID string
	stream  pb.MatchRoom_MatchingServer
	gameType pb.GameType
}

type server struct {
	pb.UnimplementedMatchRoomServer
	waitingPlayers map[pb.GameType][]waitingPlayer // ゲームタイプごとの待機プレイヤー
	mu             sync.RWMutex
}

func (s *server) addPlayer(playerID string, gameType pb.GameType, stream pb.MatchRoom_MatchingServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.waitingPlayers[gameType] = append(s.waitingPlayers[gameType], waitingPlayer{
		playerID: playerID,
		stream:   stream,
		gameType: gameType,
	})
}

func (s *server) removePlayer(playerID string, gameType pb.GameType) {
	s.mu.Lock()
	defer s.mu.Unlock()
	players := s.waitingPlayers[gameType]
	for i, p := range players {
		if p.playerID == playerID {
			s.waitingPlayers[gameType] = append(players[:i], players[i+1:]...)
			break
		}
	}
}

func (s* server) matchPlayers(gameType pb.GameType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	players := s.waitingPlayers[gameType]
	if len(players) >= 2 {
		player1 := players[0]
		player2 := players[1]

		roomID := uuid.Must(uuid.NewRandom()).String()

		// Broadcast: 全てのプレイヤーにマッチングを通知

		go func() {
			err1 := player1.stream.Send(&pb.MatchResponse{
				Message: "あなたは" + player2.playerID + "とマッチングしました",
				RoomId:  roomID,
				PlayerId: player2.playerID,
				GameType: gameType,
			})
			err2 := player2.stream.Send(&pb.MatchResponse{
				Message: "あなたは" + player1.playerID + "とマッチングしました",
				RoomId:  roomID,
				PlayerId: player1.playerID,
				GameType: gameType,
			})

			if err1 != nil {
				log.Printf("error: %v", err1)
			}
			if err2 != nil {
				log.Printf("error: %v", err2)
			}
		}()

		s.waitingPlayers[gameType] = s.waitingPlayers[gameType][2:]
	}
}

func (s *server) Matching(req *pb.MatchRequest, stream pb.MatchRoom_MatchingServer) error {
	log.Printf("Player %s requests to join game %v", req.PlayerId, req.GameType)

	s.addPlayer(req.PlayerId, req.GameType, stream)

	s.matchPlayers(req.GameType)

	defer s.removePlayer(req.PlayerId, req.GameType)

	select {}
}

func main() {
	addr := ":50051"
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterMatchRoomServer(s, &server{
		waitingPlayers: make(map[pb.GameType][]waitingPlayer),
	})

	log.Printf("Server listening at %v", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

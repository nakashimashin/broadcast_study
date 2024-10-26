package server

import (
	"log"

	pb "broadcast_study/pkg/grpc"

	"github.com/google/uuid"
)

func (s *server) Matching(req *pb.MatchRequest, stream pb.MatchRoom_MatchingServer) error {
	log.Printf("Player %s requests to join game %v", req.PlayerId, req.GameType)

	s.addPlayer(req.PlayerId, req.GameType, stream)

	s.matchPlayers(req.GameType)

	defer s.removePlayer(req.PlayerId, req.GameType)

	select {}
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

		// Room: マッチングしたプレイヤーを登録
		room := &gameRoom{
			roomID: roomID,
			players: map[string]int32{player1.playerID: 0, player2.playerID: 0},
			playerStreams: map[string]pb.MatchRoom_KeyCollectServer{},
		}

		s.activeRooms[roomID] = room

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
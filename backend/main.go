package main

import (
	pb "broadcast_study/pkg/grpc"
	"io"
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

type gameRoom struct {
	roomID string
	players map[string]int32
	playerStreams map[string]pb.MatchRoom_KeyCollectServer
	mu sync.RWMutex
}

type server struct {
	pb.UnimplementedMatchRoomServer
	waitingPlayers map[pb.GameType][]waitingPlayer // ゲームタイプごとの待機プレイヤー
	activeRooms map[string]*gameRoom
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

func (s *server) KeyCollect(stream pb.MatchRoom_KeyCollectServer) error {
    var roomID string
    var playerID string

    // 最初のストリーム受信でプレイヤーを登録
    req, err := stream.Recv()
    if err != nil {
        log.Printf("error receiving key collect request: %v", err)
        return err
    }

    roomID = req.RoomId
    playerID = req.PlayerId

    // 部屋が存在するか確認
    s.mu.RLock()
    room, exists := s.activeRooms[roomID]
    s.mu.RUnlock()

    if !exists {
        return stream.Send(&pb.KeyCollectResponse{
            Message:   "部屋が存在しません",
            RoomId:    roomID,
            IsGameOver: true,
        })
    }

    // プレイヤーをstreamに登録
    room.mu.Lock()
    room.playerStreams[playerID] = stream
    room.players[playerID] = 0

    // 全プレイヤーが揃った場合のみ初回通知を送信
    if len(room.playerStreams) == 2 {
        for pid, playerStream := range room.playerStreams {
            err := playerStream.Send(&pb.KeyCollectResponse{
                Message:    "ゲームが開始されました",
                RoomId:     roomID,
                PlayerKeys: room.players,
                IsGameOver: false,
            })
            if err != nil {
                log.Printf("error sending initial key collect response to player %s: %v", pid, err)
            }
        }
    }
    room.mu.Unlock()

    for {
        req, err := stream.Recv()
        if err != nil {
            if err == io.EOF {
                log.Printf("Player %s disconnected", playerID)
                return nil
            }
            log.Printf("error receiving key collect request: %v", err)
            return err
        }

        // 鍵の取得進捗を更新
        room.mu.Lock()
        room.players[playerID] = req.TotalKeys
        room.mu.Unlock()

        var gameOver bool
        var result string
        if req.TotalKeys >= 5 {
            gameOver = true
            result = "win"
        }

        for pid, playerStream := range room.playerStreams {
            err := playerStream.Send(&pb.KeyCollectResponse{
                Message:    playerID + "が鍵を取得しました",
                RoomId:     roomID,
                PlayerKeys: room.players,
                IsGameOver: gameOver,
                Result:     result,
            })
            if err != nil {
                log.Printf("error sending key collect response to player %s: %v", pid, err)
            }
        }

        // ゲームが終了したら部屋を削除
        if gameOver {
            s.mu.Lock()
            delete(s.activeRooms, roomID)
            s.mu.Unlock()
            break
        }
    }

    return nil
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
		activeRooms: make(map[string]*gameRoom),
	})

	log.Printf("Server listening at %v", addr)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

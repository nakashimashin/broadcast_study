package main

import (
	pb "broadcast_study/pkg/grpc"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/google/uuid"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

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
	// プレイヤーを待機プレイヤーに追加
	s.waitingPlayers[gameType] = append(s.waitingPlayers[gameType], waitingPlayer{
		playerID: playerID,
		stream:   stream,
		gameType: gameType,
	})
	log.Println(playerID, "を追加しました")
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

	// 指定したゲームタイプの待機プレイヤーを取得
	players := s.waitingPlayers[gameType]
	if len(players) >= 2 {

		// マッチングしたプレイヤーを取得
		// ここでは最初の2人を取得
		player1 := players[0]
		player2 := players[1]

		// Room: ユニークなroomIDを生成
		roomID := uuid.Must(uuid.NewRandom()).String()

		// Room: マッチングしたプレイヤーを登録
		room := &gameRoom{
			roomID: roomID,
			players: map[string]int32{player1.playerID: 0, player2.playerID: 0}, // 初期は鍵の数を0に設定
			playerStreams: map[string]pb.MatchRoom_KeyCollectServer{},
		}

		// Room: アクティブな部屋に登録
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

		// マッチングしたプレイヤーを待機プレイヤーから削除
		s.waitingPlayers[gameType] = s.waitingPlayers[gameType][2:]
	}
}

func (s *server) KeyCollect(stream pb.MatchRoom_KeyCollectServer) error {
	// プレイヤーがKeyCollect streamを開いたことを通知
	log.Println("KeyCollect stream opened")
    var roomID string
    var playerID string

	// ストリームからのリクエストを受け取る(プレイヤーからのリクエスト)
    req, err := stream.Recv()
    if err != nil {
        log.Printf("error receiving key collect request: %v", err)
        return err
    }

    roomID = req.RoomId
    playerID = req.PlayerId
	// プレイヤーがどのRoomIDにリクエストを送っているかを確認
	log.Printf("Received KeyCollect request for RoomID: %s, PlayerID: %s", roomID, playerID)


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
	log.Printf("Player %s registered in room %s", playerID, roomID)
    // 全プレイヤーが揃った場合のみ初回通知を送信
    if len(room.playerStreams) == 2 {
		log.Printf("2人のプレイヤーが揃いました")
        for pid, playerStream := range room.playerStreams {
            err := playerStream.Send(&pb.KeyCollectResponse{
                Message:    "プレイヤーが揃いました。ゲームが開始されます。",
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

        // 鍵が5個集まったらゲーム終了
		if req.TotalKeys >= 5 {
			winnerID := playerID

			// 勝者と敗者にそれぞれの結果を通知
			for pid, playerStream := range room.playerStreams {
				result := "lose"
				if pid == winnerID {
					result = "win"
				}
				log.Printf("Notifying player %s of result: %s", pid, result)
				err := playerStream.Send(&pb.KeyCollectResponse{
					Message:    winnerID + "がゲームに勝ちました",
					RoomId:     roomID,
					PlayerKeys: room.players,
					IsGameOver: true,
					Result:     result,
				})
				if err != nil {
					log.Printf("error sending key collect response to player %s: %v", pid, err)
				}
			}

			// ゲーム終了通知後に部屋を削除
			log.Printf("Deleting room %s after game over notifications", roomID)
			room.mu.Unlock()
			s.mu.Lock()
			delete(s.activeRooms, roomID)
			s.mu.Unlock()
			return nil
		}

        room.mu.Unlock()

        // 進捗を全プレイヤーに通知
        for pid, playerStream := range room.playerStreams {
            err := playerStream.Send(&pb.KeyCollectResponse{
                Message:    playerID + "が鍵を取得しました",
                RoomId:     roomID,
                PlayerKeys: room.players,
                IsGameOver: false,
            })
            if err != nil {
                log.Printf("error sending key collect response to player %s: %v", pid, err)
            }
        }
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
	port := 8081
	grpcServer := grpc.NewServer()
	pb.RegisterMatchRoomServer(grpcServer, &server{
		waitingPlayers: make(map[pb.GameType][]waitingPlayer),
		activeRooms:    make(map[string]*gameRoom),
	})
	reflection.Register(grpcServer)

	// gRPC-Web サーバーのラップ
	grpcWebServer := grpcweb.WrapServer(grpcServer, grpcweb.WithCorsForRegisteredEndpointsOnly(false), grpcweb.WithOriginFunc(func(origin string) bool { return true }))

	// HTTP サーバーの起動とリクエストハンドリング
	handler := func(resp http.ResponseWriter, req *http.Request) {
		if grpcWebServer.IsGrpcWebRequest(req) || grpcWebServer.IsGrpcWebSocketRequest(req) {
			grpcWebServer.ServeHTTP(resp, req)
			return
		}
		http.NotFound(resp, req)
	}

	httpServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handler),
	}

	log.Printf("Server started on port %v", port)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	log.Println("Stopping server...")
	grpcServer.GracefulStop()
}
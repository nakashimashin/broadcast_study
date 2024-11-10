package server

// import (
// 	"log"

// 	pb "broadcast_study/pkg/grpc"
// )

// func (s *server) PositionGame(stream pb.MatchRoom_PositionGameServer) error {
// 	var roomID string
// 	var playerID string

// 	req, err := stream.Recv()
// 	if err != nil {
// 		log.Printf("error receiving position game request: %v", err)
// 		return err
// 	}

// 	roomID = req.RoomId
// 	playerID = req.PlayerId

// 	s.mu.RLock()
// 	room, exists := s.activeRooms[roomID]
// 	s.mu.RUnlock()

// 	if !exists {
// 		return stream.Send(&pb.PositionGameResponse{
// 			Message:   "部屋が存在しません",
// 			RoomId:    roomID,
// 			IsGameOver: true,
// 		})
// 	}

// 	// プレイヤーをstreamに登録
// 	room.mu.Lock()
// 	room.playerStreams[playerID] = stream

// 	// 全プレイヤーが揃った場合のみ初回通知を送信
// 	if len(room.playerStreams) == 2 {
// 		for pid, playerStream := range room.playerStreams {
// 			err := playerStream.Send(&pb.PositionGameResponse{
// 				Message:    "ゲームが開始されました",
// 				RoomId:     roomID,
// 				IsGameOver: false,
// 			})
// 			if err != nil {
// 				log.Printf("error sending initial position game response to player %s: %v", pid, err)
// 			}
// 		}
// 	}
// 	room.mu.Unlock()
// }
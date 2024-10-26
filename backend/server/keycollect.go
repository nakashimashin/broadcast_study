package server

import (
	"io"
	"log"

	pb "broadcast_study/pkg/grpc"
)

func (s *server) KeyCollect(stream pb.MatchRoom_KeyCollectServer) error {
    var roomID string
    var playerID string

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

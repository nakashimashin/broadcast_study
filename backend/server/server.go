package server

import (
	"sync"

	pb "broadcast_study/pkg/grpc"
)

// serverのインスタンスを作成
type waitingPlayer struct {
    playerID string
    stream   pb.MatchRoom_MatchingServer
    gameType pb.GameType
}

type gameRoom struct {
    roomID         string
    players        map[string]int32
    playerStreams  map[string]pb.MatchRoom_KeyCollectServer
    mu             sync.RWMutex
}

type server struct {
    pb.UnimplementedMatchRoomServer
    waitingPlayers map[pb.GameType][]waitingPlayer
    activeRooms    map[string]*gameRoom
    mu             sync.RWMutex
}

// 新しいサーバのインスタンスを作成
func NewServer() *server {
	return &server{
		waitingPlayers: make(map[pb.GameType][]waitingPlayer),
		activeRooms:    make(map[string]*gameRoom),
	}
}

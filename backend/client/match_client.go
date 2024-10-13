package main

import (
	pb "broadcast_study/pkg/grpc"
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
)

func main() {
	// サーバーアドレス
	addr := "localhost:50051"
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewMatchRoomClient(conn)

	// プレイヤーIDを取得
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter your player ID: ")
	playerID, _ := reader.ReadString('\n')
	playerID = playerID[:len(playerID)-1] // 改行コードを削除

	// ゲーム選択
	fmt.Println("Select a game:")
	fmt.Println("0: KEY_COLLECTION")
	fmt.Println("1: Battle_Game")
	fmt.Println("2: Janken")

	gameChoice, _ := reader.ReadString('\n')
	gameChoice = gameChoice[:len(gameChoice)-1] // 改行コードを削除
	gameType, err := strconv.Atoi(gameChoice)
	if err != nil || gameType < 0 || gameType > 2 {
		log.Fatalf("Invalid game choice: %v", err)
	}

	// gRPCのコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second) // 30秒に変更
	defer cancel()

	// マッチングリクエストを送信
	stream, err := c.Matching(ctx, &pb.MatchRequest{
		PlayerId: playerID,
		GameType: pb.GameType(gameType),
	})
	if err != nil {
		log.Fatalf("could not start matching: %v", err)
	}

	// サーバーからのマッチング結果を待つ
	resp, err := stream.Recv()
	if err != nil {
		log.Fatalf("failed to receive response: %v", err)
	}

	// マッチング結果を表示し、処理を終了
	fmt.Printf("Matched: %s, RoomID: %s, Opponent: %s\n", resp.Message, resp.RoomId, resp.PlayerId)

	// マッチングが完了したため、プログラムを終了
	fmt.Println("Matching complete. Exiting...")
}

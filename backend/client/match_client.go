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
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
    // サーバーアドレス
    addr := "localhost:50051"
    conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

    // マッチング結果を表示
    fmt.Printf("Matched: %s, RoomID: %s, Opponent: %s\n", resp.Message, resp.RoomId, resp.PlayerId)

    // 鍵集めゲームに移行
    keyCollectGame(c, playerID, resp.RoomId)
}

func keyCollectGame(c pb.MatchRoomClient, playerID, roomID string) {
    fmt.Println("Starting key collection game...")

    ctx := context.Background()
    stream, err := c.KeyCollect(ctx)
    if err != nil {
        log.Fatalf("could not start key collection game: %v", err)
    }

    totalKeys := 0
    gameOver := false

    // サーバーからの進捗メッセージを非同期で受け取るためのゴルーチン
    go func() {
        for {
            resp, err := stream.Recv()
            if err != nil {
                if !gameOver {
                    log.Fatalf("failed to receive key collect response: %v", err)
                }
                break
            }

            fmt.Printf("Message: %s, RoomID: %s, Keys: %v\n", resp.Message, resp.RoomId, resp.PlayerKeys)

            // ゲーム終了メッセージの受信
            if resp.IsGameOver {
                fmt.Printf("Game over! Result: %s\n", resp.Result)
                gameOver = true
                break
            }
        }
    }()

    // 初期状態を送信
    err = stream.Send(&pb.KeyCollectRequest{
        RoomId:    roomID,
        PlayerId:  playerID,
        TotalKeys: int32(totalKeys), // 鍵の数を0に設定して送信
    })
    if err != nil {
        log.Fatalf("failed to send initial key collect request: %v", err)
    }

    reader := bufio.NewReader(os.Stdin)
    for totalKeys < 5 && !gameOver {
        fmt.Print("Type 'Get' to collect a key: ")
        input, _ := reader.ReadString('\n')
        input = input[:len(input)-1]

        // "Get" を入力した場合に鍵を取得
        if input == "Get" {
            totalKeys++
            err := stream.Send(&pb.KeyCollectRequest{
                RoomId:    roomID,
                PlayerId:  playerID,
                TotalKeys: int32(totalKeys),
            })
            if err != nil {
                log.Fatalf("failed to send key collect request: %v", err)
            }
        }
    }

    if gameOver {
        stream.CloseSend()
    }
}

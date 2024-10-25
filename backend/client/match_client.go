package main

import (
	pb "broadcast_study/pkg/grpc"
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func main() {
    // サーバーアドレス
    addr := "localhost:8081"

    // サーバー証明書を読み込む
    certFile := "../certs/cert.pem" // サーバー証明書のパスを指定
    cert, err := ioutil.ReadFile(certFile)
    if err != nil {
        log.Fatalf("failed to read server certificate: %v", err)
    }

    // 読み込んだ証明書を使って証明書プールを作成
    certPool := x509.NewCertPool()
    if !certPool.AppendCertsFromPEM(cert) {
        log.Fatalf("failed to add server certificate to cert pool")
    }

    // // TLS設定を作成
    // creds := credentials.NewClientTLSFromCert(certPool, "")

    // TLS設定を作成（サーバー名検証を無効化）
    creds := credentials.NewTLS(&tls.Config{
        RootCAs:            certPool,
        InsecureSkipVerify: true, // サーバー名検証を無効化
    })


    // gRPC接続を作成
    conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(creds))
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

    // サーバーからの進捗メッセージを非同期で受け取るためのゴルーチン
    go func() {
        for {
            resp, err := stream.Recv()
            if err == io.EOF {
                fmt.Println("Stream closed by server.")
                os.Exit(0)  // サーバー側がストリームを閉じたら終了
            }
            if err != nil {
                log.Printf("failed to receive key collect response: %v", err)
                os.Exit(1)  // エラーが発生した場合はエラーコードで終了
            }

            fmt.Printf("Message: %s, RoomID: %s, Keys: %v\n", resp.Message, resp.RoomId, resp.PlayerKeys)

            // ゲーム終了メッセージの受信
            if resp.IsGameOver {
                fmt.Printf("Game over! Result: %s\n", resp.Result)
                err = stream.CloseSend()  // ストリームを閉じる
                if err != nil {
                    log.Fatalf("failed to close stream: %v", err)
                }
                os.Exit(0)  // ゲーム終了時にプログラムを終了
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
    for {
        fmt.Print("Type 'Get' to collect a key: ")

        input, _ := reader.ReadString('\n')
        input = input[:len(input)-1]

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
}

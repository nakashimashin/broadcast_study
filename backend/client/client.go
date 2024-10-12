package main

import (
	pb "broadcast_study/pkg/grpc"
	"bufio"
	"context"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
)

func main() {
    addr := "localhost:50051"
    conn, err := grpc.Dial(addr, grpc.WithInsecure())
    if err != nil {
        log.Fatalf("did not connect: %v", err)
    }
    defer conn.Close()
    c := pb.NewChatRoomClient(conn)

    ctx := context.Background()

    // サーバーとの双方向ストリームを作成
    stream, err := c.Chat(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // 標準入力からのメッセージを送信するためのgoroutine
    go func() {
        reader := bufio.NewReader(os.Stdin)
        for {
            log.Print("Enter message: ")
            message, _ := reader.ReadString('\n')
            if err := stream.Send(&pb.ChatRequest{Message: message}); err != nil {
                log.Fatalf("failed to send message: %v", err)
            }
            time.Sleep(500 * time.Millisecond) // 連続送信を防ぐための短いスリープ
        }
    }()

    // サーバーからのメッセージを受信するループ
    for {
        resp, err := stream.Recv()
        if err != nil {
            log.Fatalf("failed to receive message: %v", err)
        }
        log.Printf("recv: %s", resp.Message)
    }
}

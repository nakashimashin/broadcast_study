## proto ファイルの作成

```
// backendのコンテナに入る
docker compose exec broadcast-grpc-server bash
// protobufディレクトリへ移動
cd protobuf
// ファイルを自動生成
protoc --go_out=../pkg/grpc --go_opt=paths=source_relative \
	--go-grpc_out=../pkg/grpc --go-grpc_opt=paths=source_relative \
	match.proto
```

#!/bin/bash

set -e

echo "Generating Protocol Buffers code..."

# プロジェクトルートディレクトリに移動
cd "$(dirname "$0")/.."

# protocが利用可能か確認
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc is not installed. Please install Protocol Buffers compiler."
    echo "See: https://grpc.io/docs/protoc-installation/"
    exit 1
fi

# protoc-gen-goが利用可能か確認
if ! command -v protoc-gen-go &> /dev/null; then
    echo "Error: protoc-gen-go is not installed."
    echo "Run: go install google.golang.org/protobuf/cmd/protoc-gen-go@latest"
    exit 1
fi

# protoc-gen-go-grpcが利用可能か確認
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo "Error: protoc-gen-go-grpc is not installed."
    echo "Run: go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest"
    exit 1
fi

# proto ファイルからコードを生成
protoc \
    --proto_path=proto \
    --go_out=proto \
    --go_opt=paths=source_relative \
    --go-grpc_out=proto \
    --go-grpc_opt=paths=source_relative \
    proto/keruberosu/v1/*.proto

echo "Protocol Buffers code generation completed successfully!"

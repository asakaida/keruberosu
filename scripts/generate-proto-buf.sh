#!/bin/bash

set -e

echo "Generating Protocol Buffers code using buf..."

# プロジェクトルートディレクトリに移動
cd "$(dirname "$0")/.."

# bufが利用可能か確認
if ! command -v buf &> /dev/null; then
    echo "Error: buf is not installed."
    echo "See: https://buf.build/docs/installation"
    exit 1
fi

# buf generate を実行
buf generate

echo "Protocol Buffers code generation completed successfully!"

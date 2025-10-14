# Example 2: データ書き込み

このサンプルでは、関係性（Relations）と属性（Attributes）を書き込む方法を示します。

### 関係性（Relations）の書き込み

関係性は、エンティティ間の関係を表現します。

```go
client.WriteRelations(ctx, &pb.WriteRelationsRequest{
    Tuples: []*pb.RelationTuple{
        {
            Entity:   &pb.Entity{Type: "document", Id: "doc1"},
            Relation: "owner",
            Subject:  &pb.Entity{Type: "user", Id: "alice"},
        },
    },
})
```

### 属性（Attributes）の書き込み

属性は、エンティティの特性を表現します（ABAC 用）。

```go
client.WriteAttributes(ctx, &pb.WriteAttributesRequest{
    Attributes: []*pb.AttributeData{
        {
            Entity: &pb.Entity{Type: "document", Id: "doc1"},
            Data: map[string]*structpb.Value{
                "public": structpb.NewBoolValue(true),
                "owner_id": structpb.NewStringValue("alice"),
            },
        },
    },
})
```

## 実行方法

```bash
# サーバーが起動していることを確認
cd examples/02_data_write
go run main.go
```

## 前提条件

- サーバーが起動していること
- スキーマが定義されていること（Example 1 を先に実行）

## 期待される出力

```
✅ スキーマが書き込まれました
✅ 関係性が書き込まれました: 3件
✅ 属性が書き込まれました
```

## 関連ドキュメント

- [PRD.md](../../PRD.md) - API 仕様の詳細

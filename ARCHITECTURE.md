# Keruberosu アーキテクチャ図

このドキュメントでは、Keruberosu のアーキテクチャを視覚的に示します。

## 目次

1. [システム全体構成](#システム全体構成)
2. [レイヤーアーキテクチャ](#レイヤーアーキテクチャ)
3. [認可エンジンの処理フロー](#認可エンジンの処理フロー)
4. [DSL パーサーの処理フロー](#dslパーサーの処理フロー)
5. [データフロー](#データフロー)

---

## システム全体構成

```mermaid
graph TB
    subgraph "クライアント"
        Client[gRPC Client]
    end

    subgraph "Keruberosu Server"
        GRPCServer[gRPC Server<br/>:50051]
        Handler[Authorization Handler]

        subgraph "Services"
            SchemaService[Schema Service]
            AuthzEngine[Authorization Engine]
            Parser[DSL Parser]
        end

        subgraph "Repositories"
            SchemaRepo[Schema Repository]
            RelationRepo[Relation Repository]
            AttributeRepo[Attribute Repository]
        end
    end

    subgraph "データストア"
        DB[(PostgreSQL)]
    end

    Client -->|gRPC| GRPCServer
    GRPCServer --> Handler
    Handler --> SchemaService
    Handler --> AuthzEngine
    SchemaService --> Parser
    SchemaService --> SchemaRepo
    AuthzEngine --> RelationRepo
    AuthzEngine --> AttributeRepo
    SchemaRepo --> DB
    RelationRepo --> DB
    AttributeRepo --> DB
```

---

## レイヤーアーキテクチャ

Keruberosu は 4 層のクリーンアーキテクチャを採用しています。

```mermaid
graph TB
    subgraph "Presentation Layer"
        Handler[gRPC Handler]
    end

    subgraph "Application Layer"
        SchemaService[Schema Service]
        AuthzEngine[Authorization Engine<br/>Checker, Evaluator, Expander, Lookup]
        Parser[DSL Parser<br/>Lexer, Parser, Validator]
    end

    subgraph "Domain Layer"
        Entities[Domain Entities<br/>Schema, Permission, Rule]
    end

    subgraph "Infrastructure Layer"
        SchemaRepo[Schema Repository<br/>PostgreSQL]
        RelationRepo[Relation Repository<br/>PostgreSQL]
        AttributeRepo[Attribute Repository<br/>PostgreSQL]
        CELEngine[CEL Engine<br/>ABAC Rule Evaluation]
    end

    Handler --> SchemaService
    Handler --> AuthzEngine
    SchemaService --> Parser
    SchemaService --> SchemaRepo
    SchemaService --> Entities
    AuthzEngine --> RelationRepo
    AuthzEngine --> AttributeRepo
    AuthzEngine --> CELEngine
    AuthzEngine --> Entities
    Parser --> Entities
```

**各層の責務:**

| Layer              | 責務                           | 主要コンポーネント                 |
| ------------------ | ------------------------------ | ---------------------------------- |
| **Presentation**   | gRPC リクエスト/レスポンス処理 | `handlers/`                        |
| **Application**    | ビジネスロジック、パース処理   | `services/`                        |
| **Domain**         | ドメインモデル定義             | `entities/`                        |
| **Infrastructure** | データアクセス、外部サービス   | `repositories/`, `infrastructure/` |

---

## 認可エンジンの処理フロー

### Check API の処理フロー

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant Checker
    participant Evaluator
    participant SchemaService
    participant RelationRepo
    participant AttributeRepo
    participant CELEngine

    Client->>Handler: Check(entity, permission, subject)
    Handler->>Checker: Check(req)
    Checker->>SchemaService: GetSchemaEntity(tenantID)
    SchemaService-->>Checker: Schema
    Checker->>Checker: GetPermission(entityType, permission)
    Checker->>Evaluator: EvaluateRule(req, rule)

    alt RelationRule
        Evaluator->>RelationRepo: Read(entity, relation, subject)
        RelationRepo-->>Evaluator: Tuples
        Evaluator-->>Checker: true/false
    else HierarchicalRule
        Evaluator->>RelationRepo: Read(entity, parent relation)
        RelationRepo-->>Evaluator: Parent entities
        Evaluator->>Evaluator: Recursive check on parent
        Evaluator-->>Checker: true/false
    else ABACRule
        Evaluator->>AttributeRepo: Read(entity attributes)
        AttributeRepo-->>Evaluator: Entity attributes
        Evaluator->>AttributeRepo: Read(subject attributes)
        AttributeRepo-->>Evaluator: Subject attributes
        Evaluator->>CELEngine: Evaluate(expression, env)
        CELEngine-->>Evaluator: Result
        Evaluator-->>Checker: true/false
    else LogicalRule (OR/AND/NOT)
        Evaluator->>Evaluator: Recursive evaluation
        Evaluator-->>Checker: true/false
    end

    Checker-->>Handler: CheckResult
    Handler-->>Client: Response
```

### 評価ルールの種類

```mermaid
graph LR
    Rule[Permission Rule]

    Rule --> RelationRule[Relation Rule<br/>例: owner]
    Rule --> HierarchicalRule[Hierarchical Rule<br/>例: parent.view]
    Rule --> ABACRule[ABAC Rule<br/>例: rule subject.level >= 3]
    Rule --> LogicalRule[Logical Rule<br/>or/and/not]

    LogicalRule --> RelationRule
    LogicalRule --> HierarchicalRule
    LogicalRule --> ABACRule
    LogicalRule --> LogicalRule
```

---

## DSL パーサーの処理フロー

```mermaid
graph TB
    Input[Schema DSL Text]
    Lexer[Lexer<br/>字句解析]
    Parser[Parser<br/>構文解析]
    Validator[Validator<br/>検証]
    Converter[Converter<br/>変換]
    Output[Domain Entities]

    Input --> Lexer
    Lexer -->|Tokens| Parser
    Parser -->|AST| Validator
    Validator -->|検証済みAST| Converter
    Converter -->|Schema Entity| Output

    subgraph "Lexer の役割"
        L1[入力文字列をトークンに分割]
        L2[キーワード・演算子を識別]
        L3[文字列リテラル・数値を抽出]
    end

    subgraph "Parser の役割"
        P1[トークンからASTを構築]
        P2[構文ルールを適用]
        P3[階層構造を表現]
    end

    subgraph "Validator の役割"
        V1[エンティティ・関係性の存在確認]
        V2[循環参照のチェック]
        V3[型の整合性確認]
    end
```

### トークン → AST → エンティティの変換例

```text
DSL:
  entity document {
    relation owner: user
    permission edit = owner or editor
  }

↓ Lexer (字句解析)

Tokens:
  ENTITY, IDENTIFIER("document"), LBRACE,
  RELATION, IDENTIFIER("owner"), COLON, IDENTIFIER("user"),
  PERMISSION, IDENTIFIER("edit"), EQUALS, IDENTIFIER("owner"), OR, IDENTIFIER("editor"),
  RBRACE

↓ Parser (構文解析)

AST:
  EntityAST {
    Name: "document"
    Relations: [
      RelationAST { Name: "owner", TargetType: "user" }
    ]
    Permissions: [
      PermissionAST {
        Name: "edit"
        Rule: LogicalPermissionAST {
          Operator: "or"
          Left: RelationPermissionAST { Relation: "owner" }
          Right: RelationPermissionAST { Relation: "editor" }
        }
      }
    ]
  }

↓ Converter (変換)

Domain Entity:
  Entity {
    Name: "document"
    Relations: [
      Relation { Name: "owner", TargetType: "user" }
    ]
    Permissions: [
      Permission {
        Name: "edit"
        Rule: LogicalRule {
          Operator: "or"
          Left: RelationRule { Relation: "owner" }
          Right: RelationRule { Relation: "editor" }
        }
      }
    ]
  }
```

---

## データフロー

### 1. スキーマ定義フロー

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant SchemaService
    participant Parser
    participant Validator
    participant SchemaRepo
    participant DB

    Client->>Handler: WriteSchema(schema_dsl)
    Handler->>SchemaService: WriteSchema(dsl)
    SchemaService->>Parser: Parse(dsl)
    Parser->>Parser: Lexical Analysis
    Parser->>Parser: Syntax Analysis
    Parser-->>SchemaService: AST
    SchemaService->>Validator: Validate(AST)
    Validator-->>SchemaService: Errors or Success
    SchemaService->>SchemaService: Convert AST to Entity
    SchemaService->>SchemaRepo: Create(tenantID, dsl, schemaEntity)
    SchemaRepo->>DB: INSERT schema
    DB-->>SchemaRepo: Success
    SchemaRepo-->>SchemaService: Success
    SchemaService-->>Handler: Success
    Handler-->>Client: WriteSchemaResponse
```

### 2. リレーション書き込みフロー

```mermaid
sequenceDiagram
    participant Client
    participant Handler
    participant RelationRepo
    participant DB

    Client->>Handler: WriteRelations(tuples)
    Handler->>RelationRepo: CreateBatch(tenantID, tuples)
    RelationRepo->>DB: BEGIN TRANSACTION
    loop For each tuple
        RelationRepo->>DB: INSERT relation tuple
    end
    RelationRepo->>DB: COMMIT
    DB-->>RelationRepo: Success
    RelationRepo-->>Handler: WrittenCount
    Handler-->>Client: WriteRelationsResponse
```

### 3. Check API の権限判定フロー

```mermaid
graph TB
    Start[Check Request]
    GetSchema[スキーマ取得]
    FindPermission[権限定義を検索]
    EvaluateRule[ルール評価]

    CheckRelation[関係性チェック]
    CheckHierarchical[階層的チェック]
    CheckABAC[ABAC評価]
    CheckLogical[論理演算]

    Success[ALLOWED]
    Denied[DENIED]

    Start --> GetSchema
    GetSchema --> FindPermission
    FindPermission --> EvaluateRule

    EvaluateRule --> CheckRelation
    EvaluateRule --> CheckHierarchical
    EvaluateRule --> CheckABAC
    EvaluateRule --> CheckLogical

    CheckRelation -->|タプル存在| Success
    CheckRelation -->|タプル不存在| Denied

    CheckHierarchical -->|親が許可| Success
    CheckHierarchical -->|親が拒否| Denied

    CheckABAC -->|CEL true| Success
    CheckABAC -->|CEL false| Denied

    CheckLogical -->|true| Success
    CheckLogical -->|false| Denied
```

---

## コンポーネント詳細

### Authorization Engine

```mermaid
classDiagram
    class Checker {
        +Check(req) CheckResult
        -schemaService SchemaServiceInterface
        -evaluator Evaluator
    }

    class Evaluator {
        +EvaluateRule(req, rule) bool
        -schemaService SchemaServiceInterface
        -relationRepo RelationRepository
        -attributeRepo AttributeRepository
        -celEngine CELEngine
    }

    class Expander {
        +Expand(req) ExpandNode
        -schemaService SchemaServiceInterface
        -relationRepo RelationRepository
    }

    class Lookup {
        +LookupEntity(req) []string
        +LookupSubject(req) []string
        -checker Checker
        -schemaService SchemaServiceInterface
        -relationRepo RelationRepository
    }

    Checker --> Evaluator
    Lookup --> Checker
    Evaluator --> CELEngine
```

### Repository Layer

```mermaid
classDiagram
    class SchemaRepository {
        <<interface>>
        +Create(tenantID, dsl, schema)
        +GetByTenant(tenantID) Schema
    }

    class RelationRepository {
        <<interface>>
        +Create(tenantID, tuple)
        +Read(tenantID, filter) []Tuple
        +Delete(tenantID, tuple)
    }

    class AttributeRepository {
        <<interface>>
        +Write(tenantID, entityType, entityID, data)
        +Read(tenantID, entityType, entityID) map
    }

    class PostgresSchemaRepository {
        +Create()
        +GetByTenant()
    }

    class PostgresRelationRepository {
        +Create()
        +Read()
        +Delete()
    }

    class PostgresAttributeRepository {
        +Write()
        +Read()
    }

    SchemaRepository <|.. PostgresSchemaRepository
    RelationRepository <|.. PostgresRelationRepository
    AttributeRepository <|.. PostgresAttributeRepository
```

---

## 技術スタック

```mermaid
graph TB
    subgraph "フロントエンド層"
        gRPC[gRPC / Protocol Buffers]
    end

    subgraph "アプリケーション層"
        Go[Go 1.21+]
        CEL[Google CEL<br/>Common Expression Language]
    end

    subgraph "データ層"
        PostgreSQL[(PostgreSQL 18+)]
        JSONB[JSONB<br/>属性データ]
    end

    subgraph "開発ツール"
        Docker[Docker Compose]
        Migrate[golang-migrate]
    end

    gRPC --> Go
    Go --> CEL
    Go --> PostgreSQL
    PostgreSQL --> JSONB
    Docker --> PostgreSQL
    Migrate --> PostgreSQL
```

---

## 参考資料

- [DESIGN.md](DESIGN.md): 設計ドキュメント（詳細な設計決定）
- [PRD.md](PRD.md): 要求仕様書（API 仕様）
- [DEVELOPMENT.md](DEVELOPMENT.md): 開発進捗管理
- [examples/](examples/): 実装サンプルコード

# Keruberosu PRD

## 概要

Permify を模した ReBAC と ABAC をサポートする認可マイクロサービス。グラフベースの関係性探索により、柔軟で強力な認可判定を実現する。

## 目的

- ブラウザ UI から設定可能な、直感的な認可システムを提供
- ReBAC と ABAC の両方をサポートし、複雑な認可要件に対応
- 高性能かつスケーラブルな認可判定
- Permify 互換の API とスキーマ DSL をサポート:
  - Check（認可チェック）
  - Expand（パーミッションツリー展開）
  - LookupEntity（データフィルタリング）
  - LookupSubject（Subject フィルタリング）
  - SubjectPermission（権限一覧）
  - metadata（snap_token, depth）
  - context（contextual tuples & attributes）
- 複数サーバーインスタンスによる冗長化・高可用性

## アーキテクチャ方針

### 単一サービスアプローチ

Keruberosu は **単一の gRPC サービス（AuthorizationService）** として実装されます。

**なぜ単一サービスなのか？**

1. **業界標準**: Google Zanzibar、Permify、Auth0 FGA、Ory Keto など、全ての主要な認可システムが単一サービスとして設計されています
2. **認可の本質**: Schema（ルール定義）、Relations（関係性データ）、Authorization（権限判定）は密接に連携する 1 つのドメインであり、分離すると複雑性が増します
3. **クライアントの利便性**: アプリケーション開発者は 1 つのサービスに接続するだけで、スキーマ定義、データ書き込み、権限チェックの全てが実行できます
4. **運用の単純化**: デプロイ、スケーリング、モニタリング、トラブルシューティングが容易です
5. **Permify 互換性**: Permify の API 設計を完全に踏襲することで、既存のツールやクライアントライブラリがそのまま使えます

**提供される API**:

単一の `AuthorizationService` が以下の全ての操作を提供：

- **Schema 管理**: WriteSchema, ReadSchema
- **Data 管理**: WriteRelations, DeleteRelations, WriteAttributes
- **Authorization**: Check, Expand, LookupEntity, LookupSubject, SubjectPermission

この設計により、クライアントは 1 つの gRPC サービスに接続するだけで、認可に必要な全ての操作を実行できます。

## API 利用ガイド（ステークホルダー向け）

このセクションでは、Keruberosu の認可 API を実際にどう使うかを、具体的な例を交えて説明します。

### 1. 認可モデルの理解

Keruberosu は ReBAC（関係性ベース）と ABAC（属性ベース）をネイティブにサポートします。また、ReBAC を使って従来の RBAC パターンも実現できます。

#### 1.1 ReBAC (Relationship-Based Access Control) ← Keruberosu のコア機能

関係性ベースの認可。ユーザーとリソースの「関係」を元に権限を判定します。

```text
alice は document#1 の owner
→ ownerは編集・削除・共有ができる

bob は document#1 を view できる relation を持つ
→ bobは閲覧のみ可能
```

メリット: リソース単位の細かい制御、動的な権限管理、階層構造のサポート
用途: Google Docs、GitHub、Notion 等のリソース共有システム

#### 1.2 ABAC (Attribute-Based Access Control) ← Keruberosu のコア機能

属性ベースの認可。リソースやユーザーの「属性」を使ってルールを定義します。

```text
ルール: ドキュメントのis_public == trueなら、誰でも閲覧可能
ルール: ドキュメントのdepartment == ユーザーのdepartmentなら、編集可能
ルール: 営業時間内（9:00-18:00）のみアクセス可能
```

メリット: 柔軟なルール定義、コンテキスト依存の制御
用途: 複雑なビジネスルール、動的な条件判定

#### 1.3 RBAC (Role-Based Access Control) ← ReBAC で実現可能

従来型のロールベース認可。ユーザーにロールを割り当て、ロールに権限を付与します。

```text
ユーザー → ロール → 権限
alice → admin → すべての操作が可能
bob → editor → 編集のみ可能
```

注意: Keruberosu は RBAC を直接サポートしているわけではありませんが、ReBAC を使って RBAC パターンを実現できます。`role`エンティティを定義し、ユーザーをロールのメンバーとして登録することで、従来の RBAC と同じ動作を実現します（ユースケース 1 参照）。

メリット: シンプルで理解しやすい、既存システムからの移行が容易
デメリット: リソース単位の細かい制御ができない
用途: 管理画面、社内ツールなどシンプルな権限管理

### 2. API の全体像

Keruberosu は以下の API を提供します：

| API               | 用途                   | 質問形式                                    |
| ----------------- | ---------------------- | ------------------------------------------- |
| Check             | 認可チェック           | 「alice は doc1 を編集できる？」            |
| Expand            | 権限ツリー展開         | 「doc1 を編集できるのは誰？（ツリー構造）」 |
| LookupEntity      | データフィルタリング   | 「alice が編集できるドキュメント一覧は？」  |
| LookupSubject     | ユーザーフィルタリング | 「doc1 を編集できるユーザー一覧は？」       |
| SubjectPermission | 権限一覧               | 「alice が doc1 に対して持つ権限は？」      |
| WriteSchema       | スキーマ定義           | 認可ルールの定義・更新                      |
| WriteRelations    | 関係性の書き込み       | 「alice を doc1 の owner にする」           |
| WriteAttributes   | 属性の書き込み         | 「doc1 の is_public を true にする」        |

### 3. ユースケース別実例

以下、現実的なユースケースごとに API の使い方を示します。

---

### ユースケース 1: ReBAC で RBAC パターンを実現 - シンプルな管理画面

シナリオ: 社内管理ツールで、admin/editor/viewer の 3 つのロールを管理したい（既存 RBAC システムからの移行パターン）。

#### ステップ 1: スキーマ定義

```text
// WriteSchemaRequest
schema_dsl: """
entity user {}

entity role {
  relation member: user

  permission admin = member
  permission edit = member
  permission view = member
}
"""
```

解説: このスキーマは ReBAC を使って RBAC パターンを実現しています。`role`エンティティを定義し、`user`をその`member`（関係性）として登録することで、従来の RBAC と同じ「ユーザー → ロール → 権限」の構造を表現できます。

#### ステップ 2: ロールの割り当て

```javascript
// TypeScriptクライアントの例
await client.writeRelations({
  tuples: [
    // aliceをadminロールのメンバーにする
    {
      entity: { type: "role", id: "admin" },
      relation: "member",
      subject: { type: "user", id: "alice" },
    },
    // bobをeditorロールのメンバーにする
    {
      entity: { type: "role", id: "editor" },
      relation: "member",
      subject: { type: "user", id: "bob" },
    },
    // charlieをviewerロールのメンバーにする
    {
      entity: { type: "role", id: "viewer" },
      relation: "member",
      subject: { type: "user", id: "charlie" },
    },
  ],
});
```

#### ステップ 3: 認可チェック

```javascript
// 「aliceはadmin権限を持っているか？」
const response = await client.check({
  entity: { type: "role", id: "admin" },
  permission: "admin",
  subject: { type: "user", id: "alice" },
});

console.log(response.can); // CHECK_RESULT_ALLOWED

// 「bobはadmin権限を持っているか？」
const response2 = await client.check({
  entity: { type: "role", id: "admin" },
  permission: "admin",
  subject: { type: "user", id: "bob" },
});

console.log(response2.can); // CHECK_RESULT_DENIED
```

#### フロントエンドでの利用例

```typescript
// Reactコンポーネントでの使用例
function AdminPanel() {
  const { user } = useAuth();
  const [canAccess, setCanAccess] = useState(false);

  useEffect(() => {
    async function checkPermission() {
      const result = await keruberosuClient.check({
        entity: { type: "role", id: "admin" },
        permission: "admin",
        subject: { type: "user", id: user.id },
      });
      setCanAccess(result.can === CheckResult.CHECK_RESULT_ALLOWED);
    }
    checkPermission();
  }, [user.id]);

  if (!canAccess) {
    return <div>アクセス権限がありません</div>;
  }

  return <div>管理画面の内容...</div>;
}
```

---

### ユースケース 2: ReBAC - ドキュメント管理システム（Google Docs ライク）

シナリオ: ドキュメントごとに owner/editor/viewer を設定でき、owner は他のユーザーを招待できる。

#### ステップ 1: スキーマ定義

```text
// WriteSchemaRequest
schema_dsl: """
entity user {}

entity document {
  // 関係性の定義
  relation owner: user
  relation editor: user
  relation viewer: user

  // 権限の定義
  permission delete = owner
  permission share = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer
}
"""
```

#### ステップ 2: ドキュメントの作成と権限設定

```javascript
// aliceが新しいドキュメントを作成
await client.writeRelations({
  tuples: [
    // aliceをdoc1のownerにする
    {
      entity: { type: "document", id: "doc1" },
      relation: "owner",
      subject: { type: "user", id: "alice" },
    },
  ],
});

// aliceがbobをeditorとして招待
await client.writeRelations({
  tuples: [
    {
      entity: { type: "document", id: "doc1" },
      relation: "editor",
      subject: { type: "user", id: "bob" },
    },
  ],
});

// charlieをviewerとして招待
await client.writeRelations({
  tuples: [
    {
      entity: { type: "document", id: "doc1" },
      relation: "viewer",
      subject: { type: "user", id: "charlie" },
    },
  ],
});
```

#### ステップ 3: 認可チェック

```javascript
// 「bobはdoc1を編集できる？」
const result = await client.check({
  entity: { type: "document", id: "doc1" },
  permission: "edit",
  subject: { type: "user", id: "bob" },
});
console.log(result.can); // CHECK_RESULT_ALLOWED（bobはeditor）

// 「charlieはdoc1を編集できる？」
const result2 = await client.check({
  entity: { type: "document", id: "doc1" },
  permission: "edit",
  subject: { type: "user", id: "charlie" },
});
console.log(result2.can); // CHECK_RESULT_DENIED（charlieはviewerのみ）

// 「charlieはdoc1を閲覧できる？」
const result3 = await client.check({
  entity: { type: "document", id: "doc1" },
  permission: "view",
  subject: { type: "user", id: "charlie" },
});
console.log(result3.can); // CHECK_RESULT_ALLOWED
```

#### ステップ 4: データフィルタリング（LookupEntity）

ユーザーがアクセスできるドキュメント一覧を取得：

```javascript
// 「aliceが編集できるドキュメント一覧は？」
const response = await client.lookupEntity({
  entity_type: "document",
  permission: "edit",
  subject: { type: "user", id: "alice" },
});

console.log(response.entity_ids); // ["doc1", "doc3", "doc5", ...]
```

フロントエンドでの活用:

```typescript
// ドキュメント一覧画面
function DocumentList() {
  const { user } = useAuth();
  const [documents, setDocuments] = useState([]);

  useEffect(() => {
    async function fetchAccessibleDocuments() {
      // 編集可能なドキュメントIDを取得
      const result = await keruberosuClient.lookupEntity({
        entity_type: "document",
        permission: "edit",
        subject: { type: "user", id: user.id },
      });

      // IDを元にドキュメントの詳細をDBから取得
      const docs = await fetchDocumentsByIds(result.entity_ids);
      setDocuments(docs);
    }
    fetchAccessibleDocuments();
  }, [user.id]);

  return (
    <ul>
      {documents.map((doc) => (
        <li key={doc.id}>{doc.title}</li>
      ))}
    </ul>
  );
}
```

#### ステップ 5: 権限一覧の取得（SubjectPermission）

特定のドキュメントに対してユーザーが持つ全権限を取得：

```javascript
// 「aliceがdoc1に対して持つ権限は？」
const response = await client.subjectPermission({
  entity: { type: "document", id: "doc1" },
  subject: { type: "user", id: "alice" },
});

console.log(response.results);
// {
//   "view": CHECK_RESULT_ALLOWED,
//   "edit": CHECK_RESULT_ALLOWED,
//   "delete": CHECK_RESULT_ALLOWED,
//   "share": CHECK_RESULT_ALLOWED
// }
```

フロントエンドでの活用:

```typescript
// ドキュメント詳細画面のアクションボタン
function DocumentActions({ documentId }) {
  const { user } = useAuth();
  const [permissions, setPermissions] = useState({});

  useEffect(() => {
    async function fetchPermissions() {
      const result = await keruberosuClient.subjectPermission({
        entity: { type: "document", id: documentId },
        subject: { type: "user", id: user.id },
      });
      setPermissions(result.results);
    }
    fetchPermissions();
  }, [documentId, user.id]);

  return (
    <div>
      {permissions.edit === CheckResult.CHECK_RESULT_ALLOWED && (
        <button>編集</button>
      )}
      {permissions.delete === CheckResult.CHECK_RESULT_ALLOWED && (
        <button>削除</button>
      )}
      {permissions.share === CheckResult.CHECK_RESULT_ALLOWED && (
        <button>共有</button>
      )}
    </div>
  );
}
```

---

### ユースケース 3: ReBAC 階層構造 - フォルダ/ドキュメント

シナリオ: フォルダの権限がドキュメントに継承される。

#### スキーマ定義

```text
schema_dsl: """
entity user {}

entity folder {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer
}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user
  relation parent: folder  // 親フォルダ

  permission delete = owner
  permission edit = owner or editor or parent.edit  // 親フォルダのedit権限を継承
  permission view = owner or editor or viewer or parent.view  // 親フォルダのview権限を継承
}
"""
```

#### 権限の設定

```javascript
// フォルダ「project-a」を作成し、aliceをownerに
await client.writeRelations({
  tuples: [
    {
      entity: { type: "folder", id: "project-a" },
      relation: "owner",
      subject: { type: "user", id: "alice" },
    },
  ],
});

// ドキュメント「spec.md」を作成し、project-aフォルダに配置
await client.writeRelations({
  tuples: [
    {
      entity: { type: "document", id: "spec.md" },
      relation: "parent",
      subject: { type: "folder", id: "project-a" },
    },
  ],
});

// bobをproject-aフォルダのeditorに追加
await client.writeRelations({
  tuples: [
    {
      entity: { type: "folder", id: "project-a" },
      relation: "editor",
      subject: { type: "user", id: "bob" },
    },
  ],
});
```

#### 継承の確認

```javascript
// 「bobはspec.mdを編集できる？」
// → project-aのeditorなので、配下のspec.mdも編集可能
const result = await client.check({
  entity: { type: "document", id: "spec.md" },
  permission: "edit",
  subject: { type: "user", id: "bob" },
});
console.log(result.can); // CHECK_RESULT_ALLOWED（parent.edit経由）
```

---

### ユースケース 4: ABAC - 属性ベースの制御

シナリオ: ドキュメントの公開状態や部署に基づいてアクセス制御。

#### スキーマ定義

```text
schema_dsl: """
entity user {}

entity document {
  relation owner: user

  // 属性の定義
  attribute is_public boolean
  attribute department string

  permission delete = owner
  permission edit = owner
  permission view = owner or check_public or check_department

  // ABACルール: is_publicがtrueなら誰でも閲覧可能
  rule check_public(is_public) {
    is_public == true
  }

  // ABACルール: ユーザーの部署とドキュメントの部署が一致すれば閲覧可能
  rule check_department(department) {
    request.user.department == department
  }
}
"""
```

#### 属性の設定

```javascript
// doc2を公開ドキュメントに設定
await client.writeAttributes({
  attributes: [
    {
      entity: { type: "document", id: "doc2" },
      data: {
        is_public: true,
      },
    },
  ],
});

// doc3を営業部のドキュメントに設定
await client.writeAttributes({
  attributes: [
    {
      entity: { type: "document", id: "doc3" },
      data: {
        department: "sales",
      },
    },
  ],
});

// ユーザーの属性も設定
await client.writeAttributes({
  attributes: [
    {
      entity: { type: "user", id: "dave" },
      data: {
        department: "sales",
      },
    },
  ],
});
```

#### 認可チェック（属性ベース）

```javascript
// 「誰でもdoc2を閲覧できる？」（is_public == true）
const result = await client.check({
  entity: { type: "document", id: "doc2" },
  permission: "view",
  subject: { type: "user", id: "anyone" },
});
console.log(result.can); // CHECK_RESULT_ALLOWED（公開ドキュメント）

// 「daveはdoc3を閲覧できる？」（部署が一致）
const result2 = await client.check({
  entity: { type: "document", id: "doc3" },
  permission: "view",
  subject: { type: "user", id: "dave" },
});
console.log(result2.can); // CHECK_RESULT_ALLOWED（営業部のドキュメント）
```

---

### ユースケース 5: 複合 - GitHub ライクな Organization/Repository 管理

シナリオ: Organization → Repository の階層構造、複数のロール。

#### スキーマ定義

```text
schema_dsl: """
entity user {}

entity organization {
  relation owner: user
  relation member: user

  permission admin = owner
  permission create_repo = owner or member
  permission view = owner or member
}

entity repository {
  relation owner: user
  relation maintainer: user
  relation contributor: user
  relation parent_org: organization

  permission delete = owner
  permission admin = owner or parent_org.admin
  permission write = owner or maintainer or contributor
  permission read = owner or maintainer or contributor or parent_org.member
}
"""
```

#### データ設定

```javascript
// "acme-corp" organizationを作成
await client.writeRelations({
  tuples: [
    {
      entity: { type: "organization", id: "acme-corp" },
      relation: "owner",
      subject: { type: "user", id: "alice" },
    },
    {
      entity: { type: "organization", id: "acme-corp" },
      relation: "member",
      subject: { type: "user", id: "bob" },
    },
  ],
});

// "backend-api" repositoryを作成し、acme-corpに所属
await client.writeRelations({
  tuples: [
    {
      entity: { type: "repository", id: "backend-api" },
      relation: "parent_org",
      subject: { type: "organization", id: "acme-corp" },
    },
    {
      entity: { type: "repository", id: "backend-api" },
      relation: "maintainer",
      subject: { type: "user", id: "charlie" },
    },
  ],
});
```

#### 認可チェック

```javascript
// 「bobはbackend-apiを読める？」
// → acme-corpのmemberなので、parent_org.member経由でreadできる
const result = await client.check({
  entity: { type: "repository", id: "backend-api" },
  permission: "read",
  subject: { type: "user", id: "bob" },
});
console.log(result.can); // CHECK_RESULT_ALLOWED

// 「aliceはbackend-apiを削除できる？」
// → acme-corpのownerだが、repositoryのownerではないので削除不可
const result2 = await client.check({
  entity: { type: "repository", id: "backend-api" },
  permission: "delete",
  subject: { type: "user", id: "alice" },
});
console.log(result2.can); // CHECK_RESULT_DENIED
```

---

### ユースケース 6: Contextual Tuples - 一時的な権限

シナリオ: ドキュメント共有リンクで一時的にアクセス許可。

```javascript
// 「guestユーザーは通常doc1にアクセスできないが、共有リンク経由ならアクセス可能」
const result = await client.check({
  entity: { type: "document", id: "doc1" },
  permission: "view",
  subject: { type: "user", id: "guest" },
  context: {
    tuples: [
      // 一時的にguestをviewerとして追加（DBには保存されない）
      {
        entity: { type: "document", id: "doc1" },
        relation: "viewer",
        subject: { type: "user", id: "guest" },
      },
    ],
  },
});
console.log(result.can); // CHECK_RESULT_ALLOWED
```

---

### ユースケース 7: Contextual Attributes - 時間ベースの制御

シナリオ: 営業時間内のみアクセス可能。

#### スキーマ定義

```text
schema_dsl: """
entity user {}

entity document {
  relation owner: user

  attribute business_hours_only boolean

  permission view = owner or check_business_hours

  rule check_business_hours(business_hours_only) {
    business_hours_only == false or
    (request.context.hour >= 9 and request.context.hour < 18)
  }
}
"""
```

#### 認可チェック（時刻を渡す）

```javascript
// 現在時刻を context として渡す
const now = new Date();
const result = await client.check({
  entity: { type: "document", id: "doc1" },
  permission: "view",
  subject: { type: "user", id: "bob" },
  context: {
    attributes: [
      {
        entity: { type: "document", id: "doc1" },
        data: {
          hour: now.getHours(), // 現在の時刻
        },
      },
    ],
  },
});

// 9:00-18:00の間ならALLOWED、それ以外はDENIED
```

---

### 4. パフォーマンスの考慮事項

#### キャッシュの活用

Keruberosu は自動的に L1/L2 キャッシュを使用します。同じチェックは高速に応答されます。

```javascript
// 1回目: DBアクセスあり（~10ms）
const result1 = await client.check({...});
console.log(result1.metadata.cached); // false

// 2回目: キャッシュヒット（~0.1ms）
const result2 = await client.check({...});
console.log(result2.metadata.cached); // true
```

#### ページネーション

大量のデータを取得する場合は `page_size` と `continuous_token` を使用：

```javascript
let allDocuments = [];
let token = "";

do {
  const response = await client.lookupEntity({
    entity_type: "document",
    permission: "view",
    subject: { type: "user", id: "alice" },
    page_size: 100,
    continuous_token: token,
  });

  allDocuments.push(...response.entity_ids);
  token = response.continuous_token;
} while (token);
```

---

### 5. よくある質問

Q: 既存の RBAC システムから Keruberosu への移行はどうすれば？

A: 段階的に移行できます。Keruberosu は ReBAC を使って RBAC パターンを実現できます（ユースケース 1 参照）。まず既存のロールを`role`エンティティとして定義し、従来と同じ動作を再現します。その後、必要に応じてリソース単位の細かい制御（ReBAC）や属性ベースのルール（ABAC）を追加していくことができます。

Q: 既存の DB に保存されているユーザー情報をどう扱う？

A: Keruberosu は認可のみを担当します。ユーザー情報（名前、メールなど）は既存 DB に保持し、Keruberosu には ID のみを渡します。

Q: パフォーマンスは？

A: キャッシュヒット時は 0.1ms 以下、キャッシュミス時でも 10ms 程度です。LookupEntity など複雑なクエリは 100ms 程度かかる場合があります。

Q: TypeScript/JavaScript クライアントは？

A: gRPC-web または Connect-web を使用してブラウザから直接呼び出せます。認証は JWT を metadata に含めることで実現します。

---

### 6. スキーマ定義 UI の構築方法（重要）

エンドユーザー（特に非技術者）が直接 DSL を書くのは困難です。そのため、ビジュアルなスキーマビルダー UI を提供することが重要です。

#### 6.1 基本的な考え方

DSL 文字列を直接書かせるのではなく、以下のステップで段階的に構築します：

1. エンティティの追加: ボタンで追加、名前を入力
2. リレーションの定義: ドロップダウンで型を選択、名前を入力
3. パーミッションの定義: チェックボックスで演算子を選択、ビジュアルエディタで組み合わせ
4. ABAC ルールの定義: フォームで CEL 式を構築
5. リアルタイムプレビュー: 入力内容から DSL を自動生成して表示

#### 6.2 シンプルな RBAC スキーマの構築例

ユーザーの操作:

```json
[新しいエンティティを追加] ボタンをクリック

┌─────────────────────────────────┐
│ エンティティ名: user            │
│ [保存]                          │
└─────────────────────────────────┘

[新しいエンティティを追加] ボタンをクリック

┌─────────────────────────────────┐
│ エンティティ名: role            │
│                                 │
│ □ リレーションを追加            │
│   名前: member                  │
│   型: user ▼                    │
│   [追加]                        │
│                                 │
│ ☑ パーミッションを追加          │
│   名前: admin                   │
│   定義: ◉ member               │
│         ○ 複雑な式              │
│   [追加]                        │
│                                 │
│ [保存]                          │
└─────────────────────────────────┘
```

自動生成される DSL (リアルタイムプレビュー):

```text
entity user {}

entity role {
  relation member: user

  permission admin = member
}
```

フロントエンド実装例:

```typescript
import { useState } from "react";

interface EntityConfig {
  name: string;
  relations: Array<{
    name: string;
    type: string;
  }>;
  permissions: Array<{
    name: string;
    expression: string;
  }>;
}

function SchemaBuilder() {
  const [entities, setEntities] = useState<EntityConfig[]>([]);
  const [generatedDSL, setGeneratedDSL] = useState("");

  // エンティティ追加
  const addEntity = () => {
    setEntities([
      ...entities,
      {
        name: "",
        relations: [],
        permissions: [],
      },
    ]);
  };

  // DSL生成
  const generateDSL = () => {
    let dsl = "";
    entities.forEach((entity) => {
      dsl += `entity ${entity.name} {\n`;

      // リレーション
      entity.relations.forEach((rel) => {
        dsl += `  relation ${rel.name}: ${rel.type}\n`;
      });

      if (entity.relations.length > 0 && entity.permissions.length > 0) {
        dsl += "\n";
      }

      // パーミッション
      entity.permissions.forEach((perm) => {
        dsl += `  permission ${perm.name} = ${perm.expression}\n`;
      });

      dsl += "}\n\n";
    });

    setGeneratedDSL(dsl);
    return dsl;
  };

  // スキーマ保存
  const saveSchema = async () => {
    const dsl = generateDSL();

    const response = await keruberosuClient.writeSchema({
      schema_dsl: dsl,
    });

    if (response.success) {
      alert("スキーマを保存しました！");
    } else {
      alert(`エラー: ${response.errors.join(", ")}`);
    }
  };

  return (
    <div>
      <h2>スキーマビルダー</h2>

      <button onClick={addEntity}>+ 新しいエンティティを追加</button>

      {entities.map((entity, idx) => (
        <EntityEditor
          key={idx}
          entity={entity}
          onChange={(updated) => {
            const newEntities = [...entities];
            newEntities[idx] = updated;
            setEntities(newEntities);
            generateDSL(); // リアルタイム更新
          }}
        />
      ))}

      <div style={{ marginTop: "2rem" }}>
        <h3>生成されたDSL（プレビュー）</h3>
        <pre style={{ background: "#f5f5f5", padding: "1rem" }}>
          {generatedDSL}
        </pre>
      </div>

      <button onClick={saveSchema}>スキーマを保存</button>
    </div>
  );
}

function EntityEditor({ entity, onChange }) {
  const [showRelationForm, setShowRelationForm] = useState(false);
  const [showPermissionForm, setShowPermissionForm] = useState(false);

  return (
    <div
      style={{ border: "1px solid #ccc", padding: "1rem", margin: "1rem 0" }}
    >
      <input
        type="text"
        placeholder="エンティティ名 (例: document)"
        value={entity.name}
        onChange={(e) => onChange({ ...entity, name: e.target.value })}
        style={{ fontSize: "1.2rem", marginBottom: "1rem" }}
      />

      {/* リレーションセクション */}
      <div>
        <h4>リレーション</h4>
        {entity.relations.map((rel, idx) => (
          <div key={idx}>
            {rel.name}: {rel.type}
          </div>
        ))}

        {showRelationForm ? (
          <RelationForm
            onAdd={(relation) => {
              onChange({
                ...entity,
                relations: [...entity.relations, relation],
              });
              setShowRelationForm(false);
            }}
            onCancel={() => setShowRelationForm(false)}
          />
        ) : (
          <button onClick={() => setShowRelationForm(true)}>
            + リレーションを追加
          </button>
        )}
      </div>

      {/* パーミッションセクション */}
      <div style={{ marginTop: "1rem" }}>
        <h4>パーミッション</h4>
        {entity.permissions.map((perm, idx) => (
          <div key={idx}>
            {perm.name} = {perm.expression}
          </div>
        ))}

        {showPermissionForm ? (
          <PermissionForm
            relations={entity.relations}
            onAdd={(permission) => {
              onChange({
                ...entity,
                permissions: [...entity.permissions, permission],
              });
              setShowPermissionForm(false);
            }}
            onCancel={() => setShowPermissionForm(false)}
          />
        ) : (
          <button onClick={() => setShowPermissionForm(true)}>
            + パーミッションを追加
          </button>
        )}
      </div>
    </div>
  );
}

function RelationForm({ onAdd, onCancel }) {
  const [name, setName] = useState("");
  const [type, setType] = useState("");

  return (
    <div style={{ background: "#f9f9f9", padding: "1rem", margin: "0.5rem 0" }}>
      <input
        type="text"
        placeholder="リレーション名 (例: owner)"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />
      <select value={type} onChange={(e) => setType(e.target.value)}>
        <option value="">型を選択</option>
        <option value="user">user</option>
        <option value="organization">organization</option>
        <option value="document">document</option>
        {/* 動的に追加されたエンティティも表示 */}
      </select>
      <button onClick={() => onAdd({ name, type })}>追加</button>
      <button onClick={onCancel}>キャンセル</button>
    </div>
  );
}

function PermissionForm({ relations, onAdd, onCancel }) {
  const [name, setName] = useState("");
  const [mode, setMode] = useState<"simple" | "advanced">("simple");
  const [selectedRelations, setSelectedRelations] = useState<string[]>([]);
  const [operator, setOperator] = useState<"or" | "and">("or");

  const generateExpression = () => {
    if (mode === "simple") {
      return selectedRelations.join(` ${operator} `);
    }
    // advanced mode: ビジュアルエディタで構築した式
    return ""; // 実装省略
  };

  return (
    <div style={{ background: "#f9f9f9", padding: "1rem", margin: "0.5rem 0" }}>
      <input
        type="text"
        placeholder="パーミッション名 (例: edit)"
        value={name}
        onChange={(e) => setName(e.target.value)}
      />

      <div>
        <label>
          <input
            type="radio"
            checked={mode === "simple"}
            onChange={() => setMode("simple")}
          />
          シンプルモード
        </label>
        <label>
          <input
            type="radio"
            checked={mode === "advanced"}
            onChange={() => setMode("advanced")}
          />
          高度な式
        </label>
      </div>

      {mode === "simple" && (
        <div>
          <p>含めるリレーション:</p>
          {relations.map((rel) => (
            <label key={rel.name}>
              <input
                type="checkbox"
                checked={selectedRelations.includes(rel.name)}
                onChange={(e) => {
                  if (e.target.checked) {
                    setSelectedRelations([...selectedRelations, rel.name]);
                  } else {
                    setSelectedRelations(
                      selectedRelations.filter((r) => r !== rel.name)
                    );
                  }
                }}
              />
              {rel.name}
            </label>
          ))}

          <div>
            <label>
              <input
                type="radio"
                checked={operator === "or"}
                onChange={() => setOperator("or")}
              />
              いずれか (or)
            </label>
            <label>
              <input
                type="radio"
                checked={operator === "and"}
                onChange={() => setOperator("and")}
              />
              すべて (and)
            </label>
          </div>

          <p>生成される式: {generateExpression()}</p>
        </div>
      )}

      <button onClick={() => onAdd({ name, expression: generateExpression() })}>
        追加
      </button>
      <button onClick={onCancel}>キャンセル</button>
    </div>
  );
}
```

#### 6.3 ReBAC 階層構造の構築例

ユーザーの操作 (ドキュメント管理システム):

```text
エンティティ: document
  リレーション:
    ☑ owner (user)
    ☑ editor (user)
    ☑ viewer (user)
    ☑ parent (folder)  ← 階層構造のために追加

  パーミッション:
    ☑ delete
      = owner

    ☑ edit
      = owner or editor or parent.edit  ← 親フォルダの権限を継承

    ☑ view
      = owner or editor or viewer or parent.view
```

自動生成される DSL:

```text
entity document {
  relation owner: user
  relation editor: user
  relation viewer: user
  relation parent: folder

  permission delete = owner
  permission edit = owner or editor or parent.edit
  permission view = owner or editor or viewer or parent.view
}
```

#### 6.4 ABAC（属性ベース）ルールの構築例

ユーザーの操作:

```text
エンティティ: document
  属性:
    ☑ is_public (boolean)
    ☑ department (string)

  ルール:
    ☑ check_public
      条件: is_public == true

    ☑ check_department
      条件: request.user.department == department

  パーミッション:
    ☑ view
      = owner or check_public or check_department
```

ルール構築 UI:

```typescript
function RuleBuilder({ onSave }) {
  const [ruleName, setRuleName] = useState('');
  const [leftOperand, setLeftOperand] = useState('');
  const [operator, setOperator] = useState('==');
  const [rightOperand, setRightOperand] = useState('');

  return (
    <div>
      <input
        placeholder="ルール名 (例: check_public)"
        value={ruleName}
        onChange={(e) => setRuleName(e.target.value)}
      />

      <div>
        <select value={leftOperand} onChange={(e) => setLeftOperand(e.target.value)}>
          <option value="">左辺を選択</option>
          <option value="is_public">is_public (この属性)</option>
          <option value="department">department (この属性)</option>
          <option value="request.user.department">request.user.department</option>
        </select>

        <select value={operator} onChange={(e) => setOperator(e.target.value)}>
          <option value="==">==(等しい)</option>
          <option value="!=">!=(等しくない)</option>
          <option value=">">>(より大きい)</option>
          <option value=">=">>=(以上)</option>
          <option value="<"><(より小さい)</option>
          <option value="<="><=(以下)</option>
          <option value="in">in(含まれる)</option>
        </select>

        <input
          placeholder="右辺の値"
          value={rightOperand}
          onChange={(e) => setRightOperand(e.target.value)}
        />
      </div>

      <p>生成される式: {leftOperand} {operator} {rightOperand}</p>

      <button onClick={() => onSave({
        name: ruleName,
        expression: `${leftOperand} ${operator} ${rightOperand}`
      })}>
        保存
      </button>
    </div>
  );
}
```

#### 6.5 実装のポイント

1. リアルタイムプレビュー: ユーザーの入力内容から常に DSL を生成して表示
2. バリデーション: 入力内容の妥当性をチェック（例: エンティティ名の重複、未定義の型の参照など）
3. インポート/エクスポート: 既存の DSL を読み込んで UI に反映、または UI から DSL をエクスポート
4. テンプレート: よくあるパターン（RBAC、Google Docs ライク、GitHub ライクなど）をテンプレートとして提供
5. 段階的な開示: 初心者向けにはシンプルモード、上級者向けには高度な機能を提供

#### 6.6 内部処理の流れ（参考）

UI で保存ボタンを押したあと、内部で何が起きるか？

```text
┌─────────────────┐
│ ユーザーの入力  │ チェックボックス、ドロップダウンなどで設定
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  DSL文字列生成  │ "entity document { relation owner: user ... }"
└────────┬────────┘
         │
         ▼ writeSchema API呼び出し
┌─────────────────┐
│  サーバー受信   │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Lexer(字句解析)│ 文字列を単語（トークン）に分解
│                 │ ["entity", "document", "{", "relation", "owner", ":", "user", ...]
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Parser(構文解析)│ トークンをツリー構造（AST）に変換
│                 │ EntityAST → RelationAST → PermissionAST ...
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Validator(検証) │ 文法や意味のチェック
│                 │ - 未定義のrelationを参照していないか？
│                 │ - エンティティ名が重複していないか？
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  Schema構造体   │ Goのデータ構造に変換
│   に変換        │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│  DBに保存       │ schemas テーブルに保存
└─────────────────┘
```

AST（抽象構文木）とは？

「Abstract Syntax Tree」の略で、プログラムの構造をツリー（木構造）で表したものです。

例えば `permission edit = owner or editor` という文を：

```text
PermissionAST
├── 名前: "edit"
└── ルール: LogicalPermissionAST (or)
    ├── RelationPermissionAST ("owner")
    └── RelationPermissionAST ("editor")
```

このようなツリー構造で表現します。

なぜ必要？

- 文字列のままでは「構造」がわからない
- ツリーにすることで「owner と editor を or で結合している」という意味が明確になる
- プログラムで処理しやすくなる（検証、変換、実行など）

フロントエンド開発者への補足:

UI から writeSchema を呼ぶときは、DSL 文字列を渡すだけで OK です。Lexer、Parser、Validator などの処理はすべてサーバー側で自動的に行われます。エラーがあれば `response.errors` に詳細が返ってきます。

```typescript
const response = await client.writeSchema({
  schema_dsl: generatedDSL, // UIで生成した文字列をそのまま渡す
});

if (!response.success) {
  // エラー表示
  console.error(response.errors);
  // 例: ["Line 5: undefined relation 'admin' referenced in permission"]
}
```

#### 6.7 完全な実装例（Next.js + React）

```typescript
// pages/schema-builder.tsx
import { useState, useEffect } from "react";
import { keruberosuClient } from "@/lib/keruberosu-client";

export default function SchemaBuilderPage() {
  const [entities, setEntities] = useState<EntityConfig[]>([]);
  const [dsl, setDSL] = useState("");
  const [saving, setSaving] = useState(false);

  // DSLを自動生成
  useEffect(() => {
    const generated = generateDSLFromEntities(entities);
    setDSL(generated);
  }, [entities]);

  const handleSave = async () => {
    setSaving(true);
    try {
      const response = await keruberosuClient.writeSchema({
        schema_dsl: dsl,
      });

      if (response.success) {
        alert("✅ スキーマを保存しました");
      } else {
        alert(`❌ エラー: ${response.errors.join("\n")}`);
      }
    } catch (error) {
      alert(`❌ 保存に失敗: ${error.message}`);
    } finally {
      setSaving(false);
    }
  };

  const loadTemplate = (template: "rbac" | "google-docs" | "github") => {
    const templates = {
      rbac: [
        {
          name: "user",
          relations: [],
          permissions: [],
          attributes: [],
          rules: [],
        },
        {
          name: "role",
          relations: [{ name: "member", type: "user" }],
          permissions: [
            { name: "admin", expression: "member" },
            { name: "edit", expression: "member" },
            { name: "view", expression: "member" },
          ],
          attributes: [],
          rules: [],
        },
      ],
      "google-docs": [
        // ... Google Docsライクなテンプレート
      ],
      github: [
        // ... GitHubライクなテンプレート
      ],
    };

    setEntities(templates[template]);
  };

  return (
    <div style={{ display: "flex", height: "100vh" }}>
      {/* 左側: エディタ */}
      <div style={{ flex: 1, padding: "2rem", overflowY: "auto" }}>
        <h1>スキーマビルダー</h1>

        <div style={{ marginBottom: "1rem" }}>
          <button onClick={() => loadTemplate("rbac")}>
            📋 RBACテンプレート
          </button>
          <button onClick={() => loadTemplate("google-docs")}>
            📄 Google Docsテンプレート
          </button>
          <button onClick={() => loadTemplate("github")}>
            🐙 GitHubテンプレート
          </button>
        </div>

        <button
          onClick={() =>
            setEntities([
              ...entities,
              {
                name: "",
                relations: [],
                permissions: [],
                attributes: [],
                rules: [],
              },
            ])
          }
        >
          + 新しいエンティティ
        </button>

        {entities.map((entity, idx) => (
          <EntityEditor
            key={idx}
            entity={entity}
            allEntities={entities}
            onChange={(updated) => {
              const newEntities = [...entities];
              newEntities[idx] = updated;
              setEntities(newEntities);
            }}
            onDelete={() => {
              setEntities(entities.filter((_, i) => i !== idx));
            }}
          />
        ))}
      </div>

      {/* 右側: プレビュー */}
      <div
        style={{
          flex: 1,
          background: "#1e1e1e",
          color: "#d4d4d4",
          padding: "2rem",
          overflowY: "auto",
        }}
      >
        <h2>DSLプレビュー</h2>
        <pre
          style={{
            fontFamily: "Monaco, monospace",
            fontSize: "14px",
            lineHeight: "1.6",
          }}
        >
          {dsl || "// エンティティを追加してください"}
        </pre>

        <button
          onClick={handleSave}
          disabled={saving || !dsl}
          style={{
            marginTop: "1rem",
            padding: "0.5rem 1rem",
            background: "#007acc",
            color: "white",
            border: "none",
            cursor: saving ? "not-allowed" : "pointer",
          }}
        >
          {saving ? "保存中..." : "スキーマを保存"}
        </button>
      </div>
    </div>
  );
}

function generateDSLFromEntities(entities: EntityConfig[]): string {
  let dsl = "";

  entities.forEach((entity) => {
    if (!entity.name) return;

    dsl += `entity ${entity.name} {\n`;

    // Relations
    entity.relations.forEach((rel) => {
      dsl += `  relation ${rel.name}: ${rel.type}\n`;
    });

    // Attributes
    entity.attributes.forEach((attr) => {
      dsl += `  attribute ${attr.name} ${attr.type}\n`;
    });

    if (
      (entity.relations.length > 0 || entity.attributes.length > 0) &&
      (entity.permissions.length > 0 || entity.rules.length > 0)
    ) {
      dsl += "\n";
    }

    // Rules
    entity.rules.forEach((rule) => {
      dsl += `  rule ${rule.name}(${rule.params.join(", ")}) {\n`;
      dsl += `    ${rule.expression}\n`;
      dsl += `  }\n`;
    });

    // Permissions
    entity.permissions.forEach((perm) => {
      dsl += `  permission ${perm.name} = ${perm.expression}\n`;
    });

    dsl += "}\n\n";
  });

  return dsl.trim();
}
```

`generateDSLFromEntities` の入出力例（TypeScript エンジニア向け）

この関数は、UI で構築したエンティティ設定（JavaScript オブジェクト）を DSL 文字列に変換します。

入力例: Google Docs ライクなドキュメント管理システムの設定

```typescript
const entities: EntityConfig[] = [
  {
    name: "user",
    relations: [],
    permissions: [],
    attributes: [],
    rules: [],
  },
  {
    name: "document",
    relations: [
      { name: "owner", type: "user" },
      { name: "editor", type: "user" },
      { name: "viewer", type: "user" },
    ],
    permissions: [
      { name: "delete", expression: "owner" },
      { name: "edit", expression: "owner or editor" },
      { name: "view", expression: "owner or editor or viewer" },
    ],
    attributes: [],
    rules: [],
  },
];

const dsl = generateDSLFromEntities(entities);
console.log(dsl);
```

出力: 以下の DSL 文字列が生成されます

```text
entity user {
}

entity document {
  relation owner: user
  relation editor: user
  relation viewer: user

  permission delete = owner
  permission edit = owner or editor
  permission view = owner or editor or viewer
}
```

ABAC（属性ベース）を含む複雑な例:

```typescript
const entitiesWithABAC: EntityConfig[] = [
  {
    name: "user",
    relations: [],
    permissions: [],
    attributes: [],
    rules: [],
  },
  {
    name: "document",
    relations: [{ name: "owner", type: "user" }],
    attributes: [
      { name: "is_public", type: "boolean" },
      { name: "department", type: "string" },
    ],
    rules: [
      {
        name: "check_public",
        params: ["is_public"],
        expression: "is_public == true",
      },
      {
        name: "check_department",
        params: ["department"],
        expression: "request.user.department == department",
      },
    ],
    permissions: [
      { name: "delete", expression: "owner" },
      { name: "edit", expression: "owner" },
      { name: "view", expression: "owner or check_public or check_department" },
    ],
  },
];

const dsl = generateDSLFromEntities(entitiesWithABAC);
```

出力:

```text
entity user {
}

entity document {
  relation owner: user
  attribute is_public boolean
  attribute department string

  rule check_public(is_public) {
    is_public == true
  }
  rule check_department(department) {
    request.user.department == department
  }

  permission delete = owner
  permission edit = owner
  permission view = owner or check_public or check_department
}
```

この DSL 文字列を `writeSchema({ schema_dsl: dsl })` でサーバーに送信すると、サーバー側でパース・検証・保存されます。

このように、技術者でないユーザーでもビジュアルな UI でスキーマを構築でき、システムが自動的に DSL 文字列を生成します。これが Keruberosu の重要な特徴です。

---

このガイドにより、ステークホルダーは Keruberosu の API を実践的に理解し、自分たちのユースケースに当てはめることができます。

## システムアーキテクチャ

### 1. 動作原理

認可システムの本質は、グラフ探索による関係性の検証である。

```text
ユーザー → [関係性グラフ] → リソース
             ↓
          認可判定
```

#### 認可判定のフロー

1. スキーマ定義: エンティティ、関係性、パーミッションルールを定義
2. データ書き込み: 関係性タプル（subject, relation, object）を保存
3. 認可チェック: グラフ探索により、ユーザーがリソースに対して特定のアクションを実行できるか判定

#### グラフ探索の例

```text
質問: user:alice は document:doc1 を view できるか？

スキーマ:
  entity document {
    relation owner @user
    relation parent @folder

    permission view = owner or parent.viewer
  }

関係性データ:
  (user:alice, owner, document:doc1)

探索:
  1. document:doc1 の view パーミッションを評価
  2. owner 関係をチェック → alice が owner → true
  3. 結果: 認可
```

### 2. データ構造

#### 2.1 スキーマ定義

Permify 互換の DSL で定義されたスキーマをシステム内部でパース・保存する。

基本構文:

```text
entity user {}

entity organization {
  relation admin @user
  relation member @user

  permission create_document = admin
  permission view_documents = admin or member
}

entity document {
  relation owner @user
  relation parent @organization
  relation viewer @user @organization#member

  permission view = owner or viewer or parent.member
  permission edit = owner or parent.admin
  permission delete = owner
}
```

ABAC 対応構文:

```text
entity document {
  relation owner @user
  relation parent @organization

  attribute classification string
  attribute is_public boolean

  rule is_public_doc(is_public boolean) {
    is_public == true
  }

  rule is_confidential(classification string) {
    classification == 'confidential'
  }

  permission view = is_public_doc(is_public) or owner or parent.member
  permission edit = (is_confidential(classification) and owner) or parent.admin
}
```

内部データ構造:

```go
type Schema struct {
    Entities map[string]*Entity
    Version  string
}

type Entity struct {
    Name        string
    Relations   map[string]*Relation
    Attributes  map[string]*Attribute
    Rules       map[string]*Rule
    Permissions map[string]*Permission
}

type Relation struct {
    Name        string
    TargetTypes []RelationTarget  // 複数の型を指定可能
}

type RelationTarget struct {
    Type     string  // "user", "organization", etc.
    Relation string  // optional: "member" in "@organization#member"
}

type Permission struct {
    Name  string
    Rule  *PermissionRule
}

type PermissionRule struct {
    Type     string  // "or", "and", "not", "relation", "nested", "rule"
    Children []*PermissionRule
    Relation string  // relation名 or nested path (e.g., "parent.member")
    RuleName string  // ABAC rule名 (e.g., "is_public")
    RuleArgs []string  // rule引数 (e.g., ["classification"])
}

type Attribute struct {
    Name     string
    DataType string  // "string", "boolean", "integer", "double", "string[]", etc.
}

type Rule struct {
    Name       string
    Parameters []RuleParameter
    Expression string  // CEL式 (e.g., "classification == 'public'")
}

type RuleParameter struct {
    Name string
    Type string
}
```

#### 2.2 関係性データ（タプル）

関係性は (subject, relation, object) の 3 つ組で表現する。

```text
(user:alice, owner, document:doc1)
(user:bob, member, organization:org1)
(document:doc1, parent, organization:org1)
```

PostgreSQL テーブル設計:

```sql
CREATE TABLE relations (
    id BIGSERIAL PRIMARY KEY,
    subject_type VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    relation VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(subject_type, subject_id, relation, entity_type, entity_id)
);

-- インデックス: エンティティから逆引き（最も重要）
CREATE INDEX idx_relations_reverse ON relations(entity_type, entity_id, relation);

-- インデックス: サブジェクトから検索
CREATE INDEX idx_relations_forward ON relations(subject_type, subject_id, relation);
```

relations テーブルの役割:

- 用途: ReBAC の関係性（誰が何とどんな関係にあるか）を保存
- 例:
  - `(user:alice, owner, document:doc1)` → alice は doc1 の owner
  - `(document:doc1, parent, organization:org1)` → doc1 は org1 に属する
- クエリパターン:
  - Check: `user:alice`が`document:doc1`の`owner`か？ → WHERE 句で直接検索
  - LookupEntity: `user:alice`が`edit`できる`document`一覧 → 逆引きインデックスを使用
  - ネスト関係: `parent.member`の展開 → 2 段階クエリ（まず parent 取得、次に member 取得）

#### 2.3 属性データ（ABAC 用）

エンティティやコンテキストの属性を保存。

```sql
CREATE TABLE attributes (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    attribute_key VARCHAR(255) NOT NULL,
    attribute_value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(entity_type, entity_id, attribute_key)
);

CREATE INDEX idx_attributes_entity ON attributes(entity_type, entity_id);
```

attributes テーブルの役割:

- 用途: ABAC の属性（エンティティの動的な性質）を保存
- 例:
  - `(document, doc1, "classification", "confidential")` → doc1 の分類は機密
  - `(document, doc1, "is_public", true)` → doc1 は公開
  - `(user, alice, "department", "engineering")` → alice の部署は engineering
- データ型: JSONB を使用することで、string、boolean、integer、array など柔軟に保存
- クエリパターン:
  - Check 時: 認可判定でルール評価に必要な属性を取得
  - 例: `is_confidential(classification)`ルール評価時、document の`classification`属性を取得

#### 2.4 スキーマストレージ

```sql
CREATE TABLE schemas (
    id INTEGER PRIMARY KEY DEFAULT 1,
    schema_dsl TEXT NOT NULL,
    schema_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (id = 1)  -- 常に1行のみを強制
);

-- 初期行を作成
INSERT INTO schemas (id, schema_dsl, schema_json)
VALUES (1, '', '{}')
ON CONFLICT DO NOTHING;
```

schemas テーブルの役割:

- 用途: 現在の認可スキーマ（entity、relation、permission 定義）を保持
- schema_dsl: Permify DSL 形式の元テキスト（人間が読める形式）
- schema_json: パース済みの構造化データ（高速な検証・参照用）
- 設計方針: 常に 1 行のみ存在（`CHECK (id = 1)` で強制）

使用フロー:

1. 書き込み: `WriteSchema` API → DSL をパース → 両形式で保存（UPDATE）
2. 読み込み: 認可チェック時、`SELECT * FROM schemas WHERE id = 1` で取得・キャッシュ
3. 更新: スキーマ更新時は既存行を UPDATE
4. 整合性チェック: データ書き込み時、現在のスキーマと照合

バージョン管理を行わない理由:

- 複数環境は別 DB: 開発・ステージング・本番で異なる DB を使用 → バージョン管理不要
- 監査は別サービス: スキーマ変更の履歴は AuditService で記録
- YAGNI 原則: ロールバック機能は未定 → 必要になったら実装
- シンプルさ: 1 行のみで管理が容易、読み込みが高速

なぜ 3 つのテーブルで済むのか？:

- `schemas`: 認可の「ルール定義」を保存（常に 1 行のみ）
- `relations`: 認可の「関係性データ」を保存（誰と誰が繋がっているか）
- `attributes`: 認可の「属性データ」を保存（エンティティの性質）

この 3 つがあれば、「ユーザー X はリソース Y に対してアクション Z ができるか？」を判定するための全情報が揃う。

#### 2.5 監査ログ（オプション）

スキーマ変更や重要な操作の履歴を記録するための別テーブル：

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,  -- 'schema_update', 'relation_write', 'permission_check' など
    actor_id VARCHAR(255),              -- 操作を行ったユーザー/サービスID
    actor_type VARCHAR(255),            -- 'user', 'service' など
    resource_type VARCHAR(255),         -- 'schema', 'relation', 'attribute' など
    resource_id VARCHAR(255),           -- 対象リソースのID
    action VARCHAR(255) NOT NULL,       -- 'create', 'update', 'delete', 'check' など
    details JSONB,                      -- 詳細情報（変更内容など）
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- 検索用インデックス
    INDEX idx_audit_logs_timestamp (timestamp),
    INDEX idx_audit_logs_actor (actor_id, actor_type),
    INDEX idx_audit_logs_event_type (event_type)
);
```

監査ログの例:

```json
// スキーマ更新
{
  "event_type": "schema_update",
  "actor_id": "admin@example.com",
  "actor_type": "user",
  "resource_type": "schema",
  "resource_id": "1",
  "action": "update",
  "details": {
    "old_schema": "entity document { ... }",
    "new_schema": "entity document { relation owner @user ... }",
    "changes": ["added relation 'owner'"]
  },
  "timestamp": "2025-01-15T10:30:00Z"
}

// 認可チェック（オプション: 高トラフィックの場合はサンプリング）
{
  "event_type": "permission_check",
  "actor_id": "user:alice",
  "actor_type": "user",
  "resource_type": "document",
  "resource_id": "doc1",
  "action": "check",
  "details": {
    "permission": "view",
    "result": "allowed",
    "latency_ms": 5
  },
  "timestamp": "2025-01-15T10:31:00Z"
}
```

AuditService API（別サービスとして実装可能）:

```protobuf
message WriteAuditLogRequest {
  string event_type = 1;
  string actor_id = 2;
  string actor_type = 3;
  string resource_type = 4;
  string resource_id = 5;
  string action = 6;
  google.protobuf.Struct details = 7;
}

message WriteAuditLogResponse {
  bool success = 1;
}

message ReadAuditLogsRequest {
  string event_type = 1;     // フィルタ（オプション）
  string actor_id = 2;        // フィルタ（オプション）
  string start_time = 3;      // ISO8601形式
  string end_time = 4;        // ISO8601形式
  int32 limit = 5;            // デフォルト: 100
  string cursor = 6;          // ページネーション用
}

message AuditLog {
  string id = 1;
  string event_type = 2;
  string actor_id = 3;
  string actor_type = 4;
  string resource_type = 5;
  string resource_id = 6;
  string action = 7;
  google.protobuf.Struct details = 8;
  string timestamp = 9;
}

message ReadAuditLogsResponse {
  repeated AuditLog logs = 1;
  string next_cursor = 2;
  int32 total_count = 3;
}
```

監査ログの運用:

- 書き込みタイミング: AuthorizationService が重要な操作後に AuditService を呼び出す
- 非同期処理: メッセージキュー（Kafka/RabbitMQ）経由で送信してパフォーマンス影響を最小化
- 保持期間: 要件に応じて設定（例: 1 年間保持）
- アーカイブ: 古いログは S3 などに移動

### 3. API 設計（gRPC）

#### 3.0 サービス設計方針

単一サービス（AuthorizationService）を採用

採用理由:

1. キャッシュ戦略との整合性:

   - 認可判定結果を LRU キャッシュで保存し、高速に返す戦略を採用
   - キャッシュヒット時はグラフ探索不要 → トラフィックの大部分をキャッシュで吸収
   - 「認可チェックだけを独立スケール」する必要性が薄い

2. クライアント体験の最適化:

   - 単一のクライアントオブジェクトですべての API にアクセス可能
   - 複数スタブの管理が不要（ユーザーフレンドリー）
   - 接続管理がシンプル

3. YAGNI 原則（You Aren't Gonna Need It）:

   - 初期段階から複数サービスに分割するのは過剰設計
   - 必要になったら分割（内部モジュール分離で将来の移行に備える）
   - 開発・運用コストを抑える

4. 内部での責務分離は維持:
   - サーバー内部では SchemaManager、DataManager、PermissionChecker をモジュール分離
   - 将来的にサービス分割が必要になったら、このモジュールを切り出すだけ

パフォーマンス戦略:

```text
リクエスト → LRUキャッシュチェック
                ↓ ヒット（90%+）
              即座に結果を返す（高速）

                ↓ ミス（10%-）
              グラフ探索 + DB問い合わせ
                ↓
              結果をキャッシュに保存
                ↓
              結果を返す
```

ネストしたグラフ探索は重いため、キャッシュヒット率を最大化することで実用的なパフォーマンスを実現。

#### 3.1 統合 API 定義

```text
Protocol Buffers 定義は以下の3ファイルに分割されています：

- proto/keruberosu/v1/common.proto: 全サービスで共有される基本型（Entity, Subject, RelationTuple など）
- proto/keruberosu/v1/authorization.proto: AuthorizationService の定義と専用メッセージ型
- proto/keruberosu/v1/audit.proto: AuditService の定義と専用メッセージ型

完全な定義は Appendix C.1 を参照してください。

この設計により：
- サービスの境界が明確になる
- 共通型は他プロジェクトから import 可能
- クライアントは必要なサービスだけを選択可能
- メンテナンス性が向上
- Google API Design Guide に準拠
```

各 API のユースケース:

| API               | 質問形式                          | 例                                       | 用途               |
| ----------------- | --------------------------------- | ---------------------------------------- | ------------------ |
| Check             | X は Y に対して Z できるか？      | alice は doc1 を edit できるか？         | アクセス制御の基本 |
| Expand            | Y の Z を持つのは誰？（ツリー）   | doc1 の edit を持つユーザーツリー        | デバッグ・可視化   |
| LookupEntity      | X が Z できる Y は何？            | alice が edit できる document は？       | UI でのリスト表示  |
| LookupSubject     | Y の Z を持つ X は誰？            | doc1 を edit できる user は？            | 共有設定 UI        |
| SubjectPermission | X は Y に対してどの権限を持つか？ | alice は doc1 に対してどの権限を持つか？ | 権限一覧表示       |

各 API の説明:

| API カテゴリ   | メソッド           | 用途                        | 例                                    |
| -------------- | ------------------ | --------------------------- | ------------------------------------- |
| スキーマ管理   | WriteSchema        | スキーマ定義の登録・更新    | DSL を送信してスキーマ作成            |
|                | ReadSchema         | スキーマ定義の取得          | 現在のスキーマを取得                  |
| データ書き込み | WriteRelations     | 関係性タプルの書き込み      | alice を doc1 の owner に             |
|                | DeleteRelations    | 関係性タプルの削除          | alice の owner 権限を削除             |
|                | WriteAttributes    | 属性データの書き込み        | doc1 を confidential に               |
| 認可チェック   | Check              | 認可判定                    | alice は doc1 を edit できる？        |
|                | Expand             | パーミッションツリー展開    | doc1 の edit 権限を持つユーザーツリー |
|                | LookupEntity       | 許可された Entity 一覧      | alice が edit できる document は？    |
|                | LookupSubject      | 許可された Subject 一覧     | doc1 を edit できる user は？         |
|                | LookupEntityStream | LookupEntity のストリーム版 | 大量結果をストリームで取得            |
|                | SubjectPermission  | Subject の権限一覧          | alice が doc1 に対して持つ権限一覧    |

クライアント使用例:

```typescript
// TypeScript
import { AuthorizationServiceClient } from "./generated/keruberosu_grpc_pb";

const client = new AuthorizationServiceClient("localhost:50051");

// スキーマの定義
await client.writeSchema(schemaRequest);

// データの書き込み
await client.writeRelations(relationsRequest);

// 認可チェック（単一）
const checkResp = await client.check({
  metadata: { depth: 20 },
  entity: { type: "document", id: "doc1" },
  permission: "edit",
  subject: { type: "user", id: "alice" },
  context: { tuples: [], attributes: [] },
});
console.log(checkResp.can); // ALLOWED or DENIED

// エンティティ検索
const entities = await client.lookupEntity({
  metadata: { depth: 20 },
  entity_type: "document",
  permission: "edit",
  subject: { type: "user", id: "alice" },
  page_size: 100,
});
console.log(entities.entity_ids); // ["doc1", "doc2", ...]

// Subject検索
const subjects = await client.lookupSubject({
  metadata: { depth: 20 },
  entity: { type: "document", id: "doc1" },
  permission: "edit",
  subject_reference: { type: "user" },
  page_size: 100,
});
console.log(subjects.subject_ids); // ["alice", "bob", ...]

// 権限一覧取得
const permissions = await client.subjectPermission({
  metadata: { depth: 20 },
  entity: { type: "document", id: "doc1" },
  subject: { type: "user", id: "alice" },
});
console.log(permissions.results); // { "edit": ALLOWED, "delete": DENIED, ... }
```

```go
// Go
import pb "github.com/asakaida/keruberosu/gen/proto"

client := pb.NewAuthorizationServiceClient(conn)

// スキーマの定義
client.WriteSchema(ctx, schemaRequest)

// データの書き込み
client.WriteRelations(ctx, relationsRequest)

// 認可チェック（単一）
checkResp, _ := client.Check(ctx, &pb.CheckRequest{
  Metadata: &pb.PermissionCheckMetadata{Depth: 20},
  Entity: &pb.Entity{Type: "document", Id: "doc1"},
  Permission: "edit",
  Subject: &pb.Subject{Type: "user", Id: "alice"},
  Context: &pb.Context{},
})
fmt.Println(checkResp.Can) // ALLOWED or DENIED

// エンティティ検索
entities, _ := client.LookupEntity(ctx, &pb.LookupEntityRequest{
  Metadata: &pb.PermissionCheckMetadata{Depth: 20},
  EntityType: "document",
  Permission: "edit",
  Subject: &pb.Subject{Type: "user", Id: "alice"},
  PageSize: 100,
})
fmt.Println(entities.EntityIds) // ["doc1", "doc2", ...]

// Subject検索
subjects, _ := client.LookupSubject(ctx, &pb.LookupSubjectRequest{
  Metadata: &pb.PermissionCheckMetadata{Depth: 20},
  Entity: &pb.Entity{Type: "document", Id: "doc1"},
  Permission: "edit",
  SubjectReference: &pb.SubjectReference{Type: "user"},
  PageSize: 100,
})
fmt.Println(subjects.SubjectIds) // ["alice", "bob", ...]

// 権限一覧取得
permissions, _ := client.SubjectPermission(ctx, &pb.SubjectPermissionRequest{
  Metadata: &pb.PermissionCheckMetadata{Depth: 20},
  Entity: &pb.Entity{Type: "document", Id: "doc1"},
  Subject: &pb.Subject{Type: "user", Id: "alice"},
})
fmt.Println(permissions.Results) // map[string]CheckResult{"edit": ALLOWED, ...}
```

### 4. グラフ探索エンジンとキャッシュ戦略

#### 4.0 L1/L2 キャッシュによるパフォーマンス最適化

基本方針: ネストしたグラフ探索は重い処理のため、認可判定結果を L1（ローカルメモリ）/L2（Redis）の 2 層キャッシュに保存して高速化。

キャッシュの詳細実装は Section 6.1 を参照。ここでは統合された`AuthorizationCache`インターフェースの使用方法を示す。

```go
type CacheKey struct {
    SubjectType string
    SubjectID   string
    Permission  string
    ObjectType  string
    ObjectID    string
    ContextHash string  // contextのハッシュ値
}

// AuthorizationCacheはL1/L2を統合したインターフェース
// 実装の詳細はSection 6.1を参照
type AuthorizationCache struct {
    l1 *L1Cache  // 自前LRU実装（sync.RWMutex + container/list）
    l2 *L2Cache  // Redis分散キャッシュ
}

// Get: L1 → L2 → DBの順でキャッシュを確認
func (c *AuthorizationCache) Get(ctx context.Context, key CacheKey) (bool, bool) {
    // L1チェック
    if entry, ok := c.l1.Get(key); ok {
        return entry.Allowed, true
    }

    // L2チェック
    if entry, ok := c.l2.Get(ctx, key); ok {
        // L2ヒット時はL1にも保存（ウォームアップ）
        c.l1.Set(key, entry)
        return entry.Allowed, true
    }

    return false, false  // キャッシュミス
}

// Set: L1とL2の両方に保存
func (c *AuthorizationCache) Set(ctx context.Context, key CacheKey, allowed bool) {
    entry := &CacheEntry{
        Allowed:   allowed,
        Timestamp: time.Now(),
    }
    c.l1.Set(key, entry)
    c.l2.Set(ctx, key, entry)  // ベストエフォート
}
```

サーバー実装:

```go
type Server struct {
    schemaManager     *SchemaManager
    dataManager       *DataManager
    permissionChecker *PermissionChecker
    cache             *AuthorizationCache  // L1/L2統合キャッシュ
}

func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
    // 1. キャッシュキーを生成
    key := CacheKey{
        SubjectType: req.Subject.Type,
        SubjectID:   req.Subject.Id,
        Permission:  req.Permission,
        ObjectType:  req.Object.Type,
        ObjectID:    req.Object.Id,
        ContextHash: hashContext(req.Context),
    }

    // 2. L1/L2キャッシュチェック（L1 → L2 → DBの順）
    if allowed, ok := s.cache.Get(ctx, key); ok {
        return &pb.CheckResponse{Allowed: allowed}, nil
    }

    // 3. キャッシュミス: グラフ探索を実行
    schema := s.schemaCache.Get()
    allowed, err := s.permissionChecker.Check(ctx, schema, req)
    if err != nil {
        return nil, err
    }

    // 4. 結果をL1/L2両方に保存
    s.cache.Set(ctx, key, allowed)

    return &pb.CheckResponse{Allowed: allowed}, nil
}
```

キャッシュ無効化戦略:

データ書き込み時に関連するキャッシュを無効化（複数インスタンス対応）：

```go
func (s *Server) WriteRelations(ctx context.Context, req *pb.WriteRelationsRequest) (*pb.WriteRelationsResponse, error) {
    // 1. データベースに書き込み
    count, err := s.dataManager.WriteRelations(ctx, req.Tuples)
    if err != nil {
        return nil, err
    }

    // 2. L1/L2両方のキャッシュを無効化
    for _, tuple := range req.Tuples {
        s.cache.InvalidateByEntity(ctx, tuple.Entity)
        // 注: 簡易実装では全クリア、本番ではセカンダリインデックスで部分無効化
    }

    return &pb.WriteRelationsResponse{WrittenCount: count}, nil
}
```

キャッシュサイズと TTL 設定:

- L1 (ローカルメモリ):

  - デフォルトサイズ: 10,000 エントリ/インスタンス
  - TTL: 1 分（短めで不整合を最小化）
  - メモリ使用量目安: 約 10MB/インスタンス

- L2 (Redis):
  - 容量: Redis の設定に依存（推奨: 1GB 以上）
  - TTL: 5 分
  - Redis Cluster で高可用性を確保

#### 4.1 探索アルゴリズム（キャッシュミス時）

```go
// Checkの基本アルゴリズム
func (e *Engine) Check(ctx context.Context, req *CheckRequest) (bool, error) {
    // 1. スキーマを取得
    schema := e.getSchema()
    entity := schema.Entities[req.Object.Type]
    permission := entity.Permissions[req.Permission]

    // 2. パーミッションルールを評価
    return e.evaluateRule(ctx, permission.Rule, req)
}

func (e *Engine) evaluateRule(ctx context.Context, rule *PermissionRule, req *CheckRequest) (bool, error) {
    switch rule.Type {
    case "or":
        // いずれかの子ルールがtrueならtrue
        for _, child := range rule.Children {
            if ok, _ := e.evaluateRule(ctx, child, req); ok {
                return true, nil
            }
        }
        return false, nil

    case "and":
        // すべての子ルールがtrueならtrue
        for _, child := range rule.Children {
            if ok, _ := e.evaluateRule(ctx, child, req); !ok {
                return false, nil
            }
        }
        return true, nil

    case "not":
        // 子ルールがfalseならtrue
        if ok, _ := e.evaluateRule(ctx, rule.Children[0], req); !ok {
            return true, nil
        }
        return false, nil

    case "relation":
        // 直接の関係性をチェック
        return e.hasRelation(ctx, req.Subject, rule.Relation, req.Object)

    case "nested":
        // ネストした関係性をチェック (e.g., "parent.member")
        return e.hasNestedRelation(ctx, req.Subject, rule.Relation, req.Object)

    case "rule":
        // ABACルールを評価 (e.g., "is_public(classification)")
        return e.evaluateABACRule(ctx, rule, req)
    }
}
```

#### 4.2 ネストした関係性の処理

```go
// "parent.member" のような関係性を処理
func (e *Engine) hasNestedRelation(ctx context.Context, subject Entity, path string, object Entity) (bool, error) {
    parts := strings.Split(path, ".")

    // 最初の関係性をトラバース
    intermediates, err := e.getRelated(ctx, object, parts[0])
    if err != nil {
        return false, err
    }

    // 中間ノードが1つの場合
    if len(parts) == 2 {
        // 各中間ノードに対してチェック
        for _, intermediate := range intermediates {
            if ok, _ := e.hasRelation(ctx, subject, parts[1], intermediate); ok {
                return true, nil
            }
        }
        return false, nil
    }

    // さらにネストしている場合は再帰
    restPath := strings.Join(parts[1:], ".")
    for _, intermediate := range intermediates {
        if ok, _ := e.hasNestedRelation(ctx, subject, restPath, intermediate); ok {
            return true, nil
        }
    }
    return false, nil
}
```

### 5. パフォーマンス最適化戦略

#### 5.1 キャッシュ戦略（最重要）

分散環境における 3 層キャッシュアーキテクチャ:

1. L1: 認可判定結果キャッシュ（ローカルメモリ LRU）

   - Check API の結果を各インスタンスのメモリに直接キャッシュ
   - 最もヒット率が高く、最速（ネットワーク I/O 不要）
   - TTL: 1 分（短めに設定し、インスタンス間の不整合を最小化）
   - 自前実装（`sync.RWMutex` + `container/list`）

2. L2: 認可判定結果の分散キャッシュ（Redis）

   - 複数のサーバーインスタンス間で共有される認可結果
   - L1 ミス時のフォールバック
   - TTL: 5 分
   - Redis Cluster 推奨（高可用性）

3. L3: スキーマキャッシュ（In-Memory）

   - アクティブなスキーマを各インスタンスのメモリに保持
   - スキーマ更新時のみ無効化
   - TTL: 無期限（明示的な無効化のみ）
   - `atomic.Pointer` でロックフリー読み込み

#### 6.2 データベースインデックス戦略

```sql
-- 最重要: エンティティから逆引き
CREATE INDEX idx_relations_reverse ON relations(entity_type, entity_id, relation);

-- サブジェクトから検索
CREATE INDEX idx_relations_forward ON relations(subject_type, subject_id, relation);

-- 属性検索
CREATE INDEX idx_attributes_entity ON attributes(entity_type, entity_id);

-- カバリングインデックス（LookupEntity最適化）
CREATE INDEX idx_relations_covering ON relations(
    subject_type, subject_id, relation, entity_type, entity_id
);
```

#### 6.3 並列処理とバッチ最適化

```go
// 複数の認可チェックを並列実行
func (s *Server) CheckBatch(ctx context.Context, requests []*CheckRequest) ([]*CheckResponse, error) {
    results := make([]*CheckResponse, len(requests))
    var wg sync.WaitGroup

    for i, req := range requests {
        wg.Add(1)
        go func(idx int, r *CheckRequest) {
            defer wg.Done()
            resp, _ := s.Check(ctx, r)
            results[idx] = resp
        }(i, req)
    }

    wg.Wait()
    return results, nil
}
```

### 6. 並行性とスレッド安全性

多数のコア（16+）を持つ CPU で並行実行する場合、データの整合性とスレッド安全性が最重要課題となる。

#### 6.1 2 層キャッシュアーキテクチャ（L1/L2）

冗長化要件: 複数のサーバーインスタンスが並行稼働する環境を想定（例: 16 コア × 複数インスタンス）

この構成では、インスタンス間でキャッシュを共有する必要があるため、自前の L1/L2 キャッシュ実装が必須となる。

```text
リクエスト → L1 (ローカルメモリ) → L2 (Redis) → DB + グラフ探索
               ↓ ヒット              ↓ ヒット       ↓ ミス
             即座に返す            高速に返す      結果を返す
                                                    ↓
                                              L2/L1に保存
```

##### L1: ローカルメモリキャッシュ（自前 LRU 実装）

各サーバーインスタンス内のインメモリキャッシュ。最も高速だが、インスタンス間で共有されない。

```go
type CacheKey struct {
    SubjectType string
    SubjectID   string
    Permission  string
    ObjectType  string
    ObjectID    string
    ContextHash string
}

type CacheEntry struct {
    Allowed   bool
    Timestamp time.Time
}

// 自前のスレッドセーフLRU実装
type L1Cache struct {
    mu       sync.RWMutex
    capacity int
    ttl      time.Duration
    items    map[CacheKey]*list.Element
    lruList  *list.List
}

type lruEntry struct {
    key   CacheKey
    value *CacheEntry
}

func NewL1Cache(capacity int, ttl time.Duration) *L1Cache {
    return &L1Cache{
        capacity: capacity,
        ttl:      ttl,
        items:    make(map[CacheKey]*list.Element),
        lruList:  list.New(),
    }
}

func (c *L1Cache) Get(key CacheKey) (*CacheEntry, bool) {
    c.mu.Lock()
    defer c.mu.Unlock()

    elem, ok := c.items[key]
    if !ok {
        return nil, false
    }

    entry := elem.Value.(*lruEntry)

    // TTLチェック
    if time.Since(entry.value.Timestamp) > c.ttl {
        c.removeElement(elem)
        return nil, false
    }

    // LRUリストの先頭に移動
    c.lruList.MoveToFront(elem)
    return entry.value, true
}

func (c *L1Cache) Set(key CacheKey, entry *CacheEntry) {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 既存エントリの更新
    if elem, ok := c.items[key]; ok {
        c.lruList.MoveToFront(elem)
        elem.Value.(*lruEntry).value = entry
        return
    }

    // 新規エントリの追加
    elem := c.lruList.PushFront(&lruEntry{key: key, value: entry})
    c.items[key] = elem

    // 容量超過時は最古のエントリを削除
    if c.lruList.Len() > c.capacity {
        oldest := c.lruList.Back()
        if oldest != nil {
            c.removeElement(oldest)
        }
    }
}

func (c *L1Cache) Delete(key CacheKey) {
    c.mu.Lock()
    defer c.mu.Unlock()

    if elem, ok := c.items[key]; ok {
        c.removeElement(elem)
    }
}

func (c *L1Cache) Purge() {
    c.mu.Lock()
    defer c.mu.Unlock()

    c.items = make(map[CacheKey]*list.Element)
    c.lruList.Init()
}

func (c *L1Cache) removeElement(elem *list.Element) {
    c.lruList.Remove(elem)
    delete(c.items, elem.Value.(*lruEntry).key)
}
```

##### L2: Redis 分散キャッシュ

複数のサーバーインスタンス間で共有されるキャッシュ。L1 よりは遅いが、DB アクセスよりは高速。

```go
import (
    "context"
    "encoding/json"
    "time"
    "github.com/redis/go-redis/v9"
)

type L2Cache struct {
    client *redis.Client
    ttl    time.Duration
}

func NewL2Cache(redisAddr string, ttl time.Duration) *L2Cache {
    client := redis.NewClient(&redis.Options{
        Addr:         redisAddr,
        PoolSize:     100,           // 並行接続数
        MinIdleConns: 10,            // 最小アイドル接続
        MaxRetries:   3,             // リトライ回数
    })

    return &L2Cache{
        client: client,
        ttl:    ttl,
    }
}

func (c *L2Cache) Get(ctx context.Context, key CacheKey) (*CacheEntry, bool) {
    cacheKeyStr := c.serializeKey(key)

    val, err := c.client.Get(ctx, cacheKeyStr).Result()
    if err == redis.Nil {
        return nil, false  // キャッシュミス
    }
    if err != nil {
        // Redisエラー時はミス扱い（フォールバック）
        return nil, false
    }

    var entry CacheEntry
    if err := json.Unmarshal([]byte(val), &entry); err != nil {
        return nil, false
    }

    return &entry, true
}

func (c *L2Cache) Set(ctx context.Context, key CacheKey, entry *CacheEntry) error {
    cacheKeyStr := c.serializeKey(key)

    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }

    return c.client.Set(ctx, cacheKeyStr, data, c.ttl).Err()
}

func (c *L2Cache) Delete(ctx context.Context, key CacheKey) error {
    cacheKeyStr := c.serializeKey(key)
    return c.client.Del(ctx, cacheKeyStr).Err()
}

func (c *L2Cache) Purge(ctx context.Context) error {
    // パターンマッチで全キーを削除（本番環境では慎重に）
    return c.client.FlushDB(ctx).Err()
}

func (c *L2Cache) serializeKey(key CacheKey) string {
    return fmt.Sprintf("keruberosu:check:%s:%s:%s:%s:%s:%s",
        key.SubjectType, key.SubjectID,
        key.Permission,
        key.ObjectType, key.ObjectID,
        key.ContextHash,
    )
}
```

##### 統合キャッシュマネージャー

L1 と L2 を組み合わせた統合インターフェース。

```go
type AuthorizationCache struct {
    l1 *L1Cache
    l2 *L2Cache
}

func NewAuthorizationCache(l1Capacity int, l1TTL time.Duration, redisAddr string, l2TTL time.Duration) *AuthorizationCache {
    return &AuthorizationCache{
        l1: NewL1Cache(l1Capacity, l1TTL),
        l2: NewL2Cache(redisAddr, l2TTL),
    }
}

func (c *AuthorizationCache) Get(ctx context.Context, key CacheKey) (bool, bool) {
    // L1チェック
    if entry, ok := c.l1.Get(key); ok {
        return entry.Allowed, true
    }

    // L2チェック
    if entry, ok := c.l2.Get(ctx, key); ok {
        // L2ヒット時はL1にも保存（ウォームアップ）
        c.l1.Set(key, entry)
        return entry.Allowed, true
    }

    return false, false
}

func (c *AuthorizationCache) Set(ctx context.Context, key CacheKey, allowed bool) {
    entry := &CacheEntry{
        Allowed:   allowed,
        Timestamp: time.Now(),
    }

    // L1とL2の両方に保存
    c.l1.Set(key, entry)
    c.l2.Set(ctx, key, entry)  // エラーは無視（ベストエフォート）
}

func (c *AuthorizationCache) InvalidateByObject(ctx context.Context, obj *Entity) {
    // 簡易実装: 全キャッシュをクリア
    // TODO: セカンダリインデックスで部分無効化を実装
    c.l1.Purge()
    c.l2.Purge(ctx)
}
```

キャッシュ無効化の通知（複数インスタンス対応）:

複数サーバーインスタンスが稼働している場合、L1 キャッシュの無効化をすべてのインスタンスに伝播させる必要がある。Redis Pub/Sub を使用。

```go
import "github.com/redis/go-redis/v9"

type CacheInvalidator struct {
    pubsub *redis.PubSub
    cache  *AuthorizationCache
}

func NewCacheInvalidator(redisClient *redis.Client, cache *AuthorizationCache) *CacheInvalidator {
    pubsub := redisClient.Subscribe(context.Background(), "keruberosu:invalidate")

    invalidator := &CacheInvalidator{
        pubsub: pubsub,
        cache:  cache,
    }

    // バックグラウンドで無効化メッセージを受信
    go invalidator.listen()

    return invalidator
}

func (ci *CacheInvalidator) listen() {
    ch := ci.pubsub.Channel()
    for msg := range ch {
        // 他のインスタンスからの無効化通知を受信
        // L1キャッシュのみをクリア（L2は既にクリア済み）
        ci.cache.l1.Purge()
    }
}

func (ci *CacheInvalidator) Invalidate(ctx context.Context) error {
    // L2をクリア
    ci.cache.l2.Purge(ctx)

    // 全インスタンスにL1クリアを通知
    return ci.pubsub.Publish(ctx, "keruberosu:invalidate", "purge").Err()
}
```

L1/L2 キャッシュの TTL 設定例:

```go
// L1: 短い TTL（インスタンス間の不整合を最小化）
l1TTL := 1 * time.Minute

// L2: 長い TTL（Redisの負荷を考慮）
l2TTL := 5 * time.Minute

cache := NewAuthorizationCache(
    10000,      // L1容量: 10,000エントリ
    l1TTL,
    "localhost:6379",  // Redis
    l2TTL,
)
```

#### 6.2 スキーマキャッシュのアトミック更新

スキーマは頻繁に更新されないが、更新時は全ゴルーチンが最新版を参照する必要がある。`atomic.Pointer` を使用してロックフリーで実現。

```go
import "sync/atomic"

type SchemaCache struct {
    current atomic.Pointer[Schema]  // ロックフリーなアトミックポインタ
}

// スキーマの読み込み（並行実行しても安全）
func (sc *SchemaCache) Get() *Schema {
    return sc.current.Load()  // アトミックな読み込み
}

// スキーマの更新（並行実行しても安全）
func (sc *SchemaCache) Set(schema *Schema) {
    sc.current.Store(schema)  // アトミックな書き込み
}

// サーバー実装例
type Server struct {
    schemaCache       SchemaCache
    authCache         *AuthorizationCache
    dataManager       *DataManager
    permissionChecker *PermissionChecker
}

func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
    // スキーマの取得（ロック不要）
    schema := s.schemaCache.Get()

    // 認可チェック処理...
    allowed, err := s.permissionChecker.Check(ctx, schema, req)
    if err != nil {
        return nil, err
    }

    return &pb.CheckResponse{Allowed: allowed}, nil
}

func (s *Server) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
    // DSLをパース
    schema, err := ParseSchema(req.SchemaDsl)
    if err != nil {
        return nil, err
    }

    // DBに保存
    if err := s.dataManager.SaveSchema(ctx, schema); err != nil {
        return nil, err
    }

    // キャッシュを更新（全ゴルーチンに即座に反映）
    s.schemaCache.Set(schema)

    // 認可キャッシュを全クリア
    s.authCache.Purge()

    return &pb.WriteSchemaResponse{Success: true}, nil
}
```

#### 6.3 データベース接続プールの設定

PostgreSQL の接続プールは並行性を考慮して適切に設定する必要がある。

```go
import (
    "database/sql"
    _ "github.com/lib/pq"
)

func NewDB(connStr string, numCPU int) (*sql.DB, error) {
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        return nil, err
    }

    // 並行性を考慮した接続プール設定
    db.SetMaxOpenConns(numCPU * 4)      // CPU数 × 4（推奨: 2〜5倍）
    db.SetMaxIdleConns(numCPU * 2)      // CPU数 × 2
    db.SetConnMaxLifetime(time.Hour)    // 接続の最大寿命
    db.SetConnMaxIdleTime(10 * time.Minute)  // アイドル接続のタイムアウト

    return db, nil
}
```

接続プール設定の指針:

| 設定項目          | 推奨値            | 理由                                     |
| ----------------- | ----------------- | ---------------------------------------- |
| `MaxOpenConns`    | CPU コア数 × 2〜5 | 並行リクエストを捌けるだけの接続数を確保 |
| `MaxIdleConns`    | CPU コア数 × 2    | 接続の再確立オーバーヘッドを削減         |
| `ConnMaxLifetime` | 30 分〜1 時間     | 接続のリークを防止                       |
| `ConnMaxIdleTime` | 5〜10 分          | 未使用接続のリソース解放                 |

16 コア CPU の場合の推奨設定:

```go
db.SetMaxOpenConns(64)   // 16 × 4
db.SetMaxIdleConns(32)   // 16 × 2
```

#### 6.4 トランザクション管理と書き込みの一貫性

複数の関係性タプルを原子的に書き込む場合、トランザクションを使用。

```go
func (dm *DataManager) WriteRelations(ctx context.Context, tuples []*RelationTuple) (int32, error) {
    // トランザクション開始
    tx, err := dm.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelReadCommitted,  // READ COMMITTEDで十分
    })
    if err != nil {
        return 0, err
    }
    defer tx.Rollback()  // エラー時は自動ロールバック

    // プリペアドステートメント（パフォーマンス最適化）
    stmt, err := tx.PrepareContext(ctx, `
        INSERT INTO relations (subject_type, subject_id, relation, entity_type, entity_id)
        VALUES ($1, $2, $3, $4, $5)
        ON CONFLICT DO NOTHING
    `)
    if err != nil {
        return 0, err
    }
    defer stmt.Close()

    var count int32
    for _, tuple := range tuples {
        result, err := stmt.ExecContext(ctx,
            tuple.Subject.Type,
            tuple.Subject.Id,
            tuple.Relation,
            tuple.Entity.Type,
            tuple.Entity.Id,
        )
        if err != nil {
            return 0, err  // エラー時はロールバック
        }

        affected, _ := result.RowsAffected()
        count += int32(affected)
    }

    // コミット（全ての書き込みが成功した場合のみ）
    if err := tx.Commit(); err != nil {
        return 0, err
    }

    return count, nil
}
```

トランザクション分離レベルの選択:

| 分離レベル         | 用途                           | パフォーマンス |
| ------------------ | ------------------------------ | -------------- |
| `READ UNCOMMITTED` | 使用しない（整合性リスク）     | 最速           |
| `READ COMMITTED`   | 推奨: 書き込みに使用           | 高速           |
| `REPEATABLE READ`  | 複雑な多段階読み込みが必要な時 | 中速           |
| `SERIALIZABLE`     | 完全な一貫性が必要な時         | 低速           |

Keruberosu では `READ COMMITTED` で十分（書き込みの原子性を保証し、ファントムリードは問題にならない）。

#### 6.5 グラフ探索のゴルーチン安全性

グラフ探索は各リクエストごとに独立して実行されるため、状態を共有しないことが重要。

```go
type PermissionChecker struct {
    dataManager *DataManager
    // グローバルな状態を持たない（ステートレス）
}

// Checkは並行実行しても安全（goroutineローカルな状態のみ使用）
func (pc *PermissionChecker) Check(ctx context.Context, schema *Schema, req *CheckRequest) (bool, error) {
    // 探索状態をgoroutineローカルに保持
    visited := make(map[string]bool)  // 循環参照防止用

    return pc.evaluate(ctx, schema, req, visited)
}

func (pc *PermissionChecker) evaluate(ctx context.Context, schema *Schema, req *CheckRequest, visited map[string]bool) (bool, error) {
    // visitedはこのgoroutine専用（他のgoroutineと共有しない）
    key := fmt.Sprintf("%s:%s:%s", req.Subject.Type, req.Subject.Id, req.Object.Id)
    if visited[key] {
        return false, nil  // 循環参照を検出
    }
    visited[key] = true

    // グラフ探索ロジック...

    return false, nil
}
```

重要な設計原則:

- ✅ 各ゴルーチンは独立した状態を持つ（`visited` マップをゴルーチンローカルに保持）
- ✅ PermissionChecker はステートレス（共有状態を持たない）
- ✅ DB クエリは並行実行可能（接続プールにより自動的に調整）

#### 6.6 並行アクセスパターンのまとめ

| コンポーネント     | 並行実行        | スレッド安全性の実現方法                    | 注意点                                       |
| ------------------ | --------------- | ------------------------------------------- | -------------------------------------------- |
| L1 キャッシュ      | 読み書き並行 OK | `sync.RWMutex` による排他制御               | 自前実装、各インスタンス独立                 |
| L2 キャッシュ      | 読み書き並行 OK | Redis の内部管理                            | ネットワーク遅延あり、エラー時フォールバック |
| スキーマキャッシュ | 読み書き並行 OK | `atomic.Pointer` によるロックフリー読み込み | 更新は稀、読み込みは頻繁                     |
| DB 接続プール      | 並行 OK         | `database/sql` の内部管理                   | `MaxOpenConns` を適切に設定                  |
| トランザクション   | 独立            | 各 TX は独立した接続を使用                  | 長時間保持しない（デッドロック防止）         |
| グラフ探索         | 並行 OK         | ゴルーチンローカルな状態のみ使用            | 共有状態を持たない設計                       |

#### 6.7 パフォーマンステスト推奨項目

16 コア並行で実行する前に、以下をテスト:

1. ロードテスト: 並行 Check リクエスト（100〜1000 RPS）
2. キャッシュヒット率: 90%+ を目標
3. データベース接続プール: 接続が枯渇しないか確認
4. メモリ使用量: キャッシュサイズに応じて監視
5. レース検出: `go test -race` でデータレースを検出

```bash
# データレース検出付きでテスト実行
go test -race ./...

# ベンチマーク実行
go test -bench=. -benchmem ./...
```

### 7. UI からの連携フロー

#### 7.1 基本的なフロー

```text
1. ブラウザUI (TypeScript)
   ↓ ユーザーがUIで設定

2. DSL生成
   entity document {
     relation owner @user
     permission view = owner
   }
   ↓

3. JSON/Protobuf変換
   {
     "schema_dsl": "entity document { ... }"
   }
   ↓

4. gRPC API呼び出し
   WriteSchema(request)
   ↓

5. サーバー側処理
   - DSLパース
   - バリデーション
   - PostgreSQLに保存
```

#### 7.2 フロントエンド UI 設計（ABAC 対応）

ABAC 設定を UI で構築する方法を段階的に設計する。

設計方針: 複雑な DSL 構文を直接書かせるのではなく、段階的な設定 UI で構築する。

##### ステップ 1: エンティティ定義

```typescript
// UIコンポーネント
interface EntityEditor {
  name: string; // "document"
  relations: RelationConfig[];
  attributes: AttributeConfig[];
  rules: RuleConfig[];
  permissions: PermissionConfig[];
}
```

UI レイアウト:

```text
┌─ Entity: document ──────────────────┐
│                                      │
│ [Relations] [Attributes] [Rules]    │
│ [Permissions]                        │
└──────────────────────────────────────┘
```

##### ステップ 2: Relations 設定

```typescript
interface RelationConfig {
  name: string; // "owner"
  targetTypes: {
    type: string; // "user"
    relation?: string; // optional for "@org#member"
  }[];
}
```

UI コンポーネント例:

```text
Relation: owner
  ├─ Target: [User ▼]
  ├─ Target: [Organization ▼] → [member ▼]  (relation locking)
  └─ [+ Add Target]
```

##### ステップ 3: Attributes 設定

```typescript
interface AttributeConfig {
  name: string; // "classification"
  dataType: string; // "string", "boolean", "integer", "double", "string[]"
}
```

UI コンポーネント例:

```text
┌─ Attributes ─────────────────┐
│ Name: classification         │
│ Type: [String ▼]             │
│                              │
│ Name: is_public              │
│ Type: [Boolean ▼]            │
└──────────────────────────────┘
```

##### ステップ 4: Rules 設定（ビジュアルルールビルダー）

```typescript
interface RuleConfig {
  name: string; // "is_public_doc"
  parameters: {
    name: string; // "is_public"
    type: string; // "boolean"
  }[];
  expression: ExpressionNode; // AST形式
}

interface ExpressionNode {
  type: "comparison" | "logical" | "literal";
  operator?: "==" | "!=" | ">" | "<" | ">=" | "<=" | "and" | "or" | "in";
  left?: ExpressionNode;
  right?: ExpressionNode;
  value?: any;
}
```

UI コンポーネント例（ビジュアルルールビルダー）:

```text
Rule: is_confidential
  Parameters:
    - classification (string)

  Expression Builder:
    ┌────────────────────────────────┐
    │ [classification ▼] [== ▼]      │
    │ ['confidential' ___]           │
    └────────────────────────────────┘

  Advanced Expression:
    ┌────────────────────────────────┐
    │ ( [balance ▼] [>= ▼]           │
    │   [context.data.amount ____] ) │
    │ [and ▼]                        │
    │ ( [context.data.amount ____]   │
    │   [<= ▼] [5000 ____] )         │
    └────────────────────────────────┘
```

##### ステップ 5: Permissions 設定（ビジュアルパーミッションビルダー）

```typescript
interface PermissionConfig {
  name: string; // "view"
  expression: PermissionNode; // AST形式
}

interface PermissionNode {
  type: "or" | "and" | "not" | "relation" | "nested" | "rule";
  children?: PermissionNode[];

  // for relation/nested
  relation?: string; // "owner" or "parent.member"

  // for rule
  ruleName?: string; // "is_public_doc"
  ruleArgs?: string[]; // ["is_public"]
}
```

UI コンポーネント例（ビジュアルパーミッションビルダー）:

```text
Permission: view

Expression Builder:
  ┌─────────────────────────────────────┐
  │ [Relation ▼] owner                  │
  │     [or ▼]                          │
  │ [Relation ▼] viewer                 │
  │     [or ▼]                          │
  │ [Nested ▼] parent.member            │
  │     [or ▼]                          │
  │ [Rule ▼] is_public_doc(is_public)   │
  │                                     │
  │ [+ Add Condition]                   │
  └─────────────────────────────────────┘

Grouping (Advanced):
  ┌─────────────────────────────────────┐
  │ ( [Rule ▼] is_confidential(...)     │
  │   [and ▼]                           │
  │   [Relation ▼] owner )              │
  │     [or ▼]                          │
  │ [Nested ▼] parent.admin             │
  └─────────────────────────────────────┘
```

##### ステップ 6: DSL 生成ロジック

```typescript
class PermifyDSLBuilder {
  // AST → DSL文字列変換
  buildEntityDSL(entity: EntityEditor): string {
    let dsl = `entity ${entity.name} {\n`;

    // Relations
    entity.relations.forEach((rel) => {
      const targets = rel.targetTypes
        .map((t) => (t.relation ? `@${t.type}#${t.relation}` : `@${t.type}`))
        .join(" ");
      dsl += `  relation ${rel.name} ${targets}\n`;
    });

    dsl += "\n";

    // Attributes
    entity.attributes.forEach((attr) => {
      dsl += `  attribute ${attr.name} ${attr.dataType}\n`;
    });

    dsl += "\n";

    // Rules
    entity.rules.forEach((rule) => {
      const params = rule.parameters
        .map((p) => `${p.name} ${p.type}`)
        .join(", ");
      const expr = this.buildExpression(rule.expression);
      dsl += `  rule ${rule.name}(${params}) {\n`;
      dsl += `    ${expr}\n`;
      dsl += `  }\n\n`;
    });

    // Permissions
    entity.permissions.forEach((perm) => {
      const expr = this.buildPermissionExpression(perm.expression);
      dsl += `  permission ${perm.name} = ${expr}\n`;
    });

    dsl += "}\n";
    return dsl;
  }

  buildExpression(node: ExpressionNode): string {
    if (node.type === "literal") {
      return typeof node.value === "string"
        ? `'${node.value}'`
        : String(node.value);
    }

    if (node.type === "comparison" || node.type === "logical") {
      const left = this.buildExpression(node.left!);
      const right = this.buildExpression(node.right!);
      return `${left} ${node.operator} ${right}`;
    }

    return "";
  }

  buildPermissionExpression(node: PermissionNode): string {
    if (node.type === "relation") {
      return node.relation!;
    }

    if (node.type === "nested") {
      return node.relation!; // "parent.member"
    }

    if (node.type === "rule") {
      const args = node.ruleArgs!.join(", ");
      return `${node.ruleName}(${args})`;
    }

    if (node.type === "or" || node.type === "and") {
      return node
        .children!.map((c) => this.buildPermissionExpression(c))
        .join(` ${node.type} `);
    }

    if (node.type === "not") {
      const child = this.buildPermissionExpression(node.children![0]);
      return `not ${child}`;
    }

    return "";
  }
}
```

##### UI 実装のポイント

1. 段階的な設定: 初心者は簡単な Relation-based から始め、上級者は ABAC ルールまで設定可能
2. リアルタイムプレビュー: 設定中の DSL をリアルタイムで表示
3. バリデーション:
   - 存在しない relation の参照チェック
   - 型の整合性チェック
   - 循環参照のチェック
4. テンプレート: よくあるパターン（RBAC、document sharing 等）のプリセット
5. インポート/エクスポート: DSL 文字列との相互変換

## 実装技術スタック

- 言語: Go (1.21+)
- API: gRPC + Protocol Buffers
- データベース: PostgreSQL (14+)
- キャッシュ:
  - L1: 自前 LRU 実装（`sync.RWMutex` + `container/list`）
  - L2: Redis (7+) / Redis Cluster
  - 無効化通知: Redis Pub/Sub
- ABAC エンジン: `github.com/google/cel-go`
- Redis クライアント: `github.com/redis/go-redis/v9`
- 並行性制御: `sync.RWMutex`, `sync/atomic.Pointer`
- DSL パーサー: 独自パーサー（Go）

## Permify DSL 構文まとめ

### 基本構文

```text
entity [entity_name] {
  // 関係性の定義
  relation [relation_name] @[type1] @[type2] @[type3]#[relation]

  // 属性の定義（ABAC用）
  attribute [attr_name] [type]  // boolean, string, integer, double, string[], etc.

  // ルールの定義（ABAC用）
  rule [rule_name]([param1] [type1], [param2] [type2]) {
    [CEL expression]  // e.g., param1 == 'value'
  }

  // パーミッションの定義
  permission [perm_name] = [expression]
}
```

### 論理演算子

- `or`: いずれかが真
- `and`: すべてが真
- `not`: 否定

### ABAC: 比較演算子と式

#### サポートする比較演算子

| 演算子 | 説明           | 例                             |
| ------ | -------------- | ------------------------------ |
| `==`   | 等しい         | `classification == 'public'`   |
| `!=`   | 等しくない     | `status != 'archived'`         |
| `>`    | より大きい     | `age > 18`                     |
| `>=`   | 以上           | `balance >= 1000`              |
| `<`    | より小さい     | `price < 100`                  |
| `<=`   | 以下           | `quantity <= 50`               |
| `in`   | 配列に含まれる | `role in ['admin', 'manager']` |

#### サポートするデータ型

- boolean: `true`, `false`
- string: `'text'`, `"text"`
- integer: `123`, `-456`
- double: `3.14`, `-2.5`
- 配列: `['a', 'b']`, `[1, 2, 3]`

#### 複合式の例

```text
// 基本的な比較
rule is_adult(age integer) {
  age >= 18
}

// 文字列比較
rule is_public(classification string) {
  classification == 'public'
}

// 複数条件（AND）
rule can_withdraw(balance double) {
  balance >= context.data.amount and context.data.amount <= 5000
}

// 複数条件（OR）
rule is_privileged_user(role string) {
  role == 'admin' or role == 'manager'
}

// 配列メンバーシップ
rule is_weekday(valid_days string[]) {
  context.data.day_of_week in valid_days
}

// ネストした条件
rule can_access(clearance_level integer, classification string) {
  (clearance_level >= 3 and classification == 'confidential') or
  (clearance_level >= 5 and classification == 'top_secret')
}
```

#### リクエストコンテキストの参照

`context.data.field` でリクエスト時の動的な値を参照可能:

```text
rule check_ip_range(allowed_ips string[]) {
  context.data.ip_address in allowed_ips
}

rule check_time_window(start_hour integer, end_hour integer) {
  context.data.current_hour >= start_hour and
  context.data.current_hour < end_hour
}

rule check_amount_limit(max_amount double) {
  context.data.requested_amount <= max_amount
}
```

#### 文字列操作（将来拡張）

CEL ベースなので、将来的に以下もサポート可能:

- `str.startsWith('prefix')`
- `str.endsWith('suffix')`
- `str.contains('substring')`
- `str.matches('regex')`

### 特殊構文

- `@type#relation`: 特定の relation を持つエンティティを参照（relation locking）
- `parent.permission`: ネストしたパーミッション参照
- `rule_name(attr)`: ABAC ルールの呼び出し
- `request.field`: リクエストコンテキストの参照

### ABAC 実装詳細

#### CEL 評価エンジンの選定

採用: google/cel-go

理由:

1. 実績: Google 内部で使用、Kubernetes でも採用
2. 機能: 型安全、サンドボックス化、パフォーマンス最適化済み
3. 拡張性: カスタム関数の追加が容易

```go
import (
    "github.com/google/cel-go/cel"
    "github.com/google/cel-go/checker/decls"
)

// CELプログラムのコンパイル
func compileRule(expression string, paramTypes map[string]string) (*cel.Program, error) {
    env, err := cel.NewEnv(
        cel.Declarations(
            decls.NewVar("context", decls.NewMapType(
                decls.String, decls.Dyn,
            )),
        ),
    )
    if err != nil {
        return nil, err
    }

    ast, issues := env.Compile(expression)
    if issues != nil && issues.Err() != nil {
        return nil, issues.Err()
    }

    prg, err := env.Program(ast)
    return &prg, err
}

// ルールの評価
func evaluateRule(prg cel.Program, attributes map[string]interface{}, context map[string]interface{}) (bool, error) {
    result, _, err := prg.Eval(map[string]interface{}{
        "context": context,
        // 属性値をマージ
        ...attributes,
    })
    if err != nil {
        return false, err
    }

    return result.Value().(bool), nil
}
```

## 今後の議論ポイント

### 既に決定した事項

✅ サービス設計: 単一の AuthorizationService（クライアント体験を最優先）
✅ パフォーマンス戦略: L1/L2 キャッシュによる高速化（キャッシュヒット率 90%+ 目標）
✅ キャッシュ実装: 自前 LRU（L1: ローカルメモリ）+ Redis（L2: 分散キャッシュ）
✅ フロントエンド UI 設計: ビジュアルビルダーで段階的に DSL を構築
✅ DB テーブル設計: schemas, relations, attributes の 3 テーブル構成
✅ List 系 API: LookupEntity、LookupSubject、SubjectPermission を実装
✅ Permify 互換 API: metadata（snap_token, depth）、context（tuples, attributes）をサポート
✅ Subject/SubjectReference: Permify と同様の型定義（type, id, relation）
✅ CheckResult enum: ALLOWED/DENIED の明示的な列挙型
✅ ページネーション: continuous_token による一貫性のあるページング
✅ ABAC 比較演算子: CEL 式で `==`, `!=`, `>`, `>=`, `<`, `<=`, `in` をサポート
✅ CEL 評価エンジン: google/cel-go を採用
✅ 並行性: 16+コアでのスレッドセーフな実装（sync.RWMutex、atomic.Pointer）

### 今後議論・実装が必要な事項

1. Metadata（snap_token, depth）の実装:

   - snap_token: スナップショット分離の実装方法（PostgreSQL の txid_snapshot 利用？）
   - depth: 再帰深さ制限の実装とデフォルト値（Permify は 50）

2. Context（contextual tuples & attributes）の実装:

   - contextual tuples: リクエスト時に一時的な関係性を追加
   - contextual attributes: リクエスト時に一時的な属性を追加
   - DB との統合方法（メモリ上でマージ？）

3. Arguments の実装:

   - optional な計算用引数の用途
   - ABAC ルール評価時の引数注入方法

4. SubjectPermission の実装詳細:

   - 全パーミッションを効率的に評価する方法
   - キャッシュ戦略（個別チェックとの統合）

5. トランザクション管理:

   - 複数の関係性を原子的に書き込む方法（実装済み）
   - ロールバック戦略

6. 整合性チェック:

   - スキーマに存在しない関係性の書き込みを拒否するか
   - 型チェックのレベル（厳密 vs 緩和）

7. パフォーマンス最適化:

   - 深いネスト関係の探索をどこまで最適化するか
   - LookupEntity の効率的なクエリ戦略
   - インデックスチューニング

8. キャッシュ無効化の最適化:

   - セカンダリインデックスによる部分無効化
   - Redis Pub/Sub による複数インスタンス間の通知（実装済み）
   - キャッシュウォームアップ戦略

9. 監査ログ:

   - 認可判定の履歴をどう記録するか
   - 保存期間・削除ポリシー
   - パフォーマンスへの影響

10. マルチテナンシー:

    - テナント分離をどう実現するか（DB 分離 vs スキーマ分離 vs テーブル内分離）
    - テナント間のデータ漏洩防止
    - Permify の tenant_id 概念の導入

11. Relation Locking 実装:

    - `@organization#member` の実装詳細
    - クエリ最適化

12. DSL パーサー実装:

    - 字句解析・構文解析の詳細
    - エラーハンドリング・エラーメッセージ

13. セキュリティ:

    - gRPC 認証・認可
    - レート制限
    - DoS 対策

14. 運用:
    - メトリクス収集（Prometheus 等）
    - ヘルスチェック
    - グレースフルシャットダウン

## 詳細設計

以下のセクションで、実装に必要な詳細設計を記述する。

### A. DSL パーサーの詳細設計

#### A.1 パーサーアーキテクチャ

```text
DSL文字列 → Lexer（字句解析） → Tokens → Parser（構文解析） → AST → Validator（検証） → Schema構造体
```

処理フロー:

1. Lexer（字句解析器）: 文字列をトークン列に分解
2. Parser（構文解析器）: トークン列を AST（抽象構文木）に変換
3. Validator（検証器）: AST の意味的な正しさを検証
4. Converter（変換器）: AST を内部の `Schema` 構造体に変換

AST とは？

AST（Abstract Syntax Tree：抽象構文木）とは、プログラムや DSL の構造をツリー（木構造）で表現したものです。

例えば、以下の DSL：

```text
entity document {
  relation owner: user
  permission edit = owner
}
```

これを AST で表現すると：

```text
SchemaAST
└── EntityAST (name: "document")
    ├── RelationAST (name: "owner")
    │   └── RelationTargetAST (type: "user")
    └── PermissionAST (name: "edit")
        └── RelationPermissionAST (relation: "owner")
```

なぜ AST が必要か？

1. 構造化されたデータ: 文字列のままでは処理しにくいが、ツリー構造にすることでプログラムで扱いやすくなる
2. 検証が容易: 「未定義の relation を参照していないか」などのチェックが簡単
3. 変換が容易: DSL → AST → 内部データ構造（Schema）という段階的な変換ができる
4. エラー報告: どの部分で問題が起きたかを正確に指摘できる

具体例で理解する:

DSL 文字列:

```text
permission edit = owner or editor
```

↓ Lexer でトークンに分解

```json
[TOKEN_PERMISSION, TOKEN_IDENT("edit"), TOKEN_ASSIGN, TOKEN_IDENT("owner"), TOKEN_OR, TOKEN_IDENT("editor")]
```

↓ Parser で AST に変換

```text
PermissionAST {
  Name: "edit"
  Rule: LogicalPermissionAST {
    Operator: "or"
    Operands: [
      RelationPermissionAST { Relation: "owner" },
      RelationPermissionAST { Relation: "editor" }
    ]
  }
}
```

↓ Validator でチェック

- "owner" という relation は定義されているか？
- "editor" という relation は定義されているか？

↓ Converter で Schema 構造体に変換

```go
Permission{
  Name: "edit",
  Rule: &PermissionRule{
    Type: "logical",
    Operator: "or",
    Operands: [...],
  }
}
```

このように、AST は文字列とプログラムで使うデータ構造の橋渡しをする重要な中間表現です。

#### A.2 字句解析（Lexer）

トークン定義:

```go
type TokenType int

const (
    // リテラル
    TOKEN_IDENT TokenType = iota  // entity, relation, permission など
    TOKEN_STRING                  // 文字列リテラル 'value'
    TOKEN_NUMBER                  // 123, 3.14
    TOKEN_BOOL                    // true, false

    // キーワード
    TOKEN_ENTITY
    TOKEN_RELATION
    TOKEN_ATTRIBUTE
    TOKEN_RULE
    TOKEN_PERMISSION
    TOKEN_OR
    TOKEN_AND
    TOKEN_NOT
    TOKEN_IN

    // 演算子
    TOKEN_AT        // @
    TOKEN_HASH      // #
    TOKEN_DOT       // .
    TOKEN_EQ        // ==
    TOKEN_NEQ       // !=
    TOKEN_LT        // <
    TOKEN_LTE       // <=
    TOKEN_GT        // >
    TOKEN_GTE       // >=
    TOKEN_ASSIGN    // =

    // 区切り文字
    TOKEN_LPAREN    // (
    TOKEN_RPAREN    // )
    TOKEN_LBRACE    // {
    TOKEN_RBRACE    // }
    TOKEN_LBRACKET  // [
    TOKEN_RBRACKET  // ]
    TOKEN_COMMA     // ,
    TOKEN_SEMICOLON // ;

    // 特殊
    TOKEN_EOF
    TOKEN_ILLEGAL
)

type Token struct {
    Type    TokenType
    Literal string
    Line    int
    Column  int
}
```

Lexer 実装:

```go
type Lexer struct {
    input        string
    position     int  // 現在位置
    readPosition int  // 次の読み取り位置
    ch           byte // 現在の文字
    line         int
    column       int
}

func NewLexer(input string) *Lexer {
    l := &Lexer{input: input, line: 1, column: 0}
    l.readChar()
    return l
}

func (l *Lexer) readChar() {
    if l.readPosition >= len(l.input) {
        l.ch = 0 // EOF
    } else {
        l.ch = l.input[l.readPosition]
    }
    l.position = l.readPosition
    l.readPosition++
    l.column++
    if l.ch == '\n' {
        l.line++
        l.column = 0
    }
}

func (l *Lexer) NextToken() Token {
    l.skipWhitespace()
    l.skipComment()

    tok := Token{Line: l.line, Column: l.column}

    switch l.ch {
    case '@':
        tok.Type = TOKEN_AT
        tok.Literal = string(l.ch)
    case '#':
        tok.Type = TOKEN_HASH
        tok.Literal = string(l.ch)
    case '.':
        tok.Type = TOKEN_DOT
        tok.Literal = string(l.ch)
    case '=':
        if l.peekChar() == '=' {
            ch := l.ch
            l.readChar()
            tok.Type = TOKEN_EQ
            tok.Literal = string(ch) + string(l.ch)
        } else {
            tok.Type = TOKEN_ASSIGN
            tok.Literal = string(l.ch)
        }
    case '!':
        if l.peekChar() == '=' {
            ch := l.ch
            l.readChar()
            tok.Type = TOKEN_NEQ
            tok.Literal = string(ch) + string(l.ch)
        } else {
            tok.Type = TOKEN_ILLEGAL
            tok.Literal = string(l.ch)
        }
    case '<':
        if l.peekChar() == '=' {
            ch := l.ch
            l.readChar()
            tok.Type = TOKEN_LTE
            tok.Literal = string(ch) + string(l.ch)
        } else {
            tok.Type = TOKEN_LT
            tok.Literal = string(l.ch)
        }
    case '>':
        if l.peekChar() == '=' {
            ch := l.ch
            l.readChar()
            tok.Type = TOKEN_GTE
            tok.Literal = string(ch) + string(l.ch)
        } else {
            tok.Type = TOKEN_GT
            tok.Literal = string(l.ch)
        }
    case '(':
        tok.Type = TOKEN_LPAREN
        tok.Literal = string(l.ch)
    case ')':
        tok.Type = TOKEN_RPAREN
        tok.Literal = string(l.ch)
    case '{':
        tok.Type = TOKEN_LBRACE
        tok.Literal = string(l.ch)
    case '}':
        tok.Type = TOKEN_RBRACE
        tok.Literal = string(l.ch)
    case '[':
        tok.Type = TOKEN_LBRACKET
        tok.Literal = string(l.ch)
    case ']':
        tok.Type = TOKEN_RBRACKET
        tok.Literal = string(l.ch)
    case ',':
        tok.Type = TOKEN_COMMA
        tok.Literal = string(l.ch)
    case '\'', '"':
        tok.Type = TOKEN_STRING
        tok.Literal = l.readString(l.ch)
        return tok
    case 0:
        tok.Type = TOKEN_EOF
        tok.Literal = ""
    default:
        if isLetter(l.ch) {
            tok.Literal = l.readIdentifier()
            tok.Type = lookupKeyword(tok.Literal)
            return tok
        } else if isDigit(l.ch) {
            tok.Literal = l.readNumber()
            tok.Type = TOKEN_NUMBER
            return tok
        } else {
            tok.Type = TOKEN_ILLEGAL
            tok.Literal = string(l.ch)
        }
    }

    l.readChar()
    return tok
}

func (l *Lexer) skipWhitespace() {
    for l.ch == ' ' || l.ch == '\t' || l.ch == '\n' || l.ch == '\r' {
        l.readChar()
    }
}

func (l *Lexer) skipComment() {
    if l.ch == '/' && l.peekChar() == '/' {
        for l.ch != '\n' && l.ch != 0 {
            l.readChar()
        }
        l.skipWhitespace()
    }
}

func (l *Lexer) readIdentifier() string {
    position := l.position
    for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
        l.readChar()
    }
    return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
    position := l.position
    for isDigit(l.ch) {
        l.readChar()
    }
    if l.ch == '.' && isDigit(l.peekChar()) {
        l.readChar()
        for isDigit(l.ch) {
            l.readChar()
        }
    }
    return l.input[position:l.position]
}

func (l *Lexer) readString(quote byte) string {
    l.readChar() // skip opening quote
    position := l.position
    for l.ch != quote && l.ch != 0 {
        if l.ch == '\\' {
            l.readChar() // skip escape char
        }
        l.readChar()
    }
    str := l.input[position:l.position]
    l.readChar() // skip closing quote
    return str
}

func (l *Lexer) peekChar() byte {
    if l.readPosition >= len(l.input) {
        return 0
    }
    return l.input[l.readPosition]
}

func isLetter(ch byte) bool {
    return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z'
}

func isDigit(ch byte) bool {
    return '0' <= ch && ch <= '9'
}

var keywords = map[string]TokenType{
    "entity":     TOKEN_ENTITY,
    "relation":   TOKEN_RELATION,
    "attribute":  TOKEN_ATTRIBUTE,
    "rule":       TOKEN_RULE,
    "permission": TOKEN_PERMISSION,
    "or":         TOKEN_OR,
    "and":        TOKEN_AND,
    "not":        TOKEN_NOT,
    "in":         TOKEN_IN,
    "true":       TOKEN_BOOL,
    "false":      TOKEN_BOOL,
}

func lookupKeyword(ident string) TokenType {
    if tok, ok := keywords[ident]; ok {
        return tok
    }
    return TOKEN_IDENT
}
```

#### A.3 構文解析（Parser）

AST 定義:

```go
// AST ノードのベースインターフェース
type Node interface {
    TokenLiteral() string
    String() string
}

// スキーマ全体
type SchemaAST struct {
    Entities []*EntityAST
}

// Entity定義
type EntityAST struct {
    Name        string
    Relations   []*RelationAST
    Attributes  []*AttributeAST
    Rules       []*RuleAST
    Permissions []*PermissionAST
}

// Relation定義
type RelationAST struct {
    Name    string
    Targets []*RelationTargetAST
}

type RelationTargetAST struct {
    Type     string  // "user", "organization"
    Relation string  // optional: "member" for "@organization#member"
}

// Attribute定義
type AttributeAST struct {
    Name     string
    DataType string  // "string", "boolean", "integer", etc.
}

// Rule定義
type RuleAST struct {
    Name       string
    Parameters []*RuleParameterAST
    Expression ExpressionAST
}

type RuleParameterAST struct {
    Name string
    Type string
}

// Expression（CEL式）
type ExpressionAST interface {
    Node
    expressionNode()
}

type BinaryExpressionAST struct {
    Left     ExpressionAST
    Operator string  // "==", "!=", ">", "<", etc.
    Right    ExpressionAST
}

type LogicalExpressionAST struct {
    Left     ExpressionAST
    Operator string  // "and", "or"
    Right    ExpressionAST
}

type IdentifierAST struct {
    Value string
}

type LiteralAST struct {
    Value interface{}  // string, int, float, bool
}

// Permission定義
type PermissionAST struct {
    Name string
    Rule PermissionRuleAST
}

type PermissionRuleAST interface {
    Node
    permissionRuleNode()
}

type RelationPermissionAST struct {
    Relation string  // "owner"
}

type NestedPermissionAST struct {
    Path string  // "parent.member"
}

type RulePermissionAST struct {
    RuleName string
    Args     []string
}

type LogicalPermissionAST struct {
    Operator string  // "or", "and", "not"
    Operands []PermissionRuleAST
}
```

Parser 実装:

```go
type Parser struct {
    lexer     *Lexer
    curToken  Token
    peekToken Token
    errors    []string
}

func NewParser(l *Lexer) *Parser {
    p := &Parser{lexer: l, errors: []string{}}
    p.nextToken()
    p.nextToken()
    return p
}

func (p *Parser) nextToken() {
    p.curToken = p.peekToken
    p.peekToken = p.lexer.NextToken()
}

func (p *Parser) ParseSchema() (*SchemaAST, error) {
    schema := &SchemaAST{Entities: []*EntityAST{}}

    for p.curToken.Type != TOKEN_EOF {
        if p.curToken.Type == TOKEN_ENTITY {
            entity, err := p.parseEntity()
            if err != nil {
                return nil, err
            }
            schema.Entities = append(schema.Entities, entity)
        } else {
            return nil, p.error(fmt.Sprintf("expected 'entity', got '%s'", p.curToken.Literal))
        }
    }

    if len(p.errors) > 0 {
        return nil, fmt.Errorf("parser errors: %s", strings.Join(p.errors, "; "))
    }

    return schema, nil
}

func (p *Parser) parseEntity() (*EntityAST, error) {
    entity := &EntityAST{}

    if !p.expectToken(TOKEN_ENTITY) {
        return nil, p.error("expected 'entity'")
    }
    p.nextToken()

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected entity name")
    }
    entity.Name = p.curToken.Literal
    p.nextToken()

    if !p.expectToken(TOKEN_LBRACE) {
        return nil, p.error("expected '{'")
    }
    p.nextToken()

    // Parse entity body
    for p.curToken.Type != TOKEN_RBRACE && p.curToken.Type != TOKEN_EOF {
        switch p.curToken.Type {
        case TOKEN_RELATION:
            rel, err := p.parseRelation()
            if err != nil {
                return nil, err
            }
            entity.Relations = append(entity.Relations, rel)
        case TOKEN_ATTRIBUTE:
            attr, err := p.parseAttribute()
            if err != nil {
                return nil, err
            }
            entity.Attributes = append(entity.Attributes, attr)
        case TOKEN_RULE:
            rule, err := p.parseRule()
            if err != nil {
                return nil, err
            }
            entity.Rules = append(entity.Rules, rule)
        case TOKEN_PERMISSION:
            perm, err := p.parsePermission()
            if err != nil {
                return nil, err
            }
            entity.Permissions = append(entity.Permissions, perm)
        default:
            return nil, p.error(fmt.Sprintf("unexpected token in entity: %s", p.curToken.Literal))
        }
    }

    if !p.expectToken(TOKEN_RBRACE) {
        return nil, p.error("expected '}'")
    }
    p.nextToken()

    return entity, nil
}

func (p *Parser) parseRelation() (*RelationAST, error) {
    rel := &RelationAST{}

    p.nextToken() // skip 'relation'

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected relation name")
    }
    rel.Name = p.curToken.Literal
    p.nextToken()

    // Parse targets (@user @organization#member)
    for p.curToken.Type == TOKEN_AT {
        p.nextToken()
        if p.curToken.Type != TOKEN_IDENT {
            return nil, p.error("expected type name after '@'")
        }

        target := &RelationTargetAST{Type: p.curToken.Literal}
        p.nextToken()

        if p.curToken.Type == TOKEN_HASH {
            p.nextToken()
            if p.curToken.Type != TOKEN_IDENT {
                return nil, p.error("expected relation name after '#'")
            }
            target.Relation = p.curToken.Literal
            p.nextToken()
        }

        rel.Targets = append(rel.Targets, target)
    }

    return rel, nil
}

func (p *Parser) parseAttribute() (*AttributeAST, error) {
    attr := &AttributeAST{}

    p.nextToken() // skip 'attribute'

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected attribute name")
    }
    attr.Name = p.curToken.Literal
    p.nextToken()

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected attribute type")
    }
    attr.DataType = p.curToken.Literal
    p.nextToken()

    return attr, nil
}

func (p *Parser) parseRule() (*RuleAST, error) {
    rule := &RuleAST{}

    p.nextToken() // skip 'rule'

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected rule name")
    }
    rule.Name = p.curToken.Literal
    p.nextToken()

    // Parse parameters
    if !p.expectToken(TOKEN_LPAREN) {
        return nil, p.error("expected '('")
    }
    p.nextToken()

    for p.curToken.Type != TOKEN_RPAREN && p.curToken.Type != TOKEN_EOF {
        param := &RuleParameterAST{}

        if p.curToken.Type != TOKEN_IDENT {
            return nil, p.error("expected parameter name")
        }
        param.Name = p.curToken.Literal
        p.nextToken()

        if p.curToken.Type != TOKEN_IDENT {
            return nil, p.error("expected parameter type")
        }
        param.Type = p.curToken.Literal
        p.nextToken()

        rule.Parameters = append(rule.Parameters, param)

        if p.curToken.Type == TOKEN_COMMA {
            p.nextToken()
        }
    }

    if !p.expectToken(TOKEN_RPAREN) {
        return nil, p.error("expected ')'")
    }
    p.nextToken()

    if !p.expectToken(TOKEN_LBRACE) {
        return nil, p.error("expected '{'")
    }
    p.nextToken()

    // Parse expression (CEL)
    expr, err := p.parseExpression()
    if err != nil {
        return nil, err
    }
    rule.Expression = expr

    if !p.expectToken(TOKEN_RBRACE) {
        return nil, p.error("expected '}'")
    }
    p.nextToken()

    return rule, nil
}

func (p *Parser) parseExpression() (ExpressionAST, error) {
    // Simplified expression parser for CEL
    // Full implementation would need precedence climbing or Pratt parsing
    return p.parseLogicalOr()
}

func (p *Parser) parseLogicalOr() (ExpressionAST, error) {
    left, err := p.parseLogicalAnd()
    if err != nil {
        return nil, err
    }

    for p.curToken.Type == TOKEN_OR {
        op := p.curToken.Literal
        p.nextToken()
        right, err := p.parseLogicalAnd()
        if err != nil {
            return nil, err
        }
        left = &LogicalExpressionAST{Left: left, Operator: op, Right: right}
    }

    return left, nil
}

func (p *Parser) parseLogicalAnd() (ExpressionAST, error) {
    left, err := p.parseComparison()
    if err != nil {
        return nil, err
    }

    for p.curToken.Type == TOKEN_AND {
        op := p.curToken.Literal
        p.nextToken()
        right, err := p.parseComparison()
        if err != nil {
            return nil, err
        }
        left = &LogicalExpressionAST{Left: left, Operator: op, Right: right}
    }

    return left, nil
}

func (p *Parser) parseComparison() (ExpressionAST, error) {
    left, err := p.parsePrimary()
    if err != nil {
        return nil, err
    }

    if p.curToken.Type == TOKEN_EQ || p.curToken.Type == TOKEN_NEQ ||
       p.curToken.Type == TOKEN_LT || p.curToken.Type == TOKEN_LTE ||
       p.curToken.Type == TOKEN_GT || p.curToken.Type == TOKEN_GTE ||
       p.curToken.Type == TOKEN_IN {
        op := p.curToken.Literal
        p.nextToken()
        right, err := p.parsePrimary()
        if err != nil {
            return nil, err
        }
        return &BinaryExpressionAST{Left: left, Operator: op, Right: right}, nil
    }

    return left, nil
}

func (p *Parser) parsePrimary() (ExpressionAST, error) {
    switch p.curToken.Type {
    case TOKEN_IDENT:
        ident := &IdentifierAST{Value: p.curToken.Literal}
        p.nextToken()
        return ident, nil
    case TOKEN_STRING:
        lit := &LiteralAST{Value: p.curToken.Literal}
        p.nextToken()
        return lit, nil
    case TOKEN_NUMBER:
        // Parse as int or float
        val, err := strconv.ParseFloat(p.curToken.Literal, 64)
        if err != nil {
            return nil, err
        }
        lit := &LiteralAST{Value: val}
        p.nextToken()
        return lit, nil
    case TOKEN_BOOL:
        val := p.curToken.Literal == "true"
        lit := &LiteralAST{Value: val}
        p.nextToken()
        return lit, nil
    case TOKEN_LPAREN:
        p.nextToken()
        expr, err := p.parseExpression()
        if err != nil {
            return nil, err
        }
        if !p.expectToken(TOKEN_RPAREN) {
            return nil, p.error("expected ')'")
        }
        p.nextToken()
        return expr, nil
    default:
        return nil, p.error(fmt.Sprintf("unexpected token in expression: %s", p.curToken.Literal))
    }
}

func (p *Parser) parsePermission() (*PermissionAST, error) {
    perm := &PermissionAST{}

    p.nextToken() // skip 'permission'

    if p.curToken.Type != TOKEN_IDENT {
        return nil, p.error("expected permission name")
    }
    perm.Name = p.curToken.Literal
    p.nextToken()

    if !p.expectToken(TOKEN_ASSIGN) {
        return nil, p.error("expected '='")
    }
    p.nextToken()

    rule, err := p.parsePermissionRule()
    if err != nil {
        return nil, err
    }
    perm.Rule = rule

    return perm, nil
}

func (p *Parser) parsePermissionRule() (PermissionRuleAST, error) {
    return p.parsePermissionOr()
}

func (p *Parser) parsePermissionOr() (PermissionRuleAST, error) {
    left, err := p.parsePermissionAnd()
    if err != nil {
        return nil, err
    }

    if p.curToken.Type == TOKEN_OR {
        operands := []PermissionRuleAST{left}
        for p.curToken.Type == TOKEN_OR {
            p.nextToken()
            right, err := p.parsePermissionAnd()
            if err != nil {
                return nil, err
            }
            operands = append(operands, right)
        }
        return &LogicalPermissionAST{Operator: "or", Operands: operands}, nil
    }

    return left, nil
}

func (p *Parser) parsePermissionAnd() (PermissionRuleAST, error) {
    left, err := p.parsePermissionPrimary()
    if err != nil {
        return nil, err
    }

    if p.curToken.Type == TOKEN_AND {
        operands := []PermissionRuleAST{left}
        for p.curToken.Type == TOKEN_AND {
            p.nextToken()
            right, err := p.parsePermissionPrimary()
            if err != nil {
                return nil, err
            }
            operands = append(operands, right)
        }
        return &LogicalPermissionAST{Operator: "and", Operands: operands}, nil
    }

    return left, nil
}

func (p *Parser) parsePermissionPrimary() (PermissionRuleAST, error) {
    if p.curToken.Type == TOKEN_NOT {
        p.nextToken()
        operand, err := p.parsePermissionPrimary()
        if err != nil {
            return nil, err
        }
        return &LogicalPermissionAST{Operator: "not", Operands: []PermissionRuleAST{operand}}, nil
    }

    if p.curToken.Type == TOKEN_LPAREN {
        p.nextToken()
        rule, err := p.parsePermissionRule()
        if err != nil {
            return nil, err
        }
        if !p.expectToken(TOKEN_RPAREN) {
            return nil, p.error("expected ')'")
        }
        p.nextToken()
        return rule, nil
    }

    if p.curToken.Type == TOKEN_IDENT {
        name := p.curToken.Literal
        p.nextToken()

        // Check for function call (rule)
        if p.curToken.Type == TOKEN_LPAREN {
            p.nextToken()
            args := []string{}
            for p.curToken.Type != TOKEN_RPAREN && p.curToken.Type != TOKEN_EOF {
                if p.curToken.Type != TOKEN_IDENT {
                    return nil, p.error("expected argument name")
                }
                args = append(args, p.curToken.Literal)
                p.nextToken()
                if p.curToken.Type == TOKEN_COMMA {
                    p.nextToken()
                }
            }
            if !p.expectToken(TOKEN_RPAREN) {
                return nil, p.error("expected ')'")
            }
            p.nextToken()
            return &RulePermissionAST{RuleName: name, Args: args}, nil
        }

        // Check for nested relation (parent.member)
        if p.curToken.Type == TOKEN_DOT {
            path := name
            for p.curToken.Type == TOKEN_DOT {
                path += "."
                p.nextToken()
                if p.curToken.Type != TOKEN_IDENT {
                    return nil, p.error("expected identifier after '.'")
                }
                path += p.curToken.Literal
                p.nextToken()
            }
            return &NestedPermissionAST{Path: path}, nil
        }

        // Simple relation reference
        return &RelationPermissionAST{Relation: name}, nil
    }

    return nil, p.error("expected permission rule")
}

func (p *Parser) expectToken(t TokenType) bool {
    return p.curToken.Type == t
}

func (p *Parser) error(msg string) error {
    errMsg := fmt.Sprintf("line %d, column %d: %s", p.curToken.Line, p.curToken.Column, msg)
    p.errors = append(p.errors, errMsg)
    return fmt.Errorf(errMsg)
}
```

#### A.4 バリデーター

```go
type Validator struct {
    errors []string
}

func NewValidator() *Validator {
    return &Validator{errors: []string{}}
}

func (v *Validator) Validate(schema *SchemaAST) error {
    // 1. Entity名の重複チェック
    entityNames := make(map[string]bool)
    for _, entity := range schema.Entities {
        if entityNames[entity.Name] {
            v.addError(fmt.Sprintf("duplicate entity name: %s", entity.Name))
        }
        entityNames[entity.Name] = true

        // 2. Entity内のバリデーション
        v.validateEntity(entity, schema)
    }

    if len(v.errors) > 0 {
        return fmt.Errorf("validation errors: %s", strings.Join(v.errors, "; "))
    }
    return nil
}

func (v *Validator) validateEntity(entity *EntityAST, schema *SchemaAST) {
    // Relation名の重複チェック
    relationNames := make(map[string]bool)
    for _, rel := range entity.Relations {
        if relationNames[rel.Name] {
            v.addError(fmt.Sprintf("duplicate relation name in %s: %s", entity.Name, rel.Name))
        }
        relationNames[rel.Name] = true

        // Relation targetの存在チェック
        for _, target := range rel.Targets {
            if !v.entityExists(schema, target.Type) {
                v.addError(fmt.Sprintf("unknown entity type in relation %s.%s: %s",
                    entity.Name, rel.Name, target.Type))
            }
        }
    }

    // Attribute名の重複チェック
    attrNames := make(map[string]bool)
    for _, attr := range entity.Attributes {
        if attrNames[attr.Name] {
            v.addError(fmt.Sprintf("duplicate attribute name in %s: %s", entity.Name, attr.Name))
        }
        attrNames[attr.Name] = true
    }

    // Rule名の重複チェック
    ruleNames := make(map[string]bool)
    for _, rule := range entity.Rules {
        if ruleNames[rule.Name] {
            v.addError(fmt.Sprintf("duplicate rule name in %s: %s", entity.Name, rule.Name))
        }
        ruleNames[rule.Name] = true
    }

    // Permission名の重複チェック
    permNames := make(map[string]bool)
    for _, perm := range entity.Permissions {
        if permNames[perm.Name] {
            v.addError(fmt.Sprintf("duplicate permission name in %s: %s", entity.Name, perm.Name))
        }
        permNames[perm.Name] = true

        // Permissionルール内の参照チェック
        v.validatePermissionRule(entity, perm.Rule, schema)
    }
}

func (v *Validator) validatePermissionRule(entity *EntityAST, rule PermissionRuleAST, schema *SchemaAST) {
    switch r := rule.(type) {
    case *RelationPermissionAST:
        // Relationの存在チェック
        if !v.relationExists(entity, r.Relation) {
            v.addError(fmt.Sprintf("unknown relation in %s: %s", entity.Name, r.Relation))
        }
    case *NestedPermissionAST:
        // Nested pathの検証（例: parent.member）
        parts := strings.Split(r.Path, ".")
        if len(parts) < 2 {
            v.addError(fmt.Sprintf("invalid nested path in %s: %s", entity.Name, r.Path))
            return
        }
        // 最初のpartはrelationでなければならない
        if !v.relationExists(entity, parts[0]) {
            v.addError(fmt.Sprintf("unknown relation in nested path %s.%s: %s",
                entity.Name, r.Path, parts[0]))
        }
    case *RulePermissionAST:
        // Ruleの存在チェック
        if !v.ruleExists(entity, r.RuleName) {
            v.addError(fmt.Sprintf("unknown rule in %s: %s", entity.Name, r.RuleName))
        }
    case *LogicalPermissionAST:
        for _, operand := range r.Operands {
            v.validatePermissionRule(entity, operand, schema)
        }
    }
}

func (v *Validator) entityExists(schema *SchemaAST, name string) bool {
    for _, entity := range schema.Entities {
        if entity.Name == name {
            return true
        }
    }
    return false
}

func (v *Validator) relationExists(entity *EntityAST, name string) bool {
    for _, rel := range entity.Relations {
        if rel.Name == name {
            return true
        }
    }
    return false
}

func (v *Validator) ruleExists(entity *EntityAST, name string) bool {
    for _, rule := range entity.Rules {
        if rule.Name == name {
            return true
        }
    }
    return false
}

func (v *Validator) addError(msg string) {
    v.errors = append(v.errors, msg)
}
```

#### A.5 AST → Schema 変換

```go
func ConvertASTToSchema(ast *SchemaAST) (*Schema, error) {
    schema := &Schema{
        Entities: make(map[string]*Entity),
    }

    for _, entityAST := range ast.Entities {
        entity := &Entity{
            Name:        entityAST.Name,
            Relations:   make(map[string]*Relation),
            Attributes:  make(map[string]*Attribute),
            Rules:       make(map[string]*Rule),
            Permissions: make(map[string]*Permission),
        }

        // Convert Relations
        for _, relAST := range entityAST.Relations {
            targets := make([]RelationTarget, len(relAST.Targets))
            for i, targetAST := range relAST.Targets {
                targets[i] = RelationTarget{
                    Type:     targetAST.Type,
                    Relation: targetAST.Relation,
                }
            }
            entity.Relations[relAST.Name] = &Relation{
                Name:        relAST.Name,
                TargetTypes: targets,
            }
        }

        // Convert Attributes
        for _, attrAST := range entityAST.Attributes {
            entity.Attributes[attrAST.Name] = &Attribute{
                Name:     attrAST.Name,
                DataType: attrAST.DataType,
            }
        }

        // Convert Rules
        for _, ruleAST := range entityAST.Rules {
            params := make([]RuleParameter, len(ruleAST.Parameters))
            for i, paramAST := range ruleAST.Parameters {
                params[i] = RuleParameter{
                    Name: paramAST.Name,
                    Type: paramAST.Type,
                }
            }

            // Convert expression AST to CEL string
            celExpr := expressionASTToCEL(ruleAST.Expression)

            entity.Rules[ruleAST.Name] = &Rule{
                Name:       ruleAST.Name,
                Parameters: params,
                Expression: celExpr,
            }
        }

        // Convert Permissions
        for _, permAST := range entityAST.Permissions {
            entity.Permissions[permAST.Name] = &Permission{
                Name: permAST.Name,
                Rule: convertPermissionRule(permAST.Rule),
            }
        }

        schema.Entities[entity.Name] = entity
    }

    return schema, nil
}

func convertPermissionRule(ruleAST PermissionRuleAST) *PermissionRule {
    switch r := ruleAST.(type) {
    case *RelationPermissionAST:
        return &PermissionRule{
            Type:     "relation",
            Relation: r.Relation,
        }
    case *NestedPermissionAST:
        return &PermissionRule{
            Type:     "nested",
            Relation: r.Path,
        }
    case *RulePermissionAST:
        return &PermissionRule{
            Type:     "rule",
            RuleName: r.RuleName,
            RuleArgs: r.Args,
        }
    case *LogicalPermissionAST:
        children := make([]*PermissionRule, len(r.Operands))
        for i, operand := range r.Operands {
            children[i] = convertPermissionRule(operand)
        }
        return &PermissionRule{
            Type:     r.Operator,
            Children: children,
        }
    }
    return nil
}

func expressionASTToCEL(expr ExpressionAST) string {
    switch e := expr.(type) {
    case *BinaryExpressionAST:
        left := expressionASTToCEL(e.Left)
        right := expressionASTToCEL(e.Right)
        return fmt.Sprintf("%s %s %s", left, e.Operator, right)
    case *LogicalExpressionAST:
        left := expressionASTToCEL(e.Left)
        right := expressionASTToCEL(e.Right)
        return fmt.Sprintf("(%s) %s (%s)", left, e.Operator, right)
    case *IdentifierAST:
        return e.Value
    case *LiteralAST:
        switch v := e.Value.(type) {
        case string:
            return fmt.Sprintf("'%s'", v)
        case bool:
            return fmt.Sprintf("%t", v)
        default:
            return fmt.Sprintf("%v", v)
        }
    }
    return ""
}
```

#### A.6 使用例

```go
func ParsePermifyDSL(dsl string) (*Schema, error) {
    // 1. Lexical analysis
    lexer := NewLexer(dsl)

    // 2. Parsing
    parser := NewParser(lexer)
    ast, err := parser.ParseSchema()
    if err != nil {
        return nil, fmt.Errorf("parse error: %w", err)
    }

    // 3. Validation
    validator := NewValidator()
    if err := validator.Validate(ast); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }

    // 4. Convert to internal schema
    schema, err := ConvertASTToSchema(ast)
    if err != nil {
        return nil, fmt.Errorf("conversion error: %w", err)
    }

    return schema, nil
}

// Example usage
func main() {
    dsl := `
entity user {}

entity document {
  relation owner @user
  relation viewer @user

  attribute classification string

  rule is_confidential(classification string) {
    classification == 'confidential'
  }

  permission view = owner or viewer
  permission edit = owner
  permission delete = is_confidential(classification) and owner
}
`

    schema, err := ParsePermifyDSL(dsl)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Parsed schema with %d entities\n", len(schema.Entities))
}
```

### B. PostgreSQL スキーマの最終化

#### B.1 設計方針

最小限で動かすことを優先:

- VARCHAR 使用、正規化なし
- 後から段階的に最適化可能
- L1/L2 キャッシュで 90%+をカバーする前提
- DB パフォーマンスの影響は相対的に小さい（全体の 10%未満）

#### B.2 テーブル設計

##### B.2.1 schemas テーブル

```sql
CREATE TABLE schemas (
    id INTEGER PRIMARY KEY DEFAULT 1,
    schema_dsl TEXT NOT NULL,
    schema_json JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (id = 1)  -- 常に1行のみを強制
);

-- 初期行を作成
INSERT INTO schemas (id, schema_dsl, schema_json)
VALUES (1, '', '{}')
ON CONFLICT (id) DO NOTHING;

-- インデックス
CREATE INDEX idx_schemas_json ON schemas USING GIN (schema_json);

-- 更新時刻自動更新トリガー
CREATE OR REPLACE FUNCTION update_schemas_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_schemas_updated_at
    BEFORE UPDATE ON schemas
    FOR EACH ROW
    EXECUTE FUNCTION update_schemas_updated_at();
```

設計理由:

- `schema_dsl`: 人間が読める形式（UI 表示用）
- `schema_json`: パース済み JSON（検証・参照用）
- JOSNB は内部でバイナリ形式なのでパフォーマンス十分
- 読み込み頻度が低い（起動時＋更新時のみ）ため、最適化不要

##### B.2.2 relations テーブル

```sql
CREATE TABLE relations (
    id BIGSERIAL PRIMARY KEY,
    subject_type VARCHAR(255) NOT NULL,
    subject_id VARCHAR(255) NOT NULL,
    relation VARCHAR(255) NOT NULL,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(subject_type, subject_id, relation, entity_type, entity_id)
);

-- インデックス: エンティティから逆引き（Check API用）
CREATE INDEX idx_relations_reverse
    ON relations(entity_type, entity_id, relation);

-- インデックス: サブジェクトから検索（LookupEntity API用）
CREATE INDEX idx_relations_forward
    ON relations(subject_type, subject_id, relation);

-- インデックス: カバリングインデックス（Index-Only Scan実現）
CREATE INDEX idx_relations_lookup_entity
    ON relations(subject_type, subject_id, relation, entity_type, entity_id);

-- インデックス: LookupSubject最適化
CREATE INDEX idx_relations_lookup_subject
    ON relations(entity_type, entity_id, relation, subject_type, subject_id);
```

設計理由:

- VARCHAR(255): 実装がシンプル、人間が読みやすい
- UNIQUE 制約: 重複を防止
- 4 つのインデックス: 各 API 用に最適化

インデックス戦略:

| インデックス                   | 用途                 | カラム順序の理由                                                |
| ------------------------------ | -------------------- | --------------------------------------------------------------- |
| `idx_relations_reverse`        | Check API            | `(entity_type, entity_id, relation)` - 最も頻繁なクエリパターン |
| `idx_relations_forward`        | LookupEntity         | `(subject_type, subject_id, relation)` - subject 起点の検索     |
| `idx_relations_lookup_entity`  | LookupEntity 最適化  | 全カラム含む - Index-Only Scan                                  |
| `idx_relations_lookup_subject` | LookupSubject 最適化 | 全カラム含む - Index-Only Scan                                  |

1 行のサイズ概算:

- BIGSERIAL: 8 bytes
- VARCHAR(255) × 5: 平均 50 bytes × 5 = 250 bytes
- TIMESTAMP: 8 bytes
- 合計: 約 266 bytes/行
- 1 億行で約 25GB（許容範囲）

##### B.2.3 attributes テーブル

```sql
CREATE TABLE attributes (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR(255) NOT NULL,
    entity_id VARCHAR(255) NOT NULL,
    attribute_key VARCHAR(255) NOT NULL,

    -- 型別カラム（パフォーマンス最適化）
    value_type SMALLINT NOT NULL,  -- 1=string, 2=integer, 3=boolean, 4=float, 5=string_array
    string_value TEXT,
    int_value BIGINT,
    bool_value BOOLEAN,
    float_value DOUBLE PRECISION,
    array_value TEXT[],

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(entity_type, entity_id, attribute_key),

    -- 型に応じて適切な値カラムが設定されていることを保証
    CHECK (
        (value_type = 1 AND string_value IS NOT NULL) OR
        (value_type = 2 AND int_value IS NOT NULL) OR
        (value_type = 3 AND bool_value IS NOT NULL) OR
        (value_type = 4 AND float_value IS NOT NULL) OR
        (value_type = 5 AND array_value IS NOT NULL)
    )
);

-- インデックス: エンティティによる検索
CREATE INDEX idx_attributes_entity
    ON attributes(entity_type, entity_id);

-- 部分インデックス: 型別の高速検索
CREATE INDEX idx_attributes_string
    ON attributes(entity_type, entity_id, attribute_key, string_value)
    WHERE value_type = 1;

CREATE INDEX idx_attributes_int
    ON attributes(entity_type, entity_id, attribute_key, int_value)
    WHERE value_type = 2;

-- 更新時刻自動更新トリガー
CREATE OR REPLACE FUNCTION update_attributes_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_attributes_updated_at
    BEFORE UPDATE ON attributes
    FOR EACH ROW
    EXECUTE FUNCTION update_attributes_updated_at();
```

設計理由:

- 型別カラム採用: ABAC はアクセス頻度が高いため、JSONB/BYTEA より高速
- `value_type`: どのカラムを使うか識別
- 部分インデックス: 型ごとに最適化
- CHECK 制約: データ整合性保証

型の定義:

| value_type | 型           | 使用カラム   | 用途           |
| ---------- | ------------ | ------------ | -------------- |
| 1          | string       | string_value | 文字列属性     |
| 2          | integer      | int_value    | 整数属性       |
| 3          | boolean      | bool_value   | 真偽値属性     |
| 4          | float        | float_value  | 浮動小数点属性 |
| 5          | string_array | array_value  | 文字列配列     |

##### B.2.4 audit_logs テーブル（オプション）

```sql
CREATE TABLE audit_logs (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(255) NOT NULL,
    actor_id VARCHAR(255),
    actor_type VARCHAR(255),
    resource_type VARCHAR(255),
    resource_id VARCHAR(255),
    action VARCHAR(255) NOT NULL,
    details JSONB,
    timestamp TIMESTAMP NOT NULL DEFAULT NOW()
);

-- インデックス
CREATE INDEX idx_audit_logs_timestamp
    ON audit_logs(timestamp DESC);

CREATE INDEX idx_audit_logs_event_type
    ON audit_logs(event_type, timestamp DESC);
```

設計理由:

- コンプライアンス用
- 高トラフィック時はサンプリング推奨
- 時系列パーティショニング検討（月次など）

#### B.3 ヘルパー関数

```sql
-- 関数: Relationの存在チェック
CREATE OR REPLACE FUNCTION relation_exists(
    p_subject_type VARCHAR(255),
    p_subject_id VARCHAR(255),
    p_relation VARCHAR(255),
    p_entity_type VARCHAR(255),
    p_entity_id VARCHAR(255)
) RETURNS BOOLEAN AS $$
BEGIN
    RETURN EXISTS (
        SELECT 1 FROM relations
        WHERE subject_type = p_subject_type
          AND subject_id = p_subject_id
          AND relation = p_relation
          AND entity_type = p_entity_type
          AND entity_id = p_entity_id
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- 関数: エンティティの属性を取得
CREATE OR REPLACE FUNCTION get_entity_attribute(
    p_entity_type VARCHAR(255),
    p_entity_id VARCHAR(255),
    p_attribute_key VARCHAR(255)
) RETURNS TABLE(
    value_type SMALLINT,
    string_value TEXT,
    int_value BIGINT,
    bool_value BOOLEAN,
    float_value DOUBLE PRECISION,
    array_value TEXT[]
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        a.value_type,
        a.string_value,
        a.int_value,
        a.bool_value,
        a.float_value,
        a.array_value
    FROM attributes a
    WHERE a.entity_type = p_entity_type
      AND a.entity_id = p_entity_id
      AND a.attribute_key = p_attribute_key;
END;
$$ LANGUAGE plpgsql STABLE;
```

#### B.4 将来の最適化オプション

測定結果に基づいて、以下の最適化を段階的に実施可能：

##### オプション 1: ハッシュカラム追加（非破壊的、ダウンタイムなし）

```sql
-- Step 1: カラム追加
ALTER TABLE relations
ADD COLUMN subject_type_hash BIGINT,
ADD COLUMN subject_id_hash BIGINT,
ADD COLUMN relation_hash BIGINT,
ADD COLUMN entity_type_hash BIGINT,
ADD COLUMN entity_id_hash BIGINT;

-- Step 2: 既存データのハッシュ計算（バックグラウンド実行）
UPDATE relations
SET subject_type_hash = ('x' || md5(subject_type))::bit(64)::bigint,
    subject_id_hash = ('x' || md5(subject_id))::bit(64)::bigint,
    relation_hash = ('x' || md5(relation))::bit(64)::bigint,
    entity_type_hash = ('x' || md5(entity_type))::bit(64)::bigint,
    entity_id_hash = ('x' || md5(entity_id))::bit(64)::bigint;

-- Step 3: インデックス追加
CREATE INDEX idx_relations_hash
    ON relations(entity_type_hash, entity_id_hash, relation_hash);

-- Step 4: アプリケーション側でハッシュ使用に切り替え
```

効果: クエリ速度 2-3 倍、インデックスサイズ 60%削減

##### オプション 2: パーティショニング（大規模時）

```sql
-- entity_type別にパーティション
CREATE TABLE relations_partitioned (
    LIKE relations INCLUDING ALL
) PARTITION BY LIST (entity_type);

CREATE TABLE relations_document PARTITION OF relations_partitioned
    FOR VALUES IN ('document');

CREATE TABLE relations_folder PARTITION OF relations_partitioned
    FOR VALUES IN ('folder');
```

効果: クエリ速度向上（関連パーティションのみスキャン）

##### オプション 3: 正規化（計画的なダウンタイムが必要）

詳細は省略（必要になった時点で検討）

#### B.5 PostgreSQL 設定推奨値

```ini
# postgresql.conf

# メモリ設定（32GBメモリの場合）
shared_buffers = 8GB              # RAM の 25%
effective_cache_size = 24GB       # RAM の 75%
work_mem = 16MB
maintenance_work_mem = 1GB

# 並列処理（16コアCPUの場合）
max_worker_processes = 16
max_parallel_workers_per_gather = 4
max_parallel_workers = 16

# WAL設定
wal_buffers = 16MB
min_wal_size = 1GB
max_wal_size = 4GB
checkpoint_completion_target = 0.9

# SSD最適化
random_page_cost = 1.1
effective_io_concurrency = 200

# 統計情報
default_statistics_target = 100
```

### C. gRPC API の完全な定義

#### C.1 Protocol Buffers 定義

Protocol Buffers 定義は 3 ファイルに分割されています。

##### C.1.1 common.proto

```protobuf
syntax = "proto3";

package keruberosu.v1;

option go_package = "github.com/asakaida/keruberosu/gen/proto/keruberosu/v1;keruberosupb";

// ========================================
// 共通メッセージ型（全サービスで共有）
// ========================================

message Entity {
  string type = 1;  // e.g., "document"
  string id = 2;    // e.g., "doc1"
}

message Subject {
  string type = 1;     // e.g., "user"
  string id = 2;       // e.g., "alice"
  string relation = 3; // optional: e.g., "member" for "@organization#member"
}

message SubjectReference {
  string type = 1;     // e.g., "user"
  string relation = 2; // optional: 特定のrelationに限定する場合
}

message RelationTuple {
  Entity entity = 1;   // 対象エンティティ（Permify互換）
  string relation = 2;
  Entity subject = 3;
}

message PermissionCheckMetadata {
  string snap_token = 1;    // スナップショットトークン（optional）
  int32 depth = 2;          // 再帰クエリの深さ制限（default: 50）
  bool only_permission = 3; // SubjectPermission用: permissionのみ返す
}

message Context {
  repeated RelationTuple tuples = 1;      // contextual tuples
  repeated AttributeData attributes = 2;  // contextual attributes
}

message AttributeData {
  Entity entity = 1;
  map<string, google.protobuf.Value> data = 2;
}

enum CheckResult {
  CHECK_RESULT_UNSPECIFIED = 0;
  CHECK_RESULT_ALLOWED = 1;
  CHECK_RESULT_DENIED = 2;
}
```

##### C.1.2 authorization.proto

```protobuf
syntax = "proto3";

package keruberosu.v1;

import "google/protobuf/struct.proto";
import "keruberosu/v1/common.proto";

option go_package = "github.com/asakaida/keruberosu/gen/proto/keruberosu/v1;keruberosupb";

// ========================================
// AuthorizationService
// ========================================

service AuthorizationService {
  // === スキーマ管理 ===
  rpc WriteSchema(WriteSchemaRequest) returns (WriteSchemaResponse);
  rpc ReadSchema(ReadSchemaRequest) returns (ReadSchemaResponse);

  // === データ書き込み ===
  rpc WriteRelations(WriteRelationsRequest) returns (WriteRelationsResponse);
  rpc DeleteRelations(DeleteRelationsRequest) returns (DeleteRelationsResponse);
  rpc WriteAttributes(WriteAttributesRequest) returns (WriteAttributesResponse);

  // === 認可チェック ===
  rpc Check(CheckRequest) returns (CheckResponse);
  rpc Expand(ExpandRequest) returns (ExpandResponse);
  rpc LookupEntity(LookupEntityRequest) returns (LookupEntityResponse);
  rpc LookupSubject(LookupSubjectRequest) returns (LookupSubjectResponse);
  rpc LookupEntityStream(LookupEntityRequest) returns (stream LookupEntityStreamResponse);
  rpc SubjectPermission(SubjectPermissionRequest) returns (SubjectPermissionResponse);
}

// ========================================
// スキーマ管理
// ========================================

message WriteSchemaRequest {
  string schema_dsl = 1;
}

message WriteSchemaResponse {
  bool success = 1;
  string message = 2;
  repeated string errors = 3;  // パースエラー詳細
}

message ReadSchemaRequest {
  // パラメータなし（常に現在のスキーマを返す）
}

message ReadSchemaResponse {
  string schema_dsl = 1;
  string updated_at = 2;  // ISO8601形式のタイムスタンプ
}

// ========================================
// データ書き込み
// ========================================

message WriteRelationsRequest {
  repeated RelationTuple tuples = 1;
}

message WriteRelationsResponse {
  int32 written_count = 1;
}

message DeleteRelationsRequest {
  repeated RelationTuple tuples = 1;
}

message DeleteRelationsResponse {
  int32 deleted_count = 1;
}

message WriteAttributesRequest {
  repeated AttributeData attributes = 1;
}

message WriteAttributesResponse {
  int32 written_count = 1;
}

// ========================================
// 認可チェック
// ========================================

message CheckRequest {
  PermissionCheckMetadata metadata = 1;  // snap_token, depth
  Entity entity = 2;                      // 対象リソース
  string permission = 3;                  // 確認するパーミッション
  Subject subject = 4;                    // 主体（type, id, relation）
  Context context = 5;                    // contextual tuples & attributes
  repeated google.protobuf.Value arguments = 6;  // optional: 計算用引数
}

message CheckResponse {
  CheckResult can = 1;                    // ALLOWED or DENIED
  CheckResponseMetadata metadata = 2;     // check_count など
}

message CheckResponseMetadata {
  int32 check_count = 1;  // 実行されたチェック数
}

message ExpandRequest {
  PermissionCheckMetadata metadata = 1;  // snap_token, depth
  Entity entity = 2;                      // 対象エンティティ
  string permission = 3;                  // 展開するパーミッション
  Context context = 4;                    // contextual tuples & attributes
  repeated google.protobuf.Value arguments = 5;  // optional: 計算用引数
}

message ExpandResponse {
  ExpandNode tree = 1;  // パーミッションツリー
}

message ExpandNode {
  string operation = 1;  // "union", "intersection", "exclusion", "leaf"
  repeated ExpandNode children = 2;
  Entity entity = 3;     // leaf nodeの場合のエンティティ
  Subject subject = 4;   // leaf nodeの場合のsubject
}

message LookupEntityRequest {
  PermissionCheckMetadata metadata = 1;  // snap_token, depth
  string entity_type = 2;                // 検索対象のentity type (e.g., "document")
  string permission = 3;                 // 権限名 (e.g., "edit")
  Subject subject = 4;                   // 主体 (type, id, relation)
  Context context = 5;                   // contextual tuples & attributes

  // ページネーション
  int32 page_size = 6;                   // 1ページあたりの結果数（1-100）
  string continuous_token = 7;           // 次ページ取得用トークン
}

message LookupEntityResponse {
  repeated string entity_ids = 1;        // 許可されたentityのIDリスト
  string continuous_token = 2;           // 次ページがある場合のトークン
}

message LookupEntityStreamResponse {
  string entity_id = 1;                  // 1件ずつストリーム
  string continuous_token = 2;           // 次ページ取得用トークン
}

message LookupSubjectRequest {
  PermissionCheckMetadata metadata = 1;  // snap_token, depth
  Entity entity = 2;                     // 対象entity (type, id)
  string permission = 3;                 // 権限名 (e.g., "edit")
  SubjectReference subject_reference = 4; // 検索対象のsubject (type, relation)
  Context context = 5;                   // contextual tuples & attributes

  // ページネーション
  int32 page_size = 6;                   // 1ページあたりの結果数（1-100）
  string continuous_token = 7;           // 次ページ取得用トークン
}

message LookupSubjectResponse {
  repeated string subject_ids = 1;       // 許可されたsubjectのIDリスト
  string continuous_token = 2;           // 次ページがある場合のトークン
}

message SubjectPermissionRequest {
  PermissionCheckMetadata metadata = 1;  // snap_token, depth, only_permission
  Entity entity = 2;                     // 対象entity (type, id)
  Subject subject = 3;                   // 主体 (type, id, relation)
  Context context = 4;                   // contextual tuples & attributes
}

message SubjectPermissionResponse {
  map<string, CheckResult> results = 1;  // permission名 -> ALLOWED/DENIED
}
```

##### C.1.3 audit.proto

```protobuf
syntax = "proto3";

package keruberosu.v1;

import "google/protobuf/struct.proto";

option go_package = "github.com/asakaida/keruberosu/gen/proto/keruberosu/v1;keruberosupb";

// ========================================
// AuditService
// ========================================

service AuditService {
  rpc WriteAuditLog(WriteAuditLogRequest) returns (WriteAuditLogResponse);
  rpc ReadAuditLogs(ReadAuditLogsRequest) returns (ReadAuditLogsResponse);
}

// ========================================
// 監査ログ
// ========================================

message WriteAuditLogRequest {
  string event_type = 1;
  string actor_id = 2;
  string actor_type = 3;
  string resource_type = 4;
  string resource_id = 5;
  string action = 6;
  google.protobuf.Struct details = 7;
}

message WriteAuditLogResponse {
  bool success = 1;
}

message ReadAuditLogsRequest {
  string event_type = 1;     // フィルタ（オプション）
  string actor_id = 2;        // フィルタ（オプション）
  string start_time = 3;      // ISO8601形式
  string end_time = 4;        // ISO8601形式
  int32 limit = 5;            // デフォルト: 100
  string cursor = 6;          // ページネーション用
}

message AuditLog {
  string id = 1;
  string event_type = 2;
  string actor_id = 3;
  string actor_type = 4;
  string resource_type = 5;
  string resource_id = 6;
  string action = 7;
  google.protobuf.Struct details = 8;
  string timestamp = 9;
}

message ReadAuditLogsResponse {
  repeated AuditLog logs = 1;
  string next_cursor = 2;
  int32 total_count = 3;
}
```

##### クライアントコード生成

```bash
protoc \
  --go_out=gen/go \
  --go-grpc_out=gen/go \
  proto/keruberosu/v1/*.proto
```

生成されたコードは単一パッケージとして利用可能：

```go
import pb "github.com/asakaida/keruberosu/gen/proto/keruberosu/v1"

client := pb.NewAuthorizationServiceClient(conn)
auditClient := pb.NewAuditServiceClient(conn)
```

#### C.2 gRPC 実装パターン

```go
package server

import (
    "context"
    pb "github.com/asakaida/keruberosu/gen/proto"
)

type Server struct {
    pb.UnimplementedAuthorizationServiceServer

    schemaCache       *SchemaCache
    authCache         *AuthorizationCache
    dataManager       *DataManager
    permissionChecker *PermissionChecker
}

func NewServer(db *sql.DB, redisAddr string) (*Server, error) {
    authCache := NewAuthorizationCache(10000, time.Minute, redisAddr, 5*time.Minute)

    return &Server{
        schemaCache:       &SchemaCache{},
        authCache:         authCache,
        dataManager:       NewDataManager(db),
        permissionChecker: NewPermissionChecker(),
    }, nil
}

func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
    // バリデーション
    if req.Entity == nil || req.Subject == nil {
        return nil, status.Error(codes.InvalidArgument, "entity and subject are required")
    }

    // キャッシュキー生成
    cacheKey := CacheKey{
        SubjectType: req.Subject.Type,
        SubjectID:   req.Subject.Id,
        Permission:  req.Permission,
        ObjectType:  req.Entity.Type,
        ObjectID:    req.Entity.Id,
        ContextHash: hashContext(req.Context),
    }

    // L1/L2キャッシュチェック
    if allowed, ok := s.authCache.Get(ctx, cacheKey); ok {
        return &pb.CheckResponse{
            Can: boolToCheckResult(allowed),
            Metadata: &pb.CheckResponseMetadata{
                Cached: true,
            },
        }, nil
    }

    // スキーマ取得
    schema := s.schemaCache.Get()

    // 認可チェック実行
    allowed, checkCount, err := s.permissionChecker.Check(ctx, schema, req)
    if err != nil {
        return nil, status.Errorf(codes.Internal, "check failed: %v", err)
    }

    // キャッシュに保存
    s.authCache.Set(ctx, cacheKey, allowed)

    return &pb.CheckResponse{
        Can: boolToCheckResult(allowed),
        Metadata: &pb.CheckResponseMetadata{
            CheckCount: int32(checkCount),
            Cached:     false,
        },
    }, nil
}

func boolToCheckResult(allowed bool) pb.CheckResult {
    if allowed {
        return pb.CheckResult_CHECK_RESULT_ALLOWED
    }
    return pb.CheckResult_CHECK_RESULT_DENIED
}
```

#### C.3 エラーハンドリング

gRPC ステータスコードの使い分け：

| エラー種別             | gRPC ステータス    | 例                           |
| ---------------------- | ------------------ | ---------------------------- |
| 必須パラメータ不足     | InvalidArgument    | entity または subject が nil |
| スキーマ未定義         | FailedPrecondition | スキーマが書き込まれていない |
| パースエラー           | InvalidArgument    | DSL の構文エラー             |
| 存在しないエンティティ | NotFound           | 未定義の entity type を参照  |
| DB 接続エラー          | Unavailable        | PostgreSQL 接続失敗          |
| 内部エラー             | Internal           | 予期しないエラー             |
| タイムアウト           | DeadlineExceeded   | 深いグラフ探索でタイムアウト |

```go
// エラーハンドリングの例
func (s *Server) WriteSchema(ctx context.Context, req *pb.WriteSchemaRequest) (*pb.WriteSchemaResponse, error) {
    if req.SchemaDsl == "" {
        return nil, status.Error(codes.InvalidArgument, "schema_dsl is required")
    }

    // DSLパース
    schema, err := ParsePermifyDSL(req.SchemaDsl)
    if err != nil {
        return &pb.WriteSchemaResponse{
            Success: false,
            Message: "Parse error",
            Errors:  []string{err.Error()},
        }, nil  // ビジネスエラーなのでnilを返す
    }

    // DB保存
    if err := s.dataManager.SaveSchema(ctx, req.SchemaDsl, schema); err != nil {
        return nil, status.Errorf(codes.Internal, "failed to save schema: %v", err)
    }

    // キャッシュ更新
    s.schemaCache.Set(schema)
    s.authCache.Purge()

    return &pb.WriteSchemaResponse{
        Success: true,
        Message: "Schema updated successfully",
    }, nil
}
```

## 次のステップ

詳細設計（A. DSL パーサー、B. PostgreSQL スキーマ、C. gRPC API）が完成しました。

次に進むべき実装：

1. `schema.sql` ファイルの作成
2. `keruberosu.proto` ファイルの作成
3. DSL パーサーの実装（`pkg/parser/`）
4. gRPC サーバーの実装（`pkg/server/`）
5. データマネージャーの実装（`pkg/datamanager/`）
6. グラフ探索エンジンの実装（`pkg/checker/`）

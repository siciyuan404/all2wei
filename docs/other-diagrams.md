# All2Wei 其他图表文档

## 1. 数据流图 (DFD)

### 1.1 上下文数据流图 (Level 0)

```mermaid
graph LR
    User["用户"]
    System["All2Wei 系统"]
    MinIO["MinIO 存储"]
    SQLite[("SQLite 数据库")]
    FS[("本地文件系统")]

    User -->|"登录/上传/播放/管理"| System
    System -->|"JWT Token"| User
    System -->|"读写视频/字幕文件"| MinIO
    System -->|"读写视频/字幕文件"| FS
    System -->|"CRUD 操作"| SQLite
```

### 1.2 一级数据流图 (Level 1)

```mermaid
graph TB
    User["用户"]

    subgraph All2Wei["All2Wei 系统"]
        Auth["1.0 认证处理"]
        Upload["2.0 上传处理"]
        Stream["3.0 视频流处理"]
        Manage["4.0 资料管理"]
        Sync["5.0 同步处理"]
    end

    SQLite[("SQLite")]
    FS[("本地存储")]
    MinIO[("MinIO")]

    User -->|"登录请求"| Auth
    Auth -->|"查询/存储用户"| SQLite
    Auth -->|"Token"| User

    User -->|"上传视频+字幕"| Upload
    Upload -->|"存储元数据"| SQLite
    Upload -->|"保存文件"| FS
    Upload -->|"上传对象"| MinIO

    User -->|"播放请求"| Stream
    Stream -->|"查询资料"| SQLite
    Stream -->|"读取文件"| FS
    Stream -->|"获取对象"| MinIO
    Stream -->|"视频流"| User

    User -->|"列表/删除/搜索"| Manage
    Manage -->|"CRUD"| SQLite
    Manage -->|"删除文件"| FS
    Manage -->|"删除对象"| MinIO

    User -->|"同步请求"| Sync
    Sync -->|"列出对象"| MinIO
    Sync -->|"导入记录"| SQLite
```

## 2. 实体关系图 (ER Diagram)

```mermaid
erDiagram
    USER ||--o{ MATERIAL : owns
    
    USER {
        uint id PK "主键"
        string username UK "用户名"
        string password "密码哈希"
        datetime created_at "创建时间"
        datetime updated_at "更新时间"
        datetime deleted_at "删除时间"
    }
    
    MATERIAL {
        uint id PK "主键"
        uint user_id FK "用户ID"
        string title "标题"
        string description "描述"
        string video_key "视频文件键"
        string subtitle_key "字幕文件键"
        int duration "时长(秒)"
        string status "状态: active/deleted"
        datetime created_at "创建时间"
        datetime updated_at "更新时间"
        datetime deleted_at "删除时间"
    }
```

## 3. 部署图

```mermaid
graph TB
    subgraph Client["客户端"]
        Browser["浏览器"]
    end

    subgraph Server["服务器"]
        subgraph AppContainer["应用容器"]
            GoApp["Go 后端服务\n:8189"]
            WebApp["React 前端\n/dist"]
        end
        
        subgraph DataStorage["数据存储"]
            SQLiteDB[("SQLite\nall2wei.db")]
            Uploads[("本地文件\nuploads/")]
        end
        
        subgraph External["外部服务"]
            MinIO["MinIO 对象存储"]
            VideoDir[("视频源文件夹")]
        end
    end

    Browser -->|"HTTP 请求"| GoApp
    Browser -->|"加载静态资源"| WebApp
    GoApp -->|"SQL"| SQLiteDB
    GoApp -->|"文件 IO"| Uploads
    GoApp -->|"S3 API"| MinIO
    GoApp -->|"文件扫描"| VideoDir
```

## 4. 状态图

### 4.1 视频资料状态图

```mermaid
stateDiagram-v2
    [*] --> 导入中: 上传/同步/扫描
    导入中 --> 可用: 处理完成
    可用 --> 播放中: 用户点击播放
    播放中 --> 暂停: 用户暂停
    暂停 --> 播放中: 用户继续
    播放中 --> 已结束: 视频播放完毕
    已结束 --> 播放中: 用户重播
    可用 --> 已删除: 用户删除
    已删除 --> [*]
```

### 4.2 用户认证状态图

```mermaid
stateDiagram-v2
    [*] --> 未认证: 打开应用
    未认证 --> 认证中: 输入凭据
    认证中 --> 已认证: 验证成功
    认证中 --> 未认证: 验证失败
    已认证 --> 未认证: Token 过期/登出
    已认证 --> 已认证: 刷新 Token
```

## 5. 用例图

```mermaid
graph TB
    actor User as 普通用户
    
    subgraph All2Wei["All2Wei 系统"]
        UC1["登录系统"]
        UC2["上传视频"]
        UC3["播放视频"]
        UC4["查看字幕"]
        UC5["搜索字幕"]
        UC6["管理资料"]
        UC7["同步 MinIO"]
        UC8["扫描视频源"]
    end
    
    User --> UC1
    User --> UC2
    User --> UC3
    User --> UC4
    User --> UC5
    User --> UC6
    User --> UC7
    User --> UC8
    
    UC2 -.->|"包含"| UC4
    UC3 -.->|"包含"| UC4
    UC3 -.->|"扩展"| UC5
    UC6 -.->|"包含"| UC2
    UC6 -.->|"包含"| UC7
    UC6 -.->|"包含"| UC8
```

## 6. 组件图

```mermaid
graph TB
    subgraph Frontend["前端组件"]
        App["App.jsx"]
        Router["BrowserRouter"]
        
        subgraph Pages["页面组件"]
            Login["Login"]
            MaterialList["MaterialList"]
            Upload["Upload"]
            Watch["Watch"]
        end
        
        subgraph Common["通用组件"]
            Button["Button"]
            Input["Input"]
            Loading["Loading"]
            Toast["Toast"]
            ErrorMsg["ErrorMessage"]
        end
        
        subgraph Layout["布局组件"]
            Header["Header"]
            Container["Container"]
            PageLayout["PageLayout"]
        end
        
        subgraph MaterialComp["资料组件"]
            MaterialCard["MaterialCard"]
            MaterialGrid["MaterialGrid"]
        end
        
        subgraph Context["上下文"]
            AuthCtx["AuthContext"]
            ToastCtx["ToastContext"]
        end
    end
    
    App --> Router
    Router --> Login
    Router --> MaterialList
    Router --> Upload
    Router --> Watch
    
    Login --> Input
    Login --> Button
    Login --> AuthCtx
    
    MaterialList --> MaterialGrid
    MaterialList --> AuthCtx
    MaterialGrid --> MaterialCard
    
    Upload --> Input
    Upload --> Button
    Upload --> ToastCtx
    
    Watch --> Button
    Watch --> ToastCtx
    
    PageLayout --> Header
    PageLayout --> Container
```

## 7. 活动图 - 视频上传流程

```mermaid
flowchart TD
    Start([开始]) --> SelectFile["用户选择视频文件"]
    SelectFile --> FillInfo["填写标题和描述"]
    FillInfo --> Submit["点击上传按钮"]
    Submit --> Validate["验证文件格式"]
    Validate -->|"格式无效"| Error["显示错误信息"]
    Error --> SelectFile
    Validate -->|"格式有效"| SaveTemp["保存到临时文件"]
    SaveTemp --> UploadStorage["上传到存储服务"]
    UploadStorage --> CheckSubtitle{"有字幕文件?"}
    CheckSubtitle -->|"是"| UploadSubtitle["上传字幕文件"]
    CheckSubtitle -->|"否"| CreateRecord["创建数据库记录"]
    UploadSubtitle --> CreateRecord
    CreateRecord --> Success["返回成功响应"]
    Success --> Refresh["刷新视频列表"]
    Refresh --> End([结束])
```

## 8. 活动图 - 视频播放流程

```mermaid
flowchart TD
    Start([开始]) --> ClickVideo["用户点击视频"]
    ClickVideo --> AuthCheck{"已登录?"}
    AuthCheck -->|"否"| Login["跳转到登录页"]
    Login --> End1([结束])
    AuthCheck -->|"是"| LoadMeta["加载视频元数据"]
    LoadMeta --> CheckFormat{"需要转码?"}
    CheckFormat -->|"是"| Transcode["FFmpeg 实时转码"]
    CheckFormat -->|"否"| DirectStream["直接流式传输"]
    Transcode --> InitPlayer["初始化 Video.js"]
    DirectStream --> InitPlayer
    InitPlayer --> LoadSubtitle["加载字幕数据"]
    LoadSubtitle --> Play["开始播放"]
    Play --> UserAction{"用户操作"}
    UserAction -->|"暂停/播放"| TogglePlay["切换播放状态"]
    UserAction -->|"快进/后退"| Seek["调整播放位置"]
    UserAction -->|"点击字幕"| JumpTime["跳转到对应时间"]
    UserAction -->|"搜索字幕"| SearchSub["过滤字幕列表"]
    UserAction -->|"退出"| SaveProgress["保存播放进度"]
    TogglePlay --> UserAction
    Seek --> UserAction
    JumpTime --> UserAction
    SearchSub --> UserAction
    SaveProgress --> End2([结束])
```

## 9. 类图

### 9.1 后端核心类图

```mermaid
classDiagram
    class Config {
        +ServerConfig Server
        +DatabaseConfig Database
        +StorageConfig Storage
        +VideoSourceConfig VideoSource
        +MinIOConfig MinIO
        +JWTConfig JWT
        +Load() *Config
    }
    
    class UserHandler {
        -userRepo *UserRepository
        -jwtCfg *JWTConfig
        +Register(c *gin.Context)
        +Login(c *gin.Context)
        +GetProfile(c *gin.Context)
    }
    
    class MaterialHandler {
        -materialRepo *MaterialRepository
        -storageSvc StorageService
        -minioSvc *MinIOService
        -jwtCfg *JWTConfig
        -sourceCfg *VideoSourceConfig
        +Upload(c)
        +List(c)
        +Get(c)
        +Delete(c)
        +StreamVideo(c)
        +GetSubtitle(c)
        +Sync(c)
        +ScanSource(c)
        +ScanSourceFolder(userID)
    }
    
    class StorageService {
        <<interface>>
        +UploadFile(ctx, key, filePath, contentType)
        +UploadBytes(ctx, key, data, contentType)
        +GetPresignedURL(ctx, key, expiry)
        +DeleteObject(ctx, key)
        +GetLocalPath(key)
    }
    
    class LocalStorage {
        -baseDir string
        -baseURL string
        +objectPath(key) string
    }
    
    class MinIOService {
        -client *minio.Client
        -bucket string
        +ListObjects(ctx, prefix)
        +GetBucketName() string
    }
    
    class UserRepository {
        -db *gorm.DB
        +Create(user)
        +GetByID(id)
        +GetByUsername(username)
        +Exists(username)
    }
    
    class MaterialRepository {
        -db *gorm.DB
        +Create(material)
        +GetByID(id)
        +GetByUserID(userID)
        +Update(material)
        +Delete(id)
    }
    
    class User {
        +ID uint
        +Username string
        +Password string
        +CreatedAt time.Time
        +UpdatedAt time.Time
    }
    
    class Material {
        +ID uint
        +UserID uint
        +Title string
        +Description string
        +VideoKey string
        +SubtitleKey string
        +Duration int
        +Status string
        +CreatedAt time.Time
    }
    
    class SubtitleEntry {
        +Index int
        +StartTime float64
        +EndTime float64
        +Text string
    }
    
    UserHandler --> UserRepository
    MaterialHandler --> MaterialRepository
    MaterialHandler --> StorageService
    StorageService <|.. LocalStorage
    StorageService <|.. MinIOService
    UserRepository --> User
    MaterialRepository --> Material
```

### 9.2 前端核心类图

```mermaid
classDiagram
    class App {
        +render()
    }
    
    class AuthProvider {
        -user User
        -loading boolean
        -isAuthenticated boolean
        +login(token)
        +logout()
        +getProfile()
    }
    
    class ToastProvider {
        -toasts Toast[]
        +show(message, type)
        +remove(id)
    }
    
    class Watch {
        -playerRef
        -subtitles Subtitle[]
        -currentSubtitleIndex number
        -material Material
        +handleSubtitleClick(time)
        +handleSearch(query)
        +handleRetry()
    }
    
    class MaterialList {
        -materials Material[]
        -loading boolean
        +fetchMaterials()
        +handleDelete(id)
        +handleSync()
    }
    
    class api {
        +axiosInstance
        +interceptors
    }
    
    App --> AuthProvider
    App --> ToastProvider
    Watch --> api
    MaterialList --> api
    AuthProvider --> api
```


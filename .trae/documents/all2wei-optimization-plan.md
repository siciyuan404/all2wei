# All2Wei 项目优化计划

## 项目概述

All2Wei 是一个视频学习资料管理系统，采用以下技术栈：
- **后端**: Go 1.25 + Gin + GORM + SQLite + MinIO
- **前端**: React 19 + Vite 8 + Video.js + React Router 7

## 一、安全性优化 (高优先级)

### 1.1 移除硬编码默认密码
**文件**: [cmd/server/main.go:132-155](file:///e:/github/all2wei/cmd/server/main.go#L132-L155)

**问题**: 默认用户密码硬编码为 `all2wei/all2wei`，存在安全隐患。

**优化方案**:
- 从配置文件或环境变量读取默认密码
- 首次启动时生成随机密码并输出到日志
- 或要求用户首次启动时设置密码

### 1.2 增强 JWT Secret 安全性
**文件**: [internal/config/config.go:54](file:///e:/github/all2wei/internal/config/config.go#L54)

**问题**: 默认 JWT Secret 为 `your-secret-key`，过于简单。

**优化方案**:
- 启动时检查是否使用默认值，若是则生成随机值或报错
- 要求生产环境必须配置强密钥
- 添加密钥长度验证

### 1.3 收紧 CORS 配置
**文件**: [internal/middleware/cors.go:9](file:///e:/github/all2wei/internal/middleware/cors.go#L9)

**问题**: `Access-Control-Allow-Origin: *` 允许任何域名访问。

**优化方案**:
- 从配置文件读取允许的域名列表
- 支持多域名配置
- 生产环境禁止使用 `*`

### 1.4 移除 MinIO SSL 跳过验证
**文件**: [internal/service/minio.go:25-28](file:///e:/github/all2wei/internal/service/minio.go#L25-L28)

**问题**: `InsecureSkipVerify: true` 跳过 SSL 证书验证。

**优化方案**:
- 从配置读取是否跳过验证（仅开发环境使用）
- 生产环境强制验证证书
- 支持自定义 CA 证书

### 1.5 优化 Token 传递方式
**文件**: [internal/handler/material.go:253-265](file:///e:/github/all2wei/internal/handler/material.go#L253-L265)

**问题**: 视频流接口通过 URL 参数传递 token，可能被日志记录。

**优化方案**:
- 使用 Cookie 传递 token
- 或使用短期一次性 token
- 添加 Referer 检查

---

## 二、代码架构优化 (中优先级)

### 2.1 引入依赖注入
**问题**: 手动创建各种依赖，代码耦合度高。

**优化方案**:
- 使用 Wire 或 Dig 等 DI 框架
- 或实现简单的手动依赖注入容器
- 统一管理依赖生命周期

### 2.2 统一错误处理
**问题**: 错误处理分散，有些只是 `log.Printf`。

**优化方案**:
- 创建统一的错误类型
- 实现全局错误处理中间件
- 规范化 API 错误响应格式

### 2.3 添加配置验证
**文件**: [internal/config/config.go:41-69](file:///e:/github/all2wei/internal/config/config.go#L41-L69)

**问题**: 配置加载后没有验证。

**优化方案**:
- 添加配置验证函数
- 检查必填字段
- 验证字段格式和范围

### 2.4 抽象 Repository 接口
**文件**: [internal/repository/user.go](file:///e:/github/all2wei/internal/repository/user.go), [internal/repository/material.go](file:///e:/github/all2wei/internal/repository/material.go)

**问题**: Repository 直接返回具体类型，不利于测试和扩展。

**优化方案**:
- 定义 Repository 接口
- 便于单元测试时 mock
- 支持未来切换数据库

---

## 三、前端优化 (中优先级)

### 3.1 移除硬编码 API 地址
**文件**: [web/src/api/axios.js:4](file:///e:/github/all2wei/web/src/api/axios.js#L4)

**问题**: `baseURL` 硬编码为 `http://localhost:8189/api`。

**优化方案**:
- 使用 Vite 环境变量 `import.meta.env.VITE_API_URL`
- 创建 `.env.development` 和 `.env.production` 文件
- 支持运行时配置

### 3.2 添加代码分割和懒加载
**文件**: [web/src/App.jsx](file:///e:/github/all2wei/web/src/App.jsx)

**问题**: 所有页面组件同步加载，首屏加载慢。

**优化方案**:
```jsx
const MaterialList = lazy(() => import('./pages/MaterialList'));
const Upload = lazy(() => import('./pages/Upload'));
const Watch = lazy(() => import('./pages/Watch'));
```

### 3.3 添加全局状态管理
**问题**: 用户状态分散在 localStorage 中。

**优化方案**:
- 使用 React Context 管理用户状态
- 或引入 Zustand/Jotai 等轻量状态库
- 统一管理认证状态

### 3.4 优化上传体验
**文件**: [web/src/pages/Upload.jsx](file:///e:/github/all2wei/web/src/pages/Upload.jsx)

**问题**: 上传没有进度显示和大小限制。

**优化方案**:
- 添加上传进度条
- 添加文件大小/类型验证
- 支持拖拽上传
- 支持多文件上传

---

## 四、功能增强 (中优先级)

### 4.1 添加分页功能
**文件**: [internal/handler/material.go:123-162](file:///e:/github/all2wei/internal/handler/material.go#L123-L162)

**问题**: 资料列表没有分页，数据量大时性能差。

**优化方案**:
- 后端添加分页参数 `page`, `page_size`
- 前端添加分页组件
- 返回总数用于分页计算

### 4.2 添加搜索功能
**问题**: 资料列表没有搜索功能。

**优化方案**:
- 支持按标题搜索
- 支持按描述搜索
- 后端使用 LIKE 或全文搜索

### 4.3 添加文件上传限制
**文件**: [internal/handler/material.go:63-121](file:///e:/github/all2wei/internal/handler/material.go#L63-L121)

**问题**: 上传没有文件大小和类型限制。

**优化方案**:
- 添加最大文件大小配置
- 验证文件类型（MIME type）
- 返回友好的错误提示

### 4.4 添加视频缩略图
**问题**: 列表页视频预览体验不好。

**优化方案**:
- 上传时使用 FFmpeg 生成缩略图
- 存储缩略图到 MinIO/本地
- 列表页显示缩略图而非视频预览

### 4.5 启用用户注册功能
**文件**: [internal/handler/user.go:25-67](file:///e:/github/all2wei/internal/handler/user.go#L25-L67)

**问题**: 注册功能已实现但未启用路由。

**优化方案**:
- 添加配置项控制是否允许注册
- 添加邀请码机制
- 或保持仅管理员创建用户

---

## 五、性能优化 (低优先级)

### 5.1 添加缓存机制
**问题**: 没有任何缓存，重复请求相同数据。

**优化方案**:
- 使用 Redis 缓存热点数据
- 或使用内存缓存（如 go-cache）
- 缓存预签名 URL（注意过期时间）

### 5.2 优化视频代理
**文件**: [internal/handler/material.go:313-374](file:///e:/github/all2wei/internal/handler/material.go#L313-L374)

**问题**: 每次请求都代理整个视频流。

**优化方案**:
- 使用 CDN 加速
- 或直接返回预签名 URL 让前端访问 MinIO
- 添加 HTTP 缓存头

### 5.3 数据库优化
**文件**: [internal/repository/database.go](file:///e:/github/all2wei/internal/repository/database.go)

**问题**: SQLite 可能成为性能瓶颈。

**优化方案**:
- 添加必要的数据库索引
- 考虑支持 PostgreSQL/MySQL
- 添加数据库连接池配置

### 5.4 前端性能优化
**问题**: 前端没有性能优化措施。

**优化方案**:
- 添加图片懒加载
- 使用虚拟列表渲染大量字幕
- 添加 Service Worker 缓存

---

## 六、代码质量 (低优先级)

### 6.1 移除调试日志
**文件**: 多处存在 `log.Printf` 调试日志

**问题**: 生产代码中有很多调试日志。

**优化方案**:
- 使用结构化日志库（如 zap, zerolog）
- 添加日志级别配置
- 生产环境使用 Info 及以上级别

### 6.2 消除魔法数字
**问题**: 代码中有硬编码数字如 `24*time.Hour`。

**优化方案**:
- 提取为常量或配置项
- 添加注释说明含义

### 6.3 添加单元测试
**问题**: 项目没有任何测试文件。

**优化方案**:
- 为关键业务逻辑添加单元测试
- 为 API 添加集成测试
- 配置测试覆盖率报告

### 6.4 添加代码注释
**问题**: 关键逻辑缺少注释。

**优化方案**:
- 为公共函数添加文档注释
- 为复杂逻辑添加行内注释
- 生成 API 文档

---

## 七、运维支持 (低优先级)

### 7.1 添加 Docker 支持
**问题**: 没有 Dockerfile 和 docker-compose.yml。

**优化方案**:
- 创建多阶段构建的 Dockerfile
- 创建 docker-compose.yml 集成 MinIO
- 支持 Docker 环境变量配置

### 7.2 添加健康检查接口
**问题**: 没有健康检查端点。

**优化方案**:
- 添加 `/health` 端点
- 检查数据库连接
- 检查 MinIO 连接（如果配置）

### 7.3 实现优雅关闭
**文件**: [cmd/server/main.go:126-128](file:///e:/github/all2wei/cmd/server/main.go#L126-L128)

**问题**: 服务器没有优雅关闭。

**优化方案**:
- 监听系统信号（SIGINT, SIGTERM）
- 等待现有请求完成
- 关闭数据库连接

### 7.4 添加 CI/CD 配置
**问题**: 没有 CI/CD 配置。

**优化方案**:
- 添加 GitHub Actions 配置
- 自动运行测试和 lint
- 自动构建和发布 Docker 镜像

---

## 八、用户体验 (低优先级)

### 8.1 添加国际化支持
**问题**: 没有多语言支持。

**优化方案**:
- 使用 react-i18next
- 支持中文/英文切换
- 提取所有硬编码文本

### 8.2 添加主题切换
**问题**: 只有暗色主题。

**优化方案**:
- 添加亮色主题
- 使用 CSS 变量实现主题切换
- 记住用户偏好

### 8.3 优化移动端体验
**问题**: 部分功能在移动端体验不好。

**优化方案**:
- 优化触摸交互
- 调整移动端布局
- 添加手势支持

---

## 实施优先级建议

| 优先级 | 类别 | 建议立即实施 |
|--------|------|-------------|
| P0 | 安全性 | 1.1-1.5 所有安全优化 |
| P1 | 前端 | 3.1 移除硬编码 API 地址 |
| P1 | 功能 | 4.3 添加文件上传限制 |
| P2 | 架构 | 2.3 添加配置验证 |
| P2 | 运维 | 7.1 添加 Docker 支持 |
| P2 | 运维 | 7.2 添加健康检查接口 |
| P3 | 其他 | 根据实际需求选择实施 |

---

## 总结

本项目是一个功能完整的视频学习资料管理系统，但在安全性、代码质量和运维支持方面还有较大优化空间。建议优先处理安全性问题，然后逐步完善其他方面。

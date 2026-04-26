# Web 界面重构计划

## 当前问题分析

### 1. 代码组织问题
- **所有组件同步导入**: 没有代码分割，首屏加载慢
- **CSS 全部在一个文件**: `App.css` 有 900+ 行，难以维护
- **没有组件复用**: Button、Input、Card 等重复代码
- **没有统一布局组件**: 每个页面重复 header/container 结构

### 2. 状态管理问题
- **用户状态分散**: token 和 user 存在 localStorage，各组件自行读取
- **没有全局状态**: 认证状态没有统一管理
- **没有状态持久化策略**: 刷新页面需要重新检查

### 3. 配置问题
- **API 地址硬编码**: `http://localhost:8189/api` 写死在代码中
- **没有环境变量**: 无法区分开发/生产环境

### 4. 用户体验问题
- **没有统一的加载状态**: 各页面自己实现
- **没有统一的错误处理**: 错误提示样式不统一
- **没有 Toast 通知**: 使用 alert() 弹窗

---

## 重构目标

1. **模块化组件**: 拆分可复用组件
2. **代码分割**: 懒加载页面组件
3. **状态管理**: 统一管理认证状态
4. **样式模块化**: CSS 按组件拆分
5. **环境配置**: 支持多环境部署
6. **用户体验**: 统一加载/错误/通知处理

---

## 重构步骤

### 阶段一：基础设施 (优先级: 高)

#### 1.1 创建环境变量配置
**新建文件**: `web/.env.development`, `web/.env.production`

```env
# .env.development
VITE_API_URL=http://localhost:8189/api

# .env.production  
VITE_API_URL=/api
```

#### 1.2 更新 axios 配置
**修改文件**: `web/src/api/axios.js`

- 使用 `import.meta.env.VITE_API_URL` 替代硬编码地址

#### 1.3 创建统一导出
**新建文件**: `web/src/api/index.js`

- 统一导出所有 API 函数

---

### 阶段二：组件拆分 (优先级: 高)

#### 2.1 创建组件目录结构
```
web/src/
├── components/
│   ├── common/
│   │   ├── Button/
│   │   │   ├── Button.jsx
│   │   │   └── Button.css
│   │   ├── Input/
│   │   │   ├── Input.jsx
│   │   │   └── Input.css
│   │   ├── Card/
│   │   │   ├── Card.jsx
│   │   │   └── Card.css
│   │   ├── Loading/
│   │   │   ├── Loading.jsx
│   │   │   └── Loading.css
│   │   ├── ErrorMessage/
│   │   │   ├── ErrorMessage.jsx
│   │   │   └── ErrorMessage.css
│   │   └── Toast/
│   │       ├── Toast.jsx
│   │       ├── ToastContainer.jsx
│   │       └── Toast.css
│   ├── layout/
│   │   ├── Header/
│   │   │   ├── Header.jsx
│   │   │   └── Header.css
│   │   ├── Container/
│   │   │   ├── Container.jsx
│   │   │   └── Container.css
│   │   └── PageLayout/
│   │       ├── PageLayout.jsx
│   │       └── PageLayout.css
│   └── material/
│       ├── MaterialCard/
│       │   ├── MaterialCard.jsx
│       │   └── MaterialCard.css
│       └── MaterialGrid/
│           ├── MaterialGrid.jsx
│           └── MaterialGrid.css
├── hooks/
│   ├── useAuth.js
│   ├── useToast.js
│   └── useApi.js
├── context/
│   └── AuthContext.jsx
└── utils/
    └── storage.js
```

#### 2.2 创建通用组件

##### Button 组件
**新建文件**: `web/src/components/common/Button/Button.jsx`

```jsx
// 统一的按钮组件，支持 primary/secondary/danger 等变体
// 支持 loading 状态、禁用状态、图标等
```

##### Input 组件
**新建文件**: `web/src/components/common/Input/Input.jsx`

```jsx
// 统一的输入框组件
// 支持 label、error、placeholder 等
```

##### Card 组件
**新建文件**: `web/src/components/common/Card/Card.jsx`

```jsx
// 通用卡片容器
```

##### Loading 组件
**新建文件**: `web/src/components/common/Loading/Loading.jsx`

```jsx
// 统一的加载指示器
// 支持全屏加载和局部加载
```

##### ErrorMessage 组件
**新建文件**: `web/src/components/common/ErrorMessage/ErrorMessage.jsx`

```jsx
// 统一的错误提示组件
```

##### Toast 组件
**新建文件**: `web/src/components/common/Toast/`

```jsx
// Toast 通知系统
// 替代 alert() 弹窗
```

#### 2.3 创建布局组件

##### Header 组件
**新建文件**: `web/src/components/layout/Header/Header.jsx`

```jsx
// 统一的页面头部
// 包含标题、用户信息、退出按钮
```

##### Container 组件
**新建文件**: `web/src/components/layout/Container/Container.jsx`

```jsx
// 统一的页面容器
// 处理最大宽度、内边距等
```

##### PageLayout 组件
**新建文件**: `web/src/components/layout/PageLayout/PageLayout.jsx`

```jsx
// 组合 Header + Container
// 统一页面布局结构
```

#### 2.4 创建业务组件

##### MaterialCard 组件
**新建文件**: `web/src/components/material/MaterialCard/MaterialCard.jsx`

```jsx
// 从 MaterialList.jsx 中提取
// 单个资料卡片
```

##### MaterialGrid 组件
**新建文件**: `web/src/components/material/MaterialGrid/MaterialGrid.jsx`

```jsx
// 资料网格列表
// 包含空状态处理
```

---

### 阶段三：状态管理 (优先级: 高)

#### 3.1 创建 AuthContext
**新建文件**: `web/src/context/AuthContext.jsx`

```jsx
import { createContext, useContext, useState, useEffect } from 'react';

const AuthContext = createContext(null);

export function AuthProvider({ children }) {
  const [user, setUser] = useState(null);
  const [token, setToken] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    // 从 localStorage 恢复状态
    const savedToken = localStorage.getItem('token');
    const savedUser = localStorage.getItem('user');
    if (savedToken && savedUser) {
      setToken(savedToken);
      setUser(JSON.parse(savedUser));
    }
    setLoading(false);
  }, []);

  const login = (token, user) => {
    setToken(token);
    setUser(user);
    localStorage.setItem('token', token);
    localStorage.setItem('user', JSON.stringify(user));
  };

  const logout = () => {
    setToken(null);
    setUser(null);
    localStorage.removeItem('token');
    localStorage.removeItem('user');
  };

  return (
    <AuthContext.Provider value={{ user, token, login, logout, loading, isAuthenticated: !!token }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  return useContext(AuthContext);
}
```

#### 3.2 创建 useAuth Hook
**新建文件**: `web/src/hooks/useAuth.js`

```jsx
// 封装认证相关逻辑
// 提供给组件使用
```

#### 3.3 创建 useApi Hook
**新建文件**: `web/src/hooks/useApi.js`

```jsx
// 封装 API 调用逻辑
// 统一处理 loading、error 状态
```

#### 3.4 创建 Toast Context
**新建文件**: `web/src/context/ToastContext.jsx`

```jsx
// 全局 Toast 通知管理
```

---

### 阶段四：代码分割 (优先级: 中)

#### 4.1 懒加载页面组件
**修改文件**: `web/src/App.jsx`

```jsx
import { lazy, Suspense } from 'react';
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Loading from './components/common/Loading/Loading';

// 懒加载页面
const Login = lazy(() => import('./pages/Login'));
const MaterialList = lazy(() => import('./pages/MaterialList'));
const Upload = lazy(() => import('./pages/Upload'));
const Watch = lazy(() => import('./pages/Watch'));

function PrivateRoute({ children }) {
  const { isAuthenticated, loading } = useAuth();
  if (loading) return <Loading fullscreen />;
  return isAuthenticated ? children : <Navigate to="/login" replace />;
}

function PublicRoute({ children }) {
  const { isAuthenticated, loading } = useAuth();
  if (loading) return <Loading fullscreen />;
  return !isAuthenticated ? children : <Navigate to="/" replace />;
}

function App() {
  return (
    <AuthProvider>
      <BrowserRouter>
        <Suspense fallback={<Loading fullscreen />}>
          <Routes>
            <Route path="/login" element={<PublicRoute><Login /></PublicRoute>} />
            <Route path="/" element={<PrivateRoute><MaterialList /></PrivateRoute>} />
            <Route path="/upload" element={<PrivateRoute><Upload /></PrivateRoute>} />
            <Route path="/watch/:id" element={<PrivateRoute><Watch /></PrivateRoute>} />
          </Routes>
        </Suspense>
      </BrowserRouter>
    </AuthProvider>
  );
}
```

---

### 阶段五：样式重构 (优先级: 中)

#### 5.1 拆分 CSS 文件

```
web/src/styles/
├── variables.css     # CSS 变量（颜色、间距等）
├── reset.css         # 重置样式
├── typography.css    # 字体样式
├── utilities.css     # 工具类
└── index.css         # 统一导入
```

#### 5.2 组件级 CSS
每个组件目录下创建对应的 CSS 文件，使用 CSS Modules 或普通 CSS。

#### 5.3 保留全局样式
`web/src/index.css` 只保留：
- CSS 变量定义
- 重置样式
- 全局字体设置

---

### 阶段六：页面重构 (优先级: 中)

#### 6.1 重构 Login 页面
**修改文件**: `web/src/pages/Login.jsx`

- 使用 AuthContext
- 使用通用组件 (Button, Input, ErrorMessage)
- 使用 Toast 替代 alert

#### 6.2 重构 MaterialList 页面
**修改文件**: `web/src/pages/MaterialList.jsx`

- 使用 AuthContext
- 使用 PageLayout 布局
- 使用 MaterialGrid 组件
- 使用 Toast 替代 alert

#### 6.3 重构 Upload 页面
**修改文件**: `web/src/pages/Upload.jsx`

- 使用 PageLayout 布局
- 使用通用组件
- 添加上传进度显示
- 使用 Toast 替代 alert

#### 6.4 重构 Watch 页面
**修改文件**: `web/src/pages/Watch.jsx`

- 拆分字幕面板为独立组件
- 使用 Toast 替代 console.log
- 优化代码结构

---

### 阶段七：优化增强 (优先级: 低)

#### 7.1 添加搜索功能
- 在 MaterialList 页面添加搜索框
- 支持按标题搜索

#### 7.2 添加分页功能
- 后端添加分页 API
- 前端添加分页组件

#### 7.3 添加主题切换
- 支持亮色/暗色主题
- 记住用户偏好

#### 7.4 添加响应式优化
- 优化移动端体验
- 添加手势支持

---

## 文件变更清单

### 新建文件
| 文件路径 | 说明 |
|---------|------|
| `web/.env.development` | 开发环境变量 |
| `web/.env.production` | 生产环境变量 |
| `web/src/context/AuthContext.jsx` | 认证上下文 |
| `web/src/context/ToastContext.jsx` | Toast 上下文 |
| `web/src/hooks/useAuth.js` | 认证 Hook |
| `web/src/hooks/useApi.js` | API Hook |
| `web/src/hooks/useToast.js` | Toast Hook |
| `web/src/components/common/Button/` | 按钮组件 |
| `web/src/components/common/Input/` | 输入框组件 |
| `web/src/components/common/Card/` | 卡片组件 |
| `web/src/components/common/Loading/` | 加载组件 |
| `web/src/components/common/ErrorMessage/` | 错误提示组件 |
| `web/src/components/common/Toast/` | Toast 通知组件 |
| `web/src/components/layout/Header/` | 头部组件 |
| `web/src/components/layout/Container/` | 容器组件 |
| `web/src/components/layout/PageLayout/` | 页面布局组件 |
| `web/src/components/material/MaterialCard/` | 资料卡片组件 |
| `web/src/components/material/MaterialGrid/` | 资料网格组件 |
| `web/src/styles/variables.css` | CSS 变量 |
| `web/src/styles/reset.css` | 重置样式 |

### 修改文件
| 文件路径 | 修改内容 |
|---------|---------|
| `web/src/App.jsx` | 添加懒加载、AuthProvider |
| `web/src/App.css` | 删除或大幅精简 |
| `web/src/index.css` | 保留全局样式 |
| `web/src/api/axios.js` | 使用环境变量 |
| `web/src/pages/Login.jsx` | 使用新组件和 AuthContext |
| `web/src/pages/MaterialList.jsx` | 使用新组件和布局 |
| `web/src/pages/Upload.jsx` | 使用新组件和布局 |
| `web/src/pages/Watch.jsx` | 拆分组件、优化结构 |

---

## 实施顺序

1. **阶段一**: 环境变量配置 (5 分钟)
2. **阶段三**: 状态管理 (15 分钟)
3. **阶段二**: 组件拆分 (30 分钟)
4. **阶段四**: 代码分割 (10 分钟)
5. **阶段五**: 样式重构 (20 分钟)
6. **阶段六**: 页面重构 (30 分钟)
7. **阶段七**: 优化增强 (可选)

**预计总时间**: 约 2 小时

---

## 注意事项

1. **渐进式重构**: 每个阶段完成后确保应用可正常运行
2. **保持向后兼容**: 重构过程中不改变现有功能
3. **测试验证**: 每个阶段完成后手动测试主要功能
4. **保留备份**: 重构前确保代码已提交到 Git

# UI界面设计

## 1. 设计风格

完全参照 AiMaMi-main 项目的设计风格：
- **UI组件库**: shadcn/ui (Radix UI + Tailwind CSS)
- **图标**: Lucide React
- **布局**: SidebarProvider + SidebarInset + SiteHeader
- **色彩**: HSL色彩系统，支持亮色/暗色模式
- **动画**: PageStage页面切换动画

## 2. 技术栈

| 技术 | 选择 | 说明 |
|------|------|------|
| CSS框架 | Tailwind CSS 3 | 原子化CSS |
| 组件库 | shadcn/ui | 基于Radix UI |
| 图标库 | Lucide React | 轻量级图标 |
| 状态管理 | React useState | 简单状态 |
| 路由 | 侧边栏切换 | 无需React Router |

## 3. 色彩系统

使用HSL变量定义，支持亮色/暗色模式：

```css
:root {
  --primary: 217 91% 60%;
  --background: 240 4% 97.6%;
  --card: 0 0% 100%;
  --border: 0 0% 89.8%;
  --sidebar-background: 0 0% 98%;
}

.dark {
  --background: 240 10% 6%;
  --card: 240 8% 11%;
  --border: 60 2.17% 18.04%;
  --sidebar-background: 240 5.9% 10%;
}
```

## 4. 布局结构

```
┌─────────────────────────────────────────────────────────┐
│ SidebarProvider                                          │
│ ┌──────────┬──────────────────────────────────────────┐ │
│ │ Sidebar  │ SidebarInset                             │ │
│ │          │ ┌──────────────────────────────────────┐ │ │
│ │ Logo     │ │ SiteHeader (SidebarTrigger + Title)  │ │ │
│ │ ──────── │ ├──────────────────────────────────────┤ │ │
│ │ 仪表盘   │ │                                      │ │ │
│ │ 服务商   │ │ PageStage (active/exiting/idle)      │ │ │
│ │ 模型     │ │   ┌──────────────────────────────┐  │ │ │
│ │ API Keys │ │   │   页面内容                    │  │ │ │
│ │ 日志     │ │   │                              │  │ │ │
│ │ 配置     │ │   └──────────────────────────────┘  │ │ │
│ │ ──────── │ │                                      │ │ │
│ │ 主题切换 │ │                                      │ │ │
│ └──────────┴──────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────┘
```

## 5. 页面设计

### 5.1 仪表盘

- 4个统计卡片 (BentoCard)
- 模型分布列表
- 服务商分布列表

### 5.2 服务商管理

- 表格展示服务商列表
- 弹窗添加服务商（支持多格式URL配置）
- 弹窗添加Token

### 5.3 模型管理

- 卡片网格展示模型
- 弹窗添加模型

### 5.4 API Key管理

- 表格展示API Key列表
- 弹窗生成新Key（显示一次性）

### 5.5 日志查询

- 表格展示日志
- 状态筛选（全部/成功/失败）
- 弹窗查看详情

### 5.6 配置

- 分组表单配置
- 开关、输入框、数字输入

## 6. 组件清单

### shadcn/ui组件
- SidebarProvider, Sidebar, SidebarContent, SidebarMenu, SidebarMenuItem, SidebarMenuButton
- Button (default, outline, ghost, destructive, soft)
- Card / BentoCard
- Dialog, AlertDialog
- Input, Select, Switch, Checkbox
- Table
- Badge
- Tabs
- Separator
- Tooltip
- Skeleton
- ScrollArea
- DropdownMenu
- Sheet

### 自定义组件
- PageHeader (标题 + 描述 + 操作按钮)
- PageStage (页面切换动画)
- SiteHeader (侧边栏触发器 + 标题)
- AppSidebar (导航菜单 + 主题切换)

## 7. 文件结构

```
ui/src/
├── components/
│   ├── ui/           # shadcn/ui组件
│   ├── layout/
│   │   ├── sidebar.tsx
│   │   ├── site-header.tsx
│   │   └── page-stage.tsx
│   └── Login.tsx
├── hooks/
│   ├── use-theme.ts
│   ├── use-mobile.ts
│   ├── use-route-transition.ts
│   └── use-toast.ts
├── lib/
│   └── utils.ts      # cn()函数
├── api/
│   ├── auth.ts
│   ├── dashboard.ts
│   ├── provider.ts
│   ├── model.ts
│   ├── apikey.ts
│   └── log.ts
├── pages/
│   ├── Dashboard.tsx
│   ├── Providers.tsx
│   ├── Models.tsx
│   ├── ApiKeys.tsx
│   ├── Logs.tsx
│   └── Config.tsx
├── App.tsx
├── main.tsx
└── index.css
```

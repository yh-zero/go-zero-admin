# @vben/vsh

一个 Shell 脚本工具集合，用于 Vue Vben Admin 项目的开发和管理。

## 功能特性

- 🚀 基于 Node.js 的现代化 Shell 工具
- 📦 支持模块化开发和按需加载
- 🔍 提供依赖检查和分析功能
- 🔄 支持循环依赖扫描
- 📝 提供包发布检查功能

## 安装

```bash
# 使用 pnpm 安装
pnpm add -D @vben/vsh

# 或者使用 npm
npm install -D @vben/vsh

# 或者使用 yarn
yarn add -D @vben/vsh
```

## 使用方法

### 全局安装

```bash
# 全局安装
pnpm add -g @vben/vsh

# 使用 vsh 命令
vsh [command]
```

### 本地使用

```bash
# 在 package.json 中添加脚本
{
  "scripts": {
    "vsh": "vsh"
  }
}

# 运行命令
pnpm vsh [command]
```

## 命令列表

- `vsh check-dep`: 检查项目依赖
- `vsh check-circular`: 扫描循环依赖
- `vsh publint`: 检查包发布配置
- `vsh lint`: 执行代码规范检查
- `vsh code-workspace`: 管理 VS Code 工作区配置

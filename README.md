# Personal Wiki

基于 Hugo 的个人知识库，配套 CLI 工具 `wiki` 管理文章和收藏，支持通过 AI Agent（Claude Code、OpenClaw 等）对话式写作和发布。

## 架构

```
┌──────────────────────────────────────────────┐
│        AI Agent (Claude Code / OpenClaw)      │
│          通过 SKILL.md 了解使用方式             │
│              ↓ 调用 CLI                       │
├──────────────────────────────────────────────┤
│                 wiki CLI                      │
│  draft · edit · bookmark · preview · push     │
│  list · search                                │
│              ↓ 操作文件 & Git                  │
├──────────────────────────────────────────────┤
│              Hugo 静态站点                     │
│         content/docs/ (Markdown 文章)          │
│              ↓ git push                       │
├──────────────────────────────────────────────┤
│      GitHub Actions → GitHub Pages 部署        │
└──────────────────────────────────────────────┘
```

## 知识分类

| 分类 | 子分类 | 说明 |
|------|--------|------|
| **AI** | `AI/LLM`、`AI/Agent` | 大语言模型、智能体、提示工程 |
| **计算机基础** | `操作系统`、`数据库`、`计算机网络` | CS 核心基础 |
| **编程语言** | `Java`、`Go`、`Python` | 语言特性、生态、最佳实践 |
| **云原生** | `Docker` | 容器化、编排、基础设施 |
| **收藏** | — | 稍后阅读的链接 |

> 分类可按需扩展，创建文章时如果指定了不存在的分类，CLI 会自动创建对应目录。

## Wiki CLI

### 安装

```bash
# 克隆仓库后
go install ./cmd/wiki/
```

### 命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `wiki draft` | 创建文章草稿 | `wiki draft --title "标题" --category "AI/LLM" --tags "T1,T2" --summary "摘要" --file content.md` |
| `wiki edit` | 修改已有文章 | `wiki edit --slug "AI/LLM/文章slug" --file updated.md` |
| `wiki bookmark` | 收藏链接 | `wiki bookmark --url "https://..." --title "标题" --desc "简介"` |
| `wiki preview` | 预览文章 | `wiki preview --slug "AI/LLM/文章slug"` |
| `wiki list` | 列出文章 | `wiki list --category AI` / `wiki list --saved` / `wiki list --tags Docker` |
| `wiki search` | 搜索 | `wiki search "关键词"` |
| `wiki push` | 发布 | `wiki push --message "docs: 更新说明"` |

## Agent 接入

本项目通过 `SKILL.md` 定义 Agent 技能，任何支持 Skill 协议的 Agent 均可接入。

### 接入步骤

1. **安装 CLI**：确保 `wiki` 和 `hugo` 已安装并在 PATH 中。
2. **注册 Skill**：将本项目的 `SKILL.md` 注册到你的 Agent 中。
3. **使用**：在 Agent 对话中即可触发，例如：
   - "帮我写一篇关于 Docker 网络模型的文章"
   - "收藏一下这个链接 https://..."
   - "看看我的知识库里有哪些 AI 相关的文章"

### Skill 能力

| 能力 | 说明 |
|------|------|
| **写文章** | Agent 先与你深度探讨知识点，确保理解透彻后再整理成文，支持预览和迭代修改 |
| **收藏链接** | Agent 自动访问 URL、提取标题、生成内容摘要并保存 |
| **查询搜索** | 通过对话浏览分类、按标签筛选、全文搜索 |
| **发布** | 确认后一键 commit + push，自动触发部署 |

## 技术栈

- **站点生成**：[Hugo](https://gohugo.io/) + [Book 主题](https://github.com/alex-shpak/hugo-book)
- **部署**：GitHub Pages + GitHub Actions

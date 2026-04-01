---
name: personal-wiki
description: "管理基于 Hugo 的个人知识库。支持创建技术文章草稿、收藏链接稍后阅读、编辑预览文章、发布到 GitHub Pages。当用户想写文章、收藏链接、浏览或搜索知识库、部署更新时触发。"
metadata:
  requires:
    bins: ["wiki", "hugo"]
---

# Wiki — 个人知识库 CLI

## 命令概览

```bash
wiki draft     # 创建文章草稿
wiki edit      # 修改已有文章
wiki bookmark  # 收藏链接，稍后阅读
wiki preview   # 渲染为 HTML 并在浏览器中预览
wiki list      # 列出文章和收藏
wiki search    # 按关键词搜索
wiki push      # 提交并推送，触发部署
```

## 分类体系

`--category` 可用分类：

| 分类 | 说明 |
|------|------|
| `AI` | AI 通用（LLM、Agent、提示工程等） |
| `AI/LLM` | 大语言模型 |
| `AI/Agent` | AI 智能体 |
| `计算机基础/操作系统` | 操作系统 |
| `计算机基础/数据库` | 数据库（MySQL、Redis 等） |
| `计算机基础/计算机网络` | 计算机网络 |
| `编程语言/Java` | Java 生态 |
| `编程语言/Go` | Go |
| `编程语言/Python` | Python |
| `云原生/Docker` | Docker、容器化 |

如果现有分类不匹配，可以创建新的子分类。

## 工作流

### 写文章

写文章的核心目标是帮助用户**深入理解知识点**，而不仅仅是产出文字。整个过程应该是一次有深度的技术探讨。

**第一阶段：深度探讨**

1. 和用户确认要写的主题。
2. 围绕主题与用户展开深入对话：
   - 用提问引导用户思考：这个概念的本质是什么？为什么要这样设计？和其他方案相比优劣在哪？
   - 对用户的理解进行"审慎挑战"：指出可能的误解、补充被忽略的细节、提供不同视角。
   - 用类比、对比、具体例子帮助用户建立直觉。
   - 确保用户真正理解了核心知识点，而不只是听了一遍。
3. 当用户对内容已经有清晰的理解后，再进入写作阶段。

**第二阶段：整理成文**

4. 基于探讨的内容，整理出结构清晰的文章。和用户确认分类、标签和摘要。
5. 将文章正文写入临时文件（纯 Markdown，不含 front matter，CLI 会自动生成）。
6. 创建草稿：
   ```bash
   wiki draft --title "文章标题" --category "AI/LLM" --tags "Tag1,Tag2" --summary "一句话摘要" --file /tmp/wiki-draft.md
   ```

**第三阶段：预览和修改**

7. 询问用户是否预览：
   ```bash
   wiki preview --slug "AI/LLM/文章slug"
   ```
8. 如需修改，将新内容写入临时文件后更新：
   ```bash
   wiki edit --slug "AI/LLM/文章slug" --file /tmp/wiki-updated.md
   ```
9. 重复 预览 → 修改，直到用户满意。

**第四阶段：发布**

10. 用户确认后发布：
    ```bash
    wiki push --message "docs: 新增 xxx 文章"
    ```

### 收藏链接

1. 用户提供 URL（可选标题）。
2. 如果用户未提供标题和描述，访问该 URL 提取标题，并生成 2-3 句内容摘要。
3. 根据内容自动判断合适的标签。
4. 保存：
   ```bash
   wiki bookmark --url "https://..." --title "标题" --tags "Tag1,Tag2" --desc "内容摘要"
   ```

### 浏览和搜索

```bash
wiki list                        # 所有文章
wiki list --category "AI"        # 按分类筛选
wiki list --saved                # 仅收藏
wiki list --tags "Docker"        # 按标签筛选
wiki search "transformer"        # 全文搜索
```

### 发布

1. 执行 `wiki list` 和 `git status` 查看待发布内容。
2. 向用户确认后再发布。
3. 推送：
   ```bash
   wiki push --message "docs: 简要描述"
   ```

## Front Matter 格式

CLI 自动生成以下格式的 front matter（文章内容中不要包含）：

```yaml
---
title: "文章标题"
date: 2026-04-01
draft: true
summary: "一句话摘要"
tags: ["Tag1", "Tag2"]
---
```

## 注意事项

- 文章内容不要包含 front matter，CLI 会根据参数自动生成。
- 执行 `wiki push` 前必须先和用户确认。
- 文章默认用中文撰写，技术术语保持英文。
- 文件名根据标题自动生成（slugify）。

---
name: personal-wiki
description: "管理基于 Hugo 的个人知识库。支持创建技术文章草稿、收藏链接稍后阅读、编辑预览文章、发布到 GitHub Pages。当用户想写文章、收藏链接、浏览或搜索知识库、部署更新，或用户直接提供了 URL 链接时触发。"
metadata:
  requires:
    bins: ["wiki", "hugo"]
---

# Wiki — 个人知识库 CLI

## 命令概览

```bash
# 文章管理
wiki article create   # 创建文章草稿
wiki article edit     # 修改已有文章
wiki article show     # 查看文章内容
wiki article list     # 列出文章
wiki article delete   # 删除文章
wiki article preview    # 渲染为 HTML 并在浏览器中预览
wiki article categories # 列出所有分类

# 书签管理
wiki bookmark add     # 收藏链接，稍后阅读
wiki bookmark list    # 列出收藏
wiki bookmark delete  # 删除收藏

# 搜索
wiki search           # 搜索文章和书签

# 发布
wiki push             # 提交并推送，触发部署
wiki push --diff      # 仅查看待发布变更
```

## 分类体系

创建文章前，先通过 `wiki article categories` 查看已有分类。如果现有分类不匹配，可以在 `--category` 中指定新分类路径，CLI 会自动创建。

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

1. 基于探讨的内容，整理出结构清晰的文章。和用户确认分类、标签和摘要。
2. 将文章正文写入临时文件（纯 Markdown，不含 front matter，CLI 会自动生成）。
3. 创建草稿：
   ```bash
   wiki article create --title "文章标题" --category "AI/LLM" --tags "Tag1,Tag2" --summary "一句话摘要" --file /tmp/wiki-draft.md
   ```

**第三阶段：预览和修改**

1. 询问用户是否预览：
   ```bash
   wiki article preview --slug "AI/LLM/文章slug"
   ```
2. 如需修改，将新内容写入临时文件后更新：
   ```bash
   wiki article edit --slug "AI/LLM/文章slug" --file /tmp/wiki-updated.md
   ```
3. 重复 预览 → 修改，直到用户满意。

**第四阶段：发布**

1. 用户确认后发布：
   ```bash
   wiki push --message "docs: 新增 xxx 文章"
   ```

### 收藏链接

1. 用户提供 URL（可选标题）。
2. 如果用户未提供标题和描述，访问该 URL 提取标题，并生成 2-3 句内容摘要。
3. 根据内容自动判断合适的标签。
4. 向用户展示标题、摘要和标签，确认后再保存。
5. 保存：
   ```bash
   wiki bookmark add --url "https://..." --title "标题" --tags "Tag1,Tag2" --desc "内容摘要"
   ```

### 浏览和搜索

```bash
# 文章
wiki article list                        # 所有文章
wiki article list --category "AI"        # 按分类筛选
wiki article list --tags "Docker"        # 按标签筛选
wiki article show --slug "AI/LLM/xxx"    # 查看文章内容

# 书签
wiki bookmark list                       # 所有书签
wiki bookmark list --tags "Go"           # 按标签筛选

# 搜索（默认搜全部）
wiki search "transformer"                        # 全文搜索
wiki search "transformer" --type article          # 仅搜文章
wiki search "transformer" --type bookmark         # 仅搜书签
wiki search "transformer" --field title            # 仅搜标题
wiki search "transformer" --field content          # 仅搜正文
```

### 发布

1. 执行 `wiki push --diff` 查看待发布变更。
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
- `wiki article list` 和 `wiki search` 输出的是 slug，可直接用于 `--slug` 参数。
- `wiki bookmark delete` 的 slug 不需要带 `收藏/` 前缀，命令会自动补全。

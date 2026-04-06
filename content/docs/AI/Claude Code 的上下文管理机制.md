---
title: "Claude Code 的上下文管理机制"
date: 2026-04-06T17:52:33+08:00
draft: true
summary: "深入解析 Claude Code 的三层上下文管理：Microcompact 微压缩、Auto Compact 自动压缩、以及 Prompt Cache 缓存优化机制"
tags: ["Claude Code", "Prompt Cache", "KV Cache", "上下文管理"]
---

Claude Code 作为一个 agentic 编码工具，单次会话可能涉及几十轮工具调用，上下文很容易膨胀到数十万 token。如果不加管理，要么撞上上下文窗口上限（200K/1M），要么每次请求都要为巨量 token 付费。

Claude Code 设计了一套**三层上下文管理机制**，从轻量到重量级依次为：Microcompact（微压缩）、Auto Compact（自动压缩）、以及贯穿全程的 Prompt Cache（提示缓存）优化。

理解这套机制需要先理解 KV Cache 和 Prompt Cache 的概念。如果你还不熟悉，建议先阅读[上一篇文章](/docs/ai/llm-token/)。

## Prompt Cache：跨请求的 KV Cache 复用

### 单次请求内的 KV Cache

在[上一篇文章](/docs/ai/llm-token/)中我们讲过，模型在生成每个新 token 时，会缓存旧 token 的 K 和 V 向量，避免重复计算。这是**单次请求内**的优化。

### 跨请求的 Prompt Cache

Anthropic 的 Prompt Cache 把这个思路扩展到了**跨请求**：如果你连续两次请求的前缀相同（system prompt + tools + 消息序列），服务端会复用上一次计算好的 KV 向量，不需要重新计算。

```
请求 N:
  输入: [system][tools][msg1][msg2][msg3]  ← 全部计算 KV，缓存到服务端

请求 N+1（5分钟内）:
  输入: [system][tools][msg1][msg2][msg3][msg4（新消息）]
         ↑________________________↑        ↑
         前缀相同，直接读缓存的 KV       只需要计算新消息的 KV
```

这带来两个好处：**更快**（跳过大量计算）和**更便宜**（缓存读取的价格远低于重新计算）。

### TTL：缓存的存活时间

服务端不可能无限期保留缓存——KV Cache 占用的是 GPU 显存，资源有限。因此缓存有 TTL（Time To Live）：

- **默认 5 分钟**：所有用户适用。5 分钟内有新请求就续期，否则缓存过期
- **1 小时**：Anthropic 内部员工、付费订阅且未超额的用户可获得。本质上是 Anthropic 在补贴缓存存储成本

TTL 是服务端行为，用户无法配置。

### 缓存失效的代价

以 200K token 上下文为例，如果缓存失效（cache miss），下次请求需要重新创建缓存。创建缓存的价格是读取缓存的十几倍，一次 cache miss 可能多花几毛钱。这就是为什么 Claude Code 花了大量精力来**避免缓存失效**。

什么会导致缓存失效？**前缀不一致**。只要 system prompt、tools 定义、消息序列中任何一个字节发生变化，从变化点往后的所有 KV 都无法复用，需要重新计算。

## Microcompact：轻量级清理

在 agentic 对话中，大量 token 来自工具调用结果——一次 `grep` 可能返回几千行，一次 `read` 可能读入整个文件。这些结果在当时有用，但随着对话推进，旧的工具结果价值递减。Microcompact 的目标就是**清理这些旧的工具结果**。

Claude Code 实现了两种 microcompact 策略，分别应对缓存热（TTL 内）和缓存冷（TTL 过期）的场景。

### Time-based Microcompact：缓存冷时的清理

当距离上次 assistant 消息**超过 60 分钟**时触发。此时服务端缓存必然已过期（即使 1h TTL 用户也过期了），下次请求无论如何都要重新创建缓存。

既然缓存已经是冷的，修改消息内容不会造成额外损失。所以直接**替换**旧的 tool_result 内容为占位符：

```
修改前: msg3 的 tool_result = "（5000行 grep 输出）"
修改后: msg3 的 tool_result = "[Old tool result content cleared]"
```

保留最近 N 个工具结果不清理，确保模型还有足够的工作上下文。这样下次重建缓存时，需要重建的内容更少，节省了 `cache_creation` 的费用。

### Cached Microcompact（Cache Edits）：缓存热时的精细手术

如果缓存还在 TTL 内（比如你一直在连续操作），直接修改消息内容会**破坏缓存**——因为前缀变了。这时候用的是 Anthropic API 提供的 `cache_edits` 机制。

核心思路：**不改消息内容，附一张"跳过清单"**。

```
消息内容: [system][tools][msg1][msg2(5K grep结果)][msg3]  ← 一字不改
额外参数: cache_edits: ["跳过 msg2 的 tool_result"]

服务端处理：
  阶段一（匹配）：消息内容和缓存一致 → cache hit ✅
  阶段二（推理）：按跳过清单过滤 → 模型看不到 msg2 的 grep 结果
```

**匹配看"你发了什么"，过滤改"模型看到什么"**。两件事解耦了。

这在 Transformer 层面的实现机制是 **attention mask**——被跳过的 token 的 KV 向量仍然存在于缓存中，但在注意力计算时，它们的分数被设为负无穷，softmax 后权重变为 0，对模型输出没有任何贡献。等效于这些 token 不存在，但缓存不需要重建。

#### 服务端缓存的始终是原始版本

一个关键细节：服务端缓存的**永远是未编辑的原始版本**。`cache_edits` 不会修改缓存内容，它只是每次请求都附带的过滤指令。

这意味着每次请求都**必须重新发送**之前所有的 cache_edits。如果某次请求漏发了，服务端会把那个工具结果重新"给模型看"——因为它一直都在消息里。

这就是 Claude Code 源码中 **pin 机制**的作用：每次发出 cache_edits 后，客户端记住这条指令和它的位置，在后续每一次请求中重新发送：

```typescript
// 每次请求都要重发所有历史 edits
export function getPinnedCacheEdits() {
  return cachedMCState.pinnedEdits  // 所有之前积累的 edits
}
```

随着 pinned edits 越积越多，维护成本上升——这时候就需要更重量级的手段了。

### 两种策略的选择逻辑

```
每次 API 调用前，microcompactMessages() 执行：
  ├─ 距上次 assistant 消息 > 60 分钟？
  │    → 缓存已冷，直接修改消息内容（Time-based）
  ├─ 缓存热 + 工具结果累积超阈值？
  │    → 生成 cache_edits 跳过清单（Cached Microcompact）
  └─ 都不触发
       → 消息原样发送
```

两者互斥，time-based 优先判断。命中 time-based 后会重置 cached microcompact 的状态，因为缓存已经冷了，之前积累的 cache_edits 全部失效。

## Auto Compact：完整压缩

当 microcompact 不足以控制上下文增长时，Auto Compact 介入——用 LLM 生成一份结构化摘要，**替换全部历史消息**。

### 触发条件

```
token 使用量 >= 有效上下文窗口 - 13K（buffer）
```

其中有效窗口 = 模型上下文窗口（200K 或 1M）- 20K（预留给摘要输出）。

以 200K 模型为例：阈值 = 200K - 20K - 13K = **167K**。在 167K 时就触发，而不是等到 200K 报错——留出余量让 compact 本身的 API 调用也不会超限。

用户也可以随时通过 `/compact` 手动触发。

### 压缩流程

1. 将当前所有消息（去除图片，节省 token）发送给模型
2. 模型生成结构化摘要，包含 9 个部分：
   - 用户意图、技术概念、文件和代码变更
   - 错误和修复、问题解决过程
   - 所有用户消息（非工具结果）
   - 待办任务、当前工作、下一步计划
3. 用摘要替换全部历史消息

### Post-compact 恢复

压缩后上下文被清空，但模型可能需要一些关键信息才能继续工作。Claude Code 会自动恢复：

- **最近读取的文件**：最多 5 个文件，受 50K token 预算限制
- **Plan 文件**：如果当前有活跃的计划
- **Skill 内容**：已调用的 skill 指令，每个最多 5K token
- **工具和 Agent 列表**：重新通告 deferred tools、agent listing 等

### 缓存优化：Fork Agent 共享缓存前缀

Compact 本身也是一次 API 调用，需要把全部消息发给模型。Claude Code 通过 **forked agent** 机制来优化这个过程：compact 请求复用主对话的 system prompt + tools + 消息前缀，这样可以命中主对话已有的 prompt cache，避免为 compact 请求单独创建缓存。

### Partial Compact：部分压缩

除了全量压缩，Claude Code 还支持部分压缩，允许用户选择一条消息作为分界点：

- **`from` 方向**：压缩分界点之后的消息，保留之前的（保留 prompt cache）
- **`up_to` 方向**：压缩分界点之前的消息，保留之后的（会导致 cache miss，但保留了最近的上下文）

### 安全机制

- **熔断器**：连续失败 3 次后停止重试，避免在上下文不可恢复时反复浪费 API 调用
- **PTL 重试**：如果 compact 请求本身也超过了 prompt-too-long 限制，会逐步丢弃最旧的消息轮次再重试
- **递归保护**：session_memory 和 compact 类型的查询不会触发 auto compact，避免死锁

## Prompt Cache Break Detection：缓存监控

除了管理上下文，Claude Code 还有一套**监控系统**来检测缓存是否意外失效。

### 工作原理

分两个阶段：

1. **Pre-call（请求前）**：记录 system prompt hash、tools schemas hash、model、betas 等状态的快照
2. **Post-call（请求后）**：比较 `cache_read_input_tokens` 是否比上次下降超过 5% 且绝对值超过 2000

如果检测到缓存断裂，会分析可能的原因：

- system prompt 变更（CLAUDE.md 改了？）
- tools 定义变更（MCP 工具加载/卸载？）
- model 切换
- fast mode 切换
- TTL 过期（5 分钟 / 1 小时无请求）
- 服务端路由变更（客户端无变化时的断裂）

检测结果记录到遥测事件中，供调试使用。

## 三层机制的协作关系

```
每次 API 调用前:
  └─ microcompactMessages()
       ├─ 缓存冷（>60min）→ 直接清空旧 tool_result
       ├─ 缓存热 + 超阈值 → 生成 cache_edits 跳过清单
       └─ 都不触发 → 原样发送

每次 API 调用后:
  └─ autoCompactIfNeeded()
       ├─ token 未超阈值 → 跳过
       └─ token 超阈值 → session memory compact → 完整 compact

贯穿全程:
  └─ Prompt Cache Break Detection
       └─ 监控每次请求的缓存命中情况，记录异常
```

三者是**递进**关系：

1. **Microcompact** 在每次请求前轻量清理，延缓上下文增长
2. **Auto Compact** 在上下文逼近上限时做重量级摘要压缩
3. **Cache Break Detection** 全程监控，确保前两者不会意外破坏缓存

这套设计的核心权衡是：**在信息保留、推理成本和缓存效率之间寻找最优平衡**。Microcompact 牺牲旧信息换空间但保护缓存，Auto Compact 牺牲全部历史换回一个干净的起点，而 cache_edits 让 microcompact 在缓存热时也能安全执行。

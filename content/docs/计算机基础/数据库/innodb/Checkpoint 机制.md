---
title: "Checkpoint 机制"
date: 2023-07-27T14:39:39+08:00
draft: false
summary: "深入介绍 InnoDB 的 Checkpoint 机制，包括脏页刷盘、Redo Log 空间回收与崩溃恢复加速的需求，Write Ahead Log（WAL）策略、LSN 日志序列号的作用，以及 Checkpoint 触发脏页刷新的流程。"
tags: [MySQL, InnoDB, checkpoint]
categories: [Database]
source: csdn
source_id: "131959644"
---

## 一、引入
 

由于页的操作首先都是在缓冲池中完成的，那么如果一条DML语句改变了页中的记录，那么此时页就是脏的，即缓冲池中页的版本要比磁盘的新。那么数据库需要将新版本的页刷新到磁盘。倘若每次一个页发生变化就刷新，那么开销会很大，若热点数据集中在某几个页中，那么数据库的性能将变得非常差。


同时如果在缓冲池将新版本的页刷新到磁盘时发生了宕机，那么数据就不能恢复了。**为了避免发生数据丢失的问题，当前事务数据库普遍都采用了 Write Ahead Log 策略，即当事务提交时，先写重做日志，再修改页**。当由于发生宕机而导致数据丢失时，通过重做日志来完成数据的恢复，从而满足事务的持久性要求。


如果说重做日志可以无限地增大，同时缓冲池也足够大，能够缓冲所有数据库的数据，那么是不需要将缓冲池中页的新版本刷回磁盘。因为发生宕机时完全可以通过重做日志来恢复数据库系统的数据到宕机发生的情况。然而现实是这两个条件是很难满足的，即使满足了，那么如果数据库运行了很久后发生宕机，那么使用重做日志进行恢复的时间也会非常的久。即缓冲池的容量和重做日志容量是有限的，所以需要定期将脏页刷回磁盘，在这样的情况下，引入了 Checkpoint（检查点）技术。


所谓 Checkpoint，是指一个触发点（时间点），**当发生 Checkpoint 时，会将脏页（数据脏页和日志脏页）写回磁盘**。总的来说，Checkpoint 是数据库管理系统中的一个操作，用于将脏页刷新到磁盘，以确保数据的持久性和一致性。


## 二、LSN


LSN 称为日志的逻辑序列号（log sequence number），是日志空间中每条日志的结束点，用字节偏移量来表示。在 InnoDB 存储引擎中，LSN 占8个字节，LSN 的值会随着日志的写入而逐渐变大。除了重做日志，每个页（在每个数据页的头部 FILE_HEADER 部分，有一个 FIL_PAGE_LSN 记录了该数据页最后被修改的日志序列位置）以及 Checkpoint 也会被分配一个LSN，以便在需要时可以按照顺序进行检索和恢复。


**即 Checkpoint 是通过LSN实现，其由一个 LSN 表示，用来记录已经刷回磁盘的最新页的版本**。


可以通过`show engine innodb status`来观察 redo log 里的 checkpoint，结果如下：


```c
......
---
LOG
---
Log sequence number          38890249625                                                                                                                                             
Log buffer assigned up to    38890249625                                                                                                                                             
Log buffer completed up to   38890249625                                                                                                                                             
Log written up to            38890249625                                                                                                                                             
Log flushed up to            38890249625                                                                                                                                             
Added dirty pages up to      38890249625                                                                                                                                             
Pages flushed up to          38890249625                                                                                                                                             
Last checkpoint at           38890249625  
......
```


* log sequence number 就是当前的 redo log (in buffer) 中的 LSN；
* log flushed up to 是刷到 redo log file 磁盘数据中的 LSN；
* pages flushed up to 是下一次即将做 checkpoint lsn 的位置，如果没有新数据写入则取 lsn 的值
* last checkpoint at 是上一次检查点所在位置的 LSN。


当我们执行一条修改语句时，InnoDB 存储引擎的执行过程大概如下：


1. 首先**修改内存中的数据页，并在数据页中记录 LSN**
2. 在**修改数据页的同时向 redo log in buffer 中写入 redo log，并记录下 LSN**
3. 写完 buffer 中的日志之后，当触发了日志刷盘的几种规则时，会向 redo log file on disk 刷入 redo 日志，并在该文件中记录下对应的 LSN
4. 数据页不可能永远只停留在内存中，在某些情况下，会触发 checkpoint来 将内存中的脏页(数据脏页和日志脏页)刷到磁盘，所以会在本次 **checkpoint 脏页刷盘结束时，在 redo log 中记录 checkpoint 的 LSN 位置。**在 Checkpoint 完成之后，checkpoint LSN 之前的 Redo Log 就不再需要了
5. 要刷入所有的数据页需要一定的时间来完成，中途刷入的每个数据页都会记下当前页所在的 LSN。


![\[外链图片转存失败,源站可能有防盗链机制,建议将图片保存下来直接上传(img-kapEiKN8-1690440141442)(assets/image-20230727120620-7r0i4f6.png)\]](https://i-blog.csdnimg.cn/blog_migrate/2ef18fb42903f034901dd330c5c1ca0d.png)



MySQL 在崩溃恢复时，会从重做日志 redo-log 的 Checkpoint 处开始执行重放操作。 它从 last Checkpoint 对应的 LSN 开始扫描 redo-log 日志，并将其应用到 buffer-pool 中，直到 last Checkpoint 对应的 LSN 等于 log flushed up to 对应的 LSN （也就是 redo-log 磁盘上存储的 LSN 值)，则恢复完成 。


## 三、触发时机


Checkpoint 所做的事情无外乎是将缓冲池中的脏页刷回到磁盘，不同之处在于每次刷新多少页到磁盘，每次从哪里获取脏页，以及什么时间触发 Checkpoint。在 InnoDB 内部有两种 Checkpoint，分别为：


* Sharp Checkpoint
* Fuzzy Checkpoint


Sharp Checkpoint 发送在数据库关闭时将**所有的脏页**都刷新回磁盘，这是默认的工作方式，即参数 innodb_fast_shutdown=1。


但是若数据库在运行时也使用 Sharp Checkpoint，那么数据库的可用性就会受到很大影响。所以 InnoDB 存储引擎**内部使用 Fuzzy Checkpoint 进行页的刷新，即每次只刷新一部分脏页**。


InnoDB 存储引擎中可能发生时会触发 Fuzzy Checkpoint：


* Master Thread Checkpoint：Master Thread 差不多以**每秒或每十秒的速度从缓冲池的脏页列表中刷新一定比例的页回磁盘**，这个过程是**异步**的，不会阻塞其他操作。
* FLUSH_LRU_LIST Checkpoint：Buffer Pool 的 LRU 列表需要保留一定数量的空闲页面，来保证 Buffer Pool 中有足够的空间应对新的数据库请求。**在空闲列表不足时，移除LRU列表尾端的页，若移除的页为脏页，则需要进行 Checkpoint**。空闲数量阈值是可以配置的（默认是1024），这个检查在一个单独的 Page Cleaner 线程中进行。
* Async/Sync Flush Checkpoint：**当重做日志不可用（即 redo log 写满）时，需要强制将一些页刷新回磁盘**，此时脏页从脏页列表中获取。

  * 定义 checkpoint_age = redo_log_in_buffer_lsn - checkpoint_lsn，即有多少脏页还未刷回磁盘
  * 定义 async_water_mark = 0.75 * total_redo_log_file_size，sync_water_mark = 0.9 * total_redo_log_file_size
  * 如果 checkpoint_age < async_water_mark，那么不需要刷新任何脏页回磁盘
  * 如果 async_water_mark < checkpoint_age < sync_water_mark，那么触发 Async Flush，从 Flush 列表刷新足够的脏页回磁盘，以满足checkpoint_age < async_water_mark
  * 如果 checkpoint_age > sync_water（种情况一般很少见，除非设置的重做日志文件太小），那么触发 Sync Flush，从 Flush 列表刷新足够的脏页回磁盘，以满足checkpoint_age < async_water_mark
  * 旧版本中 Async Flush 会阻塞发现问题的用户查询线程，Sync Flush 会阻塞所有查询线程，新版本中在独立的 Page Cleaner Thread 中执行，不会阻塞
* Dirty Page too much Checkpoint：**当脏页数量太多时会强制推进 Checkpoint，以保证缓冲区有足够的空闲页**。innodb_max_dirty_pages_pct 的默认值为75，表示当缓冲池脏页比例达到该值时，就会强制进行 Checkpoint，刷新一部分脏页到磁盘。



**参考资料**：


[LSN、Checkpoint？MySQL的崩溃恢复是怎么做的？ - 脉脉 (maimai.cn)](https://maimai.cn/article/detail?fid=1745082373&efid=VqzAC2NVx8VqJs0qWU8M6Q)


[MySQL 引擎特性 · InnoDB LSN 详解 (log sequence number) - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/570694236)


[谈谈MySQL的WAL、LSN、checkpoint - 知乎 (zhihu.com)](https://zhuanlan.zhihu.com/p/492994134#:~:text=FLUSH_LRU_LIST,Checkpoint%20%E4%B8%BA%E4%BA%86%E4%BF%9D%E8%AF%81LRU%E5%88%97%E8%A1%A8%E4%B8%AD%E5%8F%AF%E7%94%A8%E9%A1%B5%E7%9A%84%E6%95%B0%E9%87%8F%EF%BC%88%E9%80%9A%E8%BF%87%E5%8F%82%E6%95%B0innodb_lru_scan_depth%E6%8E%A7%E5%88%B6%EF%BC%8C%E9%BB%98%E8%AE%A4%E5%80%BC1024%EF%BC%89%EF%BC%8C%E5%90%8E%E5%8F%B0%E7%BA%BF%E7%A8%8B%E5%AE%9A%E6%9C%9F%E6%A3%80%E6%B5%8BLRU%E5%88%97%E8%A1%A8%E4%B8%AD%E7%A9%BA%E9%97%B2%E5%88%97%E8%A1%A8%E7%9A%84%E6%95%B0%E9%87%8F%EF%BC%8C%E8%8B%A5%E4%B8%8D%E6%BB%A1%E8%B6%B3%EF%BC%8C%E5%B0%B1%E4%BC%9A%E5%B0%86%E7%A7%BB%E9%99%A4LRU%E5%88%97%E8%A1%A8%E5%B0%BE%E7%AB%AF%E7%9A%84%E9%A1%B5%EF%BC%8C%E8%8B%A5%E7%A7%BB%E9%99%A4%E7%9A%84%E9%A1%B5%E4%B8%BA%E8%84%8F%E9%A1%B5%EF%BC%8C%E5%88%99%E9%9C%80%E8%A6%81%E8%BF%9B%E8%A1%8CCheckpoint%E3%80%82)

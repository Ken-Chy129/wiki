---
title: "Redo 日志"
date: 2023-08-08T11:24:56+08:00
draft: false
summary: "从持久性出发，解释 redo log 的必要性、类型、MTR、写入流程、checkpoint 与崩溃恢复。"
tags: ["MySQL", "InnoDB", "redo log"]
categories: ["数据库"]
source: "https://blog.csdn.net/qq_25046827/article/details/132161038"
source_id: 132161038
---

## 一、为什么需要redo日志


我们知道数据的修改首先是在Buffer Pool中进行的，之后再定时刷到磁盘中。那么如果在事务提交后还没刷新到磁盘中，系统就崩溃了，那么此时数据就丢失了，这就不满足事务的持久性了。而如果我们考虑每次提交之后，都同步将事务中所有的页面刷新到磁盘，这样确实可以保证持久性，但是这种方法存在以下两种问题：


1. **刷新一个完整的数据页太浪费了**。有时候我们可能只是对页中几个字节进行了修改，但是InnoDB是以页为单位进行磁盘IO的，也就是说我们还是不得不将完整的16KB的数据刷新到磁盘中
2. **随机IO速度太慢了**。因为一个事务可能包含多条语句，即使一条语句也可以需要产生对多个页面的修改，但是在磁盘中这些页面未必相邻。这就意味着我们将数据页刷新到磁盘时，需要进行很多的随机IO


我们只是希望这个提交的事务的修改不会丢失，即使崩溃在重启后也可能进行恢复，那么其实完全没有必要在每次提交的时候就将页面全部刷回去，**只需要对修改的内容简单的做一个记录就可以了**。因此InnoDB引入了redo log重做日志，其具有很多种类型，但是大多都有如下的通用结构：


![在这里插入图片描述](/images/mysql-innodb-redo-log/e9f9d3461049d30adc567b38ca97b60b.png)



* type：日志的类型
* space ID：表空间的ID
* page number：页号
* data：这条redo log的具体内容


事务提交时只刷新redo log到磁盘的好处：


* **redo日志占用的空间很小**：只记录需要更新的值和更新的位置等信息
* **redo日志是顺序写入磁盘的**：即使是产生多条日志，也都是按顺序写入日志文件中


## 二、redo日志的类型


### 1）简单的redo日志类型


对页面的修改是极其简单的情况下，redo日志只需要记录一下**某个页面的某个偏移量处修改了几个字节的值**、具体修改后的内容是啥就好了。InnoDB根据写入数据的多少划分了几种不同的redo日志：


1. MLOG_1BYTE：表示在页面的某个偏移量处写入1字节的redo日志类型
2. MLOG_2BYTE：表示在页面的某个偏移量处写入2字节的redo日志类型
3. MLOG_4BYTE：表示在页面的某个偏移量处写入4字节的redo日志类型
4. MLOG_8BYTE：表示在页面的某个偏移量处写入8字节的redo日志类型
5. MLOG_WRITE_STRING：表示在页面的某个偏移量处写入一个字节序列


![在这里插入图片描述](/images/mysql-innodb-redo-log/3fd1090b41d266d637ce15efc6fc6552.png)



MLOG_1BYTE、MLOG_2BYTE、MLOG_4BYTE、MLOG_8BYTE这四种类型的日志结构相似，只是具体数据包含的字节数量不同罢了。MLOG_WRITE_STRING类型的redo日志表示写入一个字节序列，但是因为不能确定写入的具体数据占用多少字节，所以还需要添加一个len字段。


![](/images/mysql-innodb-redo-log/2037c46f6d4eb7b90e785d745e78dbc6.png)



### 2）复杂的redo日志类型


在把一条记录插入到一个页面时，可能需要更改的地方非常多（比如修改数据页的Page Header、Page Diretory中的槽信息、上一条记录的next_record属性等等），如果使用前面简单的日志类型，那么要么产生很多条redo日志（在每处修改的地方都产生一条），要么产生一条记录，将页面第一个被修改的字节到最后一个被修改的字节之间的所有数据都记录下来。但显然这两种方式都有明显的缺点，因此InnoDB引入了复杂的日志类型:


* MLOG_REC_INSERT：表示在插入一条使用非紧凑行格式的记录时产生的日志类型
* MLOG_COMP_REC_INSERT：表示在插入一条使用紧凑行格式的记录时产生的日志类型
* MLOG_COMP_REC_DELETE：表示在删除一条使用紧凑行格式的记录时产生的日志类型
* MLOG_COMP_PAGE_CREATE：表示在创建一个存储紧凑行格式记录的页面时产生的日志类型


还有很多类型就不一一列举了。**这些类型的redo日志既包含物理层面的意思，也包含逻辑层面的意思**：


* 物理层面：记录了对哪个表空间的哪个页进行修改
* 逻辑层面：**崩溃重启后不能直接根据日志的记录从某个偏移量恢复数据**，而是需要调用一些事先准备好的参数，将日志中的一些信息（比如记录的各个列的信息，上一条记录的地址等待）作为参数，通过这些函数进行数据的恢复


## 三、Mini-Transaction


**MySQL把对底层页面进行一次原子访问的过程称为一个Mini-Transaction（MTR），比如向某个索引对应的B+树中插入一条记录的过程就是一个MTR。每个SQL语句可能会包含多个MTR**，比如插入语句包含对聚簇索引对应的B+树插入数据，也包含对其他二级索引对应的B+树插入数据等多个对页面的原子访问过程。而**一个MTR可能会包含多条redo日志**，比如说对一棵B+树执行插入记录操作时，可能涉及对系统页面的改动、修改各种段、区的统计信息、修改各种链表的统计信息等，因此会产生很多的redo日志。我们显然不能说一个插入操作对应的对各种页的修改执行一半就停止了（即有的页修改了有的页没修改，那显然就会出现错误），所以在进行崩溃恢复时，我们需要把**每个MTR对应的多条redo日志当为一组不可分割的整体来处理**。


为了保证这些操作的原子性，必须以组的形式来记录redo日志。**在进行恢复时，针对某个组的redo日志，要么全部恢复，要么一条也不恢复**。为了实现这个功能，InnoDB在**每组的最后一条redo日志后面加上一条特殊类型的redo日志**。这个类型称为MLOG_MULTI_REC_END，它的结构很简单，只有一个type字段。


**所以某个需要保证原子性的操作产生的一系列redo日志，必须以一条类型为MLOG_MULTI_REC_END的redo日志结尾。这样在进行数据恢复时，只有解析到这条日志才认为解析到了一组完整的redo日志，才会进行恢复，否则直接放弃前面解析到的redo日志。**


## 四、redo日志的写入过程


为了更好地管理redo日志，InnoDB将MTR生成的redo日志都放在了大小为**512字节**的页中，一个redo页（又称**redo log block**）的组成如下：


1. log block header（12字节）

    * LOG_BLOCK_HDR_NO：编号
    * LOG_BLOCK_HDR_DATA_LEN：使用了多少字节（初始值为12）
    * LOG_BLOCK_FIRST_REC_GROUP：页内第一个MTR生成的第一条redo日志的偏移量
    * LOG_BLOCK_CHECKPOINT_TO：Checkpoint序号
2. log block body（496字节）：存放redo日志
3. log block trailer（4字节）

    * LOG_BLOCK_CHECKSUM：校验和


为了解决磁盘速度过慢的问题，**redo日志也引入了缓冲区，称为redo log buffer。这片连续的内存空间被划分为若干个连续的redo log block。**


![在这里插入图片描述](/images/mysql-innodb-redo-log/2d7823ffe978fb1e5052392a6273cecc.png)



此外InnoDB还提供了一个称为**buf_free的全局变量，用来指明后续写入的日志应该写到log buffer中的那个位置**。


在MTR执行过程中可能会产生若干条redo日志，这些redo日志是一个不可分割的组，所以并不是没生成一条redo日志就将其插入到log buffer中，而是将每个MTR运行过程中产生的日志先暂存到一个地方，等MTR结束时再将这一组redo日志全部复制到log buffer中。


此外不同的事务可能是并发执行的，因此不同MTR对应的redo日志可能是交替写入log buffer的。


## 五、redo日志文件


### 1、刷盘时机


* **log buffer空间不足**：如果当前写入log buffer的redo日志量占满了log buffer总容量的50%左右，就需要把这些日志刷新到磁盘
* **事务提交**：为了保证持久性，必须再事务提交时把对应的redo日志刷新到磁盘。可以通过`innodb_flush_log_at_trx_commit`参数选择为其他策略

  * 0：只在后台进行处理，事务提交时不刷磁盘。不能保证持久性
  * **1：事务提交时同步刷到磁盘。默认值**
  * 2：事务提交时只是写道操作系统的缓存，操作系统负责刷盘。只要操作系统不宕机则可以保证持久性
* 在脏页刷新到磁盘之前，会保证先将其对应的redo日志刷到磁盘
* **后台线程定时刷新**：大约以每秒一次的频率刷新
* 正常关闭服务器
* **做checkpoint**


### 2、redo日志文件组


MySQL的数据目录下默认有名为ib_logfile0和ib_ligfile1的两个文件，log buffer中的日志在默认情况下就是刷新到这两个磁盘文件中。**磁盘上的redo日志文件是以一个日志文件组的形式出现的，组内有多少个日志文件可以进行设置。**在redo日志写入日志文件组时，首先从ib_logfile0开始写，之后接着ib_logfile1，直至最后一个文件也写满了，就重新回到ib_logfile0继续写，即采用**循环写**的方式。


在redo日志文件组中，每个文件的大小都一样，格式也一样，都是由一下两部分组成：


1. **前2048个字节（即前4个block）用来存储一些管理信息**
2. 从第2048个字节往后的字节用来存储log buffer中的block镜像


![在这里插入图片描述](/images/mysql-innodb-redo-log/8c586f81cdd4060e1682680f47d9fbbe.png)



* log file header：描述该redo日志文件的一些整体属性

  * **LOG_HEADER_START_LSN：本redo日志文件偏移量为2048字节处对应的lsn值**
  * LOG_BLOCK_CHECKSUM：每个block都有的校验和
* checkpoint1

  * LOG_CHECKPOINT_TO：服务器执行checkpoint的编号，每执行一次该值就加一
  * **LOG_CHECKPOINT_LSN：服务器结束checkpoint时对应的lsn值，系统在崩溃后恢复时从该值开始**
  * **LOG_CHECKPOINT_OFFSET：上个属性中的lsn值在redo日志文件组中的偏移量（初始时是2048）**
  * LOG_CHECKPOINT_LOG_BUF_SIZE：执行checkpoint操作时对应的log buffer的大小
  * LOG_BLOCK_CHECKSUM
* checkpoint2：结构同checkpoint1


## 六、log sequence number


### 1、lsn的引入


自系统运行开始，就在不断地修改页面，不断的产生redo日志。redo日志的量在不断递增，永不会缩减。因此InnoDB引入了一个名为**lsn（log sequence number）的全局变量，用来记录当前总共已经写入的redo日志量，也就是说lsn越小，日志产生的越早。**lsn的初始值为8704，即一条redo日志也没写入时，lsn的值就是8704。


在向log buffer中写入redo日志时并不是一条条写入的，而是以MTR生成的一组redo日志为单位写入的，并且写入到的是log block body处。**但是在统计lsn的增长量时，其实不只是统计当前redo日志的字节数，还会去加上这个MTR额外占用的log block header和log block trailer的字节数。**


系统在第一次穷后，在初始化log buffer时，buf_free就会指向第一个block的偏移量为12字节的地方，lsn也会跟着增加12。接着当插入redo日志时，如果带插入的block剩余的空间能够容纳这个MTR提交的日志组，那么lsn就只需要加上这组日志占用的字节数。如果超过了，那么其会使用到下一个block，因此还需要加上这个MTR覆盖的log block header和log block trailer占用的字节数。如果跨过了一个block，则加上12+4个字节。


### 2、flushed_to_disk_lsn


redo日志是先写到log buffer之后才会被刷新到磁盘的redo日志文件中，所以lsn表示的日志包括了写到log buffer但是没有刷新到磁盘的redo日志。因此InnoDB引入了一个名为**flushed_to_disk_lsn的全局变量用来表示刷新到磁盘中的redo日志量。**


在系统第一次启动时，该变量的值与初始的lsn是相同的，都是8704,。随着系统的运行，redo日志不断地被写入到log buffer，但是并不会立即刷新到磁盘，因此lsn的值就会和flushed_to_disk_lsn拉开差距。而当log buffer中的日志被刷新到了磁盘，flushed_to_disk_lsn的值就又随之更新。


### 3、flush链表中的lsn


MTR执行结束后除了将一组日志写入到log buffer外，还需要把执行过程中修改过的页加入到Buffer Pool的flush链表中。


当第一次修改某个已经加载到Buffer Pool中的页面时，就会把这个页面对应的控制块插入到flush链表的头部，之后再修改该页面时，由于它已经在flush链表中，所以就不再次插入了。也就是说，flush链表中的脏页是按照页面第一次修改时间进行排序的，链表前面的脏页第一次修改的时间比较晚，后面的脏页第一次修改的时间比较早。在这个过程中，会在缓冲也对应的控制块中记录两个属性：


1. **oldest_modification：第一次修改时就将修改该页面的MTR开始时对应的lsn值写入到这个属性**
2. **newest_modification：每修改一次页面，都会将修改该页面的MTR结束时对应的lsn值写入这个属性。也就是说该属性代表页面最后一次修改后对应的lsn值**


**多次更新的页面不会重复插入到flush链表中，只会更新newest_modification属性的值。**


## 七、checkpoint


由于redo日志文件组的容量是有限的，所以我们选择了循环写入redo日志，因此会造成最后写入的redo日志与最开始写入的redo日志追尾的情况。为了避免追尾导致的前面的日志被覆盖而丢失，我们需要判断某些redo日志占用的磁盘空间是否可以被覆盖，也就是说这条redo日志对应的脏页是否已经被刷到了磁盘中。因此InnoDB引入了一个**全局变量checkpoint_lsn，用来表示当前系统中可以被覆盖的redo日志总量是多少**，这个变量的初始值也是8704。


**现在如果当脏页a被刷新到磁盘上，那么这个脏页对应的redo日志就可以被覆盖了，所以可以进行一个增加checkpoint_lsn的操作，我们把这个过程称为执行一次checkpoint。**


> 有些后台线程也会不停的将脏页刷新到磁盘中，但是这和执行一次checkpoint是两回事。一般来讲，刷新脏页和执行checkpoint是在不同线程上执行的，并不是说每次有脏页要刷新就要去执行一次checkpoint。也就是说刷新脏页时会去修改flushed_to_disk_lsn的值，但是不会去修改checkpoint_lsn的值。


执行一次checkpoint可以分为两个步骤：


1. 计算当前系统可以被覆盖的redo日志对应的lsn值最大是多少

    * redo日志可以被覆盖，意味着它对应的脏页被刷新到了磁盘中。只要我们计算出当前系统中**最早修改的脏页（即flush链表最早的节点）对应的oldest_modification值**，那么凡是小于这个值的lsn对应redo日志都可以被覆盖掉，我们把该值赋给checkpoint_lsn（因为flush链表保存了所有的脏页，而oldest_modification记录的是第一次修改该页面的MTR开始时对应的lsn值，那也就是说flush链表中最早的脏页的oldest_modification之前的lsn都是已经刷入磁盘了，因此将这个值赋给checkpoint_lsn）
2. **将checkpoint_lsn与对应的redo日志文件组偏移量以及这次checkpoint的编号写到日志文件的管理信息**（也就是checkpoint1和checkpoint2中）

    * **当checkpoint_no的值是偶数就写到checkpoint1中，是奇数就写到checkpoint2中**


![在这里插入图片描述](/images/mysql-innodb-redo-log/a563b0eb41b92e75bbbbf1fe2b7d09b2.png)



其中checkpoint_lsn前的redo日志可以覆盖，因为这些日志对应的修改操作已经落盘。而checkpoint_lsn和flushed_to_disk_lsn之间的redo日志不能覆盖，它们只是日志落盘，但是对应的数据页的修改还没刷入磁盘，还指望着系统崩溃后用这部分日志来进行恢复。flushed_to_disk_lsn到lsn之间表示redo日志还在log buffer中没有落盘。


> 一般脏页都是后台的线程对LRU链表和flush链表进行的，这主要是因为刷脏操作比较慢，不想影响用户线程处理请求。但是如果当前系统修改页面的操作十分频繁，就会导致写redo日志的操作十分频繁，系统lsn值增长过快。如果后台线程的刷脏操作不能将脏页快速刷出，系统将无法即使执行checkpoint，就需要用户线程从flush链表中把那些最早修改的脏页同步刷新到磁盘。这样这些脏页对应的redo日志就没用了，然后就可以去执行checkpoint了。


## 八、系统中的lsn值


使用`show engine innodb status`命令可以查看当前InnoDB存储引擎中各种lsn值的情况：


* Log Sequence number：系统中的lsn值，也就是当前系统已经写入的redo日志量，包括写入到log buffer中的redo日志
* Log flushed up to：表示flushed_to_disk_lsn的值，也就是当前系统已经写入磁盘的redo日志量
* Pages flushed up to：表示flush链表中被最早修改的那个页面对应的oldest_modification属性值（做checkpoint的时候就用这个值更新checkpoint_lsn）
* Last checkpoint at：表示当前系统的checkpoint_lsn值


## 九、崩溃恢复


前面已经说过了，对于lsn值小于checkpoint_lsn的redo日志而言，它们对应的脏页都已经被刷新到磁盘，所以不需要恢复。而对于lsn不小于checkpoint_lsn的redo日志，它们对应的脏页可能没被刷盘，也可能被刷盘了，并不能确定，因此需要**从lsn值为checkpoint_lsn的redo日志开始恢复页面。**


在redo日志文件组第一个文件的管理信息中，有两个block都存储了checkpoint_lsn，我们**选择其中更大的那个来进行恢复，并根据其对应的checkpoint_offset来对应到redo日志的位置。**


确定完起点之后，需要确定恢复的终点。对于redo日志，它们是顺序的写入block中的，也就是写满一个之后才会去写下一个。并且block的log block header中记录了当前block中使用了多少字节的空间，也就是说我们只要恢复到该值不为512的block即可。


确定好恢复的区间之后就可以按顺序遍历redo日志进行恢复了。不过InnoDB还引入了一些机制来加速这个恢复过程：


* 使用哈希表：根据redo日志的spaceID和page number属性计算出哈希值，把哈希值相同的redo日志放到同一个槽中，使用链表连接起来（需要按原redo日志的顺序连接）。之后就可以遍历哈希表进行恢复，因为填一个页面进行修改的redo日志都放在了一个槽中，所以可以一次性将一个页面修复好，减少随机IO
* 跳过已经刷新到磁盘中页面：由于再执行完最后一次checkpoint之后，有可能后台线程又从LRU链表和flush链表中将一些脏页刷回磁盘了，只不过还没做checkpoint而已，因此对于这些页面自然也没必要再进行恢复。至于判断是否已经刷到磁盘中，则是通过每个页面的File Header中的FIL_PAGE_LSN属性，其记载了最近一次刷新页面时对应的lsn值（其实就是页面控制块的newest_modification值）。如果在执行了某次checkpoint之后，有脏页被刷新到磁盘中，那么该页对应的FIL_PAGE_LSN代表的lsn值肯定大于checkpoint_lsn的值，所以凡是lsn小于FIL_PAGE_LSN的redo日志则不需要进行恢复。

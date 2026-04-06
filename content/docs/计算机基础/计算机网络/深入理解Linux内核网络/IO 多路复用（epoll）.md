---
title: "IO 多路复用（epoll）"
date: 2023-07-10T01:30:54+08:00
draft: false
summary: "从内核层面深入分析 Linux epoll IO 多路复用机制，介绍 epoll 相比阻塞 IO 和轮询的优势，详解 epoll_create、epoll_ctl、epoll_wait 三个核心系统调用及 eventpoll 内核对象的工作原理。"
tags: [Linux, epoll]
categories: [Networking]
source: csdn
source_id: "131630312"
weight: 3
---

在上一部分的阻塞模式中（详见[深入理解Linux内核网络——内核与用户进程协作之同步阻塞方案（BIO）](https://blog.csdn.net/qq_25046827/article/details/131625686)），用户进程为了等待一个socket就得被阻塞掉，如果想要同时为多个用户提供服务要么就得创建对应数量的进程处理，要么就使用非阻塞的方式。进程不说创建，单论上下文切换就需要很大的耗时，而如果非阻塞的模式，就得轮询遍历，会导致CPU空转，并且每次轮询都需要进行一次系统调用，所以Linux提供了多路复用的机制来实现一个进程同时高效地处理多个连接。


epoll就是其中最优秀的实现方式，其提供了以下几个相关的函数：


* epoll_create：创建一个epoll对象
* epoll_ctl：向epoll对象添加要管理的连接
* epoll_wait：等待其管理的连接上的IO事件



## 一、内核和用户进程协作之epoll

### 1）epoll内核对象的创建


在用户进程调用epoll_create的时候，内核会创建一个struct eventpoll的内核对象，并把它关联到当前进程的已打开文件列表中。


![在这里插入图片描述](/images/linux-network-multiplexing-epoll/3ef9af523abfeffa1652258a7919e2a0.png)




eventpoll的定义如下：


```c
struct eventpoll {
    // sys_epoll_wait用到的等待队列
    wait_queue_head_q wq;
    // 接收就绪的描述符都会放到这里
    struct list_head rdllist;
    // 每个epoll对象都有一棵红黑树
    struct rb_root rbr;
    ......
}
```


* wq：**等待队列链表**。软中断数据就绪的时候会通过wq来**找到阻塞在epoll对象上的用户进程**
* rbr：**一棵红黑树**，为了支持对海量连接的高效查找、插入和删除，eventpoll内部使用了一棵红黑树。通过这棵树来**管理用户进程下添加进来的所有socket连接**
* rdllist：**就绪的描述符的链表**。当有**连接就绪的时候，内核就会把就绪的连接放到rdllist链表里**。这样应用进程只需要判断链表就能找到就绪的连接，而不用去遍历整棵树


> **struct task_struct**用来表示一个进程（也就是我们通常所说的任务）。这个数据结构包含了所有和进程相关的信息，例如进程的状态、进程的PID、进程的父进程、进程的子进程、进程的线程、进程的调度信息等等。以下是struct task_struct的一些主要成员：
>
> * state：进程的当前状态。进程可能处于运行状态（TASK_RUNNING）、就绪状态（TASK_INTERRUPTIBLE或TASK_UNINTERRUPTIBLE）、停止状态（TASK_STOPPED或TASK_TRACED）或者僵尸状态（EXIT_ZOMBIE）
> * pid：进程的PID（Process ID）。这是一个唯一标识进程的数字
> * parent：进程的父进程。这是一个指向struct task_struct的指针，指向了创建这个进程的父进程
> * children：进程的子进程。这是一个列表，包含了所有由这个进程创建的子进程
> * thread：进程的线程信息。这是一个struct thread_struct类型的成员，包含了进程的所有线程相关的信息
> * prio和static_prio：进程的动态优先级和静态优先级。这两个成员用于进程的调度
> * files：存储了关于当前进程打开的所有文件的信息
>
> **struct files_struct**在Linux内核中是一个用于表示进程打开的文件的数据结构，是struct task_struct的一部分。以下是struct files_struct的一些主要成员：
>
> * count：这个成员是一个原子变量，表示这个files_struct实例的引用计数。当创建一个新的进程（例如通过fork或者clone系统调用）时，新的进程会共享父进程的files_struct，这个时候count就会增加。当进程终止时，count会减少，如果count变为0，那么files_struct就会被销毁。
> * file_lock：这是一个用于同步访问files_struct的锁。
> * fdt：这是一个指向struct fdtable的指针，struct fdtable存储了所有打开的文件描述符和对应的文件指针。
> * next_fd：这是一个表示下一个可用文件描述符的整数。
>
> **struct fdtable**用于维护进程当前打开的所有文件描述符以及它们对应的文件，它是struct files_struct的一部分。以下是struct fdtable的一些主要成员：
>
> * max_fds：这个成员指示了fdtable可以容纳的文件描述符的最大数量。
> * fd：这是一个指针数组，每个元素是一个指向struct file的指针。数组中的索引就是文件描述符。
> * close_on_exec和open_fds：这两个成员都是指向位图（bitmap）的指针，位图中的每一位对应一个文件描述符。open_fds中的一位如果被设置，表示对应的文件描述符是打开的。close_on_exec中的一位如果被设置，表示对应的文件描述符在执行exec()系列函数时需要被关闭。
>
> **struct file**是一个用来表示打开的文件的数据结构。每当一个文件被打开，内核都会创建一个struct file实例来代表这个打开的文件。这个数据结构存储了很多关于打开的文件的信息，例如文件的类型（普通文件、目录、字符设备、块设备等）、文件的位置、文件的状态、文件的权限等。以下是struct file的一些主要成员：
>
> * f_path：这是一个struct path实例，表示文件的路径。struct path包含一个指向struct dentry（目录项）的指针和一个指向struct vfsmount（文件系统挂载信息）的指针。
> * f_pos：这是一个表示文件当前位置（也就是读写指针的位置）的整数。
> * f_op：这是一个指向struct file_operations的指针。struct file_operations包含了一组函数指针，这些函数用来操作文件。例如，f_op->read指向一个用来读取文件的函数，f_op->write指向一个用来写入文件的函数等。
> * f_mode：这是一个表示文件打开模式（例如读、写、追加等）的位掩码。
> * f_flags：这是一个表示文件状态的位掩码，例如是否是非阻塞的、是否是同步的等。


epoll的创建：


```c
SYSCALL_DEFINE1(epoll_create1, int, flags)
{
    struct eventpoll *ep = NULL;
    // 创建一个eventpoll对象
    error = ep_alloc(&ep);
}

static int ep_alloc(struct eventpoll **pep)
{
    struct eventpoll *ep;
    // 申请内存
    ep = kzalloc(sizeof(*ep), GFP_KERNEL);
    // 初始化等待队列头
    init_waitqueue_head(&ep->wq);
    // 初始化就绪队列
    INIT_LIST_HEAD(&ep->rdllist);
    // 初始化红黑树指针
    ep->rnr = RB_ROOT;
    ......
}
```


### 2）为epoll添加socket


假设现在和客户端的多个连接的socket都创建好了，也创建好了epoll内核对象。在使用epoll_ctl注册每一个socket的时候，内核会做如下三件事情：


1. 分配一个红黑树节点对象epiem
2. 将等待事件添加到添加到socket的等待队列中，其回调函数是ep_poll_calback
3. 将epitem插入epoll对象的红黑树


```c
SYSCALL_DEFINE4(epoll_ctl, int, epfd, int, op, int, fd, struct epoll_event __user *, event)
{
    struct eventpoll *ep;
    struct file *file, *tfile;
    // 根据epfd找到eventpoll内核对象
    file = fget(epfd);
    ep = file->private_data;
    // 根据socket句柄号，找到其file内核对象
    tfile = fget(fd);

    switch(op) {
    case EPOLL_CTL_ADD:
	if(!epi) {
 	    epds.events |= POLLERR | POLLHUP; // POLLER：指定的文件描述符发生错误。POLLHUP：指定的文件描述符挂起事件。
	    error = ep_insert(ep, &epds, tfile, fd);
	} else
	    error = -EEXIST;
	clear_tfile_check_list();
	break;
}
```


* epfd：epoll_create创建的fd
* op：操作类型，ADD DEL MOD
* fd：要加入的套接字fd
* event：关心的事件类型，只有我们注册的事件才会在epoll_wait被唤醒后传递到用户空间，否则虽然内核可以收到，但不会传递到用户空间


> struct epoll_event是Linux中用于表示一个epoll（event poll）事件的数据结构。当用户空间的应用程序调用epoll_ctl()函数来添加、修改或删除一个事件，或者调用epoll_wait()函数来等待事件发生时，这个结构就会被用到。
>
> 这个结构的定义如下：
>
> ```c
> struct epoll_event {
> 	__uint32_t events;  /* Epoll events */
> 	epoll_data_t data;  /* User data variable */
> };
> ```
>
> 其中：
>
> * events成员是一个位掩码，表示需要监视的事件类型或者已经发生的事件类型。例如，EPOLLIN表示需要监视输入事件（数据可以被读取），EPOLLOUT表示需要监视输出事件（数据可以被写入），EPOLLERR表示需要监视错误事件等。
> * data成员是一个联合体（union），用户可以将其作为一个指针或者一个整数来使用。这个成员通常被用来存储用户数据，这些数据与要被监视的文件描述符相关联。例如，应用程序可能会把一个指向自定义数据结构的指针放到data.ptr中，这个自定义数据结构包含了与文件描述符相关的上下文信息。


在epoll_ctl中首**先根据传入的fd找到eventpoll、socket相关的内核对象**。对于EPOLL_CTL_ADD操作来说，会执行到ep_insert操作。所有的注册都是在整个函数中完成的。


```c
static int ep_insert(struct eventpoll *ep,
		struct epoll_event *event,
		struct file *tfile, int ft)
{
    // 1.分配并初始化epitem
    struct epitem *epi;
    if(!(epi = kmem_cache_alloc(epi_cache, GFP_KERNEL)))
	return -ENOMEM;

    INIT_LIST_HEAD(&epi->rdllink);
    INIT_LIST_HEAD(&epi->fllink);
    INIT_LIST_HEAD(&epi->pwqlist);
    epi->ep = ep;
    // 设置epitem对象对应的socket的句柄号和file对象地址  
    ep_set_ffd(&epi->ffd, tfile, fd);
    epi->event = *event;
    epi->nwait = 0;
    epi->next = EP_UNACTIVE_PTR;

    // 2.设置socket等待队列
    // 定义并初始化ep_pqueue对象
    struct ep_pqueue epq;
    epq.epi = epi;
    // 初始化epq的pt成员，设置poll的回调为ep_ptable_queue_proc，这其中藏着epoll如此高效的原因!
    init_poll_funcptr(&epq.pt, ep_ptable_queue_proc);

    // 进行完这一步算是把epitem和这个fd连接起来了，会在事件到来的时候执行上面注册的回调
    revents = ep_item_poll(epi, &epq.pt);

    ......
    // 3.将epi插入eventpoll对象的红黑树中
    ep_rbtree_insert(ep, epi);
    ......
}
```


代码的详细剖析见下文，主要分为三部分。


#### 1. 分配并初始化epitem


对于每一个socket，调用epoll_ctl的时候，都会为之分配一个epitem，该结构的定义如下


```c
struct epitem {
    // 红黑树节点
    struct rb_node rbn;
    // socket文件描述符信息
    struct epoll_filefd ffd;
    // 所归属的eventpoll对象
    struct eventpoll *ep;
    // 等待队列，和struct eppoll_entry关联
    struct list_head pwqlist;
    // 关注的事件
    struct epoll_event event;
}
```


对epitem要进行一些初始化，首先是**初始化等待队列**，**将其ep指针指向eventpoll对象**，**设置感兴趣的事件**，然后**用要添加的socket的file、fd来填充epitem-&gt;ffd**。


```c
static inline void ep_set_ffd(struct epoll_filefd *ffd, struct file *file, int fd)
{
    ffd->file = file;
    ffd->fd = fd;
}
```


#### 2. 设置socket等待队列


在创建完epitem并初始化之后，ep_insert中第二件事就是**设置socket对象上的等待队列，并把函数ep_poll_callback设置为数据就绪时候的回调函数**。


首先是**创建了ep_pqueue，并调用init_poll_funcptr初始化它的poll_table成员**


```c
init_poll_funcptr(&epq.pt, ep_ptable_queue_proc);

static inline void init_poll_funcptr(poll_table *pt, poll_queue_proc qproc)
{
    pt->_qproc = qproc;
    pt->_key = ~0UL; // all events enabled
}
```


qproc是个函数指针，这里将其设置为ep_ptable_queue_proc，fd会在poll的时候执行这个函数。key是触发的事件。


接着**调用了ep_item_poll函数，将epitem和poll_table关联在一起，主要是设置事件到来时的回调函数ep_poll_callback**


```c
static inline unsigned int ep_item_poll(struct epitem *epi, poll_table *pt)
{
    pt_key = epi->event.events
    return epi->ffd.file->f_op->poll(epi->ffd.file, pt) & epi->event.events;
}
```


file_operations的**poll是驱动提供给应用程序探测设备文件是否有数据可读的接口**，这里的实现是sock_poll，而sock_poll函数中最后调用的是sock->ops->poll，实际上指向的是tcp_poll


```c
unsigned int tcp_poll(struct file *file, struct socket *sock, poll_table *wait)
{
    struct sock *sk = sock->sk;
    sock_poll_wait(file, sk_sleep(sk), wait);
}
```


同阻塞模式中的相同，**sk_sleep用于返回sock对象下的等待队列的列表头wait_queue_head_t，稍后的等待队列项wait就插在这里**。


```c
static inline void sock_poll_wait(struct file *filp
		wait_queue_head_t *wait_address, poll_table *p)
{
    poll_wait(filp, wait_address, p);
}

static inline void poll_wait(struct file *filp, wait_queue_head_t *wait_address, poll_table *p)
{
    if(p && p->qproc && wait_address)
	p->_qproc(filp, wait_adresss, p);
}
```


**poll_wait函数是用来将一个等待事件添加到等待队列中（通过调用前面的init_poll_funcptr中设置的ep_ptable_queue_proc），并不会使进程进入休眠状态。**


```c
static void ep_ptable_queue_proc(struct file *file, wait_queue_head_t *whead,
			poll_table *pt)
{
    struct epitem *epi = ep_item_from_epqueue(pt); // 依赖于containerof实现的
    struct eppoll_entry *pwq;
    if(epi->nwait >= 0 && (pwq = kmem_cache_alloc(pwq_cache, GFP_KERNEL))) {
 	// 初始化一个等待队列的节点，其中注册的回调为ep_poll_callback
	init_waiqueue_func_entry(&pwq->wait, ep_poll_callback);
	pwq->whead = whead;  //这个就是监控的fd的等待队列头
	pwq->base = epi; 
	// 把初始化的等待对列项插入到所监控的fd的等待对列中
    	add_wait_queue(whead, &pqw->wait);
    }
}

struct eppoll_entry {
    // 用于将该结构体链接到struct epitem的列表头
    struct list_head llink;
    // base指针指向struct epitem
    struct epitem *base;
    // 等待队列项
    wait_queue_t wait;
    // 等待队列头
    wait_queue_head_t *whead;
};
```


struct eppoll_entry就是做一个epitem和其回调之间的关联，其中定义了等待队列项和等待队列头以及指向epitem的指针。


首先通过init_waiqueue_func_entry**设置eppoll_entry中wait等待项的回调函数为ep_poll_callback**


```c
static inline void init_waitqueue_func_entry(wait_queue_t *q, wait_queue_func_t func)
{
    q->flags = 0;
    q->private = NULL;
    q->func = func;
}
```


随后通过poll_table拿到的epitem以及前面sk_sleep(sk)拿到的socket的等待队列给eppoll_entry赋值。


最后再调用add_wait_queue将等待项加入等待队列。


总的一句话，ep_ptable_queue_proc函数就是用于新建一个等待队列项并注册其回调函数为ep_poll_callback，然后再将其加入等待队列中


> ep_item_from_epqueue(pt)具体的实现依赖于 container_of 宏，该宏是用来获取包含某个成员的结构体的地址的。
>
> container_of 的基本原理是，它通过成员在结构体中的偏移量来计算出结构体的起始地址。如果你知道一个结构体中成员的地址，并且知道这个成员在结构体中的偏移量，你就可以计算出这个结构体的起始地址。
>
> 而epoll_table是包含在ep_pqueue结构体中的，根据epoll_table的地址以及其在ep_pqueue中的偏移量就可以找到包含它的ep_pqueue对象，最后就可以拿到这个对象保存的epitem


前面阻塞模式中调用recvfrom也会创建这么一个等待项加入等待队列中，不过当时是需要在数据就绪的时候就唤醒进程，所以将等待项的private设置成当前用户进程current。然而这**里socket交给epoll来管理的，不需要在一个socket就绪的时候就唤醒进程**，所以这里的private没有作用，设置为NULL。而这里的func是epoll_call_back，即就绪时会去执行这个回调函数。


总的来说设置socket等待队列有如下步骤：


1. 创建ep_pqueue对象，初始化其epitem成员
2. 初始化ep_pqueue的poll_table成员，将poll_table的proc函数指针赋值为ep_ptable_queue_proc
3. 调用ep_item_poll，最终会执行ep_ptable_queue_proc函数

    1. 创建eppoll_entry对象
    2. 初始化wait等待队列项成员，设置其回调函数ep_poll_callback
    3. 绑定epitem和等待队列
    4. 将等待队列项加入等待队列


#### 3. 插入红黑树


分配完epitem，并且设置好了对应socket的等待队列，紧接着就把它插入红黑树，具体插入方式不在这里展开。


红黑树的具体组成是这样的：


1. eventpoll中包含了struct rb_root类型的成员，即红黑树的根（整棵树的抽象）
2. struct rb_root会包含一个rb_node类型的成员，即红黑树的节点，也是红黑树的根节点
3. rb_node嵌入在了epitem结构体中，通过container_of宏可以拿到对应节点的epitem对象


> 树中的每个rb_node节点都是嵌入在epitem中的，当我们在创建epitem时，就会开辟并初始化一个rb_node类型的成员，此时rbn就被嵌入在epitem中，和epitem使用同一块内存。换句话说，rbn的地址就是epitem起始地址加上rbn在epitem中的偏移。
>
> 即使rb_node的指针在别的地方也被使用（例如被rb_node的父节点或子节点引用），container_of宏仍然能够从rb_node的地址正确地计算出epitem的地址，因为rb_node的地址就是它在epitem中的位置（当然自生效于上述说的情况，如果单独去创建一个rb_node自然不会这样）。这就是container_of宏为何能正确工作的原因。
>
> 所以，不论rb_node被如何引用，只要我们知道它是被嵌入在epitem中的，我们就可以从rb_node的地址计算出epitem的地址。这就是container_of宏的作用。


之所以使用红黑树是因为其在查找效率、插入效率、内存开销等方面比较均衡，并且内部的节点是有序的，这使得它可以很容易地进行排序操作。epoll需要按照文件描述符的顺序返回事件，这就需要使用一个可以维护排序的数据结构。而哈希表则无法提供这样的排序功能。


### 3）epoll_wait之等待接收


epoll_wait做的事情不复杂，当它被调用时会去**观察eventpoll_rdllist链表里有没有数据。有数据就返回，没有数据就创建一个等待队列项，将其添加到eventpoll的等待队列项，然后把自己阻塞掉**。


> epoll_ctl添加socket的时候也创建了等待队列项，不同的是这里的等待队列项是挂在epoll对象上的，而前者是挂在socket对象上的。


具体代码如下：


```c
SYSCALL_DEFINE4(epoll_wait, int, epfd, ...)
{
    ......
    error = ep_poll(ep, events, maxevents, timeout);
}

static int ep_poll(struct eventpoll* ep, ...)
{
    wait_queue_t wait;
    ......
fetch_events:
    // 1.判断就绪队列上是否有事件就绪
    if(!ep_events_available(ep)) {
	// 2.定义等待事件并关联到当前进程
	init_waiqueue_entry(&wait, current);
   	// 3.添加到epoll->wq等待队列链表
	_add_wait_queue_exclusive(&ep->wq, &wait);
	for (;;) {
	    ......
	    // 4.更改当前进程状态为可打断
	    set_current_state(TASK_INTERRUPTIBLE);
	    // 5.让出CPU，主动进入睡眠状态
	    if(!schedule_hrtimeout_range(to, slack, HRTIMER_MODE_ABS))
		timed_out = 1;
	    ......
}
```


等待项中设置的回调函数为default_wake_function


### 4）数据到达


在前面epoll_ctl执行的时候，内核为每一个socket都添加了一个等待队列项。在epoll_wait运行完的时候，又在event_poll对象上添加了等待队列项。


相应的函数指针：


* socket->sock->sk_data_ready设置的就绪处理函数是sock_def_readable（数据到达时触发）
* 在socket的等待队列项中，其回调函数是ep_poll_callback。另外其private指向的是空指针（sock_def_readable中调用）
* 在eventpoll的等待队列项中，其回调函数是default_wake_function。其private指向的是当前的用户进程（ep_poll_callback中调用）


详细情况如下：


当数据到达后，经过网卡、硬中断、中断不断将向上层传递，最后和同步阻塞的实现相同，又来到了tcp_rcv_established函数（针对于tcp数据包且已建立连接的情况）中去调用tcp_queue_rcv函数实现将数据包放到sock的接收队列，再调用sk->sk_data_ready，即前面说到的sock_def_readable来尝试唤起等待的进程，同样这里经过层层调用最终来到_wake_up_common方法中去调用等待队列中等待队列项的回调函数。


从这里开始才和同步阻塞方式不同，在同步阻塞方式中这里的回调函数是autoremove_wake_function，而**在epoll中这里是ep_poll_callback**，具体实现如下：


```c
static int ep_poll_callback(wait_queue_t *wait, unsigned mode, int sync, void *key) // key是events，即发生的事件
{
    int pwake = 0;
    unsigned long flags;
    struct epitem *epi = ep_item_from_wait(wait); // 由等待队列中获取对应的epitem，依赖containerof实现的
    struct eventpoll *ep = epi->ep;

    spin_lock_irqsave(&ep->lock, flags); // 上锁

    if (!(epi->event.events & ~EP_PRIVATE_BITS)) // 如果所监控的fd没有注册什么事件的话,就不加入rdllist中
	goto out_unlock;
    if (key && !((unsigned long) key & epi->event.events)) // 所触发的事件我们没有注册的话当然也不加入rdllist中
	goto out_unlock;

    /*
    * 在epoll_wait中ep->ovflist初始化为EP_UNACTIVE_PTR,而当epoll_wait被唤醒后处理rdlist时会将ep->ovflist置为NULL
    * 也就是说如果ep->ovflist != EP_UNACTIVE_PTR意味这epoll_wait已被唤醒 正在执行loop
    * 此时我们就把在rdllist遍历时发生的事件用ovflist串起来,在遍历结束后插入rdllist中
    */
    if (unlikely(ep->ovflist != EP_UNACTIVE_PTR)) {
	if (epi->next == EP_UNACTIVE_PTR) {
	    epi->next = ep->ovflist;
	    ep->ovflist = epi;
	}
	goto out_unlock;
    }

    // 将当前的epitem加入到rdllist中
    // 这就是epoll如此高效的原因,我们不必每次去遍历fd寻找触发的事件,触发事件时会触发回调自动把epitem加入到rdllist中,
    // 这使得复杂度从O(N)降到了O(有效事件集合),且我们不必每次注册事件,仅在epoll_ctl(ADD)中注册一次即可(修改除外),
    if (!ep_is_linked(&epi->rdllink)) 
	list_add_tail(&epi->rdllink, &ep->rdllist);

    // 查看eventpoll等待队列上是否有等待
    if (waitqueue_active(&ep->wq))
	wake_up_locked(&ep->wq);
    if (waitqueue_active(&ep->poll_wait))
	pwake++;

out_unlock:
    spin_unlock_irqrestore(&ep->lock, flags);
  
    if (pwake)
	ep_poll_safewake(&ep->poll_wait);

    return 1;
}
```


主要工作其实就是**把自己的epitem添加到epoll的就绪队列中（还需要事先判断发生的事件是否有注册监听，有才会放到就绪队列）**，接着再**查看eventpoll对象上的等待队列里是否有等待项**（epoll_wait执行的时候设置的），如果没有等待项，那么软中断的事情就做完了。如果有等待项，那么就通过wake_up_lockced**找到等待项里设置的回调函数**。


依次调用wake_up_locked => __wake_up_locked() => __wake_up_common。


```c
static void __wake_up_common(wait_queue_head_t *q, unsigned int mode,
		int nr_exclusive, int wake_flags, void *key)
{
    wait_queue_t *curr, *next;
    list_for_each_entry_safe(curr, next, &q->task_list, task_list) {
	unsigned flags = curr->flags;
	if(curr->func(curr, mode, wake_flags, key) &&
	    (flags & WQ_FLAG_EXCLUSIVE) && !--nr_exclusive)
	    break;
    }
}
```


即去**遍历等待队列，找到第一个合适的进程，调用它的回调函数，即default_wake_function**。


default_wake_function会传入等待项的private指针，去调用try_to_wake_up将因为等待而被阻塞的进程唤醒


唤醒之后epoll_wait进程就进入可运行队列，等待内核重新调度。这个进程重新运行之后就会从epoll_wait的后续代码继续执行，主要是**将等待项从队列中移除，设置进程状态为TASK_RUNNING，最后调用ep_send_events给用户进程返回就绪事件**。


>


### 5）小结


![在这里插入图片描述](/images/linux-network-multiplexing-epoll/0c528560c3866e53adb20e0f721eaad6.png)



epoll的使用主要步骤如下


用户进程：


1. 创建epoll对象：int epoll_create(int size) // 监听的数目
2. 注册要监听的套接字：int epoll_ctl(int epfd, int op, int fd, struct epoll_event *event) // epoll文件描述符、操作类型、socket文件描述符、要监听的事件类型

    * EPOLLIN ：表示对应的文件描述符可以读（包括对端SOCKET正常关闭）；
    * EPOLLOUT：表示对应的文件描述符可以写；
    * EPOLLPRI：表示对应的文件描述符有紧急的数据可读（这里应该表示有带外数据到来）；
    * EPOLLERR：表示对应的文件描述符发生错误；
    * EPOLLHUP：表示对应的文件描述符被挂断；
    * EPOLLET： 将EPOLL设为边缘触发(Edge Triggered)模式，这是相对于水平触发(Level Triggered)来说的。
    * EPOLLONESHOT：只监听一次事件，当监听完这次事件之后，如果还需要继续监听这个socket的话，需要再次把这个socket加入到EPOLL队列里
3. 检查就绪队列：int epoll_wait(int epfd, struct epoll_event *events, int maxevents, int timeout) // epoll文件描述符、检查的事件类型、events数组的大小（即可以接受的事件数量）、最大等待时间，返回要处理的事件数目

    * 队列有数据，直接返回
    * 队列没数据，将自己加入等待队列，放弃CPU


内核进程：


1. 数据包到达网卡
2. 一些列处理后软中断将数据放到socket的接收队列
3. 触发socket的sk_data_ready，即调用sock_def_readable（sock初始化时设置的）

    * 调用socket等待队列项的回调函数，即ep_poll_callback（调用epoll_ctl的时候设置的）

      * 调用eventpoll等待队列项的回调函数，即default_wake_function（调用epoll_wait的时候设置的）
4. 将等待项移除，唤醒进程


## 二、问题解答


1. 阻塞到底是怎么一回事

    * 进程因为等待某个事件而主动让出CPU挂起的操作
    * 在网络IO中，当进程等待的socket上的数据没有来时，就把当前进程状态从TASK_RUNNING改为TASK_INTERRUPTIPLE，然后主动让出CPU，由调度器重新调度下一个就绪状态的进程
    * socket的读写操作也可以设置为非阻塞，那么如果数据没有到达就直接返回空，而不是放弃CPU阻塞等待
2. 同步阻塞IO都需要哪些开销

    * 调用recv时如果没有数据，需要一次进程上下文切换的开销
    * 当数据到达时，需要一次进程上下文切换的开销
    * 数据拷贝的开销
    * 进程占用内存的开销，如果有很多并发，则需要很多进程
3. 多路复用epoll为什么就能提高网络性能

    * 极大程度地减少了无用地进程上下文切换，让进程更专注地处理网络请求
    * 使用epoll可以实现调用读写操作地套接字的事件都已经就绪，避免阻塞等待去切换上下文
4. epoll也是阻塞的吗

    * 是的，如果没有数据到来，没有事情可干阻塞是正常的
5. redis为什么网络性能突出

    * Redis的主要逻辑就是在本机内存上的数据结构读写，几乎没有网络IO和磁盘IO，单个请求处理起来很快。所以它把主服务端程序干脆就做成了单进程的，这样就省去了多进程之间协作的负担，也更大程度减少了进程切换。
    * 进程主要的工作就是使用epoll_wait等待时间，又了事件以后处理，处理完了之后再调用epoll_wait。一直工作到没有请求要处理或时间片用完才让出CPU，让工作效率发挥到机制。
    * 其他一些网络IO框架一般是多进程的配合，谁来等待事件，谁来处理事件，谁来发送结果，就是经常听到的各种Reactor、Proactor模型。这就会有通信开销，以及可能带来进程上下文切换CPU的消耗。



**参考资料**：


[struct socket 结构详解 - stardsd - 博客园 (cnblogs.com)](https://www.cnblogs.com/sddai/p/5790414.html)


[epoll源码解析(2) epoll_ctl - 李兆龙的博客 - 博客园 (cnblogs.com)](https://www.cnblogs.com/lizhaolong/p/16437326.html)


[LINUX epoll实现原理介绍_jerry_chg的博客-CSDN博客](https://blog.csdn.net/lickylin/article/details/123195292)


《深入理解Linux网络》—— 张彦飞

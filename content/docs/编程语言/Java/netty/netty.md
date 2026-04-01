---
title: "基础入门"
date: 2023-10-25T01:28:13+08:00
draft: false
summary: "Netty 框架的基础入门，介绍 Netty 的异步回调模型，包括对 Java Future 接口的扩展、GenericFutureListener 非阻塞回调机制，以及 Netty 自有 Future 接口的增强方法。"
tags: [Netty]
categories: [Java, Networking]
source: csdn
source_id: "134025561"
---

# 一、Netty的异步回调模式


Netty继承和扩展了JDK Future系列异步回调的API，定义了自身的Futrue系列接口和类，实现了异步任务的监控、异步执行结果的获取。总体来说Netty对Java Future异步任务的扩展如下：


1. 继承Java的Future接口，得到了一个新的属于Netty自己的Future异步任务接口，该接口对原有接口进行了增强，使得Netty异步任务能够以非阻塞的方式处理回调的结果
2. 引入了一个新街口——GenericFutureListener，用于表示异步执行完成的监听器。这个Netty使用了监听器的模式，异步任务的执行完成后的回调逻辑抽象成了Listener监听器接口。可以将Netty的GenericFutureListener监听器接口加入Netty异步任务Future中，实现对异步任务执行状态的事件监听


## 1、GenericFutureListener接口


```java
public interface GenericFutureListener<F extends Future<?>> extends EventListener {
    // 监听器的回调方法
	void operationComplete(F var1) throws Exception;
}
```


GenericFutureListener拥有一个回调方法operationComplete，表示异步任务操作完成。在Future异步任务执行完成后将回调执行此方法。在大多数情况下，Netty的异步回调代码编写在GenericFutureListener接口的实现类中的operationComplete方法中。


它的父接口EventListener是一个空接口，没有任何的抽象方法，是一个仅仅具有标识作用的接口。


## 2、Future接口


Netty也定义了自己的Future接口，对JDK原有的Future接口进行了扩展：


```java
public interface Future<V> extends java.util.concurrent.Future<V> {
	boolean isSuccess(); // 判断异步执行是否成功
	boolean isCancellable(); // 判断异步执行是否取消
 	Throwable cause(); // 获取异步任务异常的原因
	// 增加异步任务执行完成与否的监听器
	Future<V> addListener(GenericFutureListener<? extends Future<? super V>> listener);
	// 溢出异步任务执行完成与否的监听器
	Future<V> removeListener(GenericFutureListener<? extends Future<? super V>> listener);
}
```


Netty的Future接口一般不会直接使用，而是会使用子接口。Netty有一系列的子接口，代表不同类型的异步任务，如ChannelFuture接口。


## 3、ChannelFuture的使用


在Netty的网络编程中，网络连接通道的输入和输出处理都是异步进行的，都会返回一个ChannelFuture接口的实例。通过返回的异步任务实例，可以为它增加异步回调的监听器。在异步任务真正完成后，回调才会执行。


```java
// connect是异步的，仅提交异步任务
ChannelFuture future = bootstrap.conncect(new InetSocketAddress("www.wanning.com", 80));
// connect的异步任务真正执行完成后，future回调监听器才会执行
future.addListener(new ChannelFutureListener() {
	@Override
	public void operationComplete(ChannelFuture channelFuture) throws Exception {
		if(channelFuture.isSuccess()) {
			...
		}
	}
});
```


GenericFutureListener接口在Netty中是一个基础类型接口。在网络编程的异步回调中，一般使用Netty中提供的某个接口，如ChannelFutureListener接口。


# 二、Netty中的Reactor反应器模式


## 1、Channel通道组件


Channel通道主键是Netty中非常重要的主键，反应器模式和通道紧密相关，反应器的查询和分发的IO事件都来自于Channel通道组件。


Netty中不直接使用Java NIO的Channel通道组件，对Channel通道组件进行了自己的封装。在Netty中，有一系列的Channel通道组件，为了支持多种通信协议，或者说对于每一种通信连接协议，Netty都实现了自己的通道。


另外一点就是除了Java的NIO，Netty还能处理Java的面向流的OIO。总的来说Netty中每一种协议的通道，都有NIO和OIO两个版本。


* NioSocketChannel：异步非阻塞TCP Socket传输通道
* NioServerSocketChannel：异步非阻塞TCP Socket服务器端监听通道
* NioDatagramChannel：异步非阻塞的UDP传输通道
* NioSctpChannel：异步非阻塞Sctp传输通道
* NioSctpServerChannel：异步非阻塞Sctp服务器端监听通道
* OioSocketChannel：同步阻塞式TCP Socket传输通道
* OioServerSocketChannel：同步阻塞式TCP Socket服务器端监听通道
* OioDatagramChannel：同步阻塞式UDP传输通道
* OioSctpChannel：同步阻塞式Sctp传输通道
* OioSctpServerChannel：同步阻塞式Sctp服务器端监听通道


一般来说，服务器端编程用到最多的通信协议还是TCP协议，对应的传输通道类型是NioSocketChannel，服务器监听类为NioServerSocketChannel。在主要使用的方法上，其他的通道类型和这个NioSocketChannel类在原理上基本是相通的。**在Netty的NioSocketChannel内部封装了一个Java NIO的SelectableChannel成员，通过这个内部的Java NIO通道，Netty的NioSocketChannel通道上的IO操作，最终会落地到Java NIO的SelectableChanne底层通道。**


## 2、Reactor反应器


在反应器模式中，一个反应器会负责一个事件处理线程，不断地轮询，通过Selector选择器不断查询注册过的IO事件（选择键）。如果查询到IO事件，则分发给Handler业务处理器。


Netty中的反应器有多个实现类，与Channel通道类有关系。**对应于NioSocketChannel通道，Netty的反应器类为NioEventLoop。NioEventLoop类绑定了两个重要的Java成员属性：一个是Thread线程类的成员，一个是Java NIO选择器的成员属性。**


也就是说一个NioEventLoop拥有一个Thread线程，负责一个Java NIO Selector选择器的IO事件轮询。在Netty中，一个EventLoop反应器和Channel通道是一对多的关系，一个反应器可以注册成千上万的通道。


## 3、Handler处理器


Java NIO可供选择器监控的通道IO事件类型包括以下4种：


* 可读：SelectionKey.OP_READ
* 可写：SelectionKey.OP_WRITE
* 连接：SelectionKey.OP_CONNECT
* 接收：SelectionKey.OP_ACCEPT


**在Netty中，EventLoop反应器内部有一个Java NIO选择器成员执行以上事件的查询，然后进行对应的事件分发。事件分发的目标就是Netty自己的Handler处理器。**


**Netty的Handler处理器分为两大类，第一类是ChannelInboundHandler通道入站处理器，第二类ChannelOutboundHandler通道出站处理器。二者都继承了ChannelHandler处理器接口。**Netty中的入站处理，不仅仅是OP_READ输入事件的处理，还是从通道底层触发，由Netty通过层层传递，调用ChannelInboundHandler通道入站处理器进行的某个处理。以底层的Java NIO中的OP_READ输入事件为例：在通道中发生了OP_READ事件后会被EventLoop查询到，然后分发给ChannelInboundHandler通道入站处理器，调用它的入站处理的方法read。在ChannelInboundHandler通道入站处理器内部的read方法可以从通道中读取数据。


Netty中的入站处理，触发的方向为：从通道到ChannelInboundHandler通道入站处理器。


Netty中的出站处理，本来就包括Java NIO的OP_WRITE可写事件。不过OP_WRITE可写事件是Java NIO的底层概念，它和Netty的出站处理的概念不是一个维度的，Netty的出站处理是应用层维度的，具体指的是从ChannelOutboundHandler通道出站处理器到通道的某次IO操作。在应用程序完成业务处理后，可以通过ChannelOutboundHandler通道出站处理器将处理的结果写入底层通道，它的最常用的一个方法就是write()方法，把数据写入到通道。


这两个业务处理接口都有各自的默认实现：**ChannelInboundHandler的默认实现为ChannelInboundHandlerApater，叫作通道入站处理适配器。ChannelOutboundHandler的默认实现ChannelOutboundHandlerAdapter，叫作通道出站处理适配器。这两个默认的通道处理器适配器，分别实现了入站操作和出站的基本功能。如果要实现自己的业务处理器，不需要从零开始去实现处理器的接口，只需要继承通道处理器适配器即可。**


## 4、Netty的流水线（Pipeline）


首先梳理一下Netty反应器模式中各个组件之间的关系：


1. 反应器和通道之间是一对多的关系，一个反应器可以查询很多个通道的IO事件
2. 通道和Handler处理器实例之间是多对多的关系，一个通道的IO事件被多个的Handler实例处理器，一个Handler处理器实例也可以绑定到很多的通道，处理多个通道的IO事件。


问题是通道和Handler处理器实例之间的绑定关系，Netty是如何组织的呢？


**Netty设计了一个特殊的组件，叫做ChannelPipeline（通道流水线），它像一条管道，将绑定到一个通道的多个Handler处理器实例，串在一起形成一条流水线。ChannelPipeline的默认实现实际上被设计成一个双向链表。所有的Handler处理器实例被包装成了双向链表的节点，被加入到了ChannelPipeline中。也就是说一个Netty通道拥有一条Handler处理器流水线，成员的名称叫作pipeline。**


以入站处理为例，每一个来自通道的IO事件，都会进入一次ChannelPipeline通道流水线，在进入第一个Handler处理器后，这个IO事件将按照既定的从前往后次序，在流水线上不断地向后流动，流向一个Handler处理器。在向后流动的过程中，会出现3种情况：


1. 如果后面还有其他Handler入站处理器，那么IO事件可以交给下一个Handler处理器向后流动
2. 如果后面没有其他的入站处理器，就意味着这个IO事件在此次流水线中的处理结束了
3. 如果在流水线中间需要终止流动，可以选择不将IO事件交给下一个Handler处理器，流水线的执行也就被终止了


总之流水线是通道的大管家，为通道管理好了它的一大堆Handler。


# 三、Bootstrap启动器类


Bootstrap类是Netty提供的一个便利的工厂类，可以通过它来完成Netty的客户端或服务器端的Netty组件的组装以及Netty程序的初始化。当然我们也可以不用这个Bootstrap启动器，但是一点点去手动创建通道、完成各种设置和启动、并且注册到EventLoop，这个过程会非常麻烦。


在Netty中有两个启动器类，分别用在服务器和客户端，这两个启动器仅仅是使用的地方不同，它们大致的配置和使用方法都是相同的。


## 1、父子通道


在Netty中每一个NioSocketChannel通道所封装的是Java NIO通道，再往下就对应到操作系统底层的socket描述符。理论上来说，操作系统底层的socket描述符分为两类：连接监听类型和传输数据类型。在Netty中，异步非阻塞的服务器端监听通道NioServerSocketChannel，封装在Linux底层的描述符，是连接监听类型的socket描述符。而NioSocketChannel异步非阻塞TCP Socket传输通道，封装在底层Linux的描述符，是数据传输类型的socket描述符。


在Netty中，NioServerSocketChannel负责服务器连接监听和接收，也叫父通道。对于每一个接收到的NioSocketChannel传输类通道，也叫子通道。


## 2、EventLoopGroup线程组


Netty的Reactor反应器模式是多线程版本的反应器模式，在Netty中，一个EventLoop相当于一个子反应器，一个NioEventLoop子反应器拥有了一个线程，同时拥有一个Java NIO选择器。多个EventLoop线程组成一个EventLoopGroup线程组。


反过来说，Netty的EventLoopGroup线程组就是一个多线程版本的反应器。Netty的程序开发不会直接使用单个EventLoop线程，而是使用EventLoopGroup线程组。EventLoopGroup的构造函数有一个参数，用于指定内部的线程数。在构造器初始化时，会按照传入的线程数量，在内部构造多个Thread线程和多个EventLoop子反应器，进行多线程的IO时间查询和分发。如果使用无参构造函数或传入线程数为0，那么默认EventLoopGroup内部线程数为最大可用CPU处理器数量的两倍。


在服务器端一般有两个独立的反应器，一个负责新连接的监听和接收，一个负责IO事件处理。对应到Netty服务器程序中则是设置两个EventLoopGroup线程组。负责新连接的监听和接受的线程组查询父通道的IO事件，称为Boss线程组。另一个线程组负责查询所有子通道的IO事件，并且执行Handler处理器中的业务处理，例如数据的输入和输出，称为Worker线程组。


## 3、Bootstrap启动流程


1. 创建反应器线程组，并赋值给ServerBootstrap启动器实例

    1. 通过NioEventLoopGroup创建两个线程组
    2. 通过bootstrap.group配置线程组，可以只配置一个反应器线程组，这种模式下连接监听IO事件和数据传输IO事件在同一个线程中处理，会导致连接的接受被更加耗时的数据传输或业务处理所阻塞
2. 设置通道的IO类型

    1. Netty不止支持Java NIO，也支持OIO，因此需要进行配置
    2. 通过bootstrap.channel()方法，传入通道的class类文件，如bootstrap.channel(NioServerSocketChannel.class)
3. 设置监听端口：bootstrap.localAddress(new InetSocketAddress(port))
4. 设置传输通道的配置选项：bootstrap.option用于给父通道接收连接通道设置一些选项，bootstrap.childOption设置子通道设置一些通道选项

    1. bootstrap.option(ChannelOption.SO_KEEPALIVE, true)：开启心跳机制
    2. bootstrap.option(ChannelOption.SO_ALLOCATOR, PooledByteBufAllocator.DEFAULT)
5. 装配子通道的Pipeline流水线：装配子通道的Handler流水线调用childHandler()方法，传递一个ChannelInitializer通道初始化类的实例。在父通道成功接收一个连接，并创建成功一个子通道后，就会初始化子通道，这里配置的ChannelInitializer实例就会被调用

    1. 在ChannelInitializer通道初始化类的事例中，有一个initChannel初始化方法，在子通道创建后被执行到，向子通道流水线增加业务处理器
    2. 也可以调用handler为付通道设置ChannelInitializer初始化器，但是父通道接受新连接后除了初始化子通道一般不需要特别的配置
    3. ```java
        // 裝配子通道流水綫
        bootstrap.childHandler(new ChannelInitializer<SocketChannel>() {
            // 有连接到达时会创建一个通道的子通道，并初始化
            protected void initChannel(SocketChannel ch) throws Exception {
        		// 流水线管理子通道中的Handler业务处理器
        		// 向子通道流水线添加一个Handler业务处理器
        		ch.pipeline().addLast(new NettyDiscardHandler());
        	}
        });
        ```
6. 开始绑定服务器连接的监听端口

    1. 开始绑定端口，通过调用sync同步方法阻塞直到绑定成功：bootstrap.bind().sync()
    2. bind返回一个端口绑定Netty的异步任务channelFuture，在Netty中所有的IO操作都是异步执行的，可以通过自我阻塞一直到ChannelFuture异步任务执行完成或者为ChannelFuture增加事件监听器的两种方式以获得Netty中的IO操作的真正结果
7. 自我阻塞，直到通道关闭

    1. channelFuture.channel().closeFuture().sync()
    2. 如果要阻塞当前线程直到通道关闭，可以使用通道的closeFuture方法，以获取通道关闭的异步任务，当通道被关闭时，closeFuture实例的sync方法会返回
8. 关闭EventLoopGroup

    1. loopGroup.shutdownGracefully
    2. 关闭Reactor反应器线程组，同时会关闭内部的子反应器线程，也会关闭内部的Selector选择器、内部的轮询线程以及负责查询的所有的子通道。在子通道关闭后会释放所有的资源，如TCP Socket文件描述符等


## 4、ChannelOption通道选项


无论是对于NioServerSocketChannel父通道类型，还是对于NioSocketChannel子通道类型，都可以设置一系列的ChannelOption选项，在ChannelOption类中，定义了一些通道选项：


1. SO_RCVBUF，SO_SNDBUF：此为TCP参数，每个TCP套接字在内核中都有一个发送缓冲区和一个接收缓冲区，这两个选项就是用来设置TCP连接的这两个缓冲区大小的
2. TCP_NODELAY：此为TCP参数，表示立即发送数据，默认值为true（Netty默认值为true，而操作系统默认值为false），该值用于设置Nagle算法的启用，该算法将小的碎片数据连接成更大的报文，来最小化所发送报文的数量。如果需要发送一些娇小的报文，则需要禁用该算法。
3. SO_KEEPALIVE：此为TCP参数，表示底层TCP协议的心跳机制，true为连接保持心跳，默认值为false。启用该功能时TCP会主动探测空闲连接的有效性（默认心跳间隔为7200s）。Netty默认关闭该功能
4. SO_REUSEADDR：此为TCP参数，设置为true时表示地址复用，默认值为false。由四种情况需要用到这个参数设置：

    1. 当有一个由相同本地地址和端口的socket处于TIME_WAIT状态时，而我们希望启动的程序socket2要占用该地址和端口，例如在重启服务且保持先前端口时
    2. 有多块网卡或用IP Alias技术的机器在同一端口启动多个进程，但每个进程绑定的本地IP地址不能相同
    3. 单个进程绑定相同的端口到多个socket上，但每个socket绑定的IP地址不同
    4. 完全相同的地址和端口的重复绑定，但这只用于UDP的多播，不用于TCP
5. SO_LINGER：此为TCP参数，表示关闭socket的延迟时间，默认值为-1，表示禁用该功能。-1表示socke.close方法立即返回，但操作系统底层会将发送缓冲区全部发送到对端。0表示socket.close方法立即放回，操作系统放弃发送缓冲区的数据，直接向对端发送RST包，对端收到复位错误。非0整数值表示调用socket.close方法的线程被阻塞，直到延迟时间到来、发送缓冲区的数据发送完毕，若超时，则对端会收到服务复位错误。
6. SO_BACKLOG：此为TCP参数，表示服务器端接收连接的队列长度，如果队列已满，客户端连接将被拒绝。在Windows中默认值为200，其他操作系统为128.如果连接建立频繁，服务器处理新连接较慢，可以适当调大这个参数
7. SO_BROADCAST：此为TCP参数，表示设置广播模式


# 四、Channel通道


## 1、主要成员和方法


在Netty中，通道是其中的核心概念之一，代表着网络连接。它负责同对端进行网路通信，可以写入数据到对端，也可以从对端读取数据。


通道的抽象类AbstractChannel的构造函数如下：


```java
protected abstract Channel(Channel parent) {
    this.parent = parent; // 父通道
	id = newId();
	unsafe = newUnsafe() // 底层NIO通道，完成实际的IO操作
	pipeline = new ChannelPipeline(); // 一条通道，拥有一条流水线
}
```


AbstractChannel内部有一个pipeline属性，表示处理器的流水线。Netty在对通道进行初始化的时候，会将pipeline属性初始化为DefaultChannelPipeline的实例。


AbstractChannel内部有一个parent属性，表示通道的父通道。对于连接监听通道来说，其父通道为null，而对于每一个传输通道，其parent属性的值为接收到该连接的服务器连接监听通道。


几乎所有的通道实现类都继承了AbstractChannel抽象类，都拥有上面的parent和pipeline两个属性成员。


再来看一下，在通道接口中所定义的几个重要方法：


1. ChannelFuture connect(SocketAddress address)：连接远程服务器，方法的参数为远程服务器的地址，调用后会立即返回，返回值为负责连接操作的异步任务ChannelFuture。此方法在客户端的传输通道使用
2. ChannelFuture bind(SocketAddress address)：绑定监听地址，开始监听新的客户端连接，此方法在服务器的新连接监听和接收通道使用
3. ChannelFuture close()：关闭通道连接，返回连接关闭的ChannelFuture异步任务。如果需要在连接正式关闭后执行其他操作，则需要为异步任务设置回调方法，或者调用ChannelFuture异步任务的sync()方法来阻塞当前线程，一直等到通道关闭的异步任务执行完毕
4. Channel read()：读取通道数据，并且启动入站处理。具体来说从内部的Java NIO Channel通道读取数据，然后启动内部的Pipeline流水线，开始数据读取的入站处理，此方法的返回通道自身用于链式调用
5. ChannelFuture write(Object obj)：启程出站流水处理，把处理后的最终数据写到底层Java NIO通道。此方法的返回值为出站处理的一部处理任务。
6. Channel flush()：将缓冲区中的数据立即写出到对端。并不是每一次write操作都是将数据直接写出到对端，write操作的作用大部分情况下仅仅是写入到操作系统的缓冲区，操作系统会根据缓冲区的情况决定什么时候把数据写到对端，而执行flush()方法立即将缓冲区的数据写到对端。


## 2、EmbeddedChannel嵌入式通道


在Netty的实际开发中，通信的基础工作已经由Netty完成了。实际上大量的工作时设计和开发ChannelHandler通道业务处理器，而不是开发Outbound出站处理器，换句话就是开发Inbound入站处理器。开发完成之后需要投入单元测试，将Handler业务处理器加入到通道的Pipeline流水线中，然后启动Netty服务器、客户端程序，相互发送消息，测试业务处理器的效果。如果每开发一个业务处理器，都进行服务器和客户端的重复启动，那么是非常繁琐和浪费时间的。


因此Netty提供了一个专用通道EmbededChannel，它仅仅是模拟入站和出站的操作，底层不进行实际的传输，不需要启动Netty服务器和客户端。除了不进行传输之外，EmbeddedChannel的其他的事件机制和处理流程和真正的传输通道是一模一样的。因此开发人员可以在开发过程中方便快捷地进行ChannelHandler业务处理器的单元测试。具体提供的方法如下：


* writeInbound：向通道写入inbound入站数据，模拟通道收到数据。也就是说这些写入的数据会被流水线上的入站处理器处理
* readInbound：从EmbeddedChannel中读取入站数据，返回经过流水线最后一个入站处理器完成之后的入站数据，如果没有数据则返回null
* writeOutbound：向通道写入outbound出站数据，模拟通道发送数据。也就是说这些写入的数据会被流水线上的出站处理器处理
* readOutbound：从EmbeddedChannel中读取出站数据，返回经过流水线最后一个出站处理器处理之后的出站数据，如果没有数据则返回null
* finish：结束EmbeddedChannel，它会调用通道的close反复噶


最重要的两个方法为writeInbound和readOutbound方法


# 五、Handler业务处理器


在Reactor反应器经典模型中，反应器查询到IO事件后，分发到Handler业务处理器，由Handler完成IO操作和业务处理。整个的IO处理操作包括：从通道读取数据包、数据包解码、业务处理、目标数据编码、把数据包写到通道，然后由通道发送到对端。前后两个环节，从通道读取数据包和由通道发送到对端由Netty的底层完成，不需要用户程序负责。


用户程序主要在Handler业务处理器中，主要负责数据包解码、业务处理、目标数据编码、把数据包写入到通道中。前面两个环节属于入站处理器的工作，后面两个环节属于出站处理器的工作。


## 1、ChannelInboundHandler通道入站处理器


当数据或信息入站到Netty通道时，Netty将触发入站处理器ChannelInboundHandler所对应的入站API，进行入站处理操作。ChannelInboundHandler的主要操作如下：


* channelRegistered：当通道注册完成后，Netty会调用fireChannelRegistered触发通道注册事件。通道会启动该入站操作的流水线处理，在通道注册过的入站处理器Handler的channelRegistered方法会被调用到
* channelAlive：当通道激活完成后，Netty会调用fireChannelActive触发通道激活事件。通道会启动该入站处理器的流水线处理，在通道注册过的入站处理器Handler的channelActive方法会被调用到
* channelRead：当通道缓冲区可读，Netty会调用fireChannelRead触发通道可读事件。通道会启动该入站处理器的流水线处理，在通道注册过的入站处理器Handler的channelRead方法会被调用到
* channelReadComplete：当通道缓冲区读完，Netty会调用fireChannelReadComplete触发通道读完事件。通道会启动该入站处理器的流水线处理，在通道注册过的入站处理器Handler的channelReadComplete方法会被调用到
* channelInactive：当连接被断开或者不可用，Netty会调用fireChannelInactive触发通道读完事件。通道会启动该入站处理器的流水线处理，在通道注册过的入站处理器Handler的channelInactive方法会被调用到
* exceptionCaught：当通道处理过程发生异常时，Netty会调用fireExceptionCaught。通道会启动异常捕获的流水线处理，在通道注册过的处理器Handler的channelInactive方法会被调用到。这个方法是在通道处理器中的ChannelHandler定义的方法，入站处理器，出站处理接口都继承到了该方法


上述为ChannelInboundHandler的部分重要方法。在Netty中它的默认实现为ChannelInboundHandlerAdapter，在实际开发中只需要继承这个类的默认实现，重写自己需要的方法即可。


## 2、ChannelOutboundHandler通道出站处理器


当业务处理完成后，通过一系列的ChannelOutboundHandler通道出站处理器，完成Netty通道到底层通道的操作。包括建立底层连接、断开底层连接、写入底层Java NIO通道等。ChannelOutboundHandler定义了大部分的出站操作，具体如下：


* bind（监听地址（IP+端口）绑定）：完成底层Java IO通道的IP地址绑定
* connect（连接服务端）：完成底层Java IO通道的服务器端连接操作
* write（写数据到底层）：完成Netty通道向底层Java IO通道的数据写入操作，此方法仅仅是触发一下操作而已，并不是完成实际的数据写入操作
* flush：腾空缓冲区中的数据，把数据写到对端
* read（从底层读数据）：完成Netty通道从Java IO通道的数据读取
* disConnect（断开服务器连接）：断开底层Java IO通道的服务器端连接
* close：关闭底层的通道


上述为ChannelOutboundHandler的部分重要方法，在Netty中它的默认实现为ChannelOutboundHandlerAdapter，在实际开发中只需要继承这个类的默认实现，重写自己需要的方法即可。


## 3、ChannelInitializer通道初始化处理器


一条Netty的通道拥有一条Handler业务处理器流水线，负责装配自己的Handler业务处理器。装配Handler的工作发生在通道开始工作之前。那么如何向流水线中装配业务处理器呢？这就得借助通道的初始化类ChannelInitializer。


initChannel方法是ChannelInitializer定义的一个抽象方法，这个抽象方法需要开发人员实现。在父通道调用initChannel方法时，会将新接收的通道作为参数，传递给initChannel方法。initChannel方法内部大致的业务代码是拿到新连接通道作为实际参数，往它的流水线中装配Handler业务处理器。


# 六、Pipeline流水线


Netty的业务处理器流水线ChannelPipeline是基于责任链设计模式来设计的，内部是一个双向链表结构，能够支持动态地添加和删除Handler业务处理器。


## 1、Pipeline处理流程


```java
public class InPipeline {
  
    static class SimpleInHandlerA extends ChannelInboundHandlerAdapter {
        @Override
        public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
            System.out.println("HandlerA");
            super.channelRead(ctx, msg);
        }
    }
    static class SimpleInHandlerB extends ChannelInboundHandlerAdapter {
        @Override
        public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
            System.out.println("HandlerB");
            super.channelRead(ctx, msg);
        }
    }
    static class SimpleInHandlerC extends ChannelInboundHandlerAdapter {
        @Override
        public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
            System.out.println("HandlerC");
            super.channelRead(ctx, msg);
        }
    }

    public static void main(String[] args) {
        ChannelInitializer<EmbeddedChannel> initializer = new ChannelInitializer<EmbeddedChannel>() {
            @Override
            protected void initChannel(EmbeddedChannel ch) throws Exception {
                ch.pipeline().addLast(new SimpleInHandlerA())
                        .addLast(new SimpleInHandlerB())
                        .addLast(new SimpleInHandlerC());
            }
        };
        EmbeddedChannel channel = new EmbeddedChannel(initializer);
        ByteBuf buf = Unpooled.buffer();
        buf.writeInt(1);
        channel.writeInbound(buf);
    }
}
```


在channelRead()方法中调用了父类的channelRead方法，父类的channelRead方法会自动调用下一个inBoundHandler的channelRead方法，并且会把当前inBoundHandler入站处理器中处理完毕的对象传递到下一个inBoundHandler入站处理器。


在入站/出站的过程中，如果由于业务条件不满足，需要阶段流水线的处理，不让流水线进入下一站。以channelRead为例，我们只需要删除掉子类的super.channelRead()方法，不在子类中调用父类的channelRead入站方法，即可实现截断。此外入站处理传入下一站还有一种方法是调用Context上下文的ctx.fireChannelRead()方法，所以想要截断还不能调用该方法。


入站通道处理器的调用顺序会按照加入流水线的顺序调用，而对于出站通道处理器而言，先加入的处理器会在最后被调用。


对于出站处理流程而言，只要开始执行，就不能被截断。强行阶段的话Netty会抛出异常。如果业务条件不满足，可以不启动出站处理。


## 2、ChannelHandlerContext


不管我们定义的是哪种类型的Handler业务处理器，最终它们都是以双向链表的方式保存在流水线中。这里流水线的节点类型，并不是前面的Handler业务处理器基类，而是一个新的Netty类型：ChannelHandlerContext，它代表了ChannelHandler通道处理器和ChannelPipeline通道流水线之间的关联。


ChannelHandlerContext中包含了很多方法，主要分为两类：第一类是获取上下文关联的Netty组件实例，如所关联的通道、所关联的流水线、上下文内部Handler业务处理器实例，第二类是入站和出站方法。


在Channel、ChannelPipeline、ChannelHandlerContext三个类中，会有同样的入栈和出站处理方法。如果通过Channel或ChannelPipeline的实例来调用这些方法，他们就会在整条流水线中传播。然而如果是通过ChannelHandlerContext通道处理器上下文来进行调用，就只会从当前节点开始执行Handler处理器，并传播到同类型的处理器的下一站。


Channel、Handler、ChannelHandlerContext三者的关系为：Channel通道拥有一条ChannelPipeline通道流水线，每一个流水线节点为一个ChannelHandlerContext通道处理器上下文对象，每一个上下文中报过了一个ChannelHandler通道处理器。在ChannelHandler通道处理器的入站和出站处理方法中，Netty都会传递一个Context上下文实例作为实际参数。通过Context实例的实惨，在业务处理中，可以获取ChannelPipeline通道流水线的实例或者Channel通道的实例。


## 3、Handler业务处理器的热拔插


在程序执行过程中，可以动态进行业务处理器的热拔插：动态地增加、删除流水线上的业务处理器。主要的热拔插方法声明在ChannelPipeline接口中，如下：


1. ChannelPipeline addFirst(String name, ChannelHandler handler)：在头部增加一个业务处理器，名字由name指定
2. ChannelPipeline addLast(String name, ChannelHandler handler)：在头部增加一个业务处理器，名字由name指定
3. ChannelPipeline addBefore(String baseName, String name, ChannelHandler handler)：在baseName处理器前面增加一个业务处理器，名字由name指定
4. ChannelPipeline addAfter(String baseName, String name, ChannelHandler handler)：在baseName处理器后面增加一个业务处理器，名字由name指定
5. ChannelPipeline remove(ChannelHandler handler)：删除一个业务处理器实例
6. ChannelHandler remove(String handler)：删除一个业务处理器
7. ChannelHandler removeFirst()：删除第一个业务处理器
8. ChannelHandler removeLast()：删除最后一个业务处理器


# 七、ByteBuf缓冲区


Netty提供了ByteBuf来替代Java NIO的ByteBuffer缓冲区，以操纵内存缓冲区。


## 1、优势


与Java NIO的ByteBuffer相比，ByteBuf的优势如下：


* Pooling（池化，减少了内存复制和GC，提升了效率）
* 复合缓冲区类型，支持零复制
* 不需要使用flip方法去切换读/写模式
* 扩展性好，例如StringBuffer
* 可以自定义缓冲区类型
* 读取和写入索引分开
* 方法的链式调用
* 可以进行引用计数，方便重复使用


## 2、逻辑部分


ByteBuf是一个字节容器，内部是一个字节数组。从逻辑上来分，字节容器内部可以分为四个部分：


1. 废弃：已用字节，表示已经使用完的废弃的无效字节
2. 可读：ByteBuf保存的有效数据，从ByteBuf读取的数据都来自这一部分
3. 可写：写入到ByteBuf的数据都会写到这一部分中
4. 可扩容：表示该ByteBuf最多还能扩容的大小


## 3、重要属性


ByteBuf通过三个整型的属性有效区分可读数据和可写数据，使得读写之间相互没有冲突。这三个属性定义在AbstractByteBuf抽象类中，分别是：


* readerIndex：读指针，指示读取的起始位置。每读取一个字节就自动增加1，一旦和writerIndex相等则表示ByteBuf不可读了
* writerIndex：写指针，指示写入的起始位置。没写入一个字节就自动增加1，一旦和capacity容量相等则表示ByteBuf不可写了
* maxCapacity：最大容量，指示可扩容的最大容量。当向ByteBuf写数据的时候，如果容量不足，可以进行扩容。扩容的最大限度由maxCapacity的值来设定，超过就会报错


*小于readerIndex指针的的部分为废弃部分*


此外AbstractByteBuf中还有两个属性：markedWriterIndex和markedReaderIndex，相当于一个暂存属性


## 4、ByteBuf的三组方法


1. 容量系列

    * capacity()：ByteBuf的容量（废弃的字节数+可读的字节数+可写的字节数）
    * maxCapacity()：表示ByteBuf最大能够容纳的最大字节数
2. 写入系列

    * isWritable()：表示ByteBuf是否可写。如果capacity容量大于writerIndex指针的位置则表示可写，返回false并不代表不能再往ByteBuf中写数据了，会自动扩容
    * writableBytes()：取得可写入的字节数，值等于capacity减去writerIndex
    * maxWritableBytes()：取得最大的可写入字节数，值等于maxCapacity减去writerIndex
    * writeBytes(byte[] src)：把src字节数组中的数据全部写到ByteBuf
    * writeTYPE(TYPE value)：写入基础数据类型的数据，TYPE表示基础数据类型，包含了8大基础数据类型
    * setTYPE(TYPE value)：基础数据类型的设置，不改变writerIndex指针值
    * markWriterIndex()：把当前的写指针属性的值保存在markedWriterIndex属性中
    * resetWriterIndex()：把之前保存的markedWriterIndex的值恢复到写指针writerIndex属性中
3. 读取系列

    * isReadable()：表示ByteBuf是否可读。如果writerIndex指针的值大于readerIndex指针的位置则表示可读，否则表示不可读
    * readableBytes()：取得可读取的字节数，值等于writerIndex减去readerIndex
    * readBytes(byte[] dst)：读取ByteBuf中的数据，将数据从ByteBuf读取到dst字节数组中
    * readTYPE(TYPE value)：读取基础数据类型的数据，TYPE表示基础数据类型，包含了8大基础数据类型
    * getTYPE(TYPE value)：读取基础数据类型，不改变readerIndex指针值
    * markReaderIndex()：把当前的读指针属性的值保存在markedReaderIndex属性中
    * resetReaderIndex()：把之前保存的markedReaderIndex的值恢复到读指针readerIndex属性中


## 5、引用计数


Netty的ByteBuf的内存回收工作是通过引用计数的方式管理的。JVM中使用计数器（一种GC方法）来标记对象是否不可达进而回收，Netty也使用这种手段来对ByteBuf的引用进行计数。Netty采用计数器来追踪ByteBuf的生命周期，一是对Pooled ByteBuf的支持，二是能够尽快地发现那些可以回收的ByteBuf（非Pooled），以便提升ByteBuf的分配和销毁的效率。


> 什么是池化的ByteBuf缓冲区？
> 在通信程序的执行过程中，Buffer缓冲区实例会被频繁创建、使用、释放。频繁创建对象、内存分配、释放内存会使系统的开销大、性能低。因此Netty4开始新增了对象池化的机制。即创建一个Buffer对象池，将没有被引用的Buffer对象，放入对象缓冲池中。当需要时则重新从对象缓冲池中取出，而不需要重新创建。


在默认情况下，当创建完一个ByteBuf时它的引用为1。每次调用retain方法，它的引用就加1，每次调用release方法，它的引用就减1.如果引用为0，再次访问这个ByteBuf对象，将会抛出异常。如果引用为0，表示这个ByteBuf没有哪个进程引用它，它占用的内存需要回收。


为了确保引用计数不会混乱，在Netty的业务处理器开发过程中，应该坚持一个原则：retain和release方法应该结对使用。简单地说，在一个方法中，调用了retain就应该调用release。


如果retain和release这两个方法一次都不调用，那么在缓冲区使用完成后调用一次release就是释放一次。例如在Netty流水线上，中间所有的Handler业务处理器处理完ByteBuf之后直接传递给下一个，由最后一个Handler负责调用release来释放缓冲区的内存空间。


当引用计数已经为0，Netty会进行ByteBuf的回收。分为两种情况：


1. Pooled池化的ByteBuf内存，回收方法是放入可以重新分配的ByteBuf池子，等待下一次分配
2. Unpooled未池化的ByteBuf缓冲区，回收分两种情况：

    1. 如果是堆结构缓冲，会被JVM的垃圾回收机制回收
    2. 如果是Direct类型，调用本地方法释放外部内存（Unsafe.freeMemory)


## 6、Allocator分配器


Netty通过ByteBufAllocator分配器来创建缓冲区和分配内存空间。Netty提供了ByteBufAllocator的两种实现：PoolByteBufAllocator和UnpooledByteAllocator。


PoolByteBufAllocator将ByteBuf实例放入池中，提高了性能，将内存碎片减少到最小，这个池化分配器采用了jemalloc高效内存分配策略，该策略被好几种现代操作系统所采用。


UnpooledByteBufAllocator是普通的未池化ByteBuf分配器，它没有把ByteBuf放入池中，每次被调用时返回一个新的ByteBuf实例，通过Java的垃圾回收机制回收。


在Netty中默认的分配器为ByteBufAllocator.DEFAULT，可以通过Java系统参数的选项io.netty.allocator.type进行配置，配置时使用字符串值“unpooled”和“pooled”。不同Netty版本对于分配器的默认使用策略是不同的，在Netty4.0中默认的分配器为UnpooledByteBufAllocator，而在Netty4.1中默认的分配器为PooledByteBufAllocator。可以在Netty程序中设置启动器Bootstrap的时候将PooledByteBufAllocator设置为默认的分配器。


使用分脾气分配ByteBuf的方法有多种：


1. ByteBufAllocator.DEFAULT.buffer(9,100)：分配器默认分配初始容量为9，最大容量100的缓冲区
2. ByteBufAllocator.DEAULT.buffer()：分配器默认分配初始容量256，最大容量Integer.MAX_VALUE的缓冲区
3. UnpooledByteBufAllocator.DEFAULT.heapBuffer()：分池化分配器，分配基于Java的堆结构内存缓冲区
4. PooledByteBufAllocator.DEFAULT.directBuffer()：池化分配器，分配基于操作系统管理的直接内存缓冲区


## 7、缓冲区的类型


根据内存的管理方式不同，分为堆缓冲区和直接缓冲区，也就是Heap ByteBuf和Direct ByteBuf。另外为了方便缓冲区进行组合，提供了一种组合缓冲区：


1. Heap ByteBuf：内部数据为一个Java数组，存储在JVM的堆空间中，通过hasArray来判断是不是堆缓冲区。

    * 优点：未使用池化的情况下能提供快速的分配和释放
    * 缺点：写入底层传输通道之前都会复制到直接缓冲区
2. Direct ByteBuf：内部数据存储在操作系统的物理内存中

    * 优点：能获取超过JVM堆限制大小的内存空间，写入传输通道比堆缓冲区更快
    * 缺点：释放和分配空间昂贵（因为使用系统的方法），在Java中操作时需要复制一次到堆上
3. CompositeBuffer：多个缓冲区的组合表示

    * 优点：方便一次操作多个缓冲区实例


上面三种缓冲区的类型，无论哪一种，都可以通过池化、非池化两种分配器来创建和分配内存空间。


> Direct Memory（直接内存）不属于Java堆内存，所分配的内存其实是调用操作系统malloc函数来获得的，由Netty的本地内存堆Native堆进行管理。Direct Memory容量可以通过-XX:MaxDirectMemorySize来指定，如果不指定则默认与Java对的最大值-Xmx一样。
>
> Direct Memory的使用避免了Java堆和Native堆之间来回复制数据，在某些应用场景中提高了性能。
>
> 在需要频繁创建缓冲区的场合，由于创建和销毁Direct Buffer的代价比较高昂，因此不宜使用Direct Buffer，也就是说Direct Buffer尽量在池化分配器中分配和回收。如果能将Direct Buffer进行复用，在读写频繁的情况下可以大幅度改善性能。
>
> 在Java的垃圾回收机制回收Java堆时，Netty框架也会释放不再使用的Direct Buffer缓冲区，因为它的内存为堆外内存，所以清理的工作不会为Java虚拟机带来压力。注意一下垃圾回收的应用场景：
>
> 1. 垃圾回收仅在Java堆被填满以至于无法为新的堆分配请求提供服务时发生
> 2. 在Java应用程序中调用System.gc()函数来释放内存


对比Heap ByteBuf和Direct ByteBuf两类缓冲区的使用：


1. 创建的方法不同：Heap ByteBuf通过调用分配器的heapBuffer()方法来创建，而Direct ByteBuf的创建时通过调用分配器的directBuffer()方法，如果调用buffer()方法创建，在Netty4.1中默认创建的是Direct Buffer
2. Heap ByteBuf缓冲区可以直接通过array()方法读取内部数组，而Direct ByteBuf缓冲区不能读取内部数组
3. 可以调用hasArray()方法来判断是否为Heap ByteBuf类型的缓冲区，如果返回true则表示是堆缓冲，否则不是（可能是Direct缓冲或CompositeByteBuf缓冲区）
4. Direct ByteBuf尧都区缓冲数据进行业务处理需要通过getBytes/readBytes等方法先将数据复制到Java的堆内存然后进行其他的计算


```java
public static void main(String[] args) {
    ByteBuf heapBuf = ByteBufAllocator.DEFAULT.heapBuffer();
    heapBuf.writeBytes("你好".getBytes(Charset.forName("UTF-8")));
    if (!heapBuf.hasArray()) {
        // 获取内部数组
        byte[] array = heapBuf.array();
        int offset = heapBuf.arrayOffset() + heapBuf.readerIndex();
        int length = heapBuf.readableBytes();
        System.out.println(new String(array, offset, length, Charset.forName("UTF-8")));
    }
    heapBuf.release();
    ByteBuf directBuffer = ByteBufAllocator.DEFAULT.directBuffer();
    directBuffer.writeBytes("你好Direct".getBytes());
    if (!directBuffer.hasArray()) {
        int length = directBuffer.readableBytes();
        byte[] array = new byte[length];
        // 把数据读到堆内存
        directBuffer.getBytes(directBuffer.readerIndex(), array);
        System.out.println(new String(array));
    }
    directBuffer.release();
}
```


## 8、ByteBuf的自动释放


Netty的Reactor反应器线程会在底层的Java NIO通道读数据，也就是AbstractNioByteChannel.NioByteUnsafe.read()，调用ByteBufAllocator方法，创建ByteBuf实例，从操作系统缓冲区把数据读取到ByteBuf实例中，然后调用pipeline.fireChannelRead(byteBuf)方法将读取到的数据包送入到入站处理流水线中。


以上是ByteBuf的创建方式，而入站的ByteBuf的释放方式有以下两种：


### 1）TailHandler自动释放


Netty默认会在ChannelPipeline通道流水线的最添加一个TailHeader末尾处理器，它实现了默认的处理方法，在这些方法中会帮助完成ByteBuf内存释放的工作。在默认情况下，如果每个InboundHandler入站处理器，把最初的ByteBuf数据包一路往下传，那么TailHandler默认处理器会自动释放入站的ByteBuf实例。


总体来说如果自定义的InboundHandler入站处理器继承自ChannelInboundHandlerAdapter适配器，那么可以调用以下两种方法来释放ByteBuf内存：


1. 手动式发那个ByteBuf，具体的方式为调用byteBuf.release()
2. 调用父类的入站方法将msg向后传递，依赖后面的处理器释放ByteBuf。具体的方式为调用基类的入站处理方法super.channelRead


### 2）SimpleChannelInboundHandler自动释放


如果Handler业务处理器需要阶段流水线的处理流程，不将ByteBuf数据包送入后边的InboundHandler入站处理器，这是流水线末端的TailHandler末尾处理器自动释放缓冲区的工作自然就失效了。


在这种场景下，Handler业务处理器有两种选择：


1. 手动释放ByteBuf实例
2. 继承SimpleChannelInboundHandler，利用它的自动释放功能


以入站读数据为例，Handler业务处理器必须继承自SimpleChannelInboundHandler基类，并且业务处理器的代码必须移动到重写的channelRead0方法中。SimpleChannelInboundHandler类的channelRead等入站处理方法，会在调用完实际的channelRead方法后帮忙释放ByteBuf实例。


至于出站处理，在出站处理流程中，申请分配到的ByteBuf主要是通过HeadHandler完成自动释放的。出站处理用到的ByteBuf缓冲区一般是要发送的消息，通常由Handler业务处理器所申请而分配的，在每一个出站Handler业务处理器的处理完成后，最后数据包回来到出站的最后一棒HeadHandler，在数据输出完成后，ByteBuf会被释放一次，如果计数器为0，将被彻底释放掉。


在Netty开发中，必须密切关注ByteBuf缓冲区的释放，如果释放的不及时，会造成Netty的内存泄露，最终导致内存耗尽。


# 八、ByteBuf浅层复制的高级使用方式


 浅层复制是一种非常重要的操作，可以很大程度地避免内存复制。ByteBuf的浅层复制分为切片浅层复制和整体浅层复制两种。


## 1、slice切片浅层复制


ByteBuf的slice方法可以获取到一个ByteBuf的一个切片。一个ByteBuf可以进行多次的切片浅层复制，多次切片后的ByteBuf对象可以共享一个存储区域。


slice方法有两个重载版本：


1. public ByteBuf slice()
2. public ByteBuf slice(int index, int length)


第一个是不带参数的slice方法，在内部是调用了buf.slice(buf.readerIndex(), buf.readableBytes())，也就是说第一个无参数slice方法的返回值是ByteBuf实例中可读部分的切片。


调用slice()方法后，返回的切片是一个新的ByteBuf对象，该对象的几个重要属性，大致如下：


* readerIndex（读指针）的值为0
* writerIndex（写指针）的值为源ByteBuf的readableBytes可读字节数
* maxCapacity（最大容量）的值为源ByteBuf的readableBytes可读字节数


切片后的心ByteBuf有两个特点：


1. 切片不可以写入，原因是maxCapacity与writerIndex值相同
2. 切片和源ByteBuf的可读字节数相同


切片后的新ByteBuf和源ByteBuf的关联性：


* 切片不会复制源ByteBuf的底层数据，底层数据和源ByteBuf的底层数组是同一个
* 切片不会改变源ByteBuf的引用计数


从根本上说，slice无参数方法所生成的切片就是源ByteBuf可读部分的浅层复制


## 2、duplicate整体浅层复制


和slice切片不同，duplicate返回的源ByteBuf的整个对象的一个浅层复制，包括以下内容：


* duplicate的读写指针、最大容量值
* duplicate不会改变源的ByteBuf的应用技术
* duplicate不会复制源的ByteBuf的底层数据


duplicate和slice方法都是浅层复制，不同的是slice方法是切取一段的浅层复制，而duplicate是整体的浅层复制


## 3、浅层复制的问题


浅层复制方法不会实际去复制数据，也不会改变ByteBuf的引用计数，这就会导致一个问题：在源ByteBuf调用release之后，一旦引用计数为零就比拿的不能访问了。在这种场景下，源ByteBuf的所有浅层复制实例也不能进行读写了，如果强行对浅层复制实例进行读写则会报错。


因此在调用千层复制实例时，可以通过一次retain方法来增加引用，表示它们对应的底层内存多了一次引用。


# 九、实例


## 1、服务器端


```java
public class NettyEchoServer {
  
    private final int serverPort;
  
    ServerBootstrap bootstrap = new ServerBootstrap();

    public NettyEchoServer(int serverPort) {
        this.serverPort = serverPort;
    }
  
    public void runServer() {
        // 创建反应器线程组
        EventLoopGroup bossLoopGroup = new NioEventLoopGroup(1);
        EventLoopGroup workerLoopGroup = new NioEventLoopGroup();
        try {
            // 1.设置反应器线程组
            bootstrap.group(bossLoopGroup, workerLoopGroup);
            // 2.设置nio类型的通道
            bootstrap.channel(NioServerSocketChannel.class);
            // 3.设置监听端口
            bootstrap.localAddress(serverPort);
            // 4.设置通道的参数
            bootstrap.option(ChannelOption.SO_KEEPALIVE, true);
            bootstrap.option(ChannelOption.ALLOCATOR, PooledByteBufAllocator.DEFAULT);
            // 5.装配子通道流水线
            bootstrap.childHandler(new ChannelInitializer<SocketChannel>() {

                // 有连接到达时会创建一个通道
                @Override
                protected void initChannel(SocketChannel socketChannel) throws Exception {
                    // 流水线管理子通道中的Handler处理器
                    // 向子通道流水线添加一个handler处理器
                    socketChannel.pipeline().addLast(NettyEchoServerHandler.INSTANCE);
                }
            });
            // 6.开始绑定服务器
            // 通过调用sync同步方法阻塞直到绑定成功
            ChannelFuture channelFuture = bootstrap.bind().sync();
            // 7.等待通道关闭的异步任务结束
            // 服务监听通道会一直等待通道关闭的异步任务结束
            ChannelFuture closeFuture = channelFuture.channel().closeFuture();
            closeFuture.sync();
        } catch (Exception e) {
            e.printStackTrace();
        } finally {
            // 8.关闭EventLoopGroup
            // 释放掉所有资源包括创建的线程
            workerLoopGroup.shutdownGracefully();
            bossLoopGroup.shutdownGracefully();
        }
    }

    public static void main(String[] args) {
        new NettyEchoServer(8819).runServer();
    }
}
```


## 2、服务端处理器


```java
@ChannelHandler.Sharable
public class NettyEchoServerHandler extends ChannelInboundHandlerAdapter {

    public static final NettyEchoServerHandler INSTANCE = new NettyEchoServerHandler();

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        ByteBuf in = (ByteBuf) msg;
        System.out.println("msg type:" + ((in.hasArray()) ? "堆内存": "直接内存"));
        int len = in.readableBytes();
        byte[] arr = new byte[len];
        in.getBytes(0, arr);
        System.out.println("server received:" + new String(arr, "UTF-8"));
        System.out.println("写回前, msg.refCnt：" + ((ByteBuf) msg).refCnt());
        ChannelFuture f = ctx.writeAndFlush(msg);
        f.addListener(new ChannelFutureListener() {
            @Override
            public void operationComplete(ChannelFuture channelFuture) throws Exception {
                System.out.println("写回后，msg.refCnt:" + ((ByteBuf) msg).refCnt());
            }
        });
    }
}
```


这里的NettyEchoServerHandler在前面加了一个特殊的Netty注解：@ChannelHandler.Sharable。这个注解的作用是标注一个Handler实例可以被多个通道安全地共享，也就是说这个通道的流水线可以加入同一个Handler业务处理器实例，而这种操作Netty默认是不允许的。


但是如果一个服务器处理很多的通道，每个通道都新建很多重复的Handler实例，就需要很多重复的Handler实例，这就会浪费很多空间。所以在Handler实例中没有与特定通道强相关的数据或者状态，建议设计成共享的模式，即在前面加上注解@ChannelHandler.Sharable。反过来如果没有加@ChannelHandler.Sharable注解，试图将一个Handler实例添加到多个ChannelPipeline通道流水线时，Netty将会抛出异常。


默认同一个通道上的所有业务处理器，只能被同一个线程处理。所以不是@Sharable共享类型的业务处理器，在线程的层面是安全的，不需要进行线程的同步控制。而不同的通道可能绑定到多个不同的EventLoop反应器线程，因此加上@ChannelHandler.Sharable注解后的共享业务处理器的实例，可能被多个线程并发执行。这样就会导致一个结果@Sharable共享实例不是线程层面安全的，因此@Sharable共享的业务处理器，如果需要操作的数据不仅仅是局部变量，则需要进行线程的同步控制，以保证操作是线程层面安全的。


ChannelHandlerAdapater提供了使用方法isSharable()，如果其对应的实现加上了@Sharable注解，那么这个方法将返回true，表示它可以被添加到多个ChannelPipeline通道流水线中。


## 3、客户端


```java
public class NettyEchoClient {
  
    private int serverPort;
  
    private String serverIp;
  
    Bootstrap b = new Bootstrap();
  
    public NettyEchoClient(String ip, int port) {
        this.serverPort = port;
        this.serverIp = ip;
    }

    public void runClient() {
        EventLoopGroup workerLoopGroup = new NioEventLoopGroup();
        try {
            b.group(workerLoopGroup);
            b.channel(NioSocketChannel.class);
            b.remoteAddress(serverIp, serverPort);
            b.option(ChannelOption.ALLOCATOR, PooledByteBufAllocator.DEFAULT);
            b.handler(new ChannelInitializer<SocketChannel>() {
                @Override
                protected void initChannel(SocketChannel ch) throws Exception {
                    ch.pipeline().addLast(NettyEchoClientHandler.INSTANCE);
                }
            });
            ChannelFuture f = b.connect();
            f.addListener(new ChannelFutureListener() {
                @Override
                public void operationComplete(ChannelFuture channelFuture) throws Exception {
                    if (f.isSuccess()) {
                        System.out.println("连接成功");
                    } else {
                        System.out.println("连接失败");
                    } 
                }
            });
            f.sync();
            Channel channel = f.channel();
            Scanner scanner = new Scanner(System.in);
            System.out.println("请输入发送内容:");
            while (scanner.hasNext()) {
                String next = scanner.next();
                ByteBuf buffer = channel.alloc().buffer();
                buffer.writeBytes(next.getBytes("UTF-8"));
                channel.writeAndFlush(buffer);
                System.out.println("请输入发送内容:");
            }
        } catch (InterruptedException e) {
            throw new RuntimeException(e);
        } catch (UnsupportedEncodingException e) {
            throw new RuntimeException(e);
        } finally {
            workerLoopGroup.shutdownGracefully();
        }
    }
}
```


## 4、客户端处理器


```java
public class NettyEchoClientHandler extends ChannelInboundHandlerAdapter {

    public static final NettyEchoClientHandler INSTANCE = new NettyEchoClientHandler();

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        ByteBuf buf = (ByteBuf) msg;
        int len = buf.readableBytes();
        byte[] arr = new byte[len];
        buf.getBytes(0, arr);
        System.out.println("client received:" + new String(arr, "UTF-8"));
        buf.release();
    }
}
```

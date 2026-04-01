---
title: "Reactor 模式"
date: 2023-10-24T22:12:37+08:00
draft: false
summary: "介绍 Reactor 反应器模式的演进过程，从阻塞 OIO 到 Connection Per Thread 再到 Reactor 模型，详解 Reactor 线程和 Handler 的职责分工，并基于 Java NIO 给出单线程 Reactor 的实现示例。"
tags: []
categories: [Tech]
source: csdn
source_id: "134023107"
---

在Java的OIO编程中，最初和最原始的网络服务器程序使用一个while循环，不断地监听端口是否有新的连接，如果有就调用一个处理函数来处理。这种方法最大的问题就是如果前一个网络连接的处理没有结束，那么后面的连接请求没法被接收，于是后面的请求统统会被阻塞住，服务器的吞吐量就太低了。


为了解决这个严重的连接阻塞问题，出现了一个即为经典模式：Connection Per Thread。即对于每一个新的网络连接都分配一个线程，每个线程都独自处理自己负责的输入和输出，任何socket连接的输入和输出处理不会阻塞到后面新socket连接的监听和建立。早期版本的Tomcat服务器就是这样实现的。


这种模式的优点是解决了前面的新连接被严重阻塞的问题，在一定程度上极大地提高了服务器的吞吐量。但是对于大量的连接，需要消耗大量的现成资源，如果线程数太多，系统无法承受。而且线程的反复创建、销毁、线程的切换也需要代价。因此高并发应用场景下多线程OIO的缺陷是致命的，因此引入了Reactor反应器模式。


反应器模式由Reactor反应器线程、Handlers处理器两大角色组成：


1. Reactor反应器线程的职责：负责响应IO事件，并且分发到Handlers处理器
2. Handlers处理器的职责：非阻塞的执行业务处理逻辑


## 一、单线程Reactor反应器模式


Reactor反应器模式有点儿类似事件驱动模式，当有事件触发时，事件源会将事件dispatch分发到handler处理器进行事件处理。反应器模式中的反应器角色类似于事件驱动模式中的dispatcher事件分发器角色。


* Reactor反应器：负责查询IO事件，当检测到一个IO时间，将其发送给对应的Handler处理器处理，这里的IO事件就是NIO选择器监控的通道IO事件。
* Handler处理器：与IO事件绑定，负责IO事件的处理，完成真正的连接建立、通道的读取、处理业务逻辑、负责将结果写出到通道等。


基于NIO实现单线程版本的反应器模式需要用到SelectionKey选择键的几个重要的成员方法：


1. void attach(Object o)：将任何的Java对象作为附件添加到SelectionKey实例，主要是将Handler处理器实例作为附件添加到SelectionKey实例
2. Object attachment()：取出之前通过attach添加到SelectionKey选择键实例的附件，一般用于取出绑定的Handler处理器实例。


Reactor实现示例：


```java
package cn.ken.jredis;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.channels.SelectionKey;
import java.nio.channels.Selector;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.util.Set;

/**
 * <pre>
 *
 * </pre>
 *
 * @author <a href="https://github.com/Ken-Chy129">Ken-Chy129</a>
 * @since 2023/10/14 14:29
 */
public class Reactor implements Runnable {

    final private Selector selector;

    final private ServerSocketChannel serverSocketChannel;

    public Reactor() {
        try {
            this.selector = Selector.open();
            this.serverSocketChannel = ServerSocketChannel.open();
            serverSocketChannel.bind(new InetSocketAddress(8088));
            // 注册ServerSocket的accept事件
            SelectionKey sk = serverSocketChannel.register(selector, SelectionKey.OP_ACCEPT);
            // 为事件绑定处理器
            sk.attach(new AcceptHandler());
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public void run() {
        try {
            while (!Thread.interrupted()) {
                selector.select();
                Set<SelectionKey> selectionKeys = selector.selectedKeys();
                for (SelectionKey selectedKey : selectionKeys) {
                    dispatch(selectedKey);
                }
                selectionKeys.clear();
            }
        } catch (Exception e) {
            throw new RuntimeException(e);
        }
    }

    private void dispatch(SelectionKey selectedKey) {
        Runnable handler = (Runnable) selectedKey.attachment();
        // 此处返回的可能是AcceptHandler也可能是IOHandler
        handler.run();
    }

    class AcceptHandler implements Runnable {
        @Override
        public void run() {
            try {
                SocketChannel socketChannel = serverSocketChannel.accept();
                if (socketChannel != null) {
                    new IOHandler(selector, socketChannel); // 注册IO处理器，并将连接加入select列表
                }
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        }
    }

    public static void main(String[] args) {
        new Reactor().run();
    }
}
```


Handler实现示例：


```java
package cn.ken.jredis;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.channels.SelectionKey;
import java.nio.channels.Selector;
import java.nio.channels.SocketChannel;

/**
 * <pre>
 *
 * </pre>
 *
 * @author <a href="https://github.com/Ken-Chy129">Ken-Chy129</a>
 * @since 2023/10/14 14:53
 */
public class IOHandler implements Runnable {

    final private SocketChannel socketChannel;

    final private ByteBuffer buffer;


    public IOHandler(Selector selector, SocketChannel channel) {
        buffer = ByteBuffer.allocate(1024);
        socketChannel = channel;
        try {
            channel.configureBlocking(false);
            SelectionKey sk = channel.register(selector, 0); // 此处没有注册感兴趣的事件
            sk.attach(this);
            sk.interestOps(SelectionKey.OP_READ); // 注册感兴趣的事件，下一次调用select时才生效
            selector.wakeup(); // 立即唤醒当前阻塞select操作，使得迅速进入下次select，从而让上面注册的读事件监听可以立即生效
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public void run() {
        try {
            int length;
            while ((length = socketChannel.read(buffer)) > 0) {
                System.out.println(new String(buffer.array(), 0, length));
            }
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }
}
```


在单线程反应器模式中，Reactor反应器和Handler处理器都执行在同一条线程上（dispatch方法是直接调用run方法，没有创建新的线程），因此当其中某个Handler阻塞时，会导致其他所有的Handler都得不到执行。


## 二、多线程Reactor反应器模式


既然Reactor反应器和Handler处理器在一个线程会造成非常严重的性能缺陷，那么可以使用多线程对基础的反应器模式进行改造。


1. 将负责输入输出处理的IOHandler处理器的执行，放入独立的线程池中。这样业务处理线程与负责服务监听和IO时间查询的反应器线程相隔离，避免服务器的连接监听收到阻塞。
2. 如果服务器为多核的CPU，可以将反应器线程拆分为多个子反应器线程，同时引入多个选择器，每一个SubReactor子线程负责一个选择器。


MultiReactor：


```java
package cn.ken.jredis;

import java.io.IOException;
import java.net.InetSocketAddress;
import java.nio.channels.SelectionKey;
import java.nio.channels.Selector;
import java.nio.channels.ServerSocketChannel;
import java.nio.channels.SocketChannel;
import java.util.Set;
import java.util.concurrent.atomic.AtomicInteger;

/**
 * <pre>
 *
 * </pre>
 *
 * @author <a href="https://github.com/Ken-Chy129">Ken-Chy129</a>
 * @since 2023/10/14 16:51
 */
public class MultiReactor {
  
    private final ServerSocketChannel server;
  
    private final Selector[] selectors = new Selector[2];

    private final SubReactor[] reactors = new SubReactor[2];
    private final AtomicInteger index = new AtomicInteger(0);

    public MultiReactor() {
        try {
            server = ServerSocketChannel.open();
            selectors[0] = Selector.open();
            selectors[1] = Selector.open();
            server.bind(new InetSocketAddress(8080));
            server.configureBlocking(false);
            SelectionKey register = server.register(selectors[0], SelectionKey.OP_ACCEPT);
            register.attach(new AcceptHandler());
            reactors[0] = new SubReactor(selectors[0]);
            reactors[1] = new SubReactor(selectors[1]);
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }
  
    private void startService() {
        new Thread(reactors[0]).start();
        new Thread(reactors[1]).start();
    }
  
    class SubReactor implements Runnable {
        final private Selector selector;

        public SubReactor(Selector selector) {
            this.selector = selector;
        }

        @Override
        public void run() {
            while (!Thread.interrupted()) {
                try {
                    selector.select();
                    Set<SelectionKey> selectionKeys = selector.selectedKeys();
                    for (SelectionKey selectionKey : selectionKeys) {
                        dispatch(selectionKey);
                    }
                    selectionKeys.clear();
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }

            }
        }
    }

    private void dispatch(SelectionKey selectionKey) {
        Runnable attachment = (Runnable) selectionKey.attachment();
        if (attachment != null) {
            attachment.run();
        }
    }

    class AcceptHandler implements Runnable {
        @Override
        public void run() {
            try {
                SocketChannel socketChannel = server.accept();
                new MultiHandler(selectors[index.getAndIncrement()], socketChannel);
                if (index.get() == selectors.length) {
                    index.set(0);
                }
            } catch (IOException e) {
                throw new RuntimeException(e);
            }
        }
    }
}
```


MultiHandler：


```java
package cn.ken.jredis;

import java.io.IOException;
import java.nio.ByteBuffer;
import java.nio.channels.SelectionKey;
import java.nio.channels.Selector;
import java.nio.channels.SocketChannel;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

/**
 * <pre>
 *
 * </pre>
 *
 * @author <a href="https://github.com/Ken-Chy129">Ken-Chy129</a>
 * @since 2023/10/14 17:28
 */
public class MultiHandler implements Runnable {
  
    final private Selector selector;
  
    final private SocketChannel channel;
  
    final ByteBuffer buffer = ByteBuffer.allocate(1024);
  
    static ExecutorService pool = Executors.newFixedThreadPool(4);

    public MultiHandler(Selector selector, SocketChannel channel) {
        this.selector = selector;
        this.channel = channel;
        try {
            channel.configureBlocking(false);
            SelectionKey register = channel.register(selector, SelectionKey.OP_READ);
            register.attach(this);
            selector.wakeup();
        } catch (IOException e) {
            throw new RuntimeException(e);
        }
    }

    @Override
    public void run() {
        pool.execute(() -> {
            synchronized (this) {
                int length;
                try {
                    while ((length = channel.read(buffer)) > 0) {
                        System.out.println(new String(buffer.array(), 0, length));
                        buffer.clear();
                    }
                } catch (IOException e) {
                    throw new RuntimeException(e);
                }
            }   
        });
    }
}
```

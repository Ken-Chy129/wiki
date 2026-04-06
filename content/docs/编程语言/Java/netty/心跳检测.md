---
title: "心跳检测"
date: 2023-11-08T00:26:33+08:00
draft: false
summary: "客户端的心跳检测对于任何长连接的应用来说，都是一个非常基础的功能。要理解心跳的重要性，首先需要从网络连接假死的现象说起。 一、网络连接假死现象 什么是连接假死呢？如果底层的TCP连接已经断开，但是服务器端并没有正常地关闭套接字，认为这条连接仍然是存在的。 连接假死的具体表现如下： 1. 在服务器端，..."
tags: [Netty]
categories: [Java, Networking]
source: csdn
source_id: "134279275"
source_url: "https://blog.csdn.net/qq_25046827/article/details/134279275"
---

客户端的心跳检测对于任何长连接的应用来说，都是一个非常基础的功能。要理解心跳的重要性，首先需要从网络连接假死的现象说起。


## 一、网络连接假死现象


什么是连接假死呢？如果底层的TCP连接已经断开，但是服务器端并没有正常地关闭套接字，认为这条连接仍然是存在的。


连接假死的具体表现如下：


1. 在服务器端，会有一些处于TCP_ESTABLISHED状态的正常连接
2. 在客户端，TCP客户端已经显示连接已经断开
3. 客户端此时虽然可以进行断线重连操作，但是上一次连接状态依然被服务器端认为有效，并且服务器端的资源得不到正确释放，包括套接字上下文以及接受/发送缓冲区


连接假死的情况虽然不常见，但是确实存在。服务器端长时间运行后，会面临大量假死连接得不到释放的情况。由于每个连接都会消耗CPU和内存资源，因此大量假死的连接会逐渐耗光服务器的资源，使得服务器越来越慢，IO处理效率越来越低，最终导致服务器崩溃。


连接假死通常是由多个原因造成的：


1. 应用程序出现线程堵塞，无法进行连接的读写
2. 网络相关的设别出现故障
3. 网络丢包


解决假死的有效手段是客户端定时进行心跳检测，服务端定时进行空闲检测。


## 二、服务器端的空闲检测


想解决假死问题，服务器端的有效手段是空闲检测。所谓空闲检测就是每隔一段时间监测子通道是否有数据读写，如果有则子通道是正常的，如果没有则判定为假死，关闭子通道。


服务器端实现空闲检测只需要使用Netty自带的IdleStateHandler空闲状态处理器就可以实现这个功能。


```java
@Slf4j
public class HeartBeatServerHandler extends IdleStateHandler {

    private static final int READ_IDLE_GAP = 150; // 最大空闲时间(s)

    public HeartBeatServerHandler() {
        super(READ_IDLE_GAP, 0, 0, TimeUnit.SECONDS);
    }
  
    @Override
    protected void channelIdle(ChannelHandlerContext ctx, IdleStateEvent evt) throws Exception {
        log.info("{}秒内未读到数据，关闭连接", READ_IDLE_GAP);
        // 其他处理，如关闭会话
    }
  
    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        // 判断消息实例
        if (!(msg instanceof MessageProtos.Message message)) {
            super.channelRead(ctx, msg);
            return;
        }
        if (message.getType() == MessageProtos.HeadType.HEART_BEAT) {
            if (ctx.channel().isActive()) {
                // 将心跳数据包直接回给客户端
                ctx.writeAndFlush(msg);
            }
        }
        super.channelRead(ctx, msg);
    }
}
```


在HeartBeatServerHandler的构造函数中，调用了基类IdleStateHandler的构造函数，传递了四个参数：


1. 入站空闲检测时长：指的是一段时间内如果没有消息入站就判定为连接假死
2. 出站空闲检测时长：指的是一段时间内如果没有数据出站就判定为连接假死
3. 出/入站检测时长：表示在一段时间内如果没有出站或者入站就判定为连接假死
4. 时间单位


判定为假死之后IdleStateHandler会回调自己的channelIdle()方法，一般在这个方法中去进行一些连接的关闭。


HeartBeatServerHandler实现的主要功能是空闲检测，需要客户端定时发送心跳数据包（或报文、消息）进行配合，而且客户端发送心跳数据包的时间间隔需要远远小于服务器端的空闲检测时间间隔。


收到客户端的心跳数据包之后可以直接回复客户端，让客户端也能进行类似的空闲检测。由于IdleStateHandler本身是一个入站处理器，只需要重写这个子类的channelRead方法，然后将心跳数据包直接写回给客户端即可。


> 如果HeartBeatServerHandler要重写channelRead方法，一定要调用积累的channelRead方法，不然IdleStateHandler的入站空闲检测会无效。


## 三、客户端的心跳报文


与服务器端的空闲检测相配合，客户端需要定期发送数据包到服务器端，通常这个数据包称为心跳数据包。


```java
@Slf4j
public class HeartBeatClientHandler extends ChannelInboundHandlerAdapter {
  
    // 心跳的时间间隔，单位为秒
    private static final int HEART_BEAT_INTERVAL = 50;

    // 在Handler业务处理器被加入到流水线时开始发送心跳数据包
    @Override
    public void handlerAdded(ChannelHandlerContext ctx) throws Exception {
        ClientSession session = ctx.channel().attr(ClientSession.CLIENT_SESSION).get();
        MessageProtos.MessageHeartBeat heartBeat =
                MessageProtos.MessageHeartBeat.newBuilder()
                        .setSeq(0)
                        .setJson("{\"from\":\"client\"}")
                        .setUid(session.getUserDTO().getUserId())
                        .build();
        MessageProtos.Message message = MessageProtos.Message.newBuilder()
                .setType(MessageProtos.HeadType.HEART_BEAT)
                .setSessionId(session.getSessionId())
                .setMessageHeartBeat(heartBeat)
                .build();
        heartBeat(ctx, message);
        super.handlerAdded(ctx);
    }

    private void heartBeat(ChannelHandlerContext ctx, MessageProtos.Message message) {
        // 提交在给定延迟后启用的一次性任务。
        ctx.executor().schedule(() -> {
            if (ctx.channel().isActive()) {
                log.info("发送心跳消息给服务端");
                ctx.writeAndFlush(message);
                // 递归调用，发送下一次的心跳
                heartBeat(ctx, message);
            }
        }, HEART_BEAT_INTERVAL, TimeUnit.SECONDS);
    }

    // 接收到服务器的心跳回写
    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        if (!(msg instanceof MessageProtos.Message message)) {
            super.channelRead(ctx, msg);
            return;
        }
        if (message.getType() == MessageProtos.HeadType.HEART_BEAT) {
            log.info("收到会写的心跳信息");
        } else {
            super.channelRead(ctx, msg);
        } 
    }
}
```


在HeartBeatClientHandler实例被加入到流水线时，它重写的handlerAdded方法被回调。在handlerAdded方法中开始调用heartBeat方法发送心跳数据包。heartBeat是一个不断递归调用的方法，它使用了ctx.executor()获取当前通道绑定的Reactor反应器NIO线，然后通过NIO现线程的schedule定时调度方法，在50s后触发这个方法的执行，再之后递归调用同样延时50s后继续发送。


客户端的心跳间隔要比服务器端的空闲检测时间间隔要短，一般来说要比它的一半要短一些，可以直接定义为空闲检测时间间隔的1/3，以防止公网偶发的秒级抖动。


HeartBeatClientHandler实例并不是一开始就装配到流水线中的，它装配的实际实在登录成功之后。


HeartBeatClientHandler实际上也可以集成IdleStateHandler类在客户端进行空闲检测，这样客户端也可以对服务器进行假死判定，在服务器假死的情况下，客户端可以发起重连。

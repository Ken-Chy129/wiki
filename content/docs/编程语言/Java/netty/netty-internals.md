---
title: "底层原理"
date: 2023-11-12T22:16:09+08:00
draft: false
summary: "NIO有一个非常重要的组件——多路复用器，其底层有3种经典模型，分别是epoll、select和poll。与传统IO相比，一个多路复用器可以处理多个Socket连接，而传统IO对每个连接都需要一条线程去同步阻塞处理。NIO有了多路复用器后只需要一条线程即可管理多个Socket连接的接入和读写事件。 Netty的多路复用"
tags: [Netty]
categories: [Java, Networking]
source: csdn
source_id: "134367044"
---

NIO有一个非常重要的组件——多路复用器，其底层有3种经典模型，分别是epoll、select和poll。与传统IO相比，一个多路复用器可以处理多个Socket连接，而传统IO对每个连接都需要一条线程去同步阻塞处理。NIO有了多路复用器后只需要一条线程即可管理多个Socket连接的接入和读写事件。


**Netty的多路复用器默认调用的模型是epoll模型**，它除了JDK自带的epoll模型的封装，还额外封装了一套，这两者都是epoll模型的封装，只是JDK的epoll模型是水平触发的，而Netty采用JNI重写的边缘触发。


## 一、线程模型


对于服务器端而言有两个线程组，Boss线程组和Worker线程组。其中Boss线程组一般只开启一条线程（除非一个Netty服务同时监听多个端口），Worker线程数默认是CPU核数的两倍。Boss线程主要监听SocketChannel的OP_ACCEPT事件和客户端的连接。


**当Boss线程监听到有SocketChannel连接接入时，会把SocketChannel包装成NioSocketChannel，并注册到Worker线程的Selector中，同时监听其OP_WRITE和OP_READ事件。**当Worker线程监听到某个SocketChannel有就绪的读IO事件时，就会进行以下操作：


1. 向内存池中分配内存，读取IO数据流
2. 将读取后的ByteBuf传递给解码器Handler进行解码，若能解码出完整的请求数据包，就会把请求数据包交给业务逻辑处理Handler
3. 经过业务逻辑处理Handler后，在返回响应结果前，交给编码器进行数据加工
4. 最终写到缓冲区，并由IO Worker线程将缓冲区的数据输出到网络中并传输给客户端


## 二、解码和编码


使用Java NIO来实现TCP网络通讯，需要对TCP连接中的问题进行进行全面的考虑，如拆包和粘包导致的半包问题和序列化等。对于这些问题，Netty都进行了很好的处理。


客户端给服务端发送消息并受到服务端返回的结果共经历了以下6步：


1. TCP是面向字节流传输的协议，它把客户端提交的请求数据看作一连串的无结构的字节流，并不知道所传送的字节流的含义，也并不关心有多少数据流入TCP输出缓冲区中
2. **每次发多少数据到网络中与当前网络的拥塞情况和服务端返回的TCP窗口的大小有关，涉及TCP的流量控制和拥塞控制，并且与Netty的反压有关。**如果客户端发送到TCP输出缓冲区的数据块太多，那么TCP会分割成多次将其传送出去，如果太少，则会等待积累足够多的字节后发送出去。很明显TCP这种传输机制会产生粘包问题
3. 当服务端读取TCP输入缓冲区中的数据时，需要进行拆包处理，并解决粘包和拆包的问题，比较常见的方案有以下3种：

    1. 将换行符号或特殊标识符号加入数据包中，如HTTP和FTP等（LineBasedFrameDecoder）
    2. 将消息分为head和body，head中包含body长度的字段， 一般前面4个字节是body的长度值，用int表示，但也有像Dubbo协议那种head中除了body长度外还有版本号、请求类型和请求id等（LengthFieldPrepender/LengthFieldBasedFrameDecoder）
    3. 固定数据包的长度，如固定100字节，不足补空格（FixedLengthFrameDeocder）


步骤4-6和步骤1-3相似。TCP的这些机制与Netty的编码和解码有很大的关系。**Netty采用模板设计模式实现了一套编码和解码架构，高度抽象，底层解决TCP的粘包和拆包的问题。**编码器和解码器大部分都有共同的编码和解码父类，即MessageToMessageEncoder与ByteToMessageDecoder。


ByteToMessageDecoder父类在读取TCP缓冲区的数据并解码后，将剩余的数据放入了半包字节容器中，具体解码方案由子类负责。在解码的过程会遇到读半包，无法解码的数据会保存在读半包字节容器中，等待下次读取数据后继续解码。编码的逻辑比较简单，MessageToMessageEncoder父类定义了整个编码的流程，并实现了对已读内存的释放，具体的编码格式由子类负责。


Netty的编码和解码除了解决TCP协议的粘包和拆包问题，还有一些编解码器做了很多额外的事情，如StringEncode（把字符串转换为字节流）、ProtobufDecoder（对Protobuf序列化数据进行解码）；还有各种常用的协议编解码器，如HTTP2、Websocket等。


## 三、序列化


当客户端向服务器端发送数据时，如果发送的是一个Java对象，由于网络只能传输二进制数据流，所以Java对象无法直接在网络中传输，则必须对Java对象的内容进行流化，只有流化后的对象才能在网络中传输。序列化就是将Java对象转换成二进制流数据的过程，而这种转化的方式多种多样：


1. Java自带序列化：简单但较少使用，因为性能低，序列化后码流太大，且无法跨语言进行反序列化
2. 为了解决Java自带序列化的缺点，会引入比较流行的序列化方式，如Protobuf、Kryo、JSON等。由于JSON格式化数据可读性好，且浏览器对JSON数据的支持性非常好，所以一般的Web应用都会选择它。另外，市场上有Fastjson，Jackson等工具包，使得Java对象转成JSON也非常方便。但是JSON序列化后的数据体积较大，不适合网络传输和海量数据存储。Protobuf和Kryo序列化后的体积与JSON相比要小很多


Protobuf是Google提供的一个具有高效协议数据交换格式的工具库，其具有更高的转化效率，且时间效率和攻坚效率都是JSON的3-5倍。对于一个Java对象，转换成JSON格式时会写进去一些无用的信息，如{}，""等，当类的属性非常多并且包含各种对象组合时，开销会非常大。


而Protobuf对这些字段属性进行了额外处理，同类的每个属性名采用Tag值进行表示，这个Tag值在Protobuf中采用了varint编码， 当类的属性个数小于128时，每个属性名只需要1B来表示即可，同时属性值的长度也只占用1B。Protobuf对值也进行了各种编码，不同类型的数据值采用不同的编码技术，以尽量减小占用的存储空间。可以将Protobuf序列化后的数据想象成下面的格式：


tag|length|value|tag|length|value


Protobuf序列化除了占用空间小，性能还非常好，主要是它带有属性值长度，无需进行字符串匹配，这个长度值只用1B的存储空间。另外JSON都是字符串解析，而Protobuf根据不同的数据类型有不同的大小，如bool类型只需要读取1B的数据。


Protobuf的缺点如下：


1. 从Protobuf序列化后的数据中发现，Protobuf序列化不会把Java类序列化进去，当遇到对象的一个属性是泛型且有继承的情况时，Protobuf序列化无法正确地对其进行反序列化，还原子类信息
2. Protobuf需要编写.proto文件，比较麻烦，此时可以使用Protostuff来解决。Protostuff是Protobuf的升级版，无需编写.proto文件，只需要在对象属性中加入@Tag注解即可
3. 可读性差，只能通过程序反序列化解析查看具体内容


Protobuf一般用于公司内部服务信息的交换，对于数据量比较大、对象属性不是泛型且有继承的数据的场景比较合适。


## 四、零拷贝


零拷贝是Netty的一个特性，主要发生在操作数据上，无需将Buffer从一个内存区域拷贝到另一个内存区域，少一次拷贝，CPU效率就会提升。Netty的零拷贝主要应用在以下三种场景：


1. **Netty接收和发送ByteBuffer采用的都是堆外直接内存**，使用堆外直接内存进行Socket的读写，无需进行字节缓冲区的二次拷贝。如果使用传统的堆内存进行Socket的读写，则JVM会将堆内存Buffer数据拷贝到堆外直接内存中，然后才写入Socket中。与堆外直接内存相比，使用传统的堆内存，在消息的发送过程中多了一次缓冲区的数据拷贝
2. 在网络传输中，一条消息很可能会被分割成多个数据包进行发送，只有当收到一个完整的数据包后才能完成解码工作。**Netty通过组合内存的方式把这些内存数据包逻辑组合到一块，而不是对每个数据块进行一次拷贝**，这类似于数据库中的视图。CompositeByteBuf是Netty在此零拷贝方案中的组合Buffer
3. 传统拷贝文件的方法需要先把文件采用FileInputStream文件输入流读取到一个临时的byte[]数组中，然后通过FileOutputStream文件输出流把临时的byte[]数据内容写入目的文件中。当拷贝大文件时，频繁的内存拷贝操作会消耗大量的系统资源。**Netty底层运用Java NIO的FileChannel.transfer()方法，该方法依赖操作系统实现零拷贝，可以直接将文件缓冲区的数据发送到目标Channel中**，避免了传统的通过循环写方式导致的内存数据拷贝问题


## 五、背压


所谓背压，是进行流量控制的一种方案。背压就是消费者需要多少，生产者就生产多少。这有点类似于TCP里的流量控制，接收方根据自己的接收窗口的情况来控制发送方的发送速率。


> 这种方案只对于cold Observable有效。cold Observable是那些允许降低速率的发送源，比如两台机器传一个文件，速率可大可小，即使降低到每秒几个字节，只要时间足够长，还是能够完成的。相反的例子就是音视频直播，速率低于某个值整个功能就没法用了（这种类似于hot Observable）。


假如我们的底层使用Netty作为网络通信框架,业务流程在将业务数据发送到对端之前,实际先要将数据发送到Netty的缓冲区中,然后再从Netty的缓冲区发送到TCP的缓冲区,最后再到对端。业务数据不可能无限制向Netty缓冲区写入数据，TCP缓冲区也不可能无限制写入数据。**Netty通过高低水位控制向Netty缓冲区写入数据的多少，从而实现整个链路的背压。**


**它的大体流程就是向Netty缓冲区写入数据的时候，会判断写入的数据总量是否超过了设置的高水位值，如果超过了就设置通道(Channel)不可写状态。当Netty缓冲区中的数据写入到TCP缓冲区之后，Netty缓冲区的数据量变少，当低于低水位值的时候就设置通过(Channel)可写状态。**


Netty默认设置的高水位为64KB，低水位为32KB。可以通过ChannelOption进行设置。


```java
bootstrap.childOption(ChannelOption.WRITE_BUFFER_WATER_MARK, new WriteBufferWaterMark(32 * 1024, 64 * 1024));
// 或
bootstrap.childOption(ChannelOption.WRITE_BUFFER_LOW_WATER_MARK, config.getMemorySegmentSize() + 1);
bootstrap.childOption(ChannelOption.WRITE_BUFFER_HIGH_WATER_MARK, 2 * config.getMemorySegmentSize());
```

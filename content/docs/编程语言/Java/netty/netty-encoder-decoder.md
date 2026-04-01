---
title: "编码器与解码器"
date: 2023-10-28T18:47:24+08:00
draft: false
summary: "介绍 Netty 编解码器的原理与使用，包括入站方向将 ByteBuf 解码为 Java POJO 对象、出站方向将 POJO 编码为 ByteBuf 的流程，以及 ByteToMessageDecoder 基类的工作机制和自定义解码器的实现。"
tags: [Netty]
categories: [Java, Networking]
source: csdn
source_id: "134094704"
---

Netty从底层Java通道读到ByteBuf二进制数据，传入Netty通道的流水线，随后开始入站处理。在入站处理过程中，需要将ByteBuf二进制类型解码成Java POJO对象。这个解码过程可以通过Netty的Decoder解码器去完成。在出站处理过程中，业务处理后的结果需要从某个Java POJO对象编码为最终的ByteBuf二进制数据，然后通过底层 Java通道发送到对端。在编码过程中，需要用到Netty的Encoder编码器去完成数据的编码工作。


# 一、Decoder原理与实践


Netty的解码器Decoder本质上是一个InBound入站处理器，它将上一站InBound入站处理器传过来的输入数据进行数据的解码或者格式转换，然后输出到下一站InBound入站处理器。


一个标准的解码器将输入类型为ByteBuf缓冲区的数据进行解码，输出一个一个的Java POJO对象。Netty内置了这个解码器，叫做ByteToMessageDecoder。


Netty中的解码器都是Inbound入站处理器类型，都直接或间接地实现了ChannelInboundHandler接口。


## 1、ByteToMessageDecoder解码器


ByteToMessageDecoder是一个非常重要的解码器基类，它是一个抽象类，实现了解码的基础逻辑和流程。ByteToMessageDecoder继承自ChannelInboundHandlerAdapter适配器，是一个入站处理器，实现了从ByteBuf到Java POJO对象的解码功能。


ByteToMessageDecoder解码的流程大致是先将上一站传过来的输入到ByteBuf中的数据进行解码，解码出一个List<Object>对象列表，然后迭代这个列表，逐个将Java POJO对象传入下一站Inbound入站处理器。


ByteToMessageDecoder的解码方法名为decode，在该类中只是提供了一个抽象方法，具体的解码过程，即如何将ByteBuf数据变成Object数据需要子类去完成。ByteToMessageDecoder作为解码器的父类，只是提供了一个流程性质的框架，它仅仅将子类的decode方法解码后的Object结果放入自己内部的结果列表List<Object>中（这个过程也是子类的decode方法完成的），最终父类会负责将列表中的元素一个一个传递给下一个站。


如果要实现一个自己的解码器，首先继承ByteToMessageDecoder抽象类，然后实现其积累的decode抽象方法，将解码的逻辑写入此方法。总体来说，流程大致如下：


1. 继承ByteToMessageDecoder抽象类
2. 实现基类的decode抽象方法，将ByteBuf到POJO解码的逻辑写入此方法。将ByteBuf二进制数据解码成一个一个的Java POJO对象
3. 在子类的decode方法中奖解码后的Java POJO对象放入decode的List<Object>市财政，这个实惨是父类传入的，也就是父类的结果收集列表
4. 父类将List中的结果一个个分开地传递到下一站的Inbound入站处理器


> ByteToMessageDecoder传递给下一站的是解码之后的Java POJO对象（会遍历list中的所有元素，将其作为参数调用fireChannelRead方法），不是ByteBuf缓冲区。那么ByteBuf缓冲区由谁负责进行引用计数和释放管理的呢？
>
> 起始积累的ByteToMessageDecoder负责解码器的ByteBuf缓冲区的释放工作，它会自动调用release方法将之前的ByteBuf缓冲区的引用数减1，这个工作是自动完成的。
>
> 如果这个ByteBuf在后面还需要用到，那么可以在decode方法中调用一次retain方法来增加一次引用计数。


## 2、自定义整数解码器


### 1）常规方式


```java
public class Byte2IntegerDecoder extends ByteToMessageDecoder {
    @Override
    protected void decode(ChannelHandlerContext channelHandlerContext, ByteBuf byteBuf, List<Object> list) throws Exception {
        while (byteBuf.readableBytes() >= 4) {
            int i = byteBuf.readInt();
            System.out.println("解码出一个整数:" + i);
            list.add(i);
        }
    }
}

public class IntegerProcessHandler extends ChannelInboundHandlerAdapter {

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        Integer integer = (Integer) msg;
        System.out.println("打印出一个整数:" + integer);
        super.channelRead(ctx, msg);
    }
}

public class Byte2IntegerDecoderTester {

    public static void main(String[] args) {
        EmbeddedChannel embeddedChannel = new EmbeddedChannel(new ChannelInitializer<EmbeddedChannel>() {
            @Override
            protected void initChannel(EmbeddedChannel ch) throws Exception {
                ch.pipeline().addLast(new Byte2IntegerDecoder())
                        .addLast(new IntegerProcessHandler());
            }
        });
        ByteBuf buf = Unpooled.buffer();
        buf.writeInt(1);
        embeddedChannel.writeInbound(buf);
    }
}
```


使用上面的Byte2IngeterDecoder证书解码器需要先对ByteBuf的长度进行检查，如果有足够的字节，才进行整数的读取。这种长度的判断可以由Netty的ReplayingDecoder类来完成。


### 2）ReplayingDecoder解码器


ReplayingDecoder类是ByteToMessageDecoder的子类，其作用是：


* 在读取ByteBuf缓冲区的数据之前，检查缓冲区是否有足够的字节
* 若ByteBuf中有足够的字节则会正常读取，否则会停止解码


也就是说使用Replaying基类来编写整数解码器可以不用我们进行长度检测。


```java
public class Byte2IntegerReplayDecoder extends ReplayingDecoder {
    @Override
    protected void decode(ChannelHandlerContext ctx, ByteBuf in, List<Object> out) throws Exception {
        int i = in.readInt();
        System.out.println("解码出一个整数:" + i);
        out.add(i);
    }
}

```


可以看出通过继承ReplayingDecoder类来实现一个解码器就不用编写长度判断的代码。Replaying内部定义了一个新的二进制缓冲区类对ByteBuf缓冲区进行了装饰，类名为ReplayingDecoderBuffer。这个装饰器会在缓冲区真正读数据之前首先进行长度的判断，如果长度合格则读取数据，否则抛出ReplayError。ReplayingDecoder捕获到ReplayError后会留着数据，等待下一次IO时间到来时再读取。


也就是说实际上ReplayingDecoder中decode方法所得到的实参in的值并不是原始的ByteBuf类型，而是ReplayingDecoderBuffer类型，它继承了ByteBuf类，包装了大部分的读取方法，在读取前进行长度判断。


当然ReplayingDecoder的作用远远不止进行长度判断，更重要的作用是分包传输。


## 3、分包解码器


底层通信协议是分包传输的，一份数据可能分几次达到对端，也就是说发送端出去的包在传输过程会进行多次的拆分和组装。接收端所收到的包和发送端所发送的包不是一模一样的。


在Java OIO流式传输中，不会出现这样的问题，因为他的策略是不读完完整的信息就一直阻塞程序，不向后执行。但是在Java的NIO中，由于NIO的非阻塞性，就会出现接受的数据包和发送端发送的包不是一模一样的情况。比如说发送方发送的是ABC和DEF，而接收方接受到的是ABCD和EF。


对于这种问题，还是可以使用ReplayingDecoder来解决，在进行数据解析时，如果发现当前ByteBuf中所有可读的数据不够，ReplayingDecoder会结束解析直到可读数据足够。这一切都是在ReplayingDecoder内部实现的，不需要用户程序操行。与整数分包传输不同的是，字符串的长度不向整数的长度是固定的，时可变长度的。因此一般来说在Netty中进行字符串的传输可以采用普遍的Header-Content内容传输协议：


1. 在协议的Head部分放置字符串的字节长度，Head部分可以用一个整形int来描述
2. 在协议的Content部分，放置的是字符串的字节数组。


那么在实际传输过程中，一个Header-Content内容包，在发送端会被编码成为一个ByteBuf内容发送包，当到达接收端后可能被分成很多ByteBuf接收包。对于这些参差不齐的接收包，如何解码成为最初的ByteBuf内容发送包呢？


在ReplayingDecoder中有一个很重要的state成员属性，该属性的作用是保存当前解码器在解码过程中的当前阶段。该属性的类型和ReplayingDecoder的泛型一致，并且ReplayingDecoder提供了有参的构造方法初始化这个值。此外还提供了checkpoint(status)方法将状态设置为新的status值并且设置读断点指针。


> 读断点指针是ReplayingDecoder类的另一个重要的成员，它保存着装饰器内部ReplayingDecoderBuffer成员的起始读指针，有点类似mark标记。当读数据时，一旦可读数据不够，ReplayingDecoderBuffer在抛出RelayError异常之前，会把读指针的值还原到之前checkpoint方法设置的读断点指针，因此下次读取时还会从之前设置的断点位置开始。


因此我们可以将读取分为两个阶段，第一个阶段获取长度，第二个阶段获取字符串。根据前面的ReplayingDecoder提供的state属性，我们只需要采用ReplayingDecoder解码器即可实现自定义的字符串分包解码器，代码如下：


```java
public class StringReplayDecoder extends ReplayingDecoder<StringReplayDecoder.Status> {

    private int length;
    private byte[] inBytes;

    public StringReplayDecoder() {
        super(Status.PARSE_1);
    }
  
    @Override
    protected void decode(ChannelHandlerContext ctx, ByteBuf in, List<Object> out) throws Exception {
        switch (state()) {
            case PARSE_1 -> {
                length = in.readInt();
                inBytes = new byte[length];
                checkpoint(Status.PARSE_2);
            }
            case PARSE_2 -> {
                in.readBytes(inBytes, 0, length);
                out.add(new String(inBytes, StandardCharsets.UTF_8));
                checkpoint(Status.PARSE_1);
            }
        }
    }

    enum Status {
        PARSE_1, PARSE_2
    }
}

public class StringProcessHandler extends ChannelInboundHandlerAdapter {

    @Override
    public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
        System.out.println("打印出一个字符串:" + msg);
        super.channelRead(ctx, msg);
    }
}

public class StringReplayDecoderTester {

    public static void main(String[] args) {
        EmbeddedChannel embeddedChannel = new EmbeddedChannel(new ChannelInitializer<EmbeddedChannel>() {
            @Override
            protected void initChannel(EmbeddedChannel ch) throws Exception {
                ch.pipeline().addLast(new StringReplayDecoder())
                        .addLast(new StringProcessHandler());
            }
        });
        final String str = "你好世界！";
        for (int i = 0; i < 3; i++) {
            ByteBuf buf = Unpooled.buffer();
            buf.writeInt((i + 1) * str.getBytes().length);
            for (int j = 0; j < i + 1; j++) {
                buf.writeBytes(str.getBytes(StandardCharsets.UTF_8));
            }
            embeddedChannel.writeInbound(buf);
        }
    }
}
输出结果：
打印出一个字符串:你好世界！
打印出一个字符串:你好世界！你好世界！
打印出一个字符串:你好世界！你好世界！你好世界！
```


可以看到结果成功打印出了我们输入的数据，这是依赖于ReplayingDecoder的state属性实现的，创建这个解码器时默认为STATE1状态，此时会去尝试读取一个整形，读取出来的结果就是我们这次希望读取到的字符串（分包）的长度，接着改变状态到STATE2，并且decode方法结束。因为这里并没有将读出来的结果加入到out列表中，因此不会触发第二个处理器的逻辑。而当读取到字符串时，此时已经处于STATE2状态，因此会调用in.readBytes方法去进行读取指定的长度。如果因为传输过程的拆包原因，此次读取到的字符串并不完整，那么此时达不到目标读入长度，ReplayingDecoderBuffer在抛出RelayError异常之前，会把读指针的值还原到之前checkpoint方法设置的读断点指针，因此下次读取时还会从之前设置的断点位置开始。因此保证了读取到的分包的正确性。


虽然通过这种方式可以正确的解码分包后的ByteBuf数据包，但是在实际开发过程中不太建议继承这个类，原因是：


1. 不是所有的ByteBuf操作都被ReplayingDecoderBuffer装饰类所支持，可能有些操作在decode方法中被使用时就会抛出ReplayError异常
2. 在数据解析逻辑复杂的应用场景，ReplayingDecoder的解析速度相对较差（因为ByteBuf中长度不够时，ReplayingDecoder会捕获一个ReplayError异常，这时会把ByteBuf中的读指针还原到之前的读断点指针(checkpoint)，然后解析这次解析操作等待下一次IO读事件，在网络条件比较糟糕时，一个数据包的解析逻辑会被反复执行多次，如果解析过程是一个消耗CPU的操作，那么对CPU是个大的负担）


因此ReplayingDecoder更多的是应用于数据解析逻辑简单的场景，复杂的场景建议使用ByteToMessageDecoder或其子类，如下：


```java
public class StringIntegerHeaderDecoder extends ByteToMessageDecoder {
    @Override
    protected void decode(ChannelHandlerContext ctx, ByteBuf in, List<Object> out) throws Exception {
        if (in.readableBytes() < 4) {
            return;
        }
        in.markReaderIndex();
        int length = in.readInt();
        if (in.readableBytes() < length) {
            // 重置为读取长度前
            in.resetReaderIndex();
            return;
        }
        byte[] inBytes = new byte[length];
        in.readBytes(inBytes, 0, length);
        out.add(new String(inBytes, StandardCharsets.UTF_8));
    }
}
```


表面上ByteToMessageDecoder基类是无状态的，不像ReplayingDecoder需要使用状态为来保存当前的读取阶段。但是实际上ByteToMessageDecoder内部有一个二进制字节的累积器cumulation，用来保存没有解析完的二进制内容。所以ByteToMessageDecoder及其子类是有状态的业务处理器，不能共享。因此每次初始化通道的流水线时，都需要重新创建一个ByteToMessageDecoder或者它的子类的实例。


## 3、MessageToMessageDecoder解码器


前面的解码器都是讲ByteBuf缓冲区的二进制数据解码成Java的普通POJO对象，而如果想要将一种POJO对象解析成另一种POJO对象，则需要继承一个新的Netty解码器基类——MessageToMessageDecoder<I>，在继承它的时候需要明确泛型实参<I>，它表示入站消息Java POJO类型。


# 二、Netty内置的Decoder


1. 固定长度数据包解码器：FixedLengthFrameDecoder

    * 使用场景：每个接受到的数据包的长度都是固定的
    * 会把入站的ByteBuf拆分成一个个固定长度的数据包（ByteBuf）然后发往下一个channelHandler入站处理器
2. 行分割数据包解码器：LineBasedFrameDecoder

    * 使用场景：每个ByteBuf数据包，使用换行符（或回车换行符）作为数据包的边界分隔符
3. 自定义分隔符数据包解码器：DelimiterBasedFrameDecoder
4. 自定义长度数据包解码器：LengthFieldBasedFramDecoder

    * 一种灵活长度的解码器。在ByteBuf数据包中加了一个长度字段，保存了原始数据报的长度。解码的时候会根据这个长度进行原始数据包的提取


## 1、LineBasedFrameDecoder解码器


前面字符串分包解码器中内容是按照Header-Content协议进行传输的。如果不使用这种协议而是在发送端通过换行符来（"\n"或者"\r\n"）来分割每一次发送的字符串，那么久需要使用LinedBasedFrameDecoder解码器。


这个解码器的工作原理很简单，它一次遍历ByteBuf数据包中的可读字节，判断在二进制字节流中是否存在换行符"\n"或"\r\n"的字节码，如果有就以此位置为结束位置，把从可读索引到结束为止之间的字节作为解码成功后的ByteBuf数据包。


LineBasedFrameDecoder还支持配置一个最大长度值（构造函数传入），表示一行最大能包含的字节数。如果连续读取到最大长度后仍然没有发现换行符就会抛出异常。


## 2、DelimiterBasedFrameDecoder解码器


DelimiterBasedFrameDecoder解码器不仅可以使用换行符，还可以将其他的特殊字符作为数据包的分隔符，例如制表符“\t”。其构造方法如下：


```java
public DelimiterBasedFrameDecoder(int maxFrameLength, 
boolean stripDelimiter, // 解码后数据包是否去掉分隔符
ByteBuf delimiter) // 分隔符
```


分隔符是ByteBuf类型的，也就是需要将分隔符对应的字节数组用ByteBuf包装起来


## 3、LengthFieldBasedFrameDecoder解码器


LengthFieldBasedFrameDecoder解码器可以翻译为长度字段数据包解码器，传输内容中的LengthField长度字段的值，是指存放在数据包中要传输内容的字节数。普通的基于Header-Content协议的内容传输，尽量用内置的LengthFieldBasedFrameDecoder来解码。


其具体的构造函数如下：


```java
public LengthFieldBasedFrameDecoder(
    int maxFrameLength, // 发送的数据包最大长度
    int lengthFieldOffset, // 长度字段偏移值
    int lengthFieldLength， // 长度字段自己占用的字节数
    int lengthAdjustment, // 长度字段的偏移量矫正
    int initialBytesToStrip) // 丢弃的起始字节数
```


1. maxFrameLength：发送的数据包的最大长度
2. lengthFieldOffset：指的是长度字段位于整个数据包内部的字节数组中的下标志
3. lengthFieldLength：长度字段所占的字节数，如果长度字段是一个int整数则为4
4. lengthAdjustment：在传输协议比较复杂的情况下（例如包含了长度字段、协议版本号、魔数等等），解码时需要进行长度矫正。长度校正值的计算公式为：内容字段偏移量-长度字段偏移量-长度字段的字节数
5. initialBytesToStrip：在有效数据字段Content前面，还有一些其他字段的字节，作为最终的解析结果，可以丢弃。


假设我们要传输的数据包ByteBuf（58个字节）包括以下三个部分：


1. 长度字段（4个字节）：52
2. 版本字段（2个字节）：10
3. content字段（52个字节）：xxxxx


那么此时运用LengthFieldBasedFrameDecoder解码器需要传入构造器的参数为：


1. 最大长度可以设置为1024
2. 长度字段偏移量为0
3. 长度字段的长度为4
4. 长度字段的偏移量矫正为2，也就是长度字段距离内容部分的字节数为2
5. 获取最终Content内容的字节数组时，前面6个字节的内容可以抛弃


# 三、Encoder原理与实践


在Netty的业务处理完成后，业务处理的结果往往是某个Java POJO对象，需要编码成最终的ByteBuf二进制类型，通过流水线写入到底层的Java通道。


编码器和解码器相呼应，编码器是一个Outbound出站处理器，将上一站传过来的输入数据（一般是某种Java POJO对象）编码成二进制ByteBuf，或者编码成另一种Java POJO对象。


编码器是ChannelOutboundHandler出站处理器的实现类。一个编码器将出站对象编码后，数据将被传递到下一个ChannelOutboundHandler出站处理器，进行后面的出站处理。由于最后只有ByteBuf才能写入到通道中去，因此可以肯定通道流水线上装配的第一个编码器一定是把数据编码成了ByteBuf类型（出站处理的顺序是从后向前的）。


## 1、MessageToByteEncoder编码器


MessageToByteEncoder编码器是一个非常重要的编码器基类，它的功能是讲一个Java POJO对象编码成一个ByteBuf数据包。它是一个抽象类，仅仅实现了编码的基础流程，在编码过程中，通过调用encode抽象方法来完成，具体的encode逻辑需要由子类去实现。


如果需要实现一个自己的编码器，则需要继承自MessageToByteEncoder基类，实现它的encode抽象方法。继承MessageToByteEncoder时需要带上泛型实参，表示编码之前的Java POJO原类型。


## 2、MessageToMessageEncoder编码器


除了将POJO对象编码成ByteBuf二进制对象，也可以将POJO对象编码成另一种POJO对象。通过继承MessageToMessageEncoder编码器，并且实现它的encode抽象方法。在子类的encode方法实现中，完成原POJO类型到目标POJO类型的编码逻辑。在encode实现方法中，编码完成后，将对象加入到encode方法中的List实参列表中即可。


# 四、解码器和编码器的结合


在流水线处理时，数据的流动往往一进一出，进来时解码，出去时编码。所以在同一个流水线上，加了某种编码逻辑，往往需要加上一个相对应的解码逻辑。


前面讲到的编码器和解码器都是分开实现的，例如通过继承ByteToMessageDecoder基类或者它的子类完成ByteBuf到POJO的解码工作；通过继承MessageToByteEncoder基类或者它的子类完成POJO到ByteBuf数据包的编码工作。总之具有相反逻辑的编码器和解码器实现在两个不同的类中，导致相互配套的编码器和解码器在加入到通道的流水线时，需要分两次添加。


因此Netty提供了新的类型Codec类型，实现具有相互配套逻辑的编码器和解码器放在同一个类中。


## 1、ByteToMessageCodec编解码器


完成POJO到ByteBuf数据包的配套的编码器和解码器的基类，叫做ByteToMessageCodec<I>，它是一个抽象类。从功能上说，继承它就等同于继承了ByteToMessageDecoder解码器和MessageToByteEncoder编码器这两个基类。


ByteToMessageCodec同时包含了编码encode和解码decode两个抽象方法，这两个方法都需要自己实现：


1. 编码方法：encode(ChannelHandlerContext, I, ByteBuf)
2. 解码方法：decode(ChannelHandlerContext, ByteBuf, List<Object>)


## 2、CombinedChannelDuplexHandler组合器


前面的编码器和解码器相结合是通过继承完成的。将编码器和解码器的逻辑强制性地放在同一个类中，在只需要编码或者解码单边操作的流水线上，逻辑上不太合适。


编码器和解码器如果要结合起来，除了继承的方法之外，还可以通过组合的方式实现。与继承相比，组合会带来更大的灵活性：编码器和解码器可以捆绑使用，也可以单独使用。


Netty提供了一个新的组合器——CombinedChannelDuplexHandler基类，继承该类不需要像ByteToMessageCodec那样将编码逻辑和解码逻辑都挤在同一个类中，还是复用原来的编码器和解码器，具体使用方式如下：


```java
public class IntegerDuplexHandler extends CombinedChannelDuplexHandler<Byte2IntegerDecoder, Integer2ByteEncoder> {
	public IntegerDuplexHandler() {
		super(new Byte2IntegerDecoder(), new Integer2ByteEncoder());
	}
}
```


继承该类不需要像ByteToMessageCodec那样把编码解码两个逻辑放在一个类中，还是复用原来的编码器和解码器。总之使用这个类保证了相反逻辑关系的encoder编码器和decoder解码器既可以结合使用，又可以分开使用，十分方便。

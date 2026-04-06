---
title: "NIO 核心组件"
date: 2023-10-13T01:55:43+08:00
draft: false
summary: "介绍 Java NIO 的三大核心组件：Channel（通道）、Buffer（缓冲区）和 Selector（选择器），对比 NIO 与传统 OIO 在面向流/面向缓冲、阻塞/非阻塞以及 IO 多路复用方面的差异。"
tags: [Java, NIO]
categories: [Java]
source: csdn
source_id: "133802251"
---

> 用户程序进行IO的读写，依赖于底层的IO读写，基本上会用到底层的read&write两大系统调用。在不同的操作系统中，IO读写的系统调用的名称可能完全不一样，但是基本功能是一样的。
>
> read系统调用并不是直接从物理设备把数据读取到内存中，write系统调用也不是直接把数据写入到物理设备。上层应用无论是调用操作系统的read还是write，都会涉及缓冲区。**具体来说，调用操作系统的read，是把数据从内核缓冲区复制到进程缓冲区；而调用系统调用的write，是把数据从进程缓冲区复制到内核缓冲区。**因为外部设备的读写设计到操作系统的中断，引入缓冲区可以减少频繁地与设备之间的物理交换，操作系统会对内核缓冲区进行监控，等待缓冲区达到一定的数量的时候（由内核决定，用户程序无需关心），再进行IO设备的中断处理，集中执行物理设备的实际IO操作。
>
> 也就是说上层程序的IO操作，实际上不是物理设备级别的读写，而是缓存的复制。read&write两大系统调用，都不负责数据在内核缓冲区和物理设备（如磁盘）之间的交换。这项底层的读写交换，是由操作系统内核来完成的，即使不调用read&write，当有数据到达网卡时软中断也会将其拷贝到内核缓冲区。


在1.4版本之前，Java IO类库是阻塞IO，从1.4版本开始引进了新的IO库，称为Java New IO类库，简称为Java NIO。New IO类库的目标就是让Java支持非阻塞IO，弥补了原本面向流的OIO（Old IO）同步阻塞的不足，它为标准Java代码提供了高速的、面向缓冲区的IO。


Java NIO由以下三个核心组件组成：


* **Channel（通道）**

  * 在OIO中，同一个网络连接会关联到两个流，一个输入流，一个输出流，通过这两个流不断的进行输入和输出的操作
  * 在NIO中，同一个网络连接使用一个通道表示，所有NIO的IO操作都是从通道开始的，一个通道类似于OIO中的两个流的接合体，既可以从通道读取，也可以向通道写入
* **Buffer（缓冲区）**

  * 通道的读取就是将数据从通道读取到缓冲区中；通道的写入就是将数据缓冲区中写入到通道中。
* **Selector（选择器）**

  * 用于实现对多个文件描述符的监视，通过选择器，一个线程可以查询多个通道的IO事件的就绪状态。
  * 与OIO相比，使用选择器的最大优势就是系统开销小，不需要为每个网络连接（文件描述符）创建进程/线程，使用一个线程就可以管理多个通道。


在Java中，NIO和OIO的区别主要体现在三个方面：


1. **OIO是面向流的，NIO是面向缓冲区的**

    * OIO是面向字节流或字符流的，在一般的OIO操作中，我们以流式的方式**顺序地**从一个流中读取一个或多个字节，因此我们不能随意地改变读取指针的位置。
    * NIO中引入了Channel和Buffer的概念，**读取和写入只需要从通道中读取数据到缓冲区，或将数据从缓冲区中写入到通道中，可以随意地读取Buffer中任意位置的数据。**
2. **OIO的操作是阻塞的，而NIO的操作是非阻塞的**
3. **OIO没有选择器概念，而NIO有选择器的概念（IO多路复用）**


## 一、Buffer


NIO的Buffer类是一个抽象类，位于java.nio包中，提供了一组更加有效的方法，用来进行写入和读取的交替访问，本质上是一个内存块（数组），既可以写入数据，也可以从中读取数据。


需要强调的是**Buffer类是一个非线程安全类**。


在NIO这种有8种缓冲区类，分别为ByteBuffer、CharBuffer、ShortBufffer、IntBuffer、LongBuffer、FloatBuffer、DoubleBuffer、MappedByteBuffer。前7种Buffer类型覆盖了能在IO中传输的所有Java基本数据类型，第8种数据类型MappedByteBuffer是专门用于内存映射的一种ByteBuffer类型。实际上使用最多的还是ByteBuffer二进制字节缓冲区类型。


### 1、重要属性


Buffer类在其内部有一个对应类型的数组（如ByteBuffer的byte[]数组）作为内存缓冲区，为了记录读写的状态和位置，Buffer类提供了一些重要的属性，其中有三个重要的成员属性：


1. capacity：容量（一旦初始化就不能再改变）

    * 表示内部容量的大小，一旦写入的对象数量超过了capacity容量，缓冲区就满了，不能再写入了。
2. position：读写位置

    * 表示当前的位置。position属性与缓冲区的读写模式有关，在不同的模式下position属性的值是不同的，当缓冲区进行读写的模式改变时，position会进行调整。
    * 写入模式：刚进入写模式时，position值为0，表示当前写入的位置从头开始。每当一个数据写到缓冲区之后，position会向后移动到下一个可写的位置。最大可写值position为limit-1，当position值达到limit时，缓冲区就已经无空间可写了。
    * 读取模式：刚进入读模式时，position值被重置为0，表示当前读取的位置从头开始。当从缓冲区读取时，也是从position位置开始读，读取之后position向后移动到下一个可读的位置。position最大的值为最大刻度上限limit，当position达到limit时表示缓冲区已经无数据可读。
    * 当新建一个缓冲区时，缓冲区处于写入模式，这时是可以写数据的。数据写入后，如果要从缓冲区读取数据，这就要进行模式的切换，可以使用flip翻转方法，将缓冲区变为读取模式。在flip翻转过程中会将position由原来的写入位置，变为新的可读位置，也就是0，表示可以从头开始读。此外flip还会调整limit属性的值。
3. limit：读写的限制

    * 表示读写的最大上线，和缓冲区的读写模式有关。
    * 写入模式：在写模式下limit属性值的含义为可以写入的数据最大上限，在刚进入到写模式时，limit的值会被设置成缓冲区的capacity容量值，表示可以将缓冲区的容量写满。
    * 读取模式：在读模式下limit属性值的含义为最多能从缓冲区中读取到多少数据。
    * 一般来说是先写入再读取，当缓冲区写入完成后，就可以开始从Buffer读取数据，可以使用flip翻转方法，这时会将写模式下的position值设置为读模式下的limit值。


### 2、重要方法


#### 1）allocate()创建缓冲区


在使用Buffer之前，我们首先需要获取Buffer子类的实例对象，并且分配内存空间。获取一个Buffer实例对象并不是使用子类的构造器new来创建一个实例对象，而是调用子类的allocate()方法，该方法需要传入一个int类型的参数，表示缓冲区的容量。


```java
public static void main(String[] args) throws IOException {
        CharBuffer buffer = CharBuffer.allocate(20);
        System.out.println("缓冲区的capacity:" + buffer.capacity());
        System.out.println("缓冲区的position:" + buffer.position());
        System.out.println("缓冲区的limit:" + buffer.limit());
}
缓冲区的capacity:20
缓冲区的position:0
缓冲区的limit:20
```


#### 2）put()写入到缓冲区


在调用allocate方法分配内存、返回了实例对象后，缓冲区实例对象处于写模式，可以写入对象。要写入缓冲区，需要调用put方法。


put方法只有一个参数，即为所需要写入的对象，数据类型要求与缓冲区的类型保持一致。


#### 3）flip()翻转


向缓冲区写入数据之后是不可以直接从缓冲区中读取数据的，因为此时缓冲区还处于写模式，如果需要读取数据，还需要将缓冲区转换成读模式。那么此时就需要使用flip()方法进行翻转。**flip()方法的作用就是将写入模式翻转成读取模式。**


对于flip()方法的从写入到读取转换的规则：


1. 首先设置可读的长度上限limit，将写模式下缓冲区中内容的最后写入位置position值作为读模式下的limit上限值
2. 其次把读的起始位置position的值设为0，表示从头开始读
3. 最后清除之前的mark标记（mark保存的是一个临时位置）


> flip()的作用是将写入模式转换为读取模式，那么如何将缓冲区切换成读取模式呢？
>
> 一般来说可以通过调用clear()清空或者compact()压缩方法，它们可以将缓冲区转换为写模式。


#### 4）get()从缓冲区读取


调用flip方法将缓冲区切换成读取模式之后就可以开始从缓冲区中进行数据读取了。


get方法每次从position的位置读取一个数据，并且进行相应的缓冲区属性的调整。


读取操作会改变刻度位置position的值，而limit值不会改变，如果position值和limit的值相等，表示所有数据读取完成，position只想了一个没有数据的元素位置，已经不能再读了，此时再读会抛出BufferUnderflowException异常。


在读完之后不可以立即进行写入操作，必须调用clear或compact方法清空或者压缩缓冲区才能编程写入模式，让其重新可写。


#### 5）rewind()倒带


已经读完的数据如果需要再读一遍，可以调用rewin()方法，rewind()也叫倒带，就像播放磁带一样倒回去，再重新播放。


rewind()方法主要是调整了缓冲区的position属性，具体的调整规则如下：


1. position重置为0，所以可以重读缓冲区的所有数据
2. limit保持不变
3. mark标记被清理，之前的临时位置不能再用了


rewind()方法与flip很像是，区别在于rewind不会影响limit属性值，而flip会重设limit属性值。


#### 6）mark()和reset()


mark方法的作用是将当前的position的值保存起来，放在mark属性中，让mark属性记住这个临时位置，之后可以调用reset方法将mark的值恢复到position中。


#### 7）clear()清空缓冲区


在读取模式下调用clear方法将缓冲区切换为写入模式，此方法会将position清零，limit设置为capacity最大容量值。


#### 8）使用Buffer类的基本步骤


1. 使用创建子类实例对象的allocate()方法创建一个Buffer类的实例对象
2. 调用put方法将数据写入到缓冲区中
3. 写入完成后在开始读取数据前调用flip()方法将缓冲区转换为读模式
4. 调用get方法从缓冲区中读取出数据
5. 读取完成后调用clear()或compact()方法将缓冲区转换为写入模式


## 二、Channel


NIO中一个连接就是用一个Channel（通道）来表示，一个通道可以表示一个底层的文件描述符，例如硬件设备、文件、网络连接等，除此之外Java NIO的通道还可以更加细化，例如对应不同的网络传输协议类型，在Java中都有不同的NIO Channel实现。


Channel主要有四种重要的类型：


1. FileChannel：文件通道，用于文件的数据读写
2. SocketChannel：套接字通道，用于Socket套接字TCP连接的数据读写
3. ServerSocketChannel：服务器嵌套字通道（或服务器监听通道），允许我们监听TCP连接请求，为每个监听到的请求创建一个SocketChannel套接字通道
4. DatagramChannel：数据报通道，用于UDP协议的数据读写


### 1、FileChannel文件通道


FileChannel是专门操作文件的通道，它是阻塞模式的，不能设置为非阻塞模式。具体的操作如下：


1. 获取通道

    1. 通过文件的输入流、输出流获取：new FileInputStream(srcFile).getChannel() / new FileOutputStream(destFile).getChannel()
    2. 通过RandomAccessFile文件随机访问类获取：new RandomAccessFile("filename.txt", "rw").getChannel()
2. 读取通道

    1. 通过调用通道的int read(ByteBuffer buf)将通道的数据读取到ByteBuffer缓冲区中，并返回读取到的数据量
    2. 对于通道来说是读取数据，对于ByteBuffer缓冲区来说是写入数据，所以此时ByteBuffer缓冲区需要处于写入模式
3. 写入通道

    1. 通过调用通道的int write(ByteBuffer buf)方法将ByteBuffer缓冲区的数据写入到通道中，并返回写入成功的字节数
    2. 此时的ByteBuffer缓冲区要求是可读的，处于读模式下
4. 关闭通道：channel.close()
5. 强制刷新到磁盘：

    1. 在将缓冲区写入通道时，由于性能原因，操作系统不可能每次都实时将数据写入磁盘，如果需要保证数据真正落盘需要调用force()方法
    2. force方法接受一个布尔参数，如果为true，则该方法需要强制更改文件的内容和将元数据写入存储;否则，它只需要强制写入内容更改


### 2、SocketChannel套接字通道


在NIO中设计网络连接的通道有两个，一个是SocketChannel负责连接传输，一个是ServerSocketChannel负责连接的监听。


NIO中SocketChannel传输通道与OIO中的Socket类对应，NIO中的ServerSocketChannel监听通道与OIO中的ServerSocket对应。


ServerSocketChannel应用于服务器端，而SocketChannel同时处于服务器端和客户端。换句话说，对于一个连接，两端都有一个负责传输的SocketChannel传输通道。


这两种Channel都可以通过configureBlocking()方法设置是否为阻塞模式。在阻塞模式下，connect连接、read、write操作都是同步阻塞的，效率上和Java旧的OIO的面向流的阻塞式读写操作相同。


1. 获取通道

    1. 服务端：ServerSocketChannel.open() / server.accept()
    2. 客户端：SocketChannel.open()
2. 读取通道和写入通道同样为read和write
3. 关闭通道：在关闭SocketChannel传输通道前，如果传输通道用来写入数据，则建议调用shutdownOutput()终止输出方法，向对方发送一个输出的结束标志(-1)，然后再调用close方法关闭套接字


### 3、DatagramChannel数据报通道


和Socket套接字的TCP传输协议不同，UDP协议不是面向连接的协议。使用UDP协议时只要知道服务器的IP和端口就可以直接向对方发送数据。


1. 获取通道：Datagram.open()
2. 读取通道：channel.receive(buf)，返回值为SocketAddress类型，表示返回发送端的连接地址
3. 写入通道：channel.send(buffer, new InetSocketAddress(ip, port))
4. 关闭：直接调用close()即可


## 三、Selector


Selector选择器的使命是完成IO的多路复用。一个通道代表一条连接通路，通过选择器可以同时监控多个通道的IO状况。选择器和通道的关系，是监控和被监控的关系。


选择器提供了独特的API，能够选出（select）所监控的通道拥有哪些已经准备好的、就绪的IO操作事件。


通道和选择器之间的关系，通过register（注册）的方式完成，调用通道的register(Selector sel, int ops)方法，可以将通道实例注册到一个选择器中。register方法有两个参数：第一个指定通道注册到的选择器实例，第二个指定选择器要监控的IO事件类型。可供选择器监控的通道IO事件类型包括以下四种：


1. 可读：SelectionKey.OP_READ
2. 可写：SelectionKey.OP_WRITE
3. 连接：SelectionKey.OP_CONNECT
4. 接收：SelectionKey.OP_ACCEPT


如果选择器要监控通道的多种事件，可以用“按位或”运算符来实现。


> 并不是所有的通道都是可以被选择器监控或选择的。比方说FileChannel文件通道选择器就不能被选择器复用。
>
> 判断一个通道能否被选择器监控或选择有一个前提：判断它是否继承了抽象类SelectableChannel（可选择通道），如果继承了则可以被选择，否则不能被选择。该抽象类中定义了register()、configureBlocking()、isBlocking()等方法。


通道和选择器的监控关系注册成功后就可以选择就绪时间。具体的选择工作由选择器的select()方法来完成。通过该方法，选择器可以不断地选择通道中发生操作的就绪状态，返回注册过的感兴趣的那些IO事件（函数放回的是感兴趣的IO事件的数量，）。也就是说一旦通道中发生了我们在选择器中注册过的IO事件，就会被选择器选中并放入SelectionKeys选择间的集合中。SelectionKey选择键不仅可以获得通道的IO事件类型，还可以获得发生IO事件所在的通道，此外还可以获得选出选择键的选择器实例。使用方式：


```java
public static void main(String[] args) throws IOException {
    try (Selector selector = Selector.open()) {
        while (selector.select() > 0) {
            Set<SelectionKey> selectionKeys = selector.selectedKeys();
            for (SelectionKey selectionKey : selectionKeys) {
                if (selectionKey.isAcceptable()) {

                } else if (selectionKey.isReadable()) {

                } else if (selectionKey.isWritable()) {

                } else if (selectionKey.isConnectable()) {
                }
            }
        }
    }
}
```

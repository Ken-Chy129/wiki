---
title: "FileZilla 搭建公网 FTP 服务器"
date: 2022-07-23T16:08:09+08:00
draft: false
summary: "介绍如何使用 FileZilla 搭建可公网访问的 FTP 服务器，通过 Frp 或花生壳实现 NAT 穿透，详解 FTP 主动模式与被动模式的数据连接建立方式及 NAT 环境下的注意事项。"
tags: [FRP, NAT]
categories: [Networking]
source: csdn
source_id: "125948960"
---

FTP 是一种基于 TCP 的应用层协议，它不支持 UDP 协议。 FTP 工作在一种特殊的服务机制上，它使用两个端口，一个 '数据' 端口和一个 '命令' 端口（也称为控制端口）。 通常情况下，端口 21 用作命令端口，端口 20 用作数据端口。 


### 一、主动模式和被动模式


#### 1、主动模式


客户端从一个任意的非特权端口N（N>1024）连接到FTP服务器的命令端口（21端口），发送用户名和密码登录，登录成功后要list列表或者读取数据时，客户端开放N+1端口，发送 PORT N+1 命令到FTP服务器，告诉服务器客户端采用主动模式以及开放的端口；FTP服务器收到PORT主动模式命令和端口号后，通过服务器的20端口和客户端开放的端口N+1连接，发送数据。


![img](/images/filezilla-ftp-frp/dadaf97acad1a6453c3c735fb97b99cc.png)


#### 2、被动模式


为了解决服务器主动发起到客户端连接会被阻止的问题。被动模式工作的前提是客户端明确告知 FTP 服务器它使用被动模式。FTP客户端使用N(N>1023)端口连接到FTP服务器的21端口，发送用户名和密码登录，登录成功后要list列表或者读取数据时，发送PASV命令到FTP服务器， 服务器在本地开放一个端口（1024以上），然后把开放的端口告诉客户端， 客户端再通过N+1端口连接到服务器开放的端口进行数据传输。


![img](/images/filezilla-ftp-frp/c19d615b714eac927c8a27931ff2ebd9.png)


#### 3、区别


显然两者的区别就在于建立数据传输连接的方式。


主动模式的连接发起方为服务器端，服务器使用20号端口主动去连接客户端的N+1端口建立数据连接；


然而当客户端位于NAT之后，服务器是无法主动与其建立连接的


被动模式连接发起方为客户端，在客户端与21端口建立联系之后服务器端告诉客户端自己使用的数据端口号，让客户端自己来连接，这样就解决了服务端无法主动连接客户端的问题。


### 二、搭建FTP服务器


首先我们需要先安装FileZilla Server


安装完成后界面中为出现警告


![img](/images/filezilla-ftp-frp/6d41d109dfab7fc06563c6507c859d98.png)


这里的意思是当前环境位于NAT之后，所以需要使用被动模式(passive mode)运行并在路由器上设置端口转发；warning中的意思是FTP服务没有启用TLS模式，所以用户登录信息是不安全的。


在上文中我们讲到了被动模式解决了客户端位于NAT之后，服务器端无法直接连接其的情况下采用被动模式让客户端来连接服务器端即可。但是如果我们的服务器端也是位于NAT之后，那么其实客户端也无法连接到服务器端。（正如下图提示所说）那我们还需要将我们服务器FTP服务的端口暴露到公网中供客户端可以连接，这里我们就才用到内网穿透的方式来解决NAT的问题（相关知识可参考[内网穿透的原理和实现方式](https://www.ken-chy129.cn/archives/73)）


![image-20220717220555518](/images/filezilla-ftp-frp/86f913ddb5ffa5150cb4c5d5b3594139.png)


这里我们需要设置自定义端口和填入外网IP的地址


端口即为进行内网穿透的端口号，ip地址则填写公网的ip


如果我们拥有公网ip，则可以使用frp进行端口转发，如果没有则可以使用花生壳等内网穿透工具。


### 三、使用FRP实现


关于Frp的使用可以参考文章[使用frp端口实现内网穿透](https://www.ken-chy129.cn/archives/77)


在完成了基础的配置后，我们只需要在客户端的frpc.ini中增加两个配置


```
[ftp_cmd]
type = tcp
local_ip = 127.0.0.1
local_port = 21
remote_port = 6001

[ftp_data]
type = tcp
local_ip = 127.0.0.1
local_port = 15779
remote_port = 15779
```


第一个即为TCP服务所监听的21号端口及其在公网上的映射


由于我们使用被动模式，所以还需要另外设置一个进行数据通信的端口


local即内网进行数据通信的端口，remote即映射到公网的端口，**此处与File Zilla中的端口都保持一致**


之后使用公网IP+[ftp_cmd]中的remote_port即可连接TCP服务


记得需要在防火墙中放行这些端口！


### 四、使用花生壳实现


花生壳的使用同理，区别就是它会为你提供一个公网ip，不需要你自己拥有，而因此你也无法自定义使用哪个端口进行通信，而是尤其自动进行分配


我们同样创建两个映射，一个是21号端口的映射


![image-20220717224401928](/images/filezilla-ftp-frp/52c8101f129568e42c9537f4b651f505.png)


另一个我们先随便现在内网端口中填写一个端口号，如下


![image-20220717224753626](/images/filezilla-ftp-frp/0e35ad1a023debe18e0b9b9fa07a06ae.png)


随后它会为我们分配公网的端口号


![image-20220717224829062](/images/filezilla-ftp-frp/2872b7efef55d32d4d8bfd507ca66dbd.png)


我们再对这个映射进行编辑，将内网端口与分配的外网端口保持一致


![image-20220717225555181](/images/filezilla-ftp-frp/387e5e546da634c38c3f639583cd1286.png)


在得到这个端口号后再去file zilla中填写被动模式中的自定义端口范围。


即如果自己有公网ip则可以使用frp自己设置开放哪个端口进行数据通信，没有的话则用花生壳这样的工具提供的公网和分配的端口，得到端口和公网地址之后再返回去file zilla中设置ip和端口


### 五、其他设置


禁用ip检查


![image-20220717225815137](/images/filezilla-ftp-frp/ea43e4be80241e95d80e073b31e4baf3.png)


设置证书（点击生成新证书后填写相关信息即可）


![image-20220717225922212](/images/filezilla-ftp-frp/c018e992d7777da05840d66d133d9d97.png)


配置用户组


添加用户，设置密码


![image-20220717230040062](/images/filezilla-ftp-frp/911df699e94ece27385d6dd177b8b01b.png)


设置共享文件夹和访问权限


![image-20220717230122617](/images/filezilla-ftp-frp/638957a08520abb3fcc3c89fb900f2a2.png)


### 六、设置开机自启动


[服务器端(linux)自启动](https://www.ken-chy129.cn/archives/77)


[客户端(windows)自启动](https://blog.csdn.net/leadseczgw01/article/details/103298118)

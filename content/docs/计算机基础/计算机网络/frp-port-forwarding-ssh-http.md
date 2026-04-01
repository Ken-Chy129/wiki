---
title: "Frp 内网穿透实战"
date: 2022-07-18T18:52:58+08:00
draft: false
summary: "使用frp端口映射实现内网穿透(SSH、HTTP服务) 一、下载 通过的学习我们已经明白了内网穿透的原理，想要实现内网穿透就需要让内网实现与具有公网IP的设备进行绑定。 我们这里使用frp（一个专注于内网穿透的高性能的反向代理应用，支持 TCP、UDP、HTTP、HTTPS 等多种协议。可以将内网服务以安全、便捷的方式"
tags: [FRP, NAT]
categories: [Networking]
source: csdn
source_id: "125857308"
---

### 使用frp端口映射实现内网穿透(SSH、HTTP服务)

#### 一、下载

通过[内网穿透的原理和实现方式](<https://www.ken-chy129.cn/archives/73>)的学习我们已经明白了内网穿透的原理，想要实现内网穿透就需要让内网实现与具有公网IP的设备进行绑定。

我们这里使用frp（一个专注于内网穿透的高性能的反向代理应用，支持 TCP、UDP、HTTP、HTTPS 等多种协议。可以将内网服务以安全、便捷的方式通过具有公网 IP 节点的中转暴露到公网）进行内网穿透。下载地址：https://github.com/fatedier/frp/releases

我们需要在内网设备和公网设备上都进行frp的下载。

  * fprc：客户端程序；（内网程序）
  * frpc.ini：客户端程序配置文件；
  * frps：服务网器端程序；（公网程序，进行端口映射服务）
  * frps.ini：服务器端程序配置文件；

#### 二、基础配置配置

1）对于公网设备，我们需要配置frps.ini文件
[code] 
    [common]
    bind_port = 7000
    token = 129496
    
    dashboard_addr = 0.0.0.0
    dashboard_port = 7500
    dashboard_user = root
    dashboard_pwd = xxxx
    
[/code]

  1. bind_port：用户客户端与服务端连接的端口（让外网设备可以将请求信息转发给内网让内网主动与发起请求设备进行连接）
  2. token：认证密码
  3. dashboard：仪表盘 
     * addr：服务器本机
     * port：仪表盘网页绑定在服务器的端口
     * user：访问仪表盘网页的用户名

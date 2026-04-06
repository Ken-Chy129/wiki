---
title: "数据卷与数据同步"
date: 2022-11-01T17:47:18+08:00
draft: false
summary: "一、什么是数据卷 数据卷是一个可供容器使用的特殊目录，它将主机操作系统目录直接映射进容器，类似于Linux中的mount操作。 数据卷可以提供很多有用的特性，如下所示： 1. 数据卷可以在容器之间共享和重用，容器间传递数据将变得高效方便； 2. 对数据卷内数据的修改会立马生效，无论是容器内操作还是本..."
tags: [Docker]
categories: ["Cloud Native"]
source: csdn
source_id: "127638133"
source_url: "https://blog.csdn.net/qq_25046827/article/details/127638133"
---

### 一、什么是数据卷


数据卷是一个可供容器使用的特殊目录，它将主机操作系统目录直接映射进容器，类似于Linux中的mount操作。


数据卷可以提供很多有用的特性，如下所示：


1. 数据卷可以在容器之间共享和重用，容器间传递数据将变得高效方便；
2. 对数据卷内数据的修改会立马生效，无论是容器内操作还是本地操作；
3. 对数据卷的更新不会影响镜像，解耦了应用和数据；
4. 卷会一直存在，直到没有容器使用，可以安全地卸载它。


### 二、使用数据卷


#### 1、通过命令挂载 -v


```shell
docker run -d -v 主机目录:容器内目录

# 检查
docker inspect 容器

# 示例
docker run -p 3307:3306 --name mysql02 -v /data/mysql02/conf.d:/etc/mysql/conf.d -v /data/mysql02/data:/var/lib/mysql -v /data/mysql02/my.cnf:/etc/mysql/my.cnf -e MYSQL_ROOT_PASSWORD=123456 -d  --restart=always --privileged=true mysql

# 参数说明
  --restart=always： 当Docker 重启时，容器会自动启动。
  --privileged=true：容器内的root拥有真正root权限，否则容器内root只是外部普通用户权限
  -v /home/mysql/conf.d/my.cnf:/etc/my.cnf：映射配置文件
  -v /home/mysql/data/:/var/lib/mysql：映射数据目录

docker inspect mysql02
# 结果
"Mounts": [
            {
                "Type": "bind",
                "Source": "/data/mysql02/log",
                "Destination": "/var/log/mysql",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            },
            {
                "Type": "bind",
                "Source": "/data/mysql02/data",
                "Destination": "/var/lib/mysql",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            },
            {
                "Type": "bind",
                "Source": "/data/mysql02/conf",
                "Destination": "/etc/mysql",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            }
        ]
```


> 启动mysql可能会出现以下问题，这是因为MYSQL新特性secure_file_priv对读写文件的影响
>
> ```shell
> docker logs -ft mysql02
> 
> mysqld: Error on realpath() on '/var/lib/mysql-files' (Error 2 - No such file or directory)
> 2019-09-14T09:52:51.015937Z 0 [ERROR] [MY-010095] [Server] Failed to access directory for --secure-file-priv. Please make sure that directory exists and is accessible by MySQL Server. Supplied value : /var/lib/mysql-files
> 2019-09-14T09:52:51.018328Z 0 [ERROR] [MY-010119] [Server] Aborting
> ```
>
> 解决方法:
>
> -   windows下：修改my.ini 在[mysqld]内加入secure_file_priv=/var/lib/mysql
> -   linux下：修改my.cnf 在[mysqld]内加入secure_file_priv=/var/lib/mysql
>
> 我们可以通过挂载的方式修改该文件，即现在宿主机创建并修改好my.cnf文件，随后挂载到容器中
>
> ```ini
> [mysqld]
> user=mysql
> character-set-server=utf8
> default_authentication_plugin=mysql_native_password
> secure_file_priv=/var/lib/mysql
> expire_logs_days=7
> sql_mode=STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION
> max_connections=1000
> 
> [client]
> default-character-set=utf8
> 
> [mysql]
> default-character-set=utf8
> ```


#### 2、具名挂载和匿名挂载


> 匿名挂载


```shell
-v 容器内路径
docker run -d -P --name nginx01 -v /etc/nginx nginx

# 查看所有的volume的情况
docker volume ls

[root@VM-12-14-centos ~]# docker volume ls
DRIVER    VOLUME NAME
local     4040579b72a0b726cbe6448b6f22d315743d8b3f47158ca50b6e6c72c747252a
# 以上便是匿名挂载，-v只写了容器内的路径，没有写容器外的路径或卷的名称
# 于是会生成一串随机的卷名

[root@VM-12-14-centos ~]# docker volume inspect 4040579b72a0b726cbe6448b6f22d315743d8b3f47158ca50b6e6c72c747252a
[
    {
        "CreatedAt": "2022-10-30T17:34:26+08:00",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/var/lib/docker/volumes/4040579b72a0b726cbe6448b6f22d315743d8b3f47158ca50b6e6c72c747252a/_data",
        "Name": "4040579b72a0b726cbe6448b6f22d315743d8b3f47158ca50b6e6c72c747252a",
        "Options": null,
        "Scope": "local"
    }
]
# 查看这个卷，可以看到挂载的地址位于
# "/var/lib/docker/volumes/4040579b72a0b726cbe6448b6f22d315743d8b3f47158ca50b6e6c72c747252a/_data"
```


> 具名挂载


```shell
docker run -d -P --name nginx01 -v juming:/etc/nginx nginx

[root@VM-12-14-centos ~]# docker volume ls
DRIVER    VOLUME NAME
local     juming

[root@VM-12-14-centos ~]# docker volume inspect juming
[
    {
        "CreatedAt": "2022-11-01T16:24:47+08:00",
        "Driver": "local",
        "Labels": null,
        "Mountpoint": "/var/lib/docker/volumes/juming/_data",
        "Name": "juming",
        "Options": null,
        "Scope": "local"
    }
]
```


观察发现如果没有设置宿主机的路径，docker内的卷都会默认在`/var/lib/docker/volumes/xxxx(volume_name)/_data`目录下


我们通过具名挂载可以方便地找到一个卷，大多数都会采用这样的方式


#### 3、区分几种命令挂载方式


- -v 容器内路径：匿名挂载
- -v 卷名:容器内路径：具名挂载
- -v /宿主机路径:容器内路径：指定路径挂载


> 扩展
>
> 通过 -v 容器内路径：ro(readonly)、rw(readwrite)改变读写权限
>
> 默认是rw，一旦设置了ro就只能在宿主机进行写操作，在容器里只能读


#### 4、通过Dockerfile挂载


dockerfile就是用来构建docker镜像的构建文件


```shell
cd /data/volume

touch dockerfile1

vim dockerfile1
```


```shell
FROM centos

VOLUME ["volume01", "volume02"]

CMD echo "----end----"

CMD /bin/bash
```


```shell
docker build -f /data/volume/dockerfile1 -t ken/centos:1.0 .

[root@VM-12-14-centos volume]# docker run -it ken/centos:1.0 /bin/bash
[root@5fd3c2f7e99e /]# ls
bin  etc   lib    lost+found  mnt  proc  run   srv  tmp  var       volume02
dev  home  lib64  media       opt  root  sbin  sys  usr  volume01
# 可以看到出现了volume01和volume02两个目录

# 另起一个终端使用inspect查看容器也能看到
"Mounts": [
            {
                "Type": "volume",
                "Name": "c3326f0d2058d43d33597b8877b134d0b9d3ee649d4a27121306c5b4233ed0c6",
                "Source": "/var/lib/docker/volumes/c3326f0d2058d43d33597b8877b134d0b9d3ee649d4a27121306c5b4233ed0c6/_data",
                "Destination": "volume01",
                "Driver": "local",
                "Mode": "",
                "RW": true,
                "Propagation": ""
            },
            {
                "Type": "volume",
                "Name": "8160d334c70dd048bcc3176e6b0b72e68b2bda81dfe724e9a6c561e4e7324fa1",
                "Source": "/var/lib/docker/volumes/8160d334c70dd048bcc3176e6b0b72e68b2bda81dfe724e9a6c561e4e7324fa1/_data",
                "Destination": "volume02",
                "Driver": "local",
                "Mode": "",
                "RW": true,
                "Propagation": ""
            }
        ]
```


### 三、数据卷容器


如果用户需要在多个容器之间共享一些持续更新的数据，最简单的方式是使用数据卷容器。数据卷容器也是一个容器，但是它的目的是专门用来提供数据卷供其他容器挂载。


```shell
# 通过--volumes-from来挂载mysql02容器中的数据卷
docker run -d --volumes-from mysql02 --name mysql03 -e MYSQL_ROOT_PASSWORD=123456 -d --restart=always --privileged=true -p 3308:3306 mysql

# 通过inspect查看mysql03，可以看到挂载了和mysql02一样的数据卷
"Mounts": [
            {
                "Type": "bind",
                "Source": "/data/mysql02/conf.d",
                "Destination": "/etc/mysql/conf.d",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            },
            {
                "Type": "bind",
                "Source": "/data/mysql02/data",
                "Destination": "/var/lib/mysql",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            },
            {
                "Type": "bind",
                "Source": "/data/mysql02/my.cnf",
                "Destination": "/etc/mysql/my.cnf",
                "Mode": "",
                "RW": true,
                "Propagation": "rprivate"
            }
        ]
```


可以多次使用--volumes-from参数来从多个容器挂载多个数据卷。还可以从其他已经挂载了容器卷的容器来挂载数据卷。


使用--volumes-from参数所挂载数据卷的容器自身并不需要保持在运行状态。


如果删除了挂载的容器（mysql02），数据卷并不会被自动删除。如果要删除一个数据卷，必须在删除最后一个还挂载着它的容器时显式使用docker rm -v命令来指定同时删除关联的容器。




















---
title: "容器化技术"
date: 2022-10-30T22:42:54+08:00
draft: false
summary: "一、虚拟机与容器的比较 在容器化技术出来之前，使用的是虚拟机技术，虚拟机和Docker容器技术都是一种虚拟化技术 虚拟机包含的是整个操作系统的原生镜像，非常的庞大，而docker的镜像只包含最核心的环境，非常小巧。 1、虚拟机技术 缺点： - 资源占用十分多 - 冗余步骤多 - 启动慢 2、容器化技..."
tags: [Docker]
categories: ["Cloud Native"]
source: csdn
source_id: "127606661"
source_url: "https://blog.csdn.net/qq_25046827/article/details/127606661"
---

### 一、虚拟机与容器的比较


>在容器化技术出来之前，使用的是虚拟机技术，虚拟机和Docker容器技术都是一种虚拟化技术
>
>虚拟机包含的是整个操作系统的原生镜像，非常的庞大，而docker的镜像只包含最核心的环境，非常小巧。


![img](/images/docker-containerization/1b4c7c0d08515ba77584bd1c1632b13a.png)


#### 1、虚拟机技术


![image-20221028165406184](/images/docker-containerization/968497a70c8b158e23257db392f3708f.png)


缺点：


- 资源占用十分多
- 冗余步骤多
- 启动慢


#### 2、容器化技术


容器化技术不是模拟的一个完整的操作系统


![image-20221028165346100](/images/docker-containerization/716eb5d5ef8b075832387c95210e9fca.png)


比较Docker与虚拟机技术的不同：


1. 传统虚拟机，虚拟出一套硬件，运行一个完整的操作系统，然后在这个操作系统上安装和运行软件
2. Docker 容器内的应用进程直接运行在宿主机的内核（内核级虚拟化），容器内没有自己的内核且也没有进行硬件虚拟。因此容器要比传统虚拟机更为轻便。
3. 每个容器是互相隔离的，每个容器有属于自己的文件系统，容器之间进行不会相互影响，能区分计算资源


#### 3、容器的优点


- 应用更快速的交付和部署，打包镜像发布测试，一键运行
- 更快捷的升级和扩缩容
- 更简单的系统运维，开发、测试环境高度一致
- 更高效的计算资源利用


#### 4、Docker比虚拟机快的原因


- Docker有着比虚拟机更少的抽象层，Docker不需要实现硬件资源虚拟化，而是直接使用实际物理机的硬件资源，因此在Cpu、内存利用率上Docker将会在效率上有明显优势。

- Docker利用的是宿主机的内核，当新建一个容器时，不需要和虚拟机一样重新加载一个操作系统，避免了引导、加载操作系统内核这个比较费时费资源的过程，当新建一个虚拟机时，虚拟机软件需要加载Guest OS，这个新建过程是分钟级别的，而Docker由于直接利用宿主机的操作系统则省略了这个过程，因此新建一个Docker容器只需要几秒钟。


### 二、Docker的名词概念


#### 1、镜像（image）


docker镜像就好比是一个模板，可以通过这个模板来创建容器服务


如tomcat镜像 ===》run ===》tomcat01容器（提供服务）


通过一个镜像可以创建多个容器，最终服务运行或项目运行就是在容器中的


#### 2、容器（container）


Docker利用容器技术，独立运行一个或一组应用，通过镜像来创建的


拥有启动、停止、删除等基本命令


可以把容器理解为一个建议的linux系统


#### 3、仓库（repository）


仓库就是存放镜像的地方，分为共有和私有


Docker Hub（默认是国外的），阿里云等厂商都有提供容器服务


### 三、Docker安装


#### 1、卸载旧版本


```shell
$ sudo yum remove docker \
                  docker-client \
                  docker-client-latest \
                  docker-common \
                  docker-latest \
                  docker-latest-logrotate \
                  docker-logrotate \
                  docker-engine
```


#### 2、设置Docker仓库


在新主机上首次安装 Docker Engine-Community 之前，需要设置 Docker 仓库。之后，您可以从仓库安装和更新 Docker。


```shell
# 安装所需的软件包
$ sudo yum install -y yum-utils

# 设置阿里云仓库
$ sudo yum-config-manager \
    --add-repo \
    http://mirrors.aliyun.com/docker-ce/linux/centos/docker-ce.repo
```


#### 3、安装


```shell
# 安装社区版，默认安装最新版
$ sudo yum install docker-ce docker-ce-cli containerd.io docker-compose-plugin

# 通过其完整的软件包名称安装特定版本，该软件包名称是软件包名称（docker-ce）加上版本字符串，如：docker-ce-18.09.1
$ sudo yum install docker-ce-<VERSION_STRING> docker-ce-cli-<VERSION_STRING> containerd.io
```


#### 4、启动


```shell
# 测试是否安装
$ sudo docker --version

# 启动docker
$ sudo systemctl start docker

# 通过运行 hello-world 镜像来验证是否正确安装了 Docker Engine-Community
$ sudo docker run hello-world # 会尝试从本地寻找镜像，寻找不到则到仓库中寻找并下载
```


#### 5、卸载


```shell
# 删除安装包
yum remove docker-ce

# 删除镜像、容器、配置文件等内容
rm -rf /var/lib/docker
```


### 四、Docker的常用命令


#### 1、帮助命令


```shell
docker version # 显示版本
docker info # 显示系统信息，包括镜像和容器的数量
docker 命令 --help # 帮助命令
```


#### 2、镜像命令


```shell
# 查看本地所有镜像
docker images
# 可选项
-a, --all # 列出所有镜像
-q, --quiet # 只显示镜像的id


# 搜索镜像（仓库中）
docker search mysql
# 可选项，通过收藏数来过滤
--filter=STARS=3000 # 搜索出stars大于3000的镜像


# 下载镜像
docker pull 镜像名[:tag]
docker pull mysql
[root@Ken-Chy129 ~]# docker pull mysql
Using default tag: latest # 没有写tag则默认为latest
latest: Pulling from library/mysql
72a69066d2fe: Pull complete  # 采用分层下载，联合文件下载系统
93619dbc5b36: Pull complete
99da31dd6142: Pull complete
626033c43d70: Pull complete
37d5d7efb64e: Pull complete
ac563158d721: Pull complete
d2ba16033dad: Pull complete
688ba7d5c01a: Pull complete
00e060b6d11d: Pull complete
1c04857f594f: Pull complete
4d7cfa90e6ea: Pull complete
e0431212d27d: Pull complete
Digest: sha256:e9027fe4d91c0153429607251656806cc784e914937271037f7738bd5b8e7709  # 签名
Status: Downloaded newer image for mysql:latest
docker.io/library/mysql:latest  # 真实地址

docker pull mysql:5.7 # 指定版本下载,需要是仓库中存在的版本
[root@Ken-Chy129 ~]# docker pull mysql:5.7
5.7: Pulling from library/mysql
72a69066d2fe: Already exists  # 此处因为在前面已经下载过了，所以无需重新下载（联合文件下载系统）
93619dbc5b36: Already exists
99da31dd6142: Already exists
626033c43d70: Already exists
37d5d7efb64e: Already exists
ac563158d721: Already exists
d2ba16033dad: Already exists
0ceb82207cd7: Pull complete
37f2405cae96: Pull complete
e2482e017e53: Pull complete
70deed891d42: Pull complete
Digest: sha256:f2ad209efe9c67104167fc609cca6973c8422939491c9345270175a300419f94
Status: Downloaded newer image for mysql:5.7
docker.io/library/mysql:5.7


# 删除镜像
docker rmi -f 容器id # 删除指定的容器
docker rmi -f 容器id 容器id 容器id # 删除多个容器
docker rmi -f $(docker images -aq) # 删除全部的容器
```


#### 3、容器命令


说明：我们有了镜像才可以创建容器


```shell
docker pull centos

# 新建容器并启动
docker run [可选参数] image
# 参数说明
--name="name" # 容器名字 tomcat01, tomcat02, 用来区分容器
--d # 后台方式运行
--it # 使用交互方式运行，进入容器查看内容
--rm # 用完即删（没有加上的话停止后还能ps -a查到）
# -i: 允许你对容器内的标准输入 (STDIN) 进行交互 -t: 在新容器内指定一个伪终端或终端。
-p # 指定容器的端口
	-p ip:主机端口:容器端口
	-p 主机端口:容器端口（常用）
	-p 容器端口
	容器端口
-P # 随机指定端口

# 启动centos容器的命令行模式（/bin/bash）
[root@Ken-Chy129 usr]# docker run -i -t centos /bin/bash
# 此处已经是进入到容器中
[root@caa59fd3ebea /]# ls
bin  dev  etc  home  lib  lib64  lost+found  media  mnt  opt  proc  root  run  sbin  srv  sys  tmp  usr  var
# 从容器中退出
[root@caa59fd3ebea /]# exit
[root@Ken-Chy129 usr]#

# 启动centos容器并使用其中的echo指令输出hello
[root@Ken-Chy129 usr]# docker run -it centos /bin/echo "hello"
hello
# 输出结束后则结束容器了


# 列出当前所有运行的容器
docker ps 命令
-a # 列出当前所有运行的容器+带出历史运行的容器
-n=? # 显示最近创建的容器
-q # 只显示容器的编号


# 退出容器
exit # 直接停止并退出容器
ctrl + p + q # 容器不停止退出


# 删除容器
docker rm 容器id # 删除指定容器，不能删除正在运行的容器，强制删除需要加-f
docker rm -f $(docker ps -aq) # 删除所有容器
docker ps -a -q|xargs docker rm # 删除所有容器


# 启动和停止容器
docker start 容器id # 启动
docker restart 容器id # 重启
docker stop 容器id # 停止
docker kill 容器id # 强制停止
```


#### 4、其他常用命令


```shell
# 后台启动容器
docker run -d 容器名

# 问题
# 启动后通过docker ps查看发现容器停止了
# 这是因为docker发现没有容器没有进程正在运行，就会自杀自动停止
# docker run -it 容器名 /bin/bash 不会被停止，因为打开了命令行交互进程


# 查看日志
docker logs -ft --tail n 容器
# --tail n表示查看最后n条日志
# -f 表示跟随日志输出，有新的日志也会不断显示（没加的话只显示到目前为止的日志后便结束）
# -t 表示显示时间戳


# 查看容器中的进程id
docker top 容器


# 查看镜像的元数据
docker inspect 容器


# 进入当前正在运行的容器
# 方式一
docker exec -it 容器 /bin/bash
# 进入容器后开启一个新的终端，可以在里面操作

# 方式二
docker attach 容器
# 进入容器正在执行的终端，不会启动新的进程


# 从容器拷贝文件到主机上（跟容器运行不运行没关系，只要容器在就行了）
docker cp 容器id:容器内路径 目的主机的路径
# 拷贝是一个手动过程，未来我们使用-v 卷的技术可以实现自动
```


#### 5、测试部署ES+kibana


```shell
# es暴露的端口很多，且十分耗内存
# es的数据一般需要放置到安全目录，挂载！

# 增加内存限制，修改配置文件 -e 环境配置修改
docker run -d --name es -p 9200:9200 -p 9300:9300 -e "discovery.type=single-node" -e ES_JAVA_OPTS="-Xms512m -Xmx1g" elasticsearch

docker stats 容器 # 查看cpu状态
```


### 五、Docker镜像讲解


#### 1、UnionFS（联合文件系统）


UnionFS（联合文件系统）是一种分层、轻量级且高性能的文件系统，它支持对文件系统的修改作为一次提交来一层层的叠加，同时可以讲不同目录挂载到同一个虚拟文件系统下。Union文件系统是Docker的基础，镜像可以通过分层来进行继承，基于基础镜像（没有父镜像）可以制作各种具体的应用镜像。


>Docker 在镜像的设计中，引入了层（layer）的概念。也就是说，用户制作镜像的每一步操作，都会生成一个层，也就是一个增量 rootfs。UnionFS最主要的功能就是将多个不同位置的目录联合挂载（union mount）到同一个目录下。比如，我现在有两个目录 A 和 B，它们分别有两个文件：
>
>```shell
>$ tree
>.
>├── A
>│  ├── a
>│  └── x
>└── B
>  ├── b
>  └── x
>```
>
>然后，我使用联合挂载的方式，将这两个目录挂载到一个公共的目录 C 上：
>
>```shell
>$ mkdir C
>$ mount -t aufs -o dirs=./A:./B none ./C
>```
>
>这时，我再查看目录 C 的内容，就能看到目录 A 和 B 下的文件被合并到了一起：
>
>```shell
>$ tree ./C
>./C
>├── a
>├── b
>└── x
>```
>
>可以看到，在这个合并后的目录 C 里，有 a、b、x 三个文件，并且 x 文件只有一份。这，就是“合并”的含义。此外，如果你在目录 C 里对 a、b、x 文件做修改，这些修改也会在对应的目录 A、B 中生效。
>
>Docker 中最常用的联合文件系统有三种：AUFS、Devicemapper 和 OverlayFS。


特性：一次同时加载多个文件系统，但从外面看起来，只能看到一个文件系统，联合加载会把各层文件系统叠加起来，这样最终的文件系统会包含所有底层的文件和目录。


>为什么 docker 等容器系统要使用类似的联合文件系统呢?
>
>我们用来启动容器的许多镜像无论 ubuntu 是 72MB 还是 nginx 133MB 的大小都非常庞大。每次我们想从这些镜像创建一个容器时，分配这么多空间将是非常昂贵的。多亏了联合文件系统，Docker 只需要在镜像之上创建一个瘦文件层，其余的可以在所有容器之间共享。这还提供了减少启动时间的额外好处，因为无需复制镜像文件和数据。
>
>联合文件系统还提供隔离功能，因为容器对共享镜像层具有只读访问权限。如果他们需要修改任何只读共享文件，他们会使用写时复制策略（稍后讨论）将内容复制到可以安全修改的可写层。


#### 2、Docker镜像加载原理


Docker 的镜像实际上由一层一层的文件系统组成，这种层级的文件系统叫 UnionFS。


boots(boot file system）主要包含 bootloader 和 Kernel, bootloader 主要是引导加载 kernel, Linux 刚启动时会加 bootfs 文件系统，在 Docker 镜像的最底层是 boots，几乎不变。这一层与我们典型的 Linux/Unix 系统是一样的，包含 bootloader 和 Kernel。当 boot 加载完成之后，整个内核就都在内存中了，此时内存的使用权已由 bootfs 转交给内核，此时系统也会卸载 bootfs


rootfs（根文件系统）是挂载在容器根目录上，用来为容器进程提供隔离后执行环境的文件系统，就是所谓的“容器镜像”。所以，一个最常见的 rootfs，或者说容器镜像，会包括如下所示的一些目录和文件，比如 /bin，/etc，/proc 等等rootfs 就是各种不同的操作系统发行版，比如 Ubuntu, Centos 等等


需要明确的是，rootfs 只是一个操作系统所包含的文件、配置和目录，并不包括操作系统内核。在 Linux 操作系统中，这两部分是分开存放的，操作系统只有在开机启动时才会加载指定版本的内核镜像。所以说，rootfs 只包括了操作系统的“躯壳”，并没有包括操作系统的“灵魂”。


>为什么虚拟机的 CentOS 镜像都是好几个G，为什么 Docker 中的只有几百M？
>
>- 因为对于不同的 Linux 发行版， boots 基本是一致的， rootfs 会有差別，因此不同的发行版可以公用 bootfs。底层直接用主机的 kernel，自己只需要提供 rootfs 就可以了
>
>- 对于个精简的 OS , rootfs 可以很小，只需要包合最基本的命令，工具和程序库就可以了
>
>同一台机器上的所有容器，都共享宿主机操作系统的内核。这就意味着，如果你的应用程序需要配置内核参数、加载额外的内核模块，以及跟内核进行直接的交互，你就需要注意了：这些操作和依赖的对象，都是宿主机操作系统的内核，它对于该机器上的所有容器来说是一个“全局变量”，牵一发而动全身。这也是容器相比于虚拟机的主要缺陷之一：毕竟后者不仅有模拟出来的硬件机器充当沙盒，而且每个沙盒里还运行着一个完整的 Guest OS 给应用随便折腾。


#### 3、镜像分层理解


![image-20221030214356597](/images/docker-containerization/4eb7fe4ca21c46e822eb8b51fe937c85.png)


可以看到下载的时候分成了六个层级，其中第一层显示已经存在。所有的 Docker 镜像都起始于一个基础镜像 rootfs，像 Ubuntu、CentOS，那么显然将这些分层显然可以实现资源共享，重复的镜像不需要再次下载（比如有多个镜像都从相同的 Base 镜像构建而来，那么宿主机只需在磁盘上保留一份 base 镜像，同时内存中也只需要加载一份 base 镜像）。当进行修改或添加新的内容时，就会在当前镜像层之上，创建新的镜像层。


> e.g.
>
> - 基于 Ubuntu Linux16.04 创建一个新的镜像，这就是新镜像的第一层
> - 如果在该镜像中添加 Python包，就会在基础镜像层之上创建第二个镜像层
> - 如果继续添加一个安全补丁，就会创健第三个镜像层
> - 该镜像当前已经包含 3 个镜像层
> - 现在可以把它再次打包成一个新的镜像（commit）提供给其他人下载，其他人 pull 下来之后就会有三层东西
>
> 
![img](/images/docker-containerization/208d795ce06e7bce3944590d939b9d82.png)


使用docker inspect redis命令即可查看镜像分层


```shell
"RootFS": {
            "Type": "layers",
            "Layers": [
                "sha256:2edcec3590a4ec7f40cf0743c15d78fb39d8326bc029073b41ef9727da6c851f",
                "sha256:9b24afeb7c2f21e50a686ead025823cd2c6e9730c013ca77ad5f115c079b57cb",
                "sha256:4b8e2801e0f956a4220c32e2c8b0a590e6f9bd2420ec65453685246b82766ea1",
                "sha256:529cdb636f61e95ab91a62a51526a84fd7314d6aab0d414040796150b4522372",
                "sha256:9975392591f2777d6bf4d9919ad1b2c9afa12f9a9b4d260f45025ec3cc9b18ed",
                "sha256:8e5669d8329116b8444b9bbb1663dda568ede12d3dbcce950199b582f6e94952"
            ]
        }
```


可以看到分成了六层，第一层可能就是操作系统镜像如centos，所以已经存在


此外需要注意的是：


**Docker 镜像都是只读的（因为是共享的），当容器启动时，一个新的可写层会加载到镜像的顶部（所有的操作都是基于这一层），这一层就是我们通常说的容器层，容器之下的都叫镜像层。** 之后当我们在这个容器上做了什么改动或者开发了什么应用进行发布，则会将我们这一层一起封装为一个新的镜像。


![img](/images/docker-containerization/165371b42dfb005813a3ad956ee97d9d.png)


#### 4、commit镜像


```shell
docker commit 容器id # 提交容器成为一个新的镜像

# 命令和git原理类似
docker commit -m="提交的描述信息" -a="作者" 容器id 目标镜像名:[tag]
```


操作步骤


1. 启动一个默认的tomcat
2. 发现这个默认的tomcat是没有webapps应用的，进不去首页(官方镜像默认webapps下面是没有文件的，在webapp.dist目录下)
3. 将webapp.dist目录下的所有文件拷贝到webapp目录下
4. 将我们操作过的容器通过commit提交为一个镜像





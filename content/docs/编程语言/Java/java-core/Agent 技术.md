---
title: "Agent 技术"
date: 2025-05-15T01:30:34+08:00
draft: false
summary: "一、简介 Java Agent 是一种 JVM 插桩机制，它允许在 主程序 main 方法执行之前 或者在 JVM 运行时 进行字节码的 修改和增强 。我们经常使用的各种线上诊断工具如 btrace 和 arthas、代码调试、热部署等功能，都是基于 Java Agent 实现的。 Java Agent 可以理解为是一"
tags: []
categories: [Tech]
source: csdn
source_id: "147966453"
---

# 一、简介


**Java Agent** 是一种 JVM 插桩机制，它允许在**主程序 main 方法执行之前**或者在 **JVM 运行时**进行字节码的**修改和增强**。我们经常使用的各种线上诊断工具如 btrace 和 arthas、代码调试、热部署等功能，都是基于 Java Agent 实现的。


Java Agent 可以理解为是一种特殊的 Java 程序，它不是一个可以单独启动的程序，必须依附在一个Java应用程序（JVM）上，与主程序运行在同一个 JVM 进程中，它提供了两种执行方式：


* 在应用运行之前，通过premain()方法来实现「在应用启动时侵入并代理应用」，这种方式是利用Instrumentation接口实现的
* 在应用运行之后，通过Attach API和agentmain()方法来实现「在应用启动后的某一个运行阶段中侵入并代理应用」，这种方式是利用Attach接口实现的


Instrumentation接口和Attach接口底层依赖JVMTI语义的Native API，相当于给用户封装了一下，降低了使用成本。


> JVM规范定义了JVMPI（Java Virtual Machine Profiler Interface）语义，JVMPI提供了一批JVM分析接口。JVMPI 可以监控就JVM发生的各种事件，比如，JVM创建、关闭、Java类被加载、创建对象或GC回收等37种事件。除了JVMPI ，JVM规范还定义了JVMDI（Java Virtual Machine Debug Interface）语义，JVMDI提供了一批JVM调试接口。JDK 1.5及之后的版本将JVMPI和JVMDI合二为一，形成了一套JVM语义——JVMTI（JVM Tool Interface），包括JVM 分析接口和JVM调试接口。
>
> JVMTI 是开发和监控工具使用的编程接口，它提供了一种方法，用于检查状态和控制在Java 虚拟机（VM）中运行的应用程序的执行，旨在为需要访问VM状态的所有工具提供VM接口，包括但不限于：评测、调试、监视、线程分析和覆盖率分析工具。


# 二、运行前代理


运行前代理需要程序通过以下命令行选项指定代理 JAR：


```
-javaagent:<jarpath>[=<options>]
```


其中 `<jarpath>` 是代理 JAR 文件的路径，`<options>` 是代理选项。


代理 JAR 文件的主清单必须包含 `Premain-Class` 属性，该属性的值是 JAR 文件中代理类的全类名。JVM 通过加载代理类并调用其 `premain` 方法来启动代理。`premain` 方法在应用程序的 `main` 方法之前被调用。`premain` 方法有两种可能的签名形式。JVM 首先尝试在代理类上调用以下方法：


```java
public static void premain(String agentArgs, Instrumentation inst)
```


如果代理类未定义此方法，则 JVM 将尝试调用：


```java
public static void premain(String agentArgs)
```


代理通过 `agentArgs` 参数接收其代理选项。代理选项作为单个字符串传递，任何额外的解析应由代理本身执行。在第一个方法中，`inst` 参数是一个 `Instrumentation` 对象，代理可以使用它来检测代码。


如果代理无法启动（例如，代理类无法加载，代理类未定义符合要求的方法，或者方法抛出未捕获的异常或错误），JVM 将在调用应用程序的 `main` 方法之前终止。


实现不需要提供从命令行界面启动代理的方式。当实现支持时，则支持上述指定的 `-javaagent` 选项。该选项可以在同一命令行中多次使用，从而启动多个代理。`premain` 方法将按照代理在命令行中指定的顺序调用。多个代理可以使用相同的 `<jarpath>`。


代理类还可以定义 `agentmain` 方法，用于在运行中的 JVM 中启动代理（见下文）。当代理通过命令行选项启动时，`agentmain` 方法不会被调用。


# 三、运行时代理

同理使用运行时代理时，代理 JAR 文件的主清单必须包含 `Agent-Class` 属性。`agentmain` 方法有两种可能的签名形式。JVM 首先尝试在代理类上调用以下方法：


```java
public static void agentmain(String agentArgs, Instrumentation inst)
```


如果代理类未定义此方法，则 JVM 将尝试调用：


```java
public static void agentmain(String agentArgs)
```


`agentArgs` 参数的值始终为空字符串。在第一个方法中，`inst` 参数是一个 `Instrumentation` 对象，代理可以使用它来检测代码。


`agentmain` 方法应完成启动代理所需的任何初始化并返回。如果代理无法启动（例如，代理类无法加载，代理类未定义符合要求的方法，或者方法抛出未捕获的异常或错误），JVM 将在调用应用程序的 `main` 方法之前终止。


实现可以提供在运行中的 JVM 中启动代理的机制（即在 JVM 启动后）。如何启动的具体细节因实现而异，但通常应用程序已经启动，并且其 `main` 方法已经被调用。如果实现支持在运行中的 JVM 中启动代理，则适用以下规则：


1. 代理类必须打包到代理 JAR 文件中。
2. 代理 JAR 文件的主清单必须包含 `Agent-Class` 属性。该属性的值是 JAR 文件中代理类的二进制名称。
3. 代理类必须定义一个公共静态方法 `agentmain`。


# 四、文件清单属性


我们需要在 jar 包的 `MANIFEST.MF` 文件中指定 agent 的入口类是什么，以及 agent 会有哪些能力。相关属性如下：


- **Launcher-Agent-Class**
  - 如果实现支持在可执行 JAR 文件中启动应用程序的机制，则此属性（如果存在）指定与应用程序一起打包的代理类的二进制名称。代理通过调用代理类的 `agentmain` 方法启动。它在应用程序的 `main` 方法之前被调用。

- **Premain-Class**
  - 如果在 JVM 启动时指定了代理 JAR，则此属性指定 JAR 文件中代理类的二进制名称。代理通过调用代理类的 `premain` 方法启动。它在应用程序的 `main` 方法之前被调用。如果该属性不存在，JVM 将终止。

- **Agent-Class**
  - 如果实现支持在 JVM 启动后的某个时间启动代理，则此属性指定代理 JAR 文件中 Java 代理类的二进制名称。代理通过调用代理类的 `agentmain` 方法启动。此属性是必需的；如果不存在，代理将不会启动。

- **Boot-Class-Path**
  - 由引导类加载器搜索的路径列表。路径表示目录或库（在许多平台上通常称为 JAR 或 zip 库）。这些路径在平台特定的类定位机制失败后由引导类加载器搜索。路径按列出的顺序搜索。列表中的路径由一个或多个空格分隔。路径采用分层 URI 的路径组件语法。如果路径以斜杠字符（`/`）开头，则它是绝对路径，否则是相对路径。相对路径相对于代理 JAR 文件的绝对路径解析。格式错误和不存在的路径将被忽略。当代理在 JVM 启动后的某个时间启动时，不代表 JAR 文件的路径将被忽略。此属性是可选的。

- **Can-Redefine-Classes**
  - 布尔值（`true` 或 `false`，不区分大小写）。此代理是否需要重新定义类的能力。除 `true` 以外的值均被视为 `false`。此属性是可选的，默认值为 `false`。

- **Can-Retransform-Classes**
  - 布尔值（`true` 或 `false`，不区分大小写）。此代理是否需要重新转换类的能力。除 `true` 以外的值均被视为 `false`。此属性是可选的，默认值为 `false`。

- **Can-Set-Native-Method-Prefix**
  - 布尔值（`true` 或 `false`，不区分大小写）。此代理是否需要设置本地方法前缀的能力。除 `true` 以外的值均被视为 `false`。此属性是可选的，默认值为 `false`。
  - Native 方法不是字节码实现的，Agent 修改不了它的逻辑。通常修改 Native 是Proxy 的做法，把原有的 Native 方法重命名，新建同名的 Java 方法来调用老方法。此时需要修改 Native 方法前缀的能力。


代理 JAR 文件的清单中可以同时存在 `Premain-Class` 和 `Agent-Class` 属性。当通过 `-javaagent` 选项在命令行启动代理时，`Premain-Class` 属性指定代理类的二进制名称，`Agent-Class` 属性被忽略。类似地，如果代理在 JVM 启动后的某个时间启动，则 `Agent-Class` 属性指定代理类的二进制名称（`Premain-Class` 属性的值被忽略）


# 五、Instrumentation


`Java agent`与`Instrumentation`密不可分，二者也需要在一起使用。因为`Instrumentation`的实例会作为参数注入到`Java agent`的启动方法中。


`Instrumentation`是Java提供的JVM接口，该接口提供了一系列查看和操作Java类定义的方法，例如修改类的字节码、向 classLoader 的 classpath 下加入jar文件等。使得开发者可以通过Java语言来操作和监控JVM内部的一些状态，进而实现Java程序的监控分析，甚至实现一些特殊功能（如AOP、热部署）。


`Instrumentation` 的一些常用接口定义如下：


- `getAllLoadedClasses()` 获取所有加载的类，得到数组后我们可以自己筛选出关心的类
- `redefineClasses(ClassDefinition... definitions)` 使用参数中的类定义重新定义类。
- `retransformClasses(Class<?>... classes)` 对JVM已经加载的类重新触发类加载。使用的就是上面注册的Transformer。retransformClasses可以修改方法体，但是不能变更方法签名、增加和删除方法/类的成员属性
- `addTransformer(ClassFileTransformer transformer)` 注册 `transformer`
- `removeTransformer(ClassFileTransformer transformer)` 注销 `transformer`


类加载的字节码被修改后，除非再次被`retransform`，否则不会恢复。


# 六、Demo


```java
public class TestAgent {

    public static void agentmain(String args, Instrumentation inst) {
        // 指定我们自己定义的Transformer，在其中利用Javassist做字节码替换
        // 第二个参数需要指定为true，否则该转换器无法进行retransformClasses（即无法修改已加载的类）
        inst.addTransformer(new TestTransformer(), true);
        try {
            // inst.addTransformer添加的转换器指挥对未来加载的类才生效
            // 而agentmain实在应用程序启动后才加载，因此会出现transformer不生效的情况
            // 需要再调用retransformClasses方法才能对已加载的类进行转换
            inst.retransformClasses();
            System.out.println("Agent Load Done.");
        } catch (Exception e) {
            System.out.println("agent load failed!");
        }
    }

    public static void premain(String args, Instrumentation inst) {
        TestTransformer testTransformer = new TestTransformer();
        inst.addTransformer(testTransformer);
    }
}
```


仅当Can-Transform-Classes清单属性的值为true时类才支持重转换，且addTransformer时入参的canRetransform需要设置为true，否则调用retransformClasses时不会使用该transformer


```java
public class TestTransformer implements ClassFileTransformer {

    @Override
    public byte[] transform(ClassLoader loader, String className, Class<?> classBeingRedefined, ProtectionDomain protectionDomain, byte[] classfileBuffer) {
        if (!"cn/ken/agent/Base".equals(className)) {
            return classfileBuffer;
        }
        System.out.println("Transforming " + className);
        
        // 通过ASM增强字节码
        ClassNode cn = new ClassNode(Opcodes.ASM4);
        ClassReader cr = new ClassReader(classfileBuffer);
        cr.accept(cn, 0);
        for (var method : cn.methods) {
          System.out.println("patching Method: " + method.name);
          var list = new InsnList();
          list.add(new FieldInsnNode(Opcodes.GETSTATIC, "java/lang/System", "out",
              "Ljava/io/PrintStream;"));
          list.add(new LdcInsnNode(">> calling Method: " + method.name));
          list.add(new MethodInsnNode(Opcodes.INVOKEVIRTUAL, "java/io/PrintStream", "println",
              "(Ljava/lang/String;)V", false));
          method.instructions.insert(list);
        }

        ClassWriter cw = new ClassWriter(ClassWriter.COMPUTE_MAXS);
        cn.accept(cw);
        return cw.toByteArray();
    }
}
```


- 通常我们会使用各种信息来过滤掉不感兴趣的类（不想修改就直接直接返回原字节码）。
- 核心输入输出是 `class` 二进制流(`byte[]`)，即 transformer 假定字节码的修改是在二进制层面进行的。常用一些库如asm、javaassist、bytebuddy进行修改


```java
package cn.ken.agent;

import com.sun.tools.attach.AgentInitializationException;
import com.sun.tools.attach.AgentLoadException;
import com.sun.tools.attach.AttachNotSupportedException;
import com.sun.tools.attach.VirtualMachine;

import java.io.IOException;

public class AttachMain {

    public static void main(String[] args) throws AttachNotSupportedException, IOException, AgentLoadException, AgentInitializationException {
        // 传入目标 JVM pid
        String pid = args[0];
        // 低版本需要单独引入tools.jar包，高版本不需要
        VirtualMachine vm = VirtualMachine.attach(pid);
        vm.loadAgent("D:\\Java\\java-agent\\target\\agent-1.0-SNAPSHOT.jar");
    }
}
```


用于将agent attach到目标JVM进程。


```xml
<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
         xsi:schemaLocation="http://maven.apache.org/POM/4.0.0 http://maven.apache.org/xsd/maven-4.0.0.xsd">
    <modelVersion>4.0.0</modelVersion>

    <groupId>cn.ken</groupId>
    <artifactId>agent</artifactId>
    <version>1.0-SNAPSHOT</version>

    <properties>
        <maven.compiler.source>21</maven.compiler.source>
        <maven.compiler.target>21</maven.compiler.target>
        <project.build.sourceEncoding>UTF-8</project.build.sourceEncoding>
    </properties>

    <dependencies>
        <dependency>
            <groupId>org.ow2.asm</groupId>
            <artifactId>asm</artifactId>
            <version>9.8</version>
        </dependency>
        <dependency>
            <groupId>org.javassist</groupId>
            <artifactId>javassist</artifactId>
            <version>3.30.2-GA</version>
        </dependency>
    </dependencies>

    <build>
        <plugins>
            <plugin>
                <groupId>org.apache.maven.plugins</groupId>
                <artifactId>maven-jar-plugin</artifactId>
                <version>3.1.0</version>
                <configuration>
                    <archive>
                        <manifest>
                            <addClasspath>true</addClasspath>
                        </manifest>
                        <manifestEntries>
                            <Main-Class>cn.ken.agent.AttachMain</Main-Class>
                            <Premain-Class>cn.ken.agent.TestAgent</Premain-Class>
                            <Agent-Class>cn.ken.agent.TestAgent</Agent-Class>
                            <Can-Redefine-Classes>true</Can-Redefine-Classes>
                            <Can-Transform-Classes>true</Can-Transform-Classes>
                        </manifestEntries>
                    </archive>
                </configuration>
            </plugin>
        </plugins>
    </build>

</project>
```






参阅：


* [【JVM】Java agent超详细知识梳理一、开篇 在梳理SkyWalking agent、arthas、elasti - 掘金](https://juejin.cn/post/7157684112122183693#heading-10)
* [一文讲透Java Agent是什么玩意？能干啥？怎么用？ - 知乎](https://zhuanlan.zhihu.com/p/636603910)
* [Java Agent 入门教程 | 三点水](https://lotabout.me/2024/Java-Agent-101/)

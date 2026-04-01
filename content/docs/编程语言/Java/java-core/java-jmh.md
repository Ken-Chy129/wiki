---
title: "微基准测试 JMH"
date: 2025-05-18T18:24:40+08:00
draft: false
summary: "Java微基准测试工具JMH（Java MicroBenchmark Harness）负责JVM预热和代码优化路径等工作，使基准测试尽可能简单。 JVM的即时编译器会对代码进行优化，这可能会影响性能测试的结果。JMH通过控制测试环境（预热、多轮迭代、多进程测试等机制），确保测试结果的准确性。 一、快速开始 Maven依"
tags: []
categories: [Tech]
source: csdn
source_id: "148047712"
---


Java微基准测试工具JMH（Java MicroBenchmark Harness）负责JVM预热和代码优化路径等工作，使基准测试尽可能简单。


JVM的即时编译器会对代码进行优化，这可能会影响性能测试的结果。JMH通过控制测试环境（预热、多轮迭代、多进程测试等机制），确保测试结果的准确性。


## 一、快速开始


Maven依赖：


```xml
<dependency>
    <groupId>org.openjdk.jmh</groupId>
    <artifactId>jmh-core</artifactId>
    <version>1.37</version>
</dependency>
<dependency>
    <groupId>org.openjdk.jmh</groupId>
    <artifactId>jmh-generator-annprocess</artifactId>
    <version>1.37</version>
</dependency>
```


启动类：


```java
package cn.ken;

import org.openjdk.jmh.profile.GCProfiler;
import org.openjdk.jmh.results.format.ResultFormatType;
import org.openjdk.jmh.runner.Runner;
import org.openjdk.jmh.runner.RunnerException;
import org.openjdk.jmh.runner.options.Options;
import org.openjdk.jmh.runner.options.OptionsBuilder;

/**
 * @author Ken-Chy129
 * @date 2025/5/18
 */
public class BenchmarkRunner {

    public static void main(String[] args) throws RunnerException {
        Options opt = new OptionsBuilder()
                .include(StringConcatBenchmark.class.getSimpleName()) // 指定基准测试类
                .addProfiler(GCProfiler.class) // 添加性能剖析工具
                .result("result.json") // 输出结果
                .resultFormat(ResultFormatType.JSON) // 结果类型
                .build();
        new Runner(opt).run(); // 运行基准测试
    }

}
```


基准测试类：


```java
package cn.ken;

import org.openjdk.jmh.annotations.*;
import org.openjdk.jmh.infra.Blackhole;

import java.util.concurrent.TimeUnit;

/**
 * @author Ken-Chy129
 * @date 2025/5/18
 */
@BenchmarkMode(Mode.AverageTime)
@Warmup(iterations = 3, time = 1)
@Measurement(iterations = 5, time = 5)
@Threads(4)
@Fork(1)
@State(value = Scope.Benchmark)
@OutputTimeUnit(TimeUnit.NANOSECONDS)
public class MyBenchmark {

    @Benchmark
    public void measureSimpleMath(Blackhole blackhole) {
        // 基准代码
        blackhole.consume(add(1, 2));
    }

    private int add(int a, int b) {
        return a + b;
    }
}
```


## 二、核心概念和注解


### 常用注解


* @Benchmark：用于标记需要跑基准测试的方法
* @BenchmarkMode：测试模式
  * Throughout：吞吐量，单位时间内可以完成的操作数
  * AverageTime：平均时间，完成一次操作所需的平均时间
  * SampleTime：基于采样的事件，提供统计分布数据
  * SingleShotTime：单次执行事件，用于测试冷启动性能
  * ALL：运行所有模式
* @State：状态对象自然地封装了基准正在处理的状态，通常作为参数注入到Benchmark方法中，JMH负责对其进行实例化和共享。状态对象的范围定义了它在工作线程之间共享的程度。该注解用于标识测试状态的生命周期和作用域
  * Scope.Thread：每个线程一个实例
  * Scope.Benchmark：所有线程共享一个实例
  * Scope.Group：每个线程组共享一个实例
* @Warmup：预热，可以指定预热迭代次数和每次迭代的运行时间
* @Measurement：指定正式测试的迭代次数和每次迭代的运行时间
* @OutputTimeUnit：指定测试结果的时间单位
* @Threads：指定测试方法运行的线程数
* @Params：为基准测试方法提供参数，允许在单个测试中运行多个参数集
* @Fork：指定测试运行在不同的JVM进程中，以避免测试间的相互影响。通常设置为1
* @AuxCounters：提供额外的性能计数器
* @CompilerControl：控制JVM的编译优化行为
* Blackhole：JMH提供的一个机制，用于“吞噬”测试方法的输出，防止JVM的死代码消除优化


### 性能剖析工具


JMH内置了多个性能剖析工具，可以查看基准测试的消耗在什么地方。常见如下：


* ClassloaderProfiler：类加载剖析 
* CompilerProfiler：JIT 编译剖析 
* GCProfiler：GC 剖析 
* StackProfiler：栈剖析 
* PausesProfiler：停顿剖析


## 三、JMH 陷阱


在使用JMH的过程中，需要避免一些会影响测试结果的陷阱。


### 循环优化


我们可能会通过循环来实现多次调用基准方法，然而JVM非常擅长优化循环，这可能回到则最终的测试结果并不准确。
如果我们希望运行多次基准方法，不应该在方法内使用循环，而是可以通过@OperationsPerInvocation注解来告诉JMH每次迭代应该执行多少次操作。


```java
@Benchmark
@OperationsPerInvocation(1000)
public void measureLoop() {
    // ...
}
```


### 消除死代码


对于某些计算的结果如果没有使用，JVM可能会认为该计算是死代码并将其消除，从而导致基准测试在JVM优化后没有留下任何代码，导致结果有很大的偏差。
因此对于该类测试，我们可以通过如下两种方法避免被识别为死代码：


1. 从基准测试方法返回代码的结果
2. 将计算出的值传递到JMH提供的Blackhole中


```java
import org.openjdk.jmh.annotations.Benchmark;

public class MyBenchmark {

    @Benchmark
    public int testMethod1() {
        int a = 1;
        int b = 2;
        int sum = a + b;
        return sum;
    }
    
    @Benchmark
    public void testMethod2(Blackhole blackhole) {
        int a = 1;
        int b = 2;
        int sum = a + b;
        blackhole.consume(sum);
    }
}
```


### 常量折叠


常量折叠是一中常见的JVM优化。基于常量的计算无论执行多少次通常都会得到导致完全相同的结果，因此JVM可能会检测到之后直接使用计算结果替换该计算。
如下示例：


```java
import org.openjdk.jmh.annotations.Benchmark;

public class MyBenchmark {

    @Benchmark
    public int testMethod1() {
        int a = 1;
        int b = 2;
        int sum = a + b;
        return sum;
    }

    @Benchmark
    public int testMethod2() {
        int sum = 3;
        return sum;
    }

}
```


JVM会检测到sum的值是两个常量的值，从而直接将方法1优化成方法2。为了避免常量折叠，我们输入的值不应该是个硬编码的常量，而应该来自一个状态对象。如下：


```java
import org.openjdk.jmh.annotations.*;

public class MyBenchmark {

    @State(Scope.Thread)
    public static class MyState {
        public int a = 1;
        public int b = 2;
    }


    @Benchmark 
    public int testMethod(MyState state) {
        int sum = state.a + state.b;
        return sum;
    }
}
```


其实 JVM 做的优化操作远不止上面这些，还有比如常量传播（Constant Propagation）、循环展开（Loop Unwinding）、循环表达式外提（Loop Expression Hoisting）、消除公共子表达式（Common Subexpression Elimination）、本块重排序（Basic Block Reordering）、范围检查消除（Range Check Elimination）等。


## 四、结果分析


运行之后可以得到如下的测试结果：


```text
# JMH version: 1.37
# VM version: JDK 1.8.0_412, OpenJDK 64-Bit Server VM, 25.412-b08
# VM invoker: D:\Java\Jdk\corretto-1.8.0_412\jre\bin\java.exe
# VM options: -javaagent:D:\JetBrains\IntelliJ IDEA 2024.2.0.2\lib\idea_rt.jar=56800:D:\JetBrains\IntelliJ IDEA 2024.2.0.2\bin -Dfile.encoding=UTF-8
# Blackhole mode: full + dont-inline hint (auto-detected, use -Djmh.blackhole.autoDetect=false to disable)
# Warmup: 3 iterations, 1 s each
# Measurement: 5 iterations, 5 s each
# Timeout: 10 min per iteration
# Threads: 4 threads, will synchronize iterations
# Benchmark mode: Average time, time/op
# Benchmark: cn.ken.MyBenchmark.measureSimpleMath

# Run progress: 0.00% complete, ETA 00:00:28
# Fork: 1 of 1
# Warmup Iteration   1: 1.339 ±(99.9%) 0.250 ns/op
# Warmup Iteration   2: 1.443 ±(99.9%) 0.218 ns/op
# Warmup Iteration   3: 1.062 ±(99.9%) 0.259 ns/op
Iteration   1: 1.035 ±(99.9%) 0.073 ns/op
Iteration   2: 1.036 ±(99.9%) 0.080 ns/op
Iteration   3: 1.043 ±(99.9%) 0.084 ns/op
Iteration   4: 1.098 ±(99.9%) 0.039 ns/op
Iteration   5: 1.813 ±(99.9%) 0.010 ns/op


Result "cn.ken.MyBenchmark.measureSimpleMath":
  1.205 ±(99.9%) 1.311 ns/op [Average]
  (min, avg, max) = (1.035, 1.205, 1.813), stdev = 0.341
  CI (99.9%): [≈ 0, 2.517] (assumes normal distribution)


# Run complete. Total time: 00:00:28

REMEMBER: The numbers below are just data. To gain reusable insights, you need to follow up on
why the numbers are the way they are. Use profilers (see -prof, -lprof), design factorial
experiments, perform baseline and negative tests that provide experimental control, make sure
the benchmarking environment is safe on JVM/OS/HW level, ask for reviews from the domain experts.
Do not assume the numbers tell you what you want them to tell.

Benchmark                      Mode  Cnt  Score   Error  Units
MyBenchmark.measureSimpleMath  avgt    5  1.205 ± 1.311  ns/op

Benchmark result is saved to result.json
```


可以得到如下信息：


* Score（分数）：1.205表示每次操作的平均时间是1.205纳秒
* Error（误差）：+-1.311表示测试结果的误差率为1.311%，误差越小，测试结果越可靠


如果测试结果的误差很小（例如+-0.01%），那么测试结果比较稳定和可靠。如果误差较高，可能需要增加迭代次数或者预热次数来降低误差。


除此以外，如果想将测试结果以图表的形式可视化，也可以通过一些工具实现：[JMH Visualizer](https://jmh.morethan.io/)
只需要将上述测试生成的json结果文件导入，就可以实现可视化。


## 五、测试Demo


官方提供了许多测试样例：[code-tools/jmh: 2be2df7dbaf8 /jmh-samples/src/main/java/org/openjdk/jmh/samples/](https://hg.openjdk.org/code-tools/jmh/file/tip/jmh-samples/src/main/java/org/openjdk/jmh/samples/)


此处提供一个字符串拼接测试，代码如下：


```java
package cn.ken;

import org.openjdk.jmh.annotations.*;

import java.util.concurrent.TimeUnit;

/**
 * @author Ken-Chy129
 * @date 2025/5/18
 */
@BenchmarkMode(Mode.Throughput)
@OutputTimeUnit(TimeUnit.MILLISECONDS)
@Warmup(iterations = 5, time = 1)
@Measurement(iterations = 5, time = 1)
@Fork(1)
public class StringConcatBenchmark {

    @Benchmark
    public String concatByPlus() {
        String str = "";
        for (int i = 0; i < 100; i++) {
            str += i;
        }
        return str;
    }

    @Benchmark
    public String concatByStringBuilder() {
        StringBuilder str = new StringBuilder();
        for (int i = 0; i < 100; i++) {
            str.append(i);
        }
        return str.toString();
    }

    @Benchmark
    public String concatByStringBuffer() {
        StringBuffer str = new StringBuffer();
        for (int i = 0; i < 100; i++) {
            str.append(i);
        }
        return str.toString();
    }
}
```


测试结果：


```text
Benchmark                                                        Mode  Cnt      Score     Error   Units
StringConcatBenchmark.concatByPlus                              thrpt    5    756.185 ±  29.003  ops/ms
StringConcatBenchmark.concatByPlus:gc.alloc.rate                thrpt    5  16225.526 ± 619.623  MB/sec
StringConcatBenchmark.concatByPlus:gc.alloc.rate.norm           thrpt    5  22504.001 ±   0.001    B/op
StringConcatBenchmark.concatByPlus:gc.count                     thrpt    5     55.000            counts
StringConcatBenchmark.concatByPlus:gc.time                      thrpt    5     45.000                ms
StringConcatBenchmark.concatByStringBuffer                      thrpt    5   1999.696 ±  93.388  ops/ms
StringConcatBenchmark.concatByStringBuffer:gc.alloc.rate        thrpt    5   3127.063 ± 146.275  MB/sec
StringConcatBenchmark.concatByStringBuffer:gc.alloc.rate.norm   thrpt    5   1640.000 ±   0.001    B/op
StringConcatBenchmark.concatByStringBuffer:gc.count             thrpt    5     75.000            counts
StringConcatBenchmark.concatByStringBuffer:gc.time              thrpt    5     44.000                ms
StringConcatBenchmark.concatByStringBuilder                     thrpt    5   2888.358 ± 268.186  ops/ms
StringConcatBenchmark.concatByStringBuilder:gc.alloc.rate       thrpt    5   4450.531 ± 413.167  MB/sec
StringConcatBenchmark.concatByStringBuilder:gc.alloc.rate.norm  thrpt    5   1616.000 ±   0.001    B/op
StringConcatBenchmark.concatByStringBuilder:gc.count            thrpt    5     54.000            counts
StringConcatBenchmark.concatByStringBuilder:gc.time             thrpt    5     36.000                ms
```


可以看到通过加号进行字符串拼接的吞吐量最低，通过StringBuilder进行字符串拼接的吞吐量最高。


对比编译生成的字节码文件：


```text
// access flags 0x1
  public concatByPlus()Ljava/lang/String;
  @Lorg/openjdk/jmh/annotations/Benchmark;()
   L0
    LINENUMBER 20 L0
    LDC ""
    ASTORE 1
   L1
    LINENUMBER 21 L1
    ICONST_0
    ISTORE 2
   L2
   FRAME APPEND [java/lang/String I]
    ILOAD 2
    BIPUSH 100
    IF_ICMPGE L3
   L4
    LINENUMBER 22 L4
    NEW java/lang/StringBuilder
    DUP
    INVOKESPECIAL java/lang/StringBuilder.<init> ()V
    ALOAD 1
    INVOKEVIRTUAL java/lang/StringBuilder.append (Ljava/lang/String;)Ljava/lang/StringBuilder;
    ILOAD 2
    INVOKEVIRTUAL java/lang/StringBuilder.append (I)Ljava/lang/StringBuilder;
    INVOKEVIRTUAL java/lang/StringBuilder.toString ()Ljava/lang/String;
    ASTORE 1
   L5
    LINENUMBER 21 L5
    IINC 2 1
    GOTO L2
   L3
    LINENUMBER 24 L3
   FRAME CHOP 1
    ALOAD 1
    ARETURN
   L6
    LOCALVARIABLE i I L2 L3 2
    LOCALVARIABLE this Lcn/ken/StringConcatBenchmark; L0 L6 0
    LOCALVARIABLE str Ljava/lang/String; L1 L6 1
    MAXSTACK = 2
    MAXLOCALS = 3
```


可以看到通过加号拼接的字符串，编译之后会在循环内重复创建StringBuilder对象，因此会带来很大的性能损耗，故吞吐量远少于其他两种方式，产生的需要回收的对象也远超其他两种方式。

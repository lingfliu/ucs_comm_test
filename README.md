# UCS 通信测试 （基于TCP / QUIC）

## 运行程序：
1. ./mock/tcp_cli/tcp_cli.go tcp客户端
2. ./mock/tcp_srv/tcp_srv.go tcp服务端
1. ./mock/quic_cli/quic_cli.go quic客户端
2. ./mock/quic_cli/quic_srv.go quic服务端

编译后分为为 ```tcp_cli.exe，tcp_srv.exe, quic_cli.exe, quic_srv.exe```

## 配置参数

1. 客户端

``` bash
tcp_cli --host_addr 127.0.0.1 --host_port 17001 --fps 10 --log_file 202506t15_230000_tcp.log
quic_cli --host_addr 127.0.0.1 --host_port 17004 --fps 10 --log_file 202506t15_230000_quic.log
```
各个参数如下：
- --host_addr 是服务端地址，默认为127.0.0.1
- host_port 是端口，需要与服务端一致, 上述示例为默认端口
- fps 是发射间隔，10 = 100 ms间隔，默认为10
- log_file 是日志记录文件名，默认存储在当前文件夹，命名方式为yyyymmdd_hhMMss_tcp.log

2. 服务端

``` bash
tcp_srv --host_port 17001 
quic_srv --host_port 17004
```
各个参数如下：
- host_port 是设定端口

程序运行时会输出上述参数

本测试样例中，客户端定时发送一个数据包（按0.1秒一次, 或根据fps进行调整）。 每个数据包前8个字节是一个纳秒级的时间戳，后8个字节是一个计数器。服务端对接收的数据直接传回客户端。客户端接收回传的数据，解析里面的时间戳和计数器，与当前客户端的时间戳进行比较，记录环路延迟, 并且在一个100的窗口内计算平均环路延迟。

## 测试情况

未考虑网络部分，tcp环路延迟为0.1ms有可靠性，quic环路延迟为0.5ms有可靠性，udp环路延迟为0.7ms无可靠性，这与网络协议设计有出入。

### 特别注意
1. go是编译语言，需要将以上两个程序单独编译，编译后的性能指标与未编译的会有显著差异。
2. quic与udp还未处理好关闭套接字管理，单不影响客户端回环测试，如果服务端日志太多，可以先停掉（ctrl+c），再重新启动一次。客户端需要重新连接。

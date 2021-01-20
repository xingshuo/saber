基于golang实现的伪并发框架
====
    一个试验品
    
架构图
----
![flowchart](https://github.com/xingshuo/saber/blob/master/saber.png)

当前支持功能
----
    基于静态配置实现
     1. 节点内服务注册, 收发包编解码器重定向, 基于method响应函数注册
     2. 单节点内服务间notify,rpc
     3. 不同节点服务间notify,rpc
测试用例
----
    节点内服务通信
      cd examples/helloworld/single/
      go build
      ./single
    
    跨节点服务通信
      cd examples/helloworld/cluster/server
      go build
      ./server
      
      cd examples/helloworld/cluster/client
      go build
      ./client
    
    压测
      cd test/stress/helloworld/server
      go build
      ./server -svc 4
      
      cd test/stress/helloworld/client
      go build
      ./client -c 200 -n 8000 -svc 4

性能测试报告
----
    测试环境:
      Intel(R) Xeon(R) Platinum 8255C CPU @ 2.50GHz
      cpu逻辑核数: 8核 
      内存: 15G
      网络: 本机网络通信
    
    测试参数:
      序列化方式: Json
      Server端单个服务并发模式: 伪并发
      Server并行服务数: 4
      Client发包负载策略: 取模hash
    
    测试结论:
      [Finally]=======>消息类型|命令字 : SABER | ReqLogin
      ─────┬───────┬───────┬───────┬────────┬────────┬────────┬────────┬────────┬────────┬────────
       耗时│ 并发数│ 成功数│ 失败数│   qps  │最长耗时│最短耗时│平均耗时│下载字节│字节每秒│ 错误码
      ─────┼───────┼───────┼───────┼────────┼────────┼────────┼────────┼────────┼────────┼────────
        16s│    200│1600000│      0│103028.23│ 31.47ms│  0.05ms│  1.93ms│22400000B|1442395B/s│0:1600000
      Latency histogram:
          0.05ms|      1|    0.00%
          3.19ms|1494657|   93.42%
          6.34ms|  88590|    5.54%
          9.48ms|   6171|    0.39%
         12.62ms|   2729|    0.17%
         15.76ms|   1741|    0.11%
         18.90ms|   1493|    0.09%
         22.04ms|   1926|    0.12%
         25.19ms|   1979|    0.12%
         28.33ms|    656|    0.04%
         31.47ms|     57|    0.00%
      Latency distribution:
           10%     in     0.88ms
           25%     in     1.25ms
           50%     in     1.71ms
           75%     in     2.25ms
           90%     in     2.88ms
           95%     in     3.42ms
           99%     in     6.60ms

待实现
----
    1. 补充整体协议格式说明
    2. 补充完整架构图
    3. 伪并发模式压测数据统计
    4. 同时支持伪并发和真并发,优化代码以及数据结构
    5. log重构, metric, trace接入
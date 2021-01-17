基于golang实现的伪并发框架
====
    一个试验品

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
待实现
----
    1. 补充整体协议格式说明
    2. ClusterReqHead与ClusterRspHead编解码buffer长度精确判定
    3. 伪并发模式压测数据统计
    4. 同时支持伪并发和真并发
    5. log重构, metric, trace接入
# Pingmesh: A Large-Scale System for Data Center Network Latency Measurement and Analysis

# 一个用于数据中心网络延迟测量和分析的大规模系统

基于：[https://github.com/aprilmadaha/pingmesh](https://github.com/aprilmadaha/pingmesh%E3%80%82%E8%BF%9B%E8%A1%8C%E4%BA%86%E4%BF%AE%E6%94%B9%E3%80%82)， 进行了修改



主要修改的内容有：

1. 原来的代码是在客户端执行fping，这个版本是基于tcping

2. 因为fping可以支持多个destination ip，如：
   
   1. fping [10.100.0.5 10.100.0.6 -q -p 12000 -c 5]
   
   2. 返回： 10.100.0.5 : xmt/rcv/%loss = 5/5/0%,min/avg/max = 0.79/0.86/0.94
      
      10.100.0.6 :xmt/rcv/%loss = 5/5/0%, min/avg/max = 0.01/0.01/0.02
   
   3. 在这个版本中，通过编写shell脚本，在multi_tcping.sh里执行多任务，对多个目标ip执行tcping测试

3. 原来的代码会通过fping自己测试自己，即client ip是10.100.0.5, destination ip也是10.100.0.5。在这个版本里会进行判断，如果client ip = destination ip，则跳过



pingmesh架构：

![](D:\Work\GitHub\pingmesh\pingmesh-image\pingmesh-architecture.png)

主要分为三个角色：

服务器端：

1. 部署MariaDB，创建database ，名字叫ping。在这个数据库里，有三张Table：
   
   1. 表Host，用来记录客户端的IP
   
   2. 表Valu，用来记录客户端之间tcping的测试结果

2. Server端起了2个服务：
   
   1. pingmesh-s-v1.1-GetHostIp.go，用来从MariaDB数据库里读取host记录，并下发给客户端
   
   2. pingmesh-s-v1.1-GetResult.go，允许客户端把测试结果进行上报，并保存到MariaDB数据表里



客户端：

1. pingmesh-c-v1.1.go，从服务器端获得客户端IP列表，通过执行multi_tcping.sh，来进行tcping测试，并把测试结果上报给服务器端

2. multi_tcping.sh，执行60次tcping，并把结果进行汇总和统计(平均值，最大值，最小值)



展示层：

1. 对tcping的延迟结果进行展示





具体的配置步骤：

1. 请先创建1个Azure Virtual Network，并创建1个subnet。步骤略

2. 创建4台虚拟机，操作系统为Rocky 9.4。步骤略：
   
   

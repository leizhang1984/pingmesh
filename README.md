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

![](https://github.com/leizhang1984/pingmesh/blob/main/pingmesh-image/pingmesh-architecture.png)

主要分为三个角色：

**服务器端：**

1. 部署MariaDB，创建database ，名字叫ping。在这个数据库里，有三张Table：
   
   1. 表Host，用来记录客户端的IP
   
   2. 表Valu，用来记录客户端之间tcping的测试结果

2. Server端起了2个服务：
   
   1. pingmesh-s-v1.1-GetHostIp.go，用来从MariaDB数据库里读取host记录，并下发给客户端
   
   2. pingmesh-s-v1.1-GetResult.go，允许客户端把测试结果进行上报，并保存到MariaDB数据表里

**客户端：**

1. pingmesh-c-v1.1.go，从服务器端获得客户端IP列表，通过执行multi_tcping.sh，来进行tcping测试，并把测试结果上报给服务器端

2. multi_tcping.sh，执行60次tcping，并把结果进行汇总和统计(平均值，最大值，最小值)

**展示层：**

1. 对tcping的延迟结果进行展示

具体的配置步骤：

第1部分：环境准备    

1. 请先创建1个Azure Virtual Network，并创建1个subnet。步骤略

2. 创建4台虚拟机，操作系统为Rocky 9.4。步骤略：
   
   | 虚拟机名称             | 角色说明  | 内网IP地址       |
   | ----------------- | ----- | ------------ |
   | pingmesh-server   | 服务器   | 10.240.0.100 |
   | pingmesh-client01 | 客户端01 | 10.240.0.101 |
   | pingmesh-client02 | 客户端02 | 10.240.0.102 |

3. 等待虚拟机创建完毕后，查看rocky 版本：
   
   [root@pingmesh-server ~]# cat /etc/os-release
   NAME="Rocky Linux"
   VERSION="9.4 (Blue Onyx)"
   ID="rocky"
   ID_LIKE="rhel centos fedora"
   VERSION_ID="9.4"
   PLATFORM_ID="platform:el9"
   PRETTY_NAME="Rocky Linux 9.4 (Blue Onyx)"
   ANSI_COLOR="0;32"
   LOGO="fedora-logo-icon"
   CPE_NAME="cpe:/o:rocky:rocky:9::baseos"
   HOME_URL="https://rockylinux.org/"
   BUG_REPORT_URL="https://bugs.rockylinux.org/"
   SUPPORT_END="2032-05-31"
   ROCKY_SUPPORT_PRODUCT="Rocky-Linux-9"
   ROCKY_SUPPORT_PRODUCT_VERSION="9.4"
   REDHAT_SUPPORT_PRODUCT="Rocky Linux"
   REDHAT_SUPPORT_PRODUCT_VERSION="9.4"





第2-1部分：部署服务器Server-安装MySQL

1. 我们先ssh到pingmesh-server，运行：sudo yum install mariadb-server -y

2. 启动mariadb服务: sudo systemctl start mariadb

3. 在系统启动时自动启动 MariaDB 服务：sudo systemctl enable mariadb

4. 初始化mariadb: mysql_secure_installation

5. Switch to unix_socket authentication [Y/n] n

6. Change the root password? [Y/n] y //这里使用密码为123456
   
   New password: 
   Re-enter new password: 

7. 其他步骤略

8. 安装完毕后，执行mysql -u root -p

9. 先创建一个数据库，叫ping。命令是：MariaDB [(none)]> create database ping;

10. 退出mysql命令行：MariaDB [(none)]> exit;

11. 下载数据库样例表：wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh.sql

12. 把pingmesh.sql导入到mariadb，mysql -u root -p ping < pingmesh.sql

13. 确认导入表成功，MariaDB [ping]> show tables;
    +----------------+
    | Tables_in_ping |
    +----------------+
    | fw |
    | host |
    | valu |
    +----------------+
    3 rows in set (0.000 sec)

14. 最后关闭selinux: setenforce 0





第2-2部分：部署服务器Server-安装和配置服务

1. 安装go，这里的go版本必须是1.21.0

2. 运行下面的命令进行下载：
   
   wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
   
   sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

3. 设置环境变量
   
   vi ~/.bashrc

4. 增加内容：export PATH=$PATH:/usr/local/go/bin

5. 使环境变量生效：source ~/.bashrc

6. 检查go版本：[root@pingmesh-server ~]# go version
   go version go1.21.0 linux/amd64

7. 把我们2个项目文件都下载到服务器上：
   
    wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh-s-v1.1-GetHostIp.go
   
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh-s-v1.1-GetResult.go

8. 修改pingmesh-s-v1.1-GetHostIp.go里的代码，为pingmesh-server的内网ip
   
   lis,err := net.Listen("tcp","10.240.0.100:58098")               //监听端口

9. 修改pingmesh-s-v1.1-GetResult.go里的代码，为pingmesh-server的内网ip
   
   lis,err := net.Listen("tcp","10.240.0.100:58099")        //监听端口

10. 然后我们运行命令：go mod init pingmesh-server

11. 运行命令： go get github.com/go-sql-driver/mysql

12. 再执行：go mod tidy

13. 编译2个go文件，执行命令：
    
    go build pingmesh-s-v1.1-GetResult.go
    
    go build pingmesh-s-v1.1-GetHostIp.go
    
    我们就可以观察到编译的结果：
    
    ![](https://github.com/leizhang1984/pingmesh/blob/main/pingmesh-image/go-build-server.png)

14. 运行服务器端的2个服务
    
    nohup ./pingmesh-s-v1.1-GetResult > output.log 2>&1 &  
    nohup ./pingmesh-s-v1.1-GetHostIp > output.log 2>&1 &

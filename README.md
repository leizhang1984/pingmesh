# Pingmesh: A Large-Scale System for Data Center Network Latency Measurement and Analysis

# 一个用于数据中心网络延迟测量和分析的大规模系统

基于：[https://github.com/aprilmadaha/pingmesh](https://github.com/aprilmadaha/pingmesh%E3%80%82%E8%BF%9B%E8%A1%8C%E4%BA%86%E4%BF%AE%E6%94%B9%E3%80%82)， 进行了修改



# 主要修改的内容有：

1. 原来的代码是在客户端执行fping，这个版本是基于tcping。tcping监听的端口是22。

2. 因为fping可以支持多个destination ip，如：
   
   ```
   fping [10.100.0.5 10.100.0.6 -q -p 12000 -c 5]
   ```
   
   1. 返回：
   
      ```
      10.100.0.5 : xmt/rcv/%loss = 5/5/0%,min/avg/max = 0.79/0.86/0.94
      10.100.0.6 :xmt/rcv/%loss = 5/5/0%, min/avg/max = 0.01/0.01/0.02
      ```
   
   2. 在这个版本中，通过编写shell脚本，在multi_tcping.sh里执行多任务，对多个目标ip执行tcping测试
   
3. 原来的代码会通过fping自己测试自己，即client ip是10.100.0.5, destination ip也是10.100.0.5。在这个版本里会进行判断，如果client ip = destination ip，则跳过

4. 把服务器端的2个go部署到crontab里，随OS启动

5. 把客户端的服务部署到service里，随OS启动

   

# pingmesh架构：

![](https://github.com/leizhang1984/pingmesh/raw/main/pingmesh-image/pingmesh-architecture.png)

主要分为三个角色：

**服务器端：**

1. 部署MariaDB，创建database ，名字叫ping。在这个数据库里，有三张Table：
   
   1. 表Host，用来记录客户端的IP
   
   2. 表Valu，用来记录客户端之间tcping的测试结果

2. Server端起了2个服务：
   
   1. pingmesh-s-v1.1-GetHostIp.go，用来从MariaDB数据库里读取host记录，并下发给客户端
   
   2. pingmesh-s-v1.1-GetResult.go，允许客户端把测试结果进行上报，并保存到MariaDB数据表Valu里

**客户端：**

1. multi_tcping.sh，执行60次tcping，并把结果进行汇总和统计(平均值，最大值，最小值)
2. pingmesh-c-v1.1.go，从服务器端获得客户端IP列表，通过执行multi_tcping.sh，来进行tcping测试，并把测试结果上报给服务器端

**展示层：**

1. 对tcping的延迟结果进行展示





# 具体的配置步骤：

## 第1部分：环境准备    

1. 请先创建1个Azure Virtual Network，并创建1个subnet。步骤略

2. 创建4台虚拟机，操作系统为Rocky 9.4。步骤略：
   
   | 虚拟机名称        | 角色说明 | 内网IP地址   | 可用区 | 需要安装的软件  |
   | ----------------- | -------- | ------------ | ------ | --------------- |
   | pingmesh-server   | 服务器   | 10.240.0.100 | 1      | golang, mariadb |
   | pingmesh-client01 | 客户端01 | 10.240.0.101 | 1      | golang          |
   | pingmesh-client02 | 客户端02 | 10.240.0.102 | 2      | golang          |
   | pingmesh-client03 | 客户端03 | 10.240.0.103 | 3      | golang          |

3. 等待虚拟机创建完毕后，查看rocky 版本：
   
   ```
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
   ```
   
   

------



## 第2-1部分：部署服务器Server-安装MySQL

**请确保客户端和服务器端的时区都是相同的。我这里的演示环境，所有的虚拟机都是UTC时区**

1. 我们先ssh到pingmesh-server，运行：

   ```
   sudo yum install mariadb-server -y
   ```

2. 启动mariadb服务: 

   ```
   sudo systemctl start mariadb
   ```

3. 在系统启动时自动启动 MariaDB 服务：

   ```
   sudo systemctl enable mariadb
   ```

4. 初始化mariadb: 

   ```
   mysql_secure_installation
   
   Switch to unix_socket authentication [Y/n] n
   
   Change the root password? [Y/n] y 
   
   New password:   //这里使用密码为123456
   Re-enter new password: 
   ```

   

7. 其他步骤略

6. 安装完毕后，执行

   ```
   mysql -u root -p
   ```

   

7. 先创建一个数据库，叫ping (**不能用其他的database名称**)。命令是：

   ```
   MariaDB [(none)]> create database ping;
   ```

   

8. 退出mysql命令行：

   ```
   MariaDB [(none)]> exit;
   ```

   

9. 下载数据库样例表：

   ```bash
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh.sql
   ```

   

10. 把pingmesh.sql导入到mariadb，

    ```
    mysql -u root -p ping < pingmesh.sql
    ```

    

13. 把客户端的3个内网ip地址，都插入到表host里：
    
    ```sql
    use ping;
    insert into host(host) values ('10.240.0.101'),('10.240.0.102'),('10.240.0.103');
    ```

14. 确认导入表成功，
    
    ```
    MariaDB [ping]> show tables;
    +----------------+
    | Tables_in_ping |
    +----------------+
    | fw |
    | host |
    | valu |
    +----------------+
    3 rows in set (0.000 sec)
    ```
    
    
    
13. 暂时关闭selinux: 

    ```
    setenforce 0
    ```

    

14. 永久关闭selinux: 

    ```
    sudo vi /etc/selinux/config
    ```

    

17. 修改配置文件，找到 `SELINUX` 这一行，并将其值设置为 `disabled`。最后重启服务器

------



## 第2-2部分：部署服务器Server-安装和配置服务

**请确保客户端和服务器端的时区都是相同的。我这里的演示环境，所有的虚拟机都是UTC时区**

1. 安装go，这里的go版本必须是1.21.0

2. 运行下面的命令进行下载：
   
   ```
   wget https://golang.org/dl/go1.21.0.linux-amd64.tar.gz
   
   sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
   ```
   
   
   
3. 设置环境变量
   
   ```
   vi ~/.bashrc
   ```
   
   
   
4. 增加内容：

    ```
    export PATH=$PATH:/usr/local/go/bin
    ```

    

5. 使环境变量生效：

    ```
    source ~/.bashrc
    ```

    

6. 检查go版本：
   
   ```
   [root@pingmesh-server ~]# go version
   go version go1.21.0 linux/amd64
   ```
   
   
   
7. 把我们2个项目文件都下载到服务器上，这里我保存的路径是/software
   
    ```
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh-s-v1.1-GetHostIp.go
   
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Server/pingmesh-s-v1.1-GetResult.go
   ```
   
   
   
8. 修改pingmesh-s-v1.1-GetHostIp.go里的代码，为pingmesh-server的内网ip: 10.240.0.100
   
   ```
   lis,err := net.Listen("tcp","10.240.0.100:58098")               //监听端口
   ```
   
   
   
9. 修改pingmesh-s-v1.1-GetResult.go里的代码，为pingmesh-server的内网ip: 10.240.0.100
   
   ```
   lis,err := net.Listen("tcp","10.240.0.100:58099")        //监听端口
   ```
   
   
   
10. 然后我们运行命令：

    ```
    go mod init pingmesh-server
    ```

    

11. 运行命令：

     ```
      go get github.com/go-sql-driver/mysql
     ```

     

12. 再执行：

     ```
     go mod tidy
     ```

     

13. 编译2个go文件，执行命令：
    
    ```
    go build pingmesh-s-v1.1-GetResult.go
    
    go build pingmesh-s-v1.1-GetHostIp.go
    ```
    
    我们就可以观察到编译的结果：

    ![](https://github.com/leizhang1984/pingmesh/raw/main/pingmesh-image/go-build-server.png)
    
14. 把服务器端的2个服务，设置为开机自动启动
    
    ```
    crontab -e
    ```
    
    
    
15. 在crontab文件中，添加以下2行：

```shell
@reboot sleep 60 && nohup /software/pingmesh-s-v1.1-GetResult > /software/output.log 2>&1 &
@reboot sleep 60 && nohup /software/pingmesh-s-v1.1-GetHostIp >> /software/output.log 2>&1 &
```



------



## 第3部分：部署客户端服务-安装tcping和go环境

**请确保客户端和服务器端的时区都是相同的。我这里的演示环境，所有的虚拟机都是UTC时区**

1. 我们ssh登录到：pingmesh-client01

2. 暂时关闭selinux: 

   ```
   setenforce 0
   ```

3. 永久关闭selinux: 

   ```
   sudo vi /etc/selinux/config
   ```

   修改配置文件，找到 `SELINUX` 这一行，并将其值设置为 `disabled`。最后重启

4. 安装下面的步骤，安装tcping

   ```shell
   yum install -y tcptraceroute bc
   cd /usr/bin
   wget -O tcping https://soft.mengclaw.com/Bash/TCP-PING
   chmod +x tcping
   ```
   
   
   
5. 下载和安装go环境，具体可以参考上面的内容，步骤略。

6. 下载客户端程序：

   ```
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Client/pingmesh-c-v1.1.go
   ```

   

7. 修改上面go代码里的2个func，都是如下。10.240.0.100换成自己服务器端的ip

   ```
   conn, err := jsonrpc.Dial("tcp", "10.240.0.100:58099") //10.240.0.100换成自己服务器端的ip
   ```

   

8. 初始化go项目：

   ```
   go mod init pingmesh-client
   
   go get github.com/go-sql-driver/mysql
   
   go mod tidy
   ```

   

9. 编译代码：

   ```
   go build pingmesh-c-v1.1.go
   ```

   

10. 下载tcping的shell脚本：

   ```shell
   wget https://raw.githubusercontent.com/leizhang1984/pingmesh/refs/heads/main/Client/multi_tcping.sh
   ```

   

11. 设置shell脚本的权限为可执行: 

    ```
    chmod +x multi_tcping.sh
    ```

    

12. 设置客户端服务，让系统开机自动执行:

    ```
    sudo vi /etc/systemd/system/pingmeshclient.service
    ```

    

13. 设置服务配置文件

    ```
    [Unit]
    Description=Pingmesh Service
    After=network.target
    
    [Service]
    Type=simple
    User=root
    WorkingDirectory=/software
    ExecStartPre=/bin/sleep 60
    ExecStart=/software/pingmesh-c-v1.1
    StandardOutput=file:/software/output.log
    StandardError=file:/software/output.log
    Restart=on-failure
    
    [Install]
    WantedBy=multi-user.target
    ```

    重新加载：

    ```
    sudo systemctl daemon-reload
    ```

    

14. 启动服务：

    ```
    sudo systemctl enable pingmeshclient.service
    sudo systemctl start pingmeshclient.service
    ```

    

15. 在其他的客户端上，都执行上述的步骤。



------



## 第4部分：观察mariadb数据库里的tcping延迟数据

1. 我们ssh到pingmesh-server服务上，登录mariadb

2. 切换数据库：

   ```sql
   use ping;
   ```

   

3. 检查tcping延迟数据：
   
   ```sql
   select * from valu;
   ```
   
   可以看到src是源ip, dst是目标ip, loss丢包率，还有rttmin, rttavg, rttmax

   下图的日期列是tss，是个unix时间戳
   
   ![](https://github.com/leizhang1984/pingmesh/blob/main/pingmesh-image/mariadb-valu-1.png?raw=true)
   
4. 因为我这里所有的虚拟机(服务器+客户端)，都是UTC时区，如果我们想显示的日志是北京时区(UTC+8)，可以执行TSQL语句是：

   ```sql
   SELECT src, dst, loss, DATE_FORMAT(CONVERT_TZ(FROM_UNIXTIME(tss), '+00:00', '+08:00'), '%Y-%m-%d %H:%i:%s') AS tss_beijing_time, id, rttmin, rttavg, rttmax FROM valu;
   ```

   

![](https://github.com/leizhang1984/pingmesh/blob/main/pingmesh-image/mariadb-valu-2.png?raw=true)



------



## 第5部分：WebUI展示：

1. 我这里把WebUI展示的服务，部署在服务器端。当然你也可以单独部署一台Web UI
2. WebUI需要依赖python，我们首先检查python的版本：

```
[root@pingmesh-server software]# python3 --version
Python 3.9.18
```

3. 安装python包管理工具

```
yum install python3-pip -y
```

4. 安装python依赖

```
pip3 install flask
pip3 install pymysql
```

5. 然后我们下载项目文件，其中WebUI层在目录：WebUI

```
git clone https://github.com/leizhang1984/pingmesh
```

6. 我们cd WebUI目录
7. 修改pingmesh.py中的代码，因为我这里WebUI和服务器在同一台虚拟机里，所以host为localhost。你可以按照你的需要进行配置修改

```
conn = pymysql.connect(
    host='localhost',
    user='root',
    password='123456',
    db='ping',
    charset='utf8'
)
```

8. 运行WebUI服务

```
python3 pingmesh.py
```

如果需要后台运行，请执行：

```
nohup python3 pingmesh.py > pingmeshpy.log 2>&1 &
```

9. 然后我们打开浏览器，地址输入：http://[服务器端的ip]:9000/。显示效果如下：

![](https://github.com/leizhang1984/pingmesh/blob/main/pingmesh-image/webui.png)



------



## 第6部分：如何检查错误：

这里介绍一下常见的拍错步骤：

### 客户端检查

1. 请先确认客户端的服务已经启动

```shell
[root@pingmesh-client01 ~]# ps aux | grep "ping"
root         876  0.0  0.0 1228648 5268 ?        Ssl  13:51   0:00 /software/pingmesh-c-v1.1
```

2. 客户端pingmesh-c-v1.1的output.log是否有报错

```
[root@pingmesh-client01 software]# cat output.log 
[root@pingmesh-client01 software]#
```

3. 客户端的multi_tcping.sh脚本，会在/tmp产生psping统计数据的临时文件，临时文件一旦被统计完会自动删除

   ```
   [root@pingmesh-client01 tmp]# ll /tmp/
   total 36
   -rw-------. 1 root root  138 Oct  2 03:01 crontab.6j37O1
   drwx------  3 root root   17 Oct  2 13:50 systemd-private-1eccfe812c74408495ca78107b23f082-chronyd.service-Ad0tnd
   drwx------  3 root root   17 Oct  2 13:50 systemd-private-1eccfe812c74408495ca78107b23f082-dbus-broker.service-ps8r7z
   drwx------  3 root root   17 Oct  2 13:50 systemd-private-1eccfe812c74408495ca78107b23f082-systemd-logind.service-Xv8NkU
   -rw-------  1 root root  346 Oct  2 05:29 tmp.2hiM0kE8fI
   -rw-------  1 root root 2249 Oct  2 05:38 tmp.7SgH9ZRUxs
   -rw-------  1 root root  346 Oct  2 05:29 tmp.bXgm11LaIZ
   -rw-------  1 root root 2249 Oct  2 05:38 tmp.iscZ7oqGSU
   -rw-------  1 root root 5190 Oct  2 14:01 tmp.lqnVa3OYYK
   -rw-------  1 root root 5190 Oct  2 14:01 tmp.xFvA85cAxU
   ```

   

### 服务器端检查：

1. 请先确认服务器端的2个服务都已经启动

   ```shell
   [root@pingmesh-server ~]# ps aux | grep "ping"
   root        1067  0.0  0.0   7124  1424 ?        S    13:51   0:00 /bin/sh -c sleep 60 && nohup /software/pingmesh-s-v1.1-GetHostIp >> /software/output.log 2>&1 &
   root        1070  0.0  0.0   7124  1552 ?        S    13:51   0:00 /bin/sh -c sleep 60 && nohup /software/pingmesh-s-v1.1-GetResult > /software/output.log 2>&1 &
   root        1112  0.0  0.0 1230760 6936 ?        Sl   13:52   0:00 /software/pingmesh-s-v1.1-GetResult
   root        1113  0.0  0.0 1231016 7324 ?        Sl   13:52   0:00 /software/pingmesh-s-v1.1-GetHostIp
   root        1204  0.0  0.0   6408  2176 pts/0    S+   13:54   0:00 grep --color=auto ping
   ```

2. 检查output.log是否有客户端连接的日志

   ```
   [root@pingmesh-server software]# tail -f output.log 
   2025/10/02 13:59:02 <nil>
   2025/10/02 13:59:02 <nil>
   new client in comming
   2025/10/02 13:59:03 <nil>
   2025/10/02 13:59:03 <nil>
   2025/10/02 13:59:03 <nil>
   new client in comming
   2025/10/02 13:59:04 <nil>
   2025/10/02 13:59:04 <nil>
   2025/10/02 13:59:04 <nil>
   new client in comming
   2025/10/02 14:00:04 <nil>
   2025/10/02 14:00:04 <nil>
   2025/10/02 14:00:04 <nil>
   new client in comming
   2025/10/02 14:00:05 <nil>
   2025/10/02 14:00:05 <nil>
   2025/10/02 14:00:05 <nil>
   
   ```

   

   

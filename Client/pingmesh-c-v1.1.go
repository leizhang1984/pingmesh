package main

import (
	"log"
	"net"
	"net/rpc/jsonrpc"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"
)

type GetIpRequest struct{} // 得到IP请求结构体

type GetIpRespone struct { // 得到IP返回结构体
	Hostip []string // 返回所有IP地址
}

type Pingstruct struct {
	Tss       int64
	Src, Dst, Loss, Min, Avg, Max string
}

type UpIpArrayRequet struct { // 返回ping结果组结构体
	UpIparrayrequet []Pingstruct
}

type UpIpRespone struct{} // 得到IP返回结构体

var pingStructArray []Pingstruct
var mu sync.Mutex

func GetLocalIp() string { // 获取本地ip
	var IP string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Println("Error getting interface addresses:", err)
		os.Exit(1)
	}
	for _, address := range addrs {
		// 检查ip地址判断是否回环地址
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				IP = ipnet.IP.String()
				log.Println("Local IP found:", IP)
			}
		}
	}
	return IP
}

func runCommand(name string, ip string, port string, wg *sync.WaitGroup) { // 运行tcping脚本
	defer wg.Done()
	var pingstruct Pingstruct
	log.Printf("Running command: %s %s %s\n", name, ip, port)
	cmd := exec.Command(name, port, ip)
	vv, err := cmd.CombinedOutput()
	if err != nil {
		log.Println("Error running command:", err)
	}
	log.Println("Command output:", string(vv))
	re := regexp.MustCompile(`(.*) +: xmt/rcv/%loss = (.*), min/avg/max = (.*)`)
	submatchall := re.FindAllStringSubmatch(string(vv), -1)
	for _, element := range submatchall {
		pingstruct.Src = GetLocalIp()
		pingstruct.Tss = time.Now().Unix()
		pingstruct.Dst = element[1]
		pingstruct.Loss = strings.Split(element[2], "/")[2]
		pingstruct.Min = strings.Split(element[3], "/")[0]
		pingstruct.Avg = strings.Split(element[3], "/")[1]
		pingstruct.Max = strings.Split(element[3], "/")[2]
		mu.Lock()
		pingStructArray = append(pingStructArray, pingstruct)
		mu.Unlock()
	}
	log.Println("Pingstruct list:", pingStructArray)
}

func fPing(ipadd []string, port string) { // 获取目标ip,丢包率，ping平均延迟
	// 清空pingStructArray
	pingStructArray = []Pingstruct{}
	var wg sync.WaitGroup
	for _, ip := range ipadd {
		wg.Add(1)
		go runCommand("./multi_tcping.sh", ip, port, &wg)
	}
	wg.Wait()
}

func pingHost() []string { // 得到所有host组的ip
	log.Println("Starting pingHost")
	conn, err := jsonrpc.Dial("tcp", "10.100.0.4:58098") // 172.19.129.11:58098换成自己服务器的ip
	if err != nil {
		log.Fatalln("Dialing error:", err)
	}

	getiprequest := GetIpRequest{}
	var getiprespone GetIpRespone
	err = conn.Call("Ip.GetIp", getiprequest, &getiprespone)
	if err != nil {
		log.Fatalln("GetIp error: ", err)
	}
	conn.Close()
	log.Println("Received IPs:", getiprespone.Hostip)
	return getiprespone.Hostip
}

func UpIp() { // 上传tcping的结果
	log.Println("Starting UpIp")
	conn, err := jsonrpc.Dial("tcp", "10.100.0.4:58099")
	if err != nil {
		log.Fatalln("Dialing error:", err)
	}

	upip := UpIpArrayRequet{pingStructArray}
	var rippr UpIpRespone
	err = conn.Call("Ip.UpIp", upip, &rippr)
	if err != nil {
		log.Fatalln("Return error: ", err)
	}
	conn.Close()
	log.Println("UpIp response received")
}

func main() {
	log.Println("Starting program")
	ticker := time.NewTicker(time.Second * 60) // 每一分钟执行执行一次
	for {
		select {
		case <-ticker.C:
			log.Println("Ticker triggered")
			hostIPs := pingHost()
			port := "22" // 定义端口
			fPing(hostIPs, port) // 得到所有主机ip并ping
			UpIp()            // 提交tcping的结果
		}
	}
}

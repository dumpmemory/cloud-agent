package main

import (
	"io"
	"encoding/json"
	"crypto/md5"
	"strings"
	"encoding/hex"
	"io/ioutil"
	"github.com/CoiaPrant/zlog"
	"os"
	"os/exec"
	"net/http"
	"time"
	"runtime"
	"flag"
	"strconv"
	"os/signal"
	"syscall"
	"bytes"
	"math"
)

type JSONConfig struct {
	Time int
	API string
	Token string
}

type JSONInfo struct {
	Eth string
}


var ConfigFile string
var conf JSONConfig
var infomation JSONInfo
var LastTraffic uint64

func main(){
	flag.StringVar(&ConfigFile, "config", "config.json", "The config file location.")
	help := flag.Bool("h", false, "Show help")
	flag.Parse()

	if *help {
		flag.PrintDefaults()
		os.Exit(0)
	}

	zlog.Info("Start Cloud Agent...\n")
    config, err := ioutil.ReadFile(ConfigFile)
		if err != nil {
			zlog.Fatal("Cannot read the config file. (io Error) " + err.Error())
		}
    err = json.Unmarshal(config,&conf)
    if err != nil {
	zlog.Fatal("Cannot read the config file. (Parse Error) " + err.Error())
	}
	
	getInfo()
	updateInfo()

    go func(){
		for{
		saveInterval := time.Duration(conf.Time) * time.Second
		time.Sleep(saveInterval)
		updateInfo()
		}
	}()

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		done <- true
	}()
	<-done
	updateInfo()
	zlog.PrintText("Exiting\n")
}

func getInfo(){

	jsonData,_ := json.Marshal(map[string]interface{}{
		"Action" : "GetInfo",
		"Token" : md5_encode(conf.Token),
	})

	code,info,err :=sendRequest(conf.API,bytes.NewReader(jsonData),nil,"POST")
	if code != 200 {
		zlog.Fatal("Cannot read the config file. (io Error) " + err.Error())
	}
	err = json.Unmarshal(info,&infomation)
    if err != nil {
	zlog.Fatal("Cannot read the config file. (Parse Error) " + err.Error())
	}
	LastTraffic = 0
	zlog.Success("Get machine infomation.\n")
}

func updateInfo(){
var newInfo JSONInfo

result := shell_exec("cat /proc/net/dev | grep " + infomation.Eth +" | awk '{print $2}'")
intraffic,err := strconv.ParseUint(result,10,64)

if err != nil {
	zlog.Error("Bad In Traffic Value: ",result)
	return
}

result = shell_exec("cat /proc/net/dev | grep " + infomation.Eth +" | awk '{print $2}'")
outtraffic,err := strconv.ParseUint(result,10,64)

if err != nil {
	zlog.Error("Bad Out Traffic Value: ",result)
	return
}
traffic := intraffic + outtraffic

result = shell_exec("df / | grep / | awk '{print $4}'")
freedisk ,err := strconv.ParseUint(result,10,64)

if err != nil {
	zlog.Error("Bad FreeDisk Value: ",result)
	return
}

usetraffic := traffic - LastTraffic
LastTraffic = traffic
jsonData,_ := json.Marshal(map[string]interface{}{
	"Action" : "UpdateInfo",
	"Token" : md5_encode(conf.Token),
	"Traffic" : math.Ceil(float64(usetraffic) / 1048576),
	"FreeDisk" : math.Ceil(float64(freedisk) / 1024),
})

code,info,err :=sendRequest(conf.API,bytes.NewReader(jsonData),nil,"POST")
if code != 200 {
	zlog.Error("Cannot read the config file. (io Error) ")
	return
}
err = json.Unmarshal(info,&newInfo)
if err != nil {
zlog.Error("Cannot read the config file. (Parse Error) " + err.Error())
return
}
infomation = newInfo
zlog.Success("Update machine infomation.")
}

func sendRequest(url string, body io.Reader, addHeaders map[string]string, method string) (statuscode int,resp []byte,err error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return
	}

    req.Header.Set("User-Agent","Mozilla/5.0 (Windows NT 10.0; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/86.0.4240.198 Safari/537.36")

	if len(addHeaders) > 0 {
		for k, v := range addHeaders {
			req.Header.Add(k, v)
		}
	}

	client := &http.Client{}
	response, err := client.Do(req)
	if err != nil {
		return
	}
	defer response.Body.Close()

	statuscode = response.StatusCode
	resp, err = ioutil.ReadAll(response.Body)
	return
}

func shell_exec(command string) string {
    var cmd *exec.Cmd
    // 执行系统命令
    // 第一个参数是命令名称
    // 后面参数可以有多个，命令参数
    if(runtime.GOOS == "windows"){
    cmd = exec.Command("cmd.exe","/c",command)
    }else{
    cmd = exec.Command("bash","-c",command)
    }
    // 获取输出对象，可以从该对象中读取输出结果
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        zlog.Fatal(err)
    }
    // 保证关闭输出流
    defer stdout.Close()
    // 运行命令
    if err := cmd.Start(); err != nil {
        zlog.Fatal(err)
    }
    // 读取输出结果
    opBytes, err := ioutil.ReadAll(stdout)
    if err != nil {
        zlog.Fatal(err)
	}
	
	result := strings.TrimSpace(string(opBytes))
    return result
}

func md5_encode(s string)string{
	h :=md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}
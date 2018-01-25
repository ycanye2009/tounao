package main

import (
	"github.com/coreos/goproxy"
	"net/http"
	"log"
	"tounao/util"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"io/ioutil"
	"bytes"
	"tounao/lib"
	"flag"
	"strconv"
)

var (
	proxy *goproxy.ProxyHttpServer
	port  int
	mode  string
)

func init() {

	flag.IntVar(&port, "port", 8989, "-port=8989")
	flag.StringVar(&mode, "mode", "auto", "-mode=manual or -mode=auto")
	if mode != `auto` {
		mode = `manual`
		util.Auto = false
	} else {
		util.Auto = true
	}

	proxy = goproxy.NewProxyHttpServer()
	proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	//proxy.Verbose = true

	proxy.NonproxyHandler = http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Add("Content-Disposition", "attachment; filename=ca.crt")
		w.Header().Add("Content-Type", "application/octet-stream")
		w.Write(goproxy.CA_CERT)
	})

	//请求拦截
	requestHandle := func(request *http.Request, ctx *goproxy.ProxyCtx) (req *http.Request, resp *http.Response) {
		req = request

		//log.Println(ctx.Req.URL)

		if ctx.Req.URL.Path == `/question/bat/findQuiz` || ctx.Req.URL.Path == `/question/bat/choose` {

			requestBody, e := ioutil.ReadAll(req.Body)
			if util.Check(e) {

				if util.Auto {
					go lib.Injection(requestBody, ctx)
				} else {
					requestBody = lib.Injection(requestBody, ctx)
				}

				req.Body = ioutil.NopCloser(bytes.NewReader(requestBody))
			}
		}
		return
	}

	//返回拦截
	responseHandle := func(response *http.Response, ctx *goproxy.ProxyCtx) (resp *http.Response) {
		resp = response

		if ctx.Req.URL.Path == `/question/bat/findQuiz` || ctx.Req.URL.Path == `/question/bat/choose` {
			responseBody, e := ioutil.ReadAll(resp.Body)
			if util.Check(e) {

				if util.Auto {
					go lib.Injection(responseBody, ctx)
				} else {
					responseBody = lib.Injection(responseBody, ctx)
				}

				resp.Body = ioutil.NopCloser(bytes.NewReader(responseBody))
			}
		}

		return
	}
	proxy.OnRequest().DoFunc(requestHandle)
	proxy.OnResponse().DoFunc(responseHandle)

}

func main() {

	flag.Parse()
	go Run(strconv.Itoa(port))
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	fmt.Println("exiting")
}

func Run(port string) {

	go func() {
		log.Println("代理服务端口:", port)
		log.Printf("请将手机连接至同一网络，并设置代理地址为%s:%s\n", util.HostIP(), port)
		log.Printf("打开http://%s 即可安装证书\n", util.HostIP())
		log.Printf("当前模式为:%s\n", mode)

		e := http.ListenAndServe(":"+port, proxy)
		util.Check(e)
	}()
}
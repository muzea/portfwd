package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/docker/go-connections/proxy"
	"github.com/gin-gonic/gin"
)

var tcpProxyPool map[int]*proxy.TCPProxy
var udpProxyPool map[int]*proxy.UDPProxy

type config struct {
	Proxy   map[string]string `json:"proxy"`
	APIPort string            `json:"APIPort"`
}

var configResult config

func getFileContent(fileName string) []byte {
	buf, err := ioutil.ReadFile(fileName)
	if err != nil {
		log.Fatalln(err)
	}
	return buf
}

func resolveConfig() {
	configContent := getFileContent("./config.json")
	json.Unmarshal(configContent, &configResult)
	for localPort, target := range configResult.Proxy {
		go prepareTCPHandler(localPort, target)
		go prepareUDPHandler(localPort, target)
		log.Printf("proxy %s to %s", localPort, target)
	}
}

func prepareTCPHandler(localPort string, target string) {
	local := fmt.Sprintf(":%s", localPort)
	localAddr, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		log.Fatalln(err)
	}
	incomingPort, err := strconv.Atoi(localPort)
	if err != nil {
		log.Fatalln(err)
	}
	targetAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		log.Fatalln(err)
	}
	localProxy, err := proxy.NewTCPProxy(localAddr, targetAddr)
	if err != nil {
		log.Fatalln(err)
	}
	tcpProxyPool[incomingPort] = localProxy
	localProxy.Run()
}

func prepareUDPHandler(localPort string, target string) {
	local := fmt.Sprintf(":%s", localPort)
	localAddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		log.Fatalln(err)
	}
	incomingPort, err := strconv.Atoi(localPort)
	if err != nil {
		log.Fatalln(err)
	}
	targetAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		log.Fatalln(err)
	}
	localProxy, err := proxy.NewUDPProxy(localAddr, targetAddr)
	if err != nil {
		log.Fatalln(err)
	}
	udpProxyPool[incomingPort] = localProxy
	localProxy.Run()
}

func closeAndDelete(localPort int) {
	{
		tcpProxy, ok := tcpProxyPool[localPort]
		if ok {
			tcpProxy.Close()
			delete(tcpProxyPool, localPort)
		}
	}
	{
		udpProxy, ok := udpProxyPool[localPort]
		if ok {
			udpProxy.Close()
			delete(udpProxyPool, localPort)
		}
	}
}

func apiHandlePing(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

type proxyAddItem struct {
	Local  string `json:"local"  binding:"required"`
	Targrt string `json:"targrt" binding:"required"`
}

func apiHandleProxyAdd(ctx *gin.Context) {
	/**
	 * req
	 * ```json
	 * {
	 *   "local": "10086",
	 * 	 "targrt": "127.0.0.1:10010"
	 * }
	 * ```
	 */
	var item proxyAddItem
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	go prepareTCPHandler(item.Local, item.Targrt)
	go prepareUDPHandler(item.Local, item.Targrt)
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "done",
	})
}

type proxyUpdateItem struct {
	Targrt string `json:"targrt" binding:"required"`
}

func apiHandleProxyUpdate(ctx *gin.Context) {
	/**
	 * req
	 * ```json
	 * {
	 * 	 "targrt": "127.0.0.1:10010"
	 * }
	 * ```
	 */
	var item proxyUpdateItem
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	local := ctx.Param("local")
	incomingPort, err := strconv.Atoi(local)
	if err != nil {
		log.Fatalln(err)
	}
	closeAndDelete(incomingPort)
	go prepareTCPHandler(local, item.Targrt)
	go prepareUDPHandler(local, item.Targrt)
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "done",
	})
}

func apiHandleProxyDelete(ctx *gin.Context) {
	local := ctx.Param("local")
	incomingPort, err := strconv.Atoi(local)
	if err != nil {
		log.Fatalln(err)
	}
	closeAndDelete(incomingPort)
	ctx.JSON(http.StatusOK, gin.H{
		"message": "done",
	})
}

func apiHandleProxyList(ctx *gin.Context) {
	result := make(map[string]string)
	for localPort, tcpProxy := range tcpProxyPool {
		result[fmt.Sprintf("%d", localPort)] = tcpProxy.BackendAddr().String()
	}
	ctx.JSON(http.StatusOK, result)
}

func apiHandleProxyDetail(ctx *gin.Context) {
	local := ctx.Param("local")
	incomingPort, err := strconv.Atoi(local)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		log.Fatalln(err)
	}
	tcpProxy := tcpProxyPool[incomingPort]
	ctx.JSON(http.StatusOK, gin.H{
		"local":  local,
		"targrt": tcpProxy.BackendAddr().String(),
	})
}

func addAPIHandler(app *gin.Engine) {
	app.GET("/ping", apiHandlePing)
	app.POST("/proxy", apiHandleProxyAdd)
	app.PATCH("/proxy/:local", apiHandleProxyUpdate)
	app.DELETE("/proxy/:local", apiHandleProxyDelete)
	app.GET("/proxy", apiHandleProxyList)
	app.GET("/proxy/:local", apiHandleProxyDetail)
}

func main() {
	tcpProxyPool = make(map[int]*proxy.TCPProxy)
	udpProxyPool = make(map[int]*proxy.UDPProxy)
	resolveConfig()
	app := gin.Default()
	addAPIHandler(app)
	app.Run(configResult.APIPort)
}


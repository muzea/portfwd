package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/docker/go-connections/proxy"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var portSep = "/"

var tcpProxyPool map[int]*proxy.TCPProxy
var udpProxyPool map[int]*proxy.UDPProxy

var lock = sync.RWMutex{}

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

func addProxyItem(localPort string, target string) {
	lock.Lock()
	defer lock.Unlock()
	targetTCPAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		log.Fatalln(err)
	}
	var iLocalPortStart int
	var iLocalPortEnd int
	if strings.Index(localPort, portSep) >= 0 {
		_, err := fmt.Sscanf(localPort, "%d/%d", &iLocalPortStart, &iLocalPortEnd)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		iLocalPortStart, err = strconv.Atoi(localPort)
		if err != nil {
			log.Fatalln(err)
		}
		iLocalPortEnd = iLocalPortStart
	}
	for i := 0; (iLocalPortStart + i) <= iLocalPortEnd; i++ {
		targetTCPAddrCurrent := &net.TCPAddr{IP: targetTCPAddr.IP, Port: targetTCPAddr.Port + i, Zone: targetTCPAddr.Zone}
		targetUDPAddrCurrent := &net.UDPAddr{IP: targetTCPAddr.IP, Port: targetTCPAddr.Port + i, Zone: targetTCPAddr.Zone}
		prepareTCPHandler(iLocalPortStart+i, targetTCPAddrCurrent)
		prepareUDPHandler(iLocalPortStart+i, targetUDPAddrCurrent)
	}
	configResult.Proxy[localPort] = target
	log.Printf("proxy %s to %s", localPort, target)
}

func resolveConfig() {
	configContent := getFileContent("./config.json")
	json.Unmarshal(configContent, &configResult)
	for localPort, target := range configResult.Proxy {
		go addProxyItem(localPort, target)
	}
}

func prepareTCPHandler(localPort int, targetAddr *net.TCPAddr) {
	local := fmt.Sprintf(":%d", localPort)
	localAddr, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		log.Fatalln(err)
	}
	localProxy, err := proxy.NewTCPProxy(localAddr, targetAddr)
	if err != nil {
		log.Fatalln(err)
	}
	tcpProxyPool[localPort] = localProxy
	localProxy.Run()
}

func prepareUDPHandler(localPort int, targetAddr *net.UDPAddr) {
	local := fmt.Sprintf(":%d", localPort)
	localAddr, err := net.ResolveUDPAddr("udp", local)
	if err != nil {
		log.Fatalln(err)
	}
	localProxy, err := proxy.NewUDPProxy(localAddr, targetAddr)
	if err != nil {
		log.Fatalln(err)
	}
	udpProxyPool[localPort] = localProxy
	localProxy.Run()
}

func closeAndDelete(localPort string) {
	var iPortStart int
	var iPortEnd int
	var err error
	if strings.Index(localPort, portSep) >= 0 {
		_, err = fmt.Sscanf(localPort, "%d/%d", &iPortStart, &iPortEnd)
		if err != nil {
			log.Fatalln(err)
		}
	} else {
		iPortStart, err = strconv.Atoi(localPort)
		if err != nil {
			log.Fatalln(err)
		}
		iPortEnd = iPortStart
	}
	for i := 0; (iPortStart + i) <= iPortEnd; i++ {
		current := iPortStart + i
		{
			tcpProxy, ok := tcpProxyPool[current]
			if ok {
				tcpProxy.Close()
				delete(tcpProxyPool, current)
			}
		}
		{
			udpProxy, ok := udpProxyPool[current]
			if ok {
				udpProxy.Close()
				delete(udpProxyPool, current)
			}
		}
	}
	_, ok := configResult.Proxy[localPort]
	if ok {
		delete(configResult.Proxy, localPort)
	}
}

func apiHandlePing(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"message": "pong",
	})
}

type proxyAddItem struct {
	Local  string `json:"local"  binding:"required"`
	Target string `json:"target" binding:"required"`
}

func apiHandleProxyAdd(ctx *gin.Context) {
	/**
	 * req
	 * ```json
	 * {
	 *   "local": "10086",
	 * 	 "target": "127.0.0.1:10010"
	 * }
	 * ```
	 */
	var item proxyAddItem
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	go addProxyItem(item.Local, item.Target)
	// go prepareTCPHandler(item.Local, item.Target)
	// go prepareUDPHandler(item.Local, item.Target)
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "done",
	})
}

type proxyUpdateItem struct {
	Target string `json:"target" binding:"required"`
}

func apiHandleProxyUpdate(ctx *gin.Context) {
	/**
	 * req
	 * ```json
	 * {
	 * 	 "target": "127.0.0.1:10010"
	 * }
	 * ```
	 */
	var item proxyUpdateItem
	if err := ctx.ShouldBindJSON(&item); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	local := ctx.Param("local")
	closeAndDelete(local)
	go addProxyItem(local, item.Target)
	ctx.JSON(http.StatusCreated, gin.H{
		"message": "done",
	})
}

func apiHandleProxyDelete(ctx *gin.Context) {
	local := ctx.Param("local")
	closeAndDelete(local)
	ctx.JSON(http.StatusOK, gin.H{
		"message": "done",
	})
}

func apiHandleProxyList(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, configResult.Proxy)
}

func apiHandleProxyDetail(ctx *gin.Context) {
	local := ctx.Param("local")
	ctx.JSON(http.StatusOK, gin.H{
		"local":  local,
		"target": configResult.Proxy[local],
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
	app.UseRawPath = true
	app.Use(cors.Default())
	addAPIHandler(app)
	app.Run(configResult.APIPort)
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

var dstTCPMap map[int]*net.TCPAddr

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
	}
}

func prepareTCPHandler(localPort string, target string) {
	local := fmt.Sprintf(":%s", localPort)
	localTCPAddr, err := net.ResolveTCPAddr("tcp", local)
	if err != nil {
		log.Fatalln(err)
	}
	listener, err := net.ListenTCP("tcp", localTCPAddr)
	incomingPort, err := strconv.Atoi(localPort)
	if err != nil {
		log.Fatalln(err)
	}
	targetAddr, err := net.ResolveTCPAddr("tcp", target)
	if err != nil {
		log.Fatalln(err)
	}
	dstTCPMap[incomingPort] = targetAddr
	log.Printf("proxy %d to %s", incomingPort, target)
	for {
		conn, err := listener.AcceptTCP()
		if err != nil {
			if _, ok := dstTCPMap[incomingPort]; ok {
				log.Fatalln(err)
			}
			return
		}
		go handleTCPConn(conn, incomingPort)
	}
}

func handleTCPConn(client *net.TCPConn, incomingPort int) {
	log.Printf("Client '%v' connected!\n", client.RemoteAddr())
	targetAddr, ok := dstTCPMap[incomingPort]
	if !ok {
		client.Close()
		return
	}
	target, err := net.DialTCP("tcp", nil, targetAddr)
	if err != nil {
		log.Fatalln("Could not connect to remote server:", err)
		client.Close()
		return
	}
	log.Printf("Connection to server '%v' established!\n", target.RemoteAddr())

	go func() {
		_, err := io.Copy(target, client)
		defer target.Close()
		if err != nil {
			// @todo
			log.Println("[Copy Error][write to target]", err)
		}
	}()

	go func() {
		_, err := io.Copy(client, target)
		defer client.Close()
		if err != nil {
			// @todo
			log.Println("[Copy Error][write to client]", err)
		}
	}()
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
	targetAddr, err := net.ResolveTCPAddr("tcp", item.Targrt)
	if err != nil {
		log.Fatalln(err)
	}
	dstTCPMap[incomingPort] = targetAddr
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
	_, ok := dstTCPMap[incomingPort]
	if ok {
		delete(dstTCPMap, incomingPort)
	}
	ctx.JSON(http.StatusOK, gin.H{
		"message": "done",
	})
}

func apiHandleProxyList(ctx *gin.Context) {
	result := make(map[string]string)
	for localPort, targetAddr := range dstTCPMap {
		result[fmt.Sprintf("%d", localPort)] = fmt.Sprintf("%s:%d", targetAddr.IP.String(), targetAddr.Port)
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
	addr := dstTCPMap[incomingPort]
	ctx.JSON(http.StatusOK, gin.H{
		"local":  local,
		"targrt": fmt.Sprintf("%s:%d", addr.IP.String(), addr.Port),
	})
}

func addAPIHandler(app *gin.Engine) {
	app.GET("/ping", apiHandlePing)
	app.POST("/proxy", apiHandleProxyAdd)
	app.PATCH("/proxy/:local", apiHandleProxyAdd)
	app.DELETE("/proxy/:local", apiHandleProxyDelete)
	app.GET("/proxy", apiHandleProxyList)
	app.GET("/proxy/:local", apiHandleProxyDetail)
}

func main() {
	dstTCPMap = make(map[int]*net.TCPAddr)
	resolveConfig()

	app := gin.Default()
	addAPIHandler(app)
	app.Run(configResult.APIPort)
}

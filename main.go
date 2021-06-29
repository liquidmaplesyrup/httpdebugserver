package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/gin-gonic/gin"

	"net/http/httputil"
)

type Request struct {
	Timestamp time.Time
	Request   string
}

func main() {
	requestStore := make(map[string](chan Request))
	r := gin.Default()
	r.LoadHTMLGlob("templates/**/*")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(c *gin.Context) {
		uuid := uuid.NewString()
		requestStore[uuid] = make(chan Request, 10)
		c.HTML(http.StatusOK, "main/index.tmpl", gin.H{
			"title": "httpdebugserver.com",
			"uuid":  uuid,
		})
	})

	r.GET("/sse", func(c *gin.Context) {
		for i := 0; i < 5; i++ {
			fmt.Println("Emitting now")
			c.SSEvent("Hello", "World:")
			time.Sleep(1 * time.Second)
		}
	})

	r.GET("/request_stream/:uuid", func(c *gin.Context) {
		/*	chanStream := make(chan int, 10)
			go func() {
				defer close(chanStream)
				for i := 0; i < 5; i++ {
					chanStream <- i
					time.Sleep(time.Second * 1)
				}

			}() */
		uuid := c.Param("uuid")
		chanStream := requestStore[uuid]
		// ensure nginx doesn't buffer this response but sends the chunks rightaway
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		c.Stream(func(w io.Writer) bool {
			if msg, ok := <-chanStream; ok {
				fmt.Println("emitting")
				c.SSEvent("message", msg)
				return true
			}
			return false
		})
	})

	// this renders
	r.GET("/request/:uuid", func(c *gin.Context) {
		uuid := c.Param("uuid")
		c.HTML(http.StatusOK, "main/requests.tmpl", gin.H{
			"uuid":     uuid,
			"requests": []Request{},
			"mockUrl":  "http://localhost:8080/mock/" + uuid,
		})
	})

	// this accepts new request from client
	r.Any("/mock/:uuid/*rest", func(c *gin.Context) {
		uuid := c.Param("uuid")
		requests := requestStore[uuid]
		bytes, _ := httputil.DumpRequest(c.Request, true)

		requests <- Request{Timestamp: time.Now(), Request: string(bytes)}

		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

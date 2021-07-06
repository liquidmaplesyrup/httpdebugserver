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
	requestStore := make(map[string]map[(chan Request)]bool)
	r := gin.Default()
	r.LoadHTMLGlob("templates/**/*")

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	r.GET("/", func(c *gin.Context) {
		uuid := uuid.NewString()

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
		if _,ok := requestStore[uuid] ; !ok {
			requestStore[uuid] = make(map[(chan Request)]bool)
		}

		chanStream := make(chan Request, 10)
		requestStore[uuid][chanStream] = true

		// ensure nginx doesn't buffer this response but sends the chunks rightaway
		c.Writer.Header().Set("X-Accel-Buffering", "no")

		c.Stream(func(w io.Writer) bool {

			fmt.Println("emitting")

			select {
			case res := <-c.Request.Context().Done():
				fmt.Println(res)
				fmt.Println("Done executing")
				delete(requestStore[uuid], chanStream)
				fmt.Println("Length", len(requestStore[uuid]))
				return false
			case msg := <-chanStream:
				c.SSEvent("message", msg)
				return true
			}

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

		for key, _ := range requests {
			// index is the index where we are
			// element is the element from someSlice for where we are
			key <- Request{Timestamp: time.Now(), Request: string(bytes)}
		}

		c.JSON(200, gin.H{
			"message": "ok",
		})
	})

	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

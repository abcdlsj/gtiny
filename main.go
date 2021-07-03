package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

const (
	ALPHABET = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	BASE     = int64(len(ALPHABET))
)

func Encode(num int64) string {
	var sb strings.Builder
	for ; num > 0; num /= BASE {
		sb.WriteByte(ALPHABET[num%BASE])
	}
	return sb.String()
}

func Decode(str string) int64 {
	var num int64
	len := len(str)
	for i := len - 1; i >= 0; i-- {
		num = num*BASE + int64(strings.Index(ALPHABET, string(str[i])))
	}
	return num
}

var rdb *redis.Client

var ctx = context.Background()

func initRedisClient() {
	rdb = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	var increaseNum int64 = 100000000
	_, err := rdb.Set(ctx, "IncreaseNum", strconv.FormatInt(increaseNum, 10), 0).Result()
	if err != nil {
		log.Fatal(err)
	}
}

func increaseGlobalNumber() {
	_, err := rdb.Incr(ctx, "IncreaseNum").Result()
	if err != nil {
		log.Fatal(err)
	}
}

type Data struct {
	Url string `json:"url"`
}

type Item struct {
	ID        string
	Url       string
	ShortLink string
}

func save(item *Item) error {
	data, _ := json.Marshal(item)
	_, err := rdb.Set(ctx, item.ID, data, 0).Result()
	if err != nil {
		return err
	}
	return nil
}

func runServer() {
	r := gin.Default()

	r.POST("/gtiny", func(c *gin.Context) {
		data := Data{}
		c.BindJSON(&data)
		curStr, _ := rdb.Get(ctx, "IncreaseNum").Result()
		curNum, _ := strconv.ParseInt(curStr, 10, 64)
		link := Encode(curNum)

		//
		increaseGlobalNumber()
		// save to redis
		item := &Item{
			ID:        curStr,
			Url:       data.Url,
			ShortLink: link,
		}
		err := save(item)
		if err != nil {
			log.Fatal(err)
		}
		c.JSON(200, gin.H{
			"shortLink": link,
			"id":        curNum,
			"saveItem":  item,
		})
	})
	r.GET("/s/:link", func(c *gin.Context) {
		link := c.Param("link")
		id := Decode(link)
		data, _ := rdb.Get(ctx, strconv.FormatInt(id, 10)).Result()
		var item Item
		_ = json.Unmarshal([]byte(data), &item)
		c.Redirect(301, item.Url)
	})

	r.Run()
}

func main() {
	initRedisClient()
	runServer()
}

package main

import(
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
)

const (
	REDIS_HOST = ""
	REDIS_PORT = "6379"
	REDIS_PREFIX = "getshorty:"
)

type Url struct {
	id string
	destination string
}

func (this *Url) Save() error {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	defer c.Close()
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", REDIS_PREFIX + "url:" + this.id, this.id)

	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func GetUrl(id string) (*Url, error) {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	if err != nil {
		return nil, err
	}

	defer c.Close()

	reply, err := c.Do("GET", REDIS_PREFIX + "url:" + id)
	if reply == nil {
		return nil, nil
	}

	reply, err = redis.String(reply, err)
	if err != nil {
		return nil, err
	}

	fmt.Println("GET RESPONSE:")
	fmt.Println(reply)

	return &Url{id: id}, nil
}


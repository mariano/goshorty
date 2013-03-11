package main

import(
	"encoding/json"
	"errors"
	"github.com/garyburd/redigo/redis"
)

const (
	REDIS_HOST = ""
	REDIS_PORT = "6379"
	REDIS_PREFIX = "getshorty:"
)

type Url struct {
	Id string
	Destination string
}

func (this *Url) Save() error {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	defer c.Close()
	if err != nil {
		return err
	}

	data, err := json.Marshal(this)
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", REDIS_PREFIX + "url:" + this.Id, data)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func (this *Url) Delete() error {
	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	defer c.Close()
	if err != nil {
		return err
	}

	reply, err := c.Do("DEL", REDIS_PREFIX + "url:" + this.Id)
	if err == nil && reply != "OK" {
		return errors.New("Invalid Redis response")
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

	
	data, err := redis.Bytes(reply, err)
	if err != nil {
		return nil, err
	}

	var url Url
	if err := json.Unmarshal(data, &url); err != nil {
		return nil, err
	}

	return &url, nil
}

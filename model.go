package main

import(
	"crypto/rand"
	"encoding/json"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"github.com/garyburd/redigo/redis"
)

const (
	REDIS_HOST = ""
	REDIS_PORT = "6379"
	REDIS_PREFIX = "getshorty:"
	RESTRICT_DOMAIN = "workana.com"
	ID_LENGTH = 6
	alphanum = "0123456789abcdefghijklmnopqrstuvwxyz-"
)

type Url struct {
	Id string
	Destination string
}

func NewUrl(data string) (entity *Url, err error) {
	data = strings.TrimSpace(data)
	if len(data) == 0 {
		err = errors.New("Please specify an URL")
		return
	}

	if matches, _ := regexp.MatchString("^https?", data); !matches {
		data = "http://" + data
	}

	u, err := url.Parse(data)
	if err != nil {
		return
	} else if matches, _ := regexp.MatchString("[.]+", u.Host); !matches {
		err = errors.New("No valid domain in URL: " + u.Host)
		return
	}

	if matches, _ := regexp.MatchString("^[A-Za-z0-9.]*" + RESTRICT_DOMAIN, u.Host); len(RESTRICT_DOMAIN) > 0 && !matches {
		err = errors.New("Only URLs on " + RESTRICT_DOMAIN + " domain allowed")
		return
	}

	entity = &Url{Destination: u.String()}

	// Generate id

	c, err := redis.Dial("tcp", REDIS_HOST + ":" + REDIS_PORT)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, ID_LENGTH)
	for {
		rand.Read(bytes)
		for i, b := range bytes {
			bytes[i] = alphanum[b % byte(len(alphanum))]
		}
		id := string(bytes)
		if exists, _ := redis.Bool(c.Do("EXISTS", REDIS_PREFIX + "url:" + id)); !exists {
			entity.Id = id
			break
		}

		if exists, _ := GetUrl(id); exists == nil {
			entity.Id = id
			break
		}
	}

	entity.Save()

	return entity, nil
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

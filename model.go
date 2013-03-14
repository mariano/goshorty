package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
)

type Url struct {
	Id          string
	Destination string
	Created     time.Time
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

	if matches, _ := regexp.MatchString("^[A-Za-z0-9.]*"+settings.RestrictDomain, u.Host); len(settings.RestrictDomain) > 0 && !matches {
		err = errors.New("Only URLs on " + settings.RestrictDomain + " domain allowed")
		return
	}

	entity = &Url{Destination: u.String(), Created: time.Now()}

	// Generate id

	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	bytes := make([]byte, settings.UrlLength)
	for {
		rand.Read(bytes)
		for i, b := range bytes {
			bytes[i] = alphanum[b%byte(len(alphanum))]
		}
		id := string(bytes)
		if exists, _ := redis.Bool(c.Do("EXISTS", settings.RedisPrefix+"url:"+id)); !exists {
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

func GetUrl(id string) (*Url, error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	if err != nil {
		return nil, err
	}

	defer c.Close()

	reply, err := c.Do("GET", settings.RedisPrefix+"url:"+id)
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

func (this *Url) Save() error {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return err
	}

	data, err := json.Marshal(this)
	if err != nil {
		return err
	}

	reply, err := c.Do("SET", settings.RedisPrefix+"url:"+this.Id, data)
	if err == nil && reply != "OK" {
		err = errors.New("Invalid Redis response")
	}

	if err != nil {
		return err
	}

	return nil
}

func (this *Url) Delete() error {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return err
	}

	reply, err := c.Do("DEL", settings.RedisPrefix+"url:"+this.Id)
	if err == nil && reply != "OK" {
		return errors.New("Invalid Redis response")
	}

	return nil
}

func (this *Url) Hit() (err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return
	}

	now := time.Now()
	year, month, day := now.Date()
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	c.Send("INCR", prefix + "hits")
	c.Send("INCR", fmt.Sprintf(prefix + "year:%d", year))
	c.Send("INCR", fmt.Sprintf(prefix + "month:%d-%d", year, month))
	c.Send("INCR", fmt.Sprintf(prefix + "day:%d-%d-%d", year, month, day))
	c.Send("INCR", fmt.Sprintf(prefix + "hour:%d-%d-%d %d", year, month, day, now.Hour()))
	c.Flush()

	fmt.Println("HIT!", settings.RedisPrefix + "url:" + this.Id)
	return
}

func (this *Url) Stats() (stats map[string]int64, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month, day := now.Date()
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	reply, err := redis.Values(c.Do(
		"MGET", 
		prefix + "hits", 
		fmt.Sprintf(prefix + "year:%d", year),
		fmt.Sprintf(prefix + "month:%d-%d", year, month),
		fmt.Sprintf(prefix + "day:%d-%d-%d", year, month, day),
		fmt.Sprintf(prefix + "hour:%d-%d-%d %d", year, month, day, now.Hour()),
	))

	stats = map[string]int64{
		"hits": 0,
		"year": 0,
		"day": 0,
		"hour": 0,
	}

	var (
		tHits int64
		tYear int64
		tDay int64
		tHour int64
	)
	if _, err = redis.Scan(reply, &tHits, &tYear, &tDay, &tHour); err != nil {
		return nil, err
	}

	stats["hits"] = tHits
	stats["year"] = tYear
	stats["day"] = tDay
	stats["hour"] = tHour

	return
}

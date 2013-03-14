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
	keyy = "year:%d"
	keym = "month:%d-%d"
	keyd = "day:%d-%d-%d"
	keyh = "hour:%d-%d-%d %d"
	keyi = "minute:%d-%d-%d %d:%d"
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
	hour := now.Hour()
	var minute int
	switch {
	case now.Minute() < 15:
		minute = 0
	case now.Minute() < 30:
		minute = 15
	case now.Minute() < 45:
		minute = 30
	default:
		minute = 45
	}

	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	c.Send("INCR", prefix + "hits")
	c.Send("INCR", fmt.Sprintf(prefix + keyy, year))
	c.Send("INCR", fmt.Sprintf(prefix + keym, year, month))
	c.Send("INCR", fmt.Sprintf(prefix + keyd, year, month, day))
	c.Send("INCR", fmt.Sprintf(prefix + keyh, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(prefix + keyi, year, month, day, hour, minute))

	c.Flush()
	return
}

func (this *Url) Stats(what string) (stats map[string]int, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month, day := now.Date()
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	stats = map[string]int{"hits": 0}
	reply, err := redis.Int(c.Do("GET", prefix + "hits"))
	if err != nil {
		return stats, err
	}
	stats["hits"] = reply

	var (
		separator string
		search string
		moment int
	)

	switch {
	case what == "hour":
		search = prefix + keyi
		separator = " %d:"
		moment = now.Hour()
		search = fmt.Sprintf(search[0:strings.LastIndex(search, ":")] + ":*", year, month, day, moment)
		stats["0"], stats["15"], stats["30"], stats["45"] = 0, 0, 0, 0
	case what == "day":
		search = prefix + keyh
		separator = "-%d"
		moment = day
		search = fmt.Sprintf(search[0:strings.LastIndex(search, " ")] + "*", year, month, moment)
		for i := 0; i < 24; i++ {
			stats[fmt.Sprintf("%d", i)] = 0
		}
	case what == "month":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%d")] + "*", year, moment)
		for i := 0; i < 31; i++ {
			stats[fmt.Sprintf("%d", i)] = 0
		}
	case what == "year":
		search = prefix + keym
		separator = "%d-"
		moment = year
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%d")] + "*", moment)
		for i := 0; i < 12; i++ {
			stats[fmt.Sprintf("%d", i)] = 0
		}
	}

	stats, err = getStats(c, search, separator, moment, stats)
	if err != nil {
		return stats, err
	}

	return stats, nil
}

func sliceToInterface(slice []string) []interface{} {
	e := make([]interface{}, len(slice))
	for i, v := range slice {
		e[i] = v
	}
	return e
}

func getStats(c redis.Conn, search string, separator string, moment int, stats map[string]int) (map[string]int, error) {
	values, err := redis.Values(c.Do("KEYS", search))
	if err != nil {
		return stats, err
	} else if len(values) == 0 {
		return stats, err
	}
	var keys []string
	for _, value := range values {
		key, err := redis.String(value, nil)
		if err == nil {
			keys = append(keys, key)
		}
	}

	args := sliceToInterface(keys)
	values, err = redis.Values(c.Do("MGET", args...))
	if err != nil {
		return stats, err
	}

	separator = fmt.Sprintf(separator, moment)

	for i, value := range values {
		total, err := redis.Int(value, nil)
		if err == nil {
			key := strings.TrimSpace(keys[i][strings.LastIndex(keys[i], separator) + len(separator):])
			stats[key] = total
		}
	}

	return stats, nil
}


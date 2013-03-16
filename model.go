package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	keyy     = "year:%d"
	keym     = "month:%0d-%0.2d"
	keyd     = "day:%d-%0.2d-%0.2d"
	keyh     = "hour:%d-%0.2d-%0.2d %0.2d"
	keyi     = "minute:%d-%0.2d-%0.2d %0.2d:%0.2d"
)

type Url struct {
	Id          string
	Destination string
	Created     time.Time
}

type Stat struct {
	key   string
	Name  string
	Value int
}

type Format func(string) (string, error)

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
	minute := 5 * int(math.Abs(float64(now.Minute()/5)))
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	c.Send("INCR", prefix+"hits")
	c.Send("INCR", fmt.Sprintf(prefix+keyy, year))
	c.Send("INCR", fmt.Sprintf(prefix+keym, year, month))
	c.Send("INCR", fmt.Sprintf(prefix+keyd, year, month, day))
	c.Send("INCR", fmt.Sprintf(prefix+keyh, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(prefix+keyi, year, month, day, hour, minute))

	c.Flush()
	return
}

func (this *Url) Hits() (total int, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return total, err
	}

	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"
	result, err := c.Do("GET", prefix+"hits")
	if result == nil {
		return 0, nil
	}
	return redis.Int(result, err)
}

func (this *Url) Stats(past string) (stats []*Stat, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month, day := now.Date()
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"

	var (
		separator string
		search    string
		moment    int
		start     int
		limit     int
		increment int
	)

	start = 1
	increment = 1
	format := func(value string) (string, error) {
		return value, nil
	}
	switch {
	case past == "hour":
		search = prefix + keyi
		separator = " %d:"
		moment = now.Hour()
		search = fmt.Sprintf(search[0:strings.LastIndex(search, ":")]+":*", year, month, day, moment)
		start = 0
		limit = 60
		increment = 5
		format = func(value string) (string, error) {
			return fmt.Sprintf("%0.2d:%s", moment, value), nil
		}
	case past == "day":
		search = prefix + keyh
		separator = "-%d"
		moment = day
		search = fmt.Sprintf(search[0:strings.LastIndex(search, " ")]+"*", year, month, moment)
		limit = 24
		format = func(value string) (string, error) {
			return fmt.Sprintf("%s:00", value), nil
		}
	case past == "week":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"*", year, moment)
		start = day - (int(now.Weekday()) - 1)
		limit = start + 7
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %s", time.Weekday().String(), value), nil
		}
	case past == "month":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"*", year, moment)
		limit = time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			month, _ := strconv.ParseInt(value, 10, 32)
			return fmt.Sprintf("%s %d", time.Month().String(), month), nil
		}
	case past == "year":
		search = prefix + keym
		separator = "%d-"
		moment = year
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"*", moment)
		limit = 12
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%s-01", year, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				return "", err
			}
			return fmt.Sprintf("%s %d", time.Month().String(), year), nil
		}
	case past == "all":
		search = prefix + keyy
		separator = ":"
		search = search[0:strings.LastIndex(search, "%d")] + "*"
		start = now.Year() - 10
		limit = int(now.Year()) + 1
	default:
		return nil, errors.New(fmt.Sprintf("Invalid stat requested: %s", past))
	}

	stats, err = getStats(c, search, separator, moment, start, limit, increment, format)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

func getStats(c redis.Conn, search string, separator string, moment int, start int, limit int, increment int, format Format) ([]*Stat, error) {
	length := int(math.Ceil(float64((limit - start) / increment)))
	stats := make([]*Stat, length)

	keys := make(map[string]int, length)
	j := 0
	for i := start; i < limit; i += increment {
		key := fmt.Sprintf("%0.2d", i)
		name, err := format(key)
		if err != nil {
			name = key
		}

		keys[key] = j
		stats[j] = &Stat{
			key:   key,
			Name:  name,
			Value: 0,
		}
		j++
	}

	values, err := redis.Values(c.Do("KEYS", search))
	if err != nil {
		return stats, err
	} else if len(values) == 0 {
		return stats, err
	}

	redisKeys := make([]string, len(values))
	i := 0
	for _, value := range values {
		key, err := redis.String(value, nil)
		if err == nil {
			redisKeys[i] = key
			i++
		}
	}

	// Convert slice to interface

	args := make([]interface{}, len(redisKeys))
	for i, v := range redisKeys {
		args[i] = v
	}
	values, err = redis.Values(c.Do("MGET", args...))
	if err != nil {
		return stats, err
	}

	if strings.Index(separator, "%d") >= 0 {
		separator = fmt.Sprintf(separator, moment)
	}

	for i, value := range values {
		total, err := redis.Int(value, nil)
		if err == nil {
			redisKey := redisKeys[i]
			key := strings.TrimSpace(redisKey[(strings.LastIndex(redisKey, separator) + len(separator)):])
			index, present := keys[key]
			if present {
				stats[index].Value = total
			}
		}
	}

	return stats, nil
}

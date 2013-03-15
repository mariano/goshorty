package main

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/garyburd/redigo/redis"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	alphanum = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	keyy = "year:%d"
	keym = "month:%0d-%0.2d"
	keyd = "day:%d-%0.2d-%0.2d"
	keyh = "hour:%d-%0.2d-%0.2d %0.2d"
	keyi = "minute:%d-%0.2d-%0.2d %0.2d:%0.2d"
)

type Url struct {
	Id          string
	Destination string
	Created     time.Time
}

type Stat struct {
	Name string
	Value int
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

func (this *Url) Hits() (total int, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return total, err
	}

	prefix := settings.RedisPrefix + "stats:" + this.Id + ":"
	return redis.Int(c.Do("GET", prefix + "hits"))
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
		search string
		moment int
		start int
		limit int
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
		search = fmt.Sprintf(search[0:strings.LastIndex(search, ":")] + ":*", year, month, day, moment)
		start = 0
		limit = 45
		increment = 15
		format = func(value string) (string, error) {
			return fmt.Sprintf("%0.2d:%s", moment, value), nil
		}
	case past == "day":
		search = prefix + keyh
		separator = "-%d"
		moment = day
		search = fmt.Sprintf(search[0:strings.LastIndex(search, " ")] + "*", year, month, moment)
		limit = 24
		format = func(value string) (string, error) {
			return fmt.Sprintf("%s:00", value), nil
		}
	case past == "week":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")] + "*", year, moment)
		start = day - (int(now.Weekday()) - 1)
		limit = start + 6
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				fmt.Println("ERROR:", date, err.Error())
				return "", err
			}
			return fmt.Sprintf("%s %s", time.Weekday().String(), value), nil
		}
	case past == "month":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")] + "*", year, moment)
		limit = time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%0.2d-%s", year, month, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				fmt.Println("ERROR:", date, err.Error())
				return "", err
			}
			month, _ := strconv.ParseInt(value, 10, 32)
			return fmt.Sprintf("%s %d", time.Month().String(), month), nil
		}
	case past == "year":
		search = prefix + keym
		separator = "%d-"
		moment = year
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")] + "*", moment)
		limit = 12
		format = func(value string) (string, error) {
			date := fmt.Sprintf("%d-%s-01", year, value)
			time, err := time.Parse("2006-01-02", date)
			if err != nil {
				fmt.Println("ERROR:", date, err.Error())
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

	result := make(map[string]int)
	for i := start; i < limit; i += increment {
		result[fmt.Sprintf("%0.2d", i)] = 0
	}
	result, err = getStats(c, search, separator, moment, result)
	if err != nil {
		return nil, err
	}

	keys := make([]string, len(result))
	i := 0
	for k, _ := range result {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	length := len(keys)
	stats = make([]*Stat, length)

	i = 0
	for i = 0; i < length; i++ {
		name, err := format(keys[i])
		if err != nil {
			name = keys[i]
		}
		stats[i] = &Stat{Name: name, Value: result[keys[i]]}
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

	if strings.Index(separator, "%d") >= 0 {
		separator = fmt.Sprintf(separator, moment)
	}

	for i, value := range values {
		total, err := redis.Int(value, nil)
		if err == nil {
			key := strings.TrimSpace(keys[i][(strings.LastIndex(keys[i], separator) + len(separator)):])
			stats[key] = total
		}
	}

	return stats, nil
}


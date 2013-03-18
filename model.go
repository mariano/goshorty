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
	"sort"
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
	Name  string
	Value int
}

type Stats []*Stat
type Descending Stats
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

func (this *Url) Hit(r *Request) (err error) {
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
	hitsPrefix := prefix + "hits:"
	countriesPrefix := prefix + "countries:"
	browsersPrefix := prefix + "browsers:"
	osPrefix := prefix + "os:"

	c.Send("INCR", hitsPrefix+"total")
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyy, year))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keym, year, month))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyd, year, month, day))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyh, year, month, day, hour))
	c.Send("INCR", fmt.Sprintf(hitsPrefix+keyi, year, month, day, hour, minute))

	if r.Country != "" {
		c.Send("INCR", countriesPrefix+"total:" + r.Country)
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyy+":"+r.Country, year))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keym+":"+r.Country, year, month))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyd+":"+r.Country, year, month, day))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyh+":"+r.Country, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(countriesPrefix+keyi+":"+r.Country, year, month, day, hour, minute))
	}

	if !r.Bot {
		c.Send("INCR", browsersPrefix+"total:" + r.Browser)
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyy+":"+r.Browser, year))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keym+":"+r.Browser, year, month))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyd+":"+r.Browser, year, month, day))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyh+":"+r.Browser, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(browsersPrefix+keyi+":"+r.Browser, year, month, day, hour, minute))

		c.Send("INCR", osPrefix+"total:" + r.OS)
		c.Send("INCR", fmt.Sprintf(osPrefix+keyy+":"+r.OS, year))
		c.Send("INCR", fmt.Sprintf(osPrefix+keym+":"+r.OS, year, month))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyd+":"+r.OS, year, month, day))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyh+":"+r.OS, year, month, day, hour))
		c.Send("INCR", fmt.Sprintf(osPrefix+keyi+":"+r.OS, year, month, day, hour, minute))
	}

	c.Flush()
	return
}

func (this *Url) Hits() (total int, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return total, err
	}

	prefix := settings.RedisPrefix + "stats:" + this.Id + ":hits:"
	result, err := c.Do("GET", prefix+"total")
	if result == nil {
		return 0, nil
	}
	return redis.Int(result, err)
}

func (this *Url) Countries() (Stats, error) {
	return this.keyStats(settings.RedisPrefix + "stats:" + this.Id + ":countries:total:*")
}

func (this *Url) Browsers() (Stats, error) {
	return this.keyStats(settings.RedisPrefix + "stats:" + this.Id + ":browsers:total:*")
}

func (this *Url) OS() (Stats, error) {
	return this.keyStats(settings.RedisPrefix + "stats:" + this.Id + ":os:total:*")
}

func (this *Url) keyStats(search string) (stats Stats, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	values, err := redis.Values(c.Do("KEYS", search))
	if err != nil {
		return nil, err
	} else if len(values) == 0 {
		return nil, nil
	}

	keys := make([]interface{}, len(values))
	i := 0
	for _, value := range values {
		key, err := redis.String(value, nil)
		if err == nil {
			keys[i] = key
			i++
		}
	}
	
	values, err = redis.Values(c.Do("MGET", keys...))
	if err != nil {
		return nil, err
	}

	stats = make(Stats, len(values))

	for i, value := range values {
		key := keys[i].(string)
		total, err := redis.Int(value, nil)
		if err == nil {
			stats[i] = &Stat{ Name: key[strings.LastIndex(key, ":")+1:], Value: total }
		}
	}

	sort.Sort(stats)

	return stats, nil
}

func (this *Url) Stats(past string) (stats Stats, err error) {
	c, err := redis.Dial("tcp", settings.RedisUrl)
	defer c.Close()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	year, month, day := now.Date()
	prefix := settings.RedisPrefix + "stats:" + this.Id + ":hits:"

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
		search = fmt.Sprintf(search[0:strings.LastIndex(search, " ")]+" *", year, month, moment)
		limit = 24
		format = func(value string) (string, error) {
			return fmt.Sprintf("%s:00", value), nil
		}
	case past == "week":
		search = prefix + keyd
		separator = "%d-"
		moment = int(month)
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", year, moment)
		if int(now.Weekday()) == 0 {
			start = day - 6
		} else {
			start = day - (int(now.Weekday()) - 1)
		}
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
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", year, moment)
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
		search = fmt.Sprintf(search[0:strings.LastIndex(search, "-%0.2d")]+"-*", moment)
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

func getStats(c redis.Conn, search string, separator string, moment int, start int, limit int, increment int, format Format) (Stats, error) {
	length := int(math.Ceil(float64((limit - start) / increment)))
	stats := make(Stats, length)

	redisKeys := make([]interface{}, length)
	j := 0

	prefix := search[:strings.LastIndex(search, "*")]
	for i := start; i < limit; i += increment {
		key := fmt.Sprintf("%0.2d", i)
		name, err := format(key)
		if err != nil {
			name = key
		}

		redisKeys[j] = prefix + key
		stats[j] = &Stat{
			Name:  name,
			Value: 0,
		}
		j++
	}

	values, err := redis.Values(c.Do("MGET", redisKeys...))
	if err != nil {
		return stats, err
	}

	if strings.Index(separator, "%d") >= 0 {
		separator = fmt.Sprintf(separator, moment)
	}

	for i, value := range values {
		total, err := redis.Int(value, nil)
		if err == nil {
			stats[i].Value = total
		}
	}

	return stats, nil
}

func (s Stats) Len() int {
	return len(s)
}

func (s Stats) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s Stats) Less(i, j int) bool {
	return s[i].Value >= s[j].Value
}

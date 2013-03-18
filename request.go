package main

import (
	"errors"
	"github.com/mssola/user_agent"
	"github.com/nranchev/go-libGeoIP"
	"net/http"
	"strings"
)

type RequestParser struct {
	gi *libgeo.GeoIP
}

type Request struct {
	Country string
	Bot bool
	Mobile bool
	OS string
	Browser string
	Version string
}

func NewRequestParser(geoDb string) (parser *RequestParser, err error){
	parser = &RequestParser{}
	parser.gi, err = libgeo.Load(geoDb)
	if err != nil {
		return nil, err
	}
	return parser, nil
}

func (this *RequestParser) Parse(req *http.Request) (r *Request, err error) {
	r = &Request{}
	r.Country, err = this.geo(req)
	ua := new(user_agent.UserAgent)
	ua.Parse(req.UserAgent())
	r.Bot = ua.Bot()
	r.Mobile = ua.Mobile()
	r.OS = ua.OS()
	r.Browser, r.Version = ua.Browser()
	return r, err
}

func (this *RequestParser) geo(req *http.Request) (string, error) {
	ip := req.Header.Get("X-Real-Ip")
	forwarded := req.Header.Get("X-Forwarded-For")
	if ip == "" && forwarded == "" {
		i := strings.LastIndex(req.RemoteAddr, ":")
		if i != -1 {
			ip = req.RemoteAddr[:i]
		} else {
			ip = req.RemoteAddr
		}
	} else if forwarded != "" {
		// X-Forwarded-For is potentially a list of addresses separated with ","
		parts := strings.Split(forwarded, ",")
		// TODO: should return first non-local address 
		ip = parts[0]
	}

	ip = strings.TrimSpace(ip)
	if ip == "" {
		return "", errors.New("Could not obtain IP address from request")
	} else if ip == "[::1]" {
		// TODO: faked request
		ip = "190.50.75.97"
		//return "", nil
	}

	location := this.gi.GetLocationByIP(ip)
	if location == nil {
		return "", nil
	}
	return location.CountryCode, nil
}

func (this *RequestParser) Browser(req *http.Request) (bot bool, mobile bool, os string, browser string, version string) {
	ua := new(user_agent.UserAgent)
	ua.Parse(req.UserAgent())
	browserName, browserVersion := ua.Browser()
	return ua.Bot(), ua.Mobile(), ua.OS(), browserName, browserVersion
}

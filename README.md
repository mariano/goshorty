A URL shortener written in Go.

Lyrics by 50 Cent ;]

> Go, shorty
>
> It's your birthday
>
> We gon' party like it's your birthday
>
> We gon' sip Bacardi like it's your birthday

# Requirements #

Go, obviously :)

Some gorilla packages:

```bash
$ go get github.com/gorilla/mux
```

Redigo:

```bash
$ go get github.com/garyburd/redigo/redis
```

User agent parser:

```bash
$ go get github.com/mssola/user_agent
```

Go's implementation of Maxmind GeoIP API:

```bash
$ go get github.com/nranchev/go-libGeoIP
```

Download and extract MaxMind's GeoIP Country Database in binary format:

```bash
$ wget -N http://geolite.maxmind.com/download/geoip/database/GeoLiteCountry/GeoIP.dat.gz
$ gunzip GeoIP.dat.gz
```

# Build & Run #

```bash
$ go run *.go
```

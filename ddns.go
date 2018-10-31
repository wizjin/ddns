package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ddns struct {
	// config
	fetchURL string
	username string
	token    string
	domain   string
	client   *http.Client
	interval time.Duration
	// data
	hosts map[string]string
}

const apiprifex = "https://api.name.com/v4/domains/"

type record struct {
	ID     uint32 `json:"id,omitempty"`
	Host   string `json:"host,omitempty"`
	Type   string `json:"type,omitempty"`
	Answer string `json:"answer,omitempty"`
	TTL    uint32 `json:"ttl,omitempty"`
}

func (d *ddns) getWarnIP() (string, error) {
	resp, err := http.Get(d.fetchURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	return strings.TrimSpace(string(data)), err
}

func (d *ddns) updateIP(ip string) error {
	u := fmt.Sprintf("https://api.name.com/v4/domains/%s/records", d.domain)
	c := &http.Client{}
	r, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	r.SetBasicAuth(d.username, d.token)
	resp, err := c.Do(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	var res struct {
		Records []*record `json:"records,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return err
	}
	rds := map[string]*record{}
	for i, rec := range res.Records {
		if rec.Type == "A" {
			rds[rec.Host] = res.Records[i]
		}
	}
	for h := range d.hosts {
		rec, ok := rds[h]
		if ok {
			if rec.Answer == ip {
				r = nil
			} else {
				rec.Answer = ip
				b, _ := json.Marshal(rec)
				r, _ = http.NewRequest(http.MethodPut, fmt.Sprintf("%s/%d", u, rec.ID), bytes.NewReader(b))
			}
		} else {
			rec = &record{
				Host:   h,
				Type:   "A",
				Answer: ip,
				TTL:    300,
			}
			b, _ := json.Marshal(rec)
			r, _ = http.NewRequest(http.MethodPost, u, bytes.NewReader(b))
		}
		if r != nil {
			r.SetBasicAuth(d.username, d.token)
			resp, err := c.Do(r)
			if err != nil {
				return err
			}
			resp.Body.Close()
			d.hosts[h] = ip
		}
	}
	return nil
}

func initDDNS() *ddns {
	d := &ddns{}
	var hosts, proxy, interval string
	flag.StringVar(&d.fetchURL, "fetch-ip", "", "fetch ip url")
	flag.StringVar(&d.username, "username", "", "name.com api username")
	flag.StringVar(&d.token, "token", "", "name.com api token")
	flag.StringVar(&d.domain, "domain", "", "domain name")
	flag.StringVar(&hosts, "hosts", "", "host list")
	flag.StringVar(&proxy, "proxy", "", "name.com api proxy")
	flag.StringVar(&interval, "interval", "", "check ip interval")
	flag.Parse()
	d.hosts = map[string]string{}
	for _, h := range strings.Split(hosts, ",") {
		d.hosts[h] = ""
	}
	d.client = &http.Client{}
	if len(proxy) > 0 {
		p, err := url.Parse(proxy)
		if err != nil {
			log.Println("parse proxy failed:", err.Error())
		} else {
			d.client.Transport = &http.Transport{Proxy: http.ProxyURL(p)}
			log.Println("enable proxy:", proxy)
		}
	}
	d.interval = time.Second * 30
	if len(interval) > 0 {
		du, err := time.ParseDuration(interval)
		if err == nil {
			d.interval = du
		}
	}
	return d
}

func main() {
	d := initDDNS()
	for {
		if ip, err := d.getWarnIP(); err == nil {
			for _, v := range d.hosts {
				if v != ip {
					log.Println("find new ip:", ip)
					if err := d.updateIP(ip); err != nil {
						log.Println("update ip failed:", err.Error())
					} else {
						log.Println("update ip success.")
					}
					break
				}
			}
		}
		time.Sleep(d.interval)
	}
}

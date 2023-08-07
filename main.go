package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/nadoo/ipset"
	"github.com/oschwald/maxminddb-golang"
)

var (
	proxy     string
	ipsetName string
	mmdbURL   string
	timeout   int
)

func main() {
	flag.StringVar(&proxy, "proxy", "http://127.0.0.1:7788", "http proxy")
	flag.StringVar(&ipsetName, "ipset", "chnroute", "ipset name")
	flag.IntVar(&timeout, "timeout", 86400, "ipset timeout")
	flag.StringVar(&mmdbURL, "url", "https://cdn.jsdelivr.net/gh/Loyalsoldier/geoip@release/Country-only-cn-private.mmdb", "mmdb download url")
	flag.Parse()

	if err := downloadFile(mmdbURL, "Country-only-cn-private.mmdb", proxy); err != nil {
		log.Fatal(err)
	}

	db, err := maxminddb.Open("Country-only-cn-private.mmdb")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := ipset.Init(); err != nil {
		log.Fatal(err)
	}
	ipset.Flush(ipsetName)

	record := struct {
		Domain string `maxminddb:"connection_type"`
	}{}

	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		subnet, err := networks.Network(&record)
		if err != nil {
			log.Panic(err)
		}
		if subnet.IP.To4() != nil {
			if timeout > 0 {
				fmt.Printf("ipset add %s %s timeout %d\n", ipsetName, subnet.String(), timeout)
				ipset.Add(ipsetName, subnet.String(), ipset.OptTimeout(uint32(timeout)))
			} else {
				ipset.Add(ipsetName, subnet.String())
				fmt.Printf("ipset add %s %s\n", ipsetName, subnet.String())
			}
		}
	}
	if networks.Err() != nil {
		log.Panic(networks.Err())
	}
}

func downloadFile(downloadURL string, fileName string, proxy string) error {
	if strings.TrimSpace(proxy) != "" {
		proxyUrl, err := url.Parse(proxy)
		if err != nil {
			return err
		}
		http.DefaultTransport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	}
	resp, err := http.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

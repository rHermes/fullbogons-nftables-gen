package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode"
)

const tmplText = `
# Auto generated at {{.Date.UTC.Format "2006-01-02 15:04:05Z07:00"}} by fullbogons-nftables-gen
# Based on https://team-cymru.com/community-services/bogon-reference/
# Source at https://github.com/rhermes/fullbogons-nftables-gen

{{- /* IPv4 Bogons */}}
{{if .IPv4s }}
define IPV4_BOGONS = {
{{- range .IPv4s }}
  {{.}},
{{- end }}
}
{{ else }}
define IPV4_BOGONS = {}
{{- end -}}

{{- /* IPv6 Bogons */}}
{{- if .IPv6s }}
define IPV6_BOGONS = {
{{- range .IPv6s }}
  {{.}},
{{- end }}
}
{{ else }}
define IPV6_BOGONS = {}
{{- end -}}
`

const (
	// IPv4ListUrl = "https://www.team-cymru.org/Services/Bogons/fullbogons-ipv4.txt"
	// IPv6ListUrl = "https://www.team-cymru.org/Services/Bogons/fullbogons-ipv6.txt"
	IPv4ListUrl = "http://localhost:8080/fullbogons-ipv4.txt"
	IPv6ListUrl = "http://localhost:8080/fullbogons-ipv6.txt"
)

// Data is the data passed to the tmplText
type Data struct {
	Date  time.Time
	IPv4s []net.IPNet
	IPv6s []net.IPNet
}

func main() {
	// Load the template
	tRoot := template.New("main")
	tmpl, err := tRoot.Parse(strings.TrimLeftFunc(tmplText, unicode.IsSpace))
	if err != nil {
		log.Fatalf("parse template: %v", err)
	}

	// We can't wait all day!
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	ctx = ctx

	// Execute the template
	data := Data{Date: time.Now()}

	var wg sync.WaitGroup

	// IPv4
	wg.Add(1)
	go func() {
		defer wg.Done()
		ips, err := fetchIpList(ctx, ValidIPv4, IPv4ListUrl)
		if err != nil {
			log.Fatalf("fetch ipv4 list: %v", err)
		}
		if len(ips) > 3 {
			data.IPv4s = ips[:4]
		} else {
			data.IPv4s = ips
		}
	}()

	// IPv6
	wg.Add(1)
	go func() {
		defer wg.Done()
		ips, err := fetchIpList(ctx, ValidIPv6, IPv6ListUrl)
		if err != nil {
			log.Fatalf("fetch ipv6 list: %v", err)
		}
		// data.IPv6s = ips
		if len(ips) > 3 {
			data.IPv6s = ips[:4]
		} else {
			data.IPv6s = ips
		}
	}()

	wg.Wait()

	if err := tmpl.Execute(os.Stdout, data); err != nil {
		log.Fatalf("execute template: %v", err)
	}
}

// fetchIpList fetches the IP list and validates the contents
func fetchIpList(ctx context.Context, validator IpValidator, listUrl string) ([]net.IPNet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, listUrl, http.NoBody)
	if err != nil {
		return nil, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	ips := make([]net.IPNet, 0)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Parse line
		_, ip, err := net.ParseCIDR(line)
		if err != nil {
			return nil, err
		}

		if !validator(ip) {
			return nil, fmt.Errorf("invalid ip by validator: %s", ip.String())
		}

		ips = append(ips, *ip)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return ips, nil
}

// An IpValidator returns if the ip is a valid ip for the context.
type IpValidator func(ip *net.IPNet) bool

// Taken from https://github.com/asaskevich/govalidator
func ValidIPv4(ip *net.IPNet) bool {
	return ip != nil && strings.Contains(ip.IP.String(), ".")
}

func ValidIPv6(ip *net.IPNet) bool {
	return ip != nil && strings.Contains(ip.IP.String(), ":")
}

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/renameio/v2"
)

const (
	IPv4ListUrl = "https://www.team-cymru.org/Services/Bogons/fullbogons-ipv4.txt"
	IPv6ListUrl = "https://www.team-cymru.org/Services/Bogons/fullbogons-ipv6.txt"
)

// Data is the data passed to the tmplText
type Data struct {
	Date  time.Time
	IPv4s []net.IPNet
	IPv6s []net.IPNet
}

func main() {
	if len(os.Args) != 2 {
		log.Fatalf("We require one argument, that being the filename")
	}

	fd, err := renameio.NewPendingFile(os.Args[1],
		renameio.WithExistingPermissions(),
		renameio.WithPermissions(0644),
	)
	if err != nil {
		log.Fatalf("new pending file: %v", err)
	}
	defer fd.Cleanup()

	// We can't wait all day!
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

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
		data.IPv4s = ips
	}()

	// IPv6
	wg.Add(1)
	go func() {
		defer wg.Done()
		ips, err := fetchIpList(ctx, ValidIPv6, IPv6ListUrl)
		if err != nil {
			log.Fatalf("fetch ipv6 list: %v", err)
		}
		data.IPv6s = ips
	}()
	wg.Wait()

	if err := writeDefFile(fd, data); err != nil {
		log.Fatalf("writeDefFile: %v", err)
	}

	if err := fd.CloseAtomicallyReplace(); err != nil {
		log.Fatalf("closing file atomically: %v", err)
	}
}

// writeDefFile writes the tempalte to the given writer based on the
// data in d
func writeDefFile(w io.Writer, d Data) error {
	bw := bufio.NewWriter(w)
	// Write the header
	if _, err := fmt.Fprintf(bw, strings.TrimSpace(`
# Generated by fullbogons-nftables-gen at %s
# Based on https://team-cymru.com/community-services/bogon-reference/
# Source at https://github.com/rhermes/fullbogons-nftables-gen 
	`), d.Date.Format("2006-01-02 15:04:05Z07:00")); err != nil {
		return err
	}

	// newline
	if _, err := fmt.Fprintln(bw, "\n"); err != nil {
		return err
	}

	// Write IPV4_BOGONS
	if err := writeIpList(bw, "IPV4_BOGONS", d.IPv4s); err != nil {
		return err
	}

	// newline
	if _, err := fmt.Fprintln(bw, ""); err != nil {
		return err
	}

	// Write IPV6_BOGONS
	if err := writeIpList(bw, "IPV6_BOGONS", d.IPv6s); err != nil {
		return err
	}

	// remember to flush the buffer
	return bw.Flush()
}

// writeIpList writes
func writeIpList(w io.Writer, name string, ips []net.IPNet) error {
	if _, err := fmt.Fprintf(w, "define %s = {", name); err != nil {
		return err
	}

	for i, ip := range ips {
		s := "  %s,\n"
		if i == 0 {
			s = "\n  %s,\n"
		}
		if _, err := fmt.Fprintf(w, s, ip.String()); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}

	return nil
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

// Taken from https://github.com/asaskevich/govalidator
func ValidIPv6(ip *net.IPNet) bool {
	return ip != nil && strings.Contains(ip.IP.String(), ":")
}

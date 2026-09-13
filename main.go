package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

type geoResponse struct {
	City        string  `json:"city"`
	Region      string  `json:"region"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	ip := r.RemoteAddr
	if host, _, err := net.SplitHostPort(ip); err == nil {
		ip = host
	}
	return strings.Trim(ip, "[]")
}

func getLocation(ip string) string {
	if ip == "" || ip == "::1" || ip == "127.0.0.1" {
		return "local system"
	}
	if strings.HasPrefix(ip, "10.") || strings.HasPrefix(ip, "192.168.") || strings.HasPrefix(ip, "172.") {
		return "private network"
	}

	url := fmt.Sprintf("https://ipapi.co/%s/json/", ip)
	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Get(url)
	if err != nil || resp == nil {
		return "unknown"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "unknown"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "unknown"
	}

	var geo geoResponse
	if err := json.Unmarshal(body, &geo); err != nil {
		return "unknown"
	}

	parts := []string{}
	if geo.City != "" {
		parts = append(parts, geo.City)
	}
	if geo.Region != "" {
		parts = append(parts, geo.Region)
	}
	if geo.Country != "" {
		parts = append(parts, geo.Country)
	}
	if len(parts) == 0 {
		return "unknown"
	}
	return strings.Join(parts, ", ")
}

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		location := getLocation(ip)

		log.Printf("Incoming request: method=%s remote=%s host=%s path=%s user_agent=%q referer=%q location=%s",
			r.Method,
			ip,
			r.Host,
			r.URL.Path,
			r.UserAgent(),
			r.Referer(),
			location,
		)

		w.Header().Set("Content-Type", "image/png")
		http.ServeFile(w, r, "plug-ev-stay-tuned.png")
	})

	log.Println("Serving EV charging landing image on http://localhost:8081")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type Config struct {
	ZoneID  string   `json:"zoneId"`
	APIToken string   `json:"apiToken"`
	Records []Record `json:"records"`
}

type Record struct {
	Endpoint              string `json:"endpoint"`
	ID                    string `json:"id"`
	Name                  string `json:"name"`
	UpdateIntervalMinutes int    `json:"updateIntervalMinutes"`
}

type IPResponse struct {
	IP string `json:"origin"`
}

func main() {
	log.Println("Starting TINA DDNS ...")

	config, err := loadConfig("config.json")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	for _, record := range config.Records {
		r := record
		go startWorker(config, r)
	}

	select {} // block forever
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.Unmarshal(data, &config)
	return &config, err
}

func startWorker(config *Config, record Record) {
	ticker := time.NewTicker(time.Duration(record.UpdateIntervalMinutes) * time.Minute)
	defer ticker.Stop()

	log.Printf("Worker started for %s (%s)", record.Name, record.Endpoint)
    
    var lastIP string

	// run once immediately
	lastIP = runUpdate(config, record, lastIP)

	for range ticker.C {
		lastIP = runUpdate(config, record, lastIP)
	}
}

func runUpdate(config *Config, record Record, lastIP string) string {
	log.Printf("INFO: Fetching IP from %s", record.Endpoint)

	ip, err := fetchIP(record.Endpoint)
	if err != nil || ip == "" {
		log.Printf("ERROR: Failed to fetch IP: %v", err)
		return lastIP
	}

	log.Printf("DEBUG: Received IP: %s", ip)

	// 🚨 Compare with last known IP
	if ip == lastIP {
		log.Printf("INFO: IP unchanged for %s (%s), skipping update", record.Name, ip)
		return lastIP
	}

	log.Printf("INFO: IP changed for %s: %s -> %s", record.Name, lastIP, ip)

	err = updateCloudflare(config, record, ip)
	if err != nil {
		log.Printf("ERROR: Failed to update DNS for %s: %v", record.Name, err)
		return lastIP
	}

	log.Printf("INFO: Successfully updated %s -> %s", record.Name, ip)

	return ip
}

func fetchIP(endpoint string) (string, error) {
	resp, err := http.Get(endpoint)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	log.Printf("DEBUG: Raw response from endpoint: %s", string(body))

	var ipResp IPResponse
	err = json.Unmarshal(body, &ipResp)
	if err != nil {
		return "", err
	}

	if ipResp.IP == "" {
		return "", fmt.Errorf("empty IP in response")
	}

	return ipResp.IP, nil
}

func updateCloudflare(config *Config, record Record, ip string) error {
	url := fmt.Sprintf("https://api.cloudflare.com/client/v4/zones/%s/dns_records/%s",
		config.ZoneID, record.ID)

	payload := map[string]interface{}{
		"type":    "A",
		"name":    record.Name,
		"content": ip,
		"proxied": false,
	}

	data, _ := json.Marshal(payload)

	req, _ := http.NewRequest("PATCH", url, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+config.APIToken)

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	log.Printf("DEBUG: Cloudflare response: %s", string(body))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("cloudflare API returned status: %d", resp.StatusCode)
}

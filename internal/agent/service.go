package agent

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/motadata/motadata-host-agent/internal/config"
	"github.com/motadata/motadata-host-agent/internal/system"
)

// Service orchestrates periodic discovery and reporting of bare metal services.
type Service struct {
	cfg        *config.Config
	cache      *system.ServiceCache
	httpClient *http.Client
}

// NewService constructs a Service with the provided configuration.
func NewService(cfg *config.Config) *Service {
	return &Service{
		cfg:   cfg,
		cache: &system.ServiceCache{},
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec
			},
		},
	}
}

// Run starts the discovery and reporting loops and blocks until they exit.
func (s *Service) Run() error {
	outCh := make(chan []system.Service, 1)

	go s.discoveryLoop(outCh)
	go s.cacheUpdateLoop(outCh)

	if s.cfg.ServerURL != "" && s.cfg.HostName != "" {
		go s.reportLoop()
	} else {
		log.Println("MOTADATA_SERVER_URL or HOST_NAME not set – periodic reporting disabled")
	}

	select {} // block until the process is signalled
}

// discoveryLoop runs DiscoverServices on a fixed interval and sends results
// to outCh. A tick fires immediately so the first report comes without delay.
func (s *Service) discoveryLoop(outCh chan<- []system.Service) {
	interval := time.Duration(s.cfg.PostIntervalSeconds) * time.Second

	// Trigger immediately, then repeat.
	s.runDiscovery(outCh)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		s.runDiscovery(outCh)
	}
}

func (s *Service) runDiscovery(outCh chan<- []system.Service) {
	services, err := system.DiscoverServices()
	if err != nil {
		log.Printf("discovery error: %v", err)
		return
	}
	outCh <- services
}

// cacheUpdateLoop reads discovered services and updates the in-memory cache.
func (s *Service) cacheUpdateLoop(outCh <-chan []system.Service) {
	for services := range outCh {
		for _, svc := range services {
			s.cache.Store(svc.Name, svc)
		}
		log.Printf("cache updated: %d services", len(services))
	}
}

// reportLoop periodically POSTs the current cache snapshot to the server.
func (s *Service) reportLoop() {
	interval := time.Duration(s.cfg.PostIntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := s.postServices(); err != nil {
			log.Printf("report error: %v", err)
		}
	}
}

type reportPayload struct {
	HostName   string           `json:"host.name"`
	Deployment string           `json:"deployment"`
	Services   []system.Service `json:"services"`
}

func (s *Service) postServices() error {
	services := s.cache.All()
	if len(services) == 0 {
		return nil
	}

	payload := reportPayload{
		HostName:   s.cfg.HostName,
		Deployment: s.cfg.Deployment,
		Services:   services,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	resp, err := s.httpClient.Post(s.cfg.ServerURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("HTTP POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("server returned %s", resp.Status)
	}

	log.Printf("reported %d services to %s", len(services), s.cfg.ServerURL)
	return nil
}

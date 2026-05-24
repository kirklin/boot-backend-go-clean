package geoip

import (
	"fmt"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/service"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
)

// Resolver implements gateway.GeoIPResolver using the ip2region offline database.
// It is thread-safe: the underlying ip2region searcher uses a fully memory-cached
// xdb buffer and performs no I/O after initialisation.
type Resolver struct {
	ip2r *service.Ip2Region
}

// Ensure Resolver implements gateway.GeoIPResolver at compile time.
var _ gateway.GeoIPResolver = (*Resolver)(nil)

// NewResolver creates a new GeoIP resolver by loading the ip2region xdb data files.
// v4XdbPath and v6XdbPath are paths to the IPv4 and IPv6 database files respectively.
// Pass an empty string to skip a version (e.g., v6XdbPath="" if only IPv4 is needed).
//
// Returns an error if neither database can be loaded.
func NewResolver(v4XdbPath, v6XdbPath string) (*Resolver, error) {
	ip2r, err := service.NewIp2RegionWithPath(v4XdbPath, v6XdbPath)
	if err != nil {
		return nil, fmt.Errorf("geoip: failed to init ip2region: %w", err)
	}
	return &Resolver{ip2r: ip2r}, nil
}

// Resolve looks up the geographic location for the given IP address.
// Returns an empty GeoLocation if the IP cannot be resolved.
func (r *Resolver) Resolve(ip string) gateway.GeoLocation {
	if r == nil || r.ip2r == nil {
		return gateway.GeoLocation{}
	}

	region, err := r.ip2r.Search(ip)
	if err != nil {
		logger.GetLogger().Debugf("GeoIP: resolve %s failed: %v", ip, err)
		return gateway.GeoLocation{}
	}

	return parseRegion(region)
}

// Close releases resources held by the resolver.
func (r *Resolver) Close() error {
	if r == nil || r.ip2r == nil {
		return nil
	}
	r.ip2r.Close()
	return nil
}

// parseRegion parses the ip2region result string.
//
// ip2region format: "国家|区域|省份|城市|ISP"
//
//	Index:             0     1    2    3    4
//
// Example: "中国|0|北京|北京市|电信"
// The "0" sentinel means the field is empty/unknown and is cleaned to "".
func parseRegion(region string) gateway.GeoLocation {
	parts := strings.Split(region, "|")
	if len(parts) < 5 {
		return gateway.GeoLocation{}
	}

	return gateway.GeoLocation{
		Country:  clean(parts[0]),
		Region:   clean(parts[1]),
		Province: clean(parts[2]),
		City:     clean(parts[3]),
		ISP:      clean(parts[4]),
	}
}

// clean replaces the ip2region "0" sentinel with an empty string.
func clean(s string) string {
	if s == "0" {
		return ""
	}
	return s
}

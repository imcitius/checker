// SPDX-License-Identifier: BUSL-1.1

package checks

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ErrDNSEmptyDomain        = "empty domain"
	ErrDNSLookupFailed    = "dns lookup failed: %s"
	ErrDNSExpectedMissing = "expected value %q not found in DNS results: %v"
	ErrInvalidRecordType  = "unsupported DNS record type: %s"
)

// DNSCheck represents a DNS health check.
type DNSCheck struct {
	Host       string // Custom DNS resolver address (optional, e.g. "8.8.8.8:53")
	Domain     string // Domain to look up
	RecordType string // A, AAAA, MX, TXT, NS, CNAME
	Timeout    string
	Expected   string // Expected value in results (optional)
	Logger     *logrus.Entry
}

// Run executes the DNS health check.
func (check *DNSCheck) Run() (time.Duration, error) {
	start := time.Now()

	if check.Domain == "" {
		return time.Since(start), errors.New(ErrDNSEmptyDomain)
	}

	// Default record type to A
	recordType := strings.ToUpper(check.RecordType)
	if recordType == "" {
		recordType = "A"
	}

	// Validate record type
	switch recordType {
	case "A", "AAAA", "MX", "TXT", "NS", "CNAME":
		// valid
	default:
		return time.Since(start), fmt.Errorf(ErrInvalidRecordType, recordType)
	}

	// Parse timeout
	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	// Ensure logger is initialized
	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "dns")
	}

	// Set up resolver
	resolver := net.DefaultResolver
	if check.Host != "" {
		resolver = &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{Timeout: timeout}
				host := check.Host
				// Add default DNS port if not specified
				if !strings.Contains(host, ":") {
					host = host + ":53"
				}
				return d.DialContext(ctx, "udp", host)
			},
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	var results []string

	switch recordType {
	case "A", "AAAA":
		addrs, err := resolver.LookupHost(ctx, check.Domain)
		if err != nil {
			return time.Since(start), fmt.Errorf(ErrDNSLookupFailed, err)
		}
		results = addrs

	case "MX":
		mxRecords, err := net.LookupMX(check.Domain)
		if check.Host != "" {
			// Use custom resolver - need to use the resolver's context
			mxRecords, err = lookupMXWithResolver(ctx, resolver, check.Domain)
		}
		if err != nil {
			return time.Since(start), fmt.Errorf(ErrDNSLookupFailed, err)
		}
		for _, mx := range mxRecords {
			results = append(results, fmt.Sprintf("%s %d", mx.Host, mx.Pref))
		}

	case "TXT":
		txtRecords, err := resolver.LookupTXT(ctx, check.Domain)
		if err != nil {
			return time.Since(start), fmt.Errorf(ErrDNSLookupFailed, err)
		}
		results = txtRecords

	case "NS":
		nsRecords, err := net.LookupNS(check.Domain)
		if check.Host != "" {
			nsRecords, err = lookupNSWithResolver(ctx, resolver, check.Domain)
		}
		if err != nil {
			return time.Since(start), fmt.Errorf(ErrDNSLookupFailed, err)
		}
		for _, ns := range nsRecords {
			results = append(results, ns.Host)
		}

	case "CNAME":
		cname, err := resolver.LookupCNAME(ctx, check.Domain)
		if err != nil {
			return time.Since(start), fmt.Errorf(ErrDNSLookupFailed, err)
		}
		results = []string{cname}
	}

	check.Logger.Debugf("DNS %s lookup for %s: %v", recordType, check.Domain, results)

	// Check expected value if set
	if check.Expected != "" {
		found := false
		for _, r := range results {
			if strings.Contains(r, check.Expected) {
				found = true
				break
			}
		}
		if !found {
			return time.Since(start), fmt.Errorf(ErrDNSExpectedMissing, check.Expected, results)
		}
	}

	return time.Since(start), nil
}

// lookupMXWithResolver performs MX lookup using a custom resolver
func lookupMXWithResolver(ctx context.Context, resolver *net.Resolver, domain string) ([]*net.MX, error) {
	// The net.Resolver doesn't have a direct LookupMX method with context in older Go versions,
	// but LookupMX is available on the default resolver. For custom resolvers, we use LookupHost
	// as a workaround, but since Go 1.18+ net.Resolver has LookupMX.
	return resolver.LookupMX(ctx, domain)
}

// lookupNSWithResolver performs NS lookup using a custom resolver
func lookupNSWithResolver(ctx context.Context, resolver *net.Resolver, domain string) ([]*net.NS, error) {
	return resolver.LookupNS(ctx, domain)
}

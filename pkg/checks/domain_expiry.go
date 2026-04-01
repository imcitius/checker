package checks

import (
	"bufio"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ErrEmptyDomain          = "empty domain"
	ErrWhoisServerNotFound  = "no WHOIS server known for TLD: %s"
	ErrWhoisConnect         = "WHOIS connection error: %s"
	ErrWhoisRead            = "WHOIS read error: %s"
	ErrExpiryDateNotFound   = "could not parse expiry date from WHOIS response"
	ErrDomainExpiringSoon   = "domain %s expires in %d days (warning threshold: %d days)"
	InfoDomainDaysRemaining = "domain %s: %d days until expiry"
)

// whoisServers maps common TLDs to their WHOIS servers.
var whoisServers = map[string]string{
	"com":   "whois.verisign-grs.com",
	"net":   "whois.verisign-grs.com",
	"org":   "whois.pir.org",
	"io":    "whois.nic.io",
	"info":  "whois.afilias.net",
	"biz":   "whois.biz",
	"me":    "whois.nic.me",
	"co":    "whois.nic.co",
	"us":    "whois.nic.us",
	"uk":    "whois.nic.uk",
	"de":    "whois.denic.de",
	"eu":    "whois.eu",
	"fr":    "whois.nic.fr",
	"nl":    "whois.sidn.nl",
	"ru":    "whois.tcinet.ru",
	"au":    "whois.auda.org.au",
	"ca":    "whois.cira.ca",
	"in":    "whois.registry.in",
	"jp":    "whois.jprs.jp",
	"cn":    "whois.cnnic.cn",
	"br":    "whois.registro.br",
	"it":    "whois.nic.it",
	"pl":    "whois.dns.pl",
	"se":    "whois.iis.se",
	"ch":    "whois.nic.ch",
	"at":    "whois.nic.at",
	"be":    "whois.dns.be",
	"app":   "whois.nic.google",
	"dev":   "whois.nic.google",
	"xyz":   "whois.nic.xyz",
	"cloud": "whois.nic.cloud",
}

// expiryPatterns are regex patterns to extract expiry dates from WHOIS responses.
// Each pattern has an associated date layout for parsing.
var expiryPatterns = []struct {
	pattern *regexp.Regexp
	layouts []string
}{
	{
		// RFC3339: "Expiry Date: 2025-03-15T00:00:00Z" or "Registry Expiry Date: 2025-03-15T00:00:00Z"
		pattern: regexp.MustCompile(`(?i)(?:registry\s+)?expir(?:y|ation)\s+date:\s*(\d{4}-\d{2}-\d{2}T[\d:]+Z?)`),
		layouts: []string{time.RFC3339, "2006-01-02T15:04:05"},
	},
	{
		// ISO date: "Expiry Date: 2025-03-15" or "Registry Expiry Date: 2025-03-15"
		pattern: regexp.MustCompile(`(?i)(?:registry\s+)?expir(?:y|ation)\s+date:\s*(\d{4}-\d{2}-\d{2})`),
		layouts: []string{"2006-01-02"},
	},
	{
		// Day-Month-Year: "Expiration Date: 15-Mar-2025"
		pattern: regexp.MustCompile(`(?i)expir(?:y|ation)\s+date:\s*(\d{2}-[A-Za-z]{3}-\d{4})`),
		layouts: []string{"02-Jan-2006"},
	},
	{
		// "paid-till: 2025-03-15" (common in .ru)
		pattern: regexp.MustCompile(`(?i)paid-till:\s*(\d{4}-\d{2}-\d{2})`),
		layouts: []string{"2006-01-02"},
	},
	{
		// "Expiry date: 2025/03/15" (slash-separated)
		pattern: regexp.MustCompile(`(?i)expir(?:y|ation)\s+date:\s*(\d{4}/\d{2}/\d{2})`),
		layouts: []string{"2006/01/02"},
	},
	{
		// "Expiry Date: 15/03/2025" (DD/MM/YYYY)
		pattern: regexp.MustCompile(`(?i)expir(?:y|ation)\s+date:\s*(\d{2}/\d{2}/\d{4})`),
		layouts: []string{"02/01/2006"},
	},
	{
		// "[Expires on] 2025/03/15" (jprs style)
		pattern: regexp.MustCompile(`(?i)\[expires\s+on\]\s*(\d{4}/\d{2}/\d{2})`),
		layouts: []string{"2006/01/02"},
	},
}

// DomainExpiryCheck represents a domain expiry health check via WHOIS.
type DomainExpiryCheck struct {
	Domain          string
	ExpiryWarningDays int
	Timeout         string
	Logger          *logrus.Entry
}

// Run executes the domain expiry check.
func (check *DomainExpiryCheck) Run() (time.Duration, error) {
	start := time.Now()

	if check.Logger == nil {
		check.Logger = logrus.WithField("check", "domain_expiry")
	}

	if check.Domain == "" {
		return time.Since(start), fmt.Errorf(ErrEmptyDomain)
	}

	timeout, err := parseCheckTimeout(check.Timeout, 10*time.Second)
	if err != nil {
		return time.Since(start), err
	}

	warningDays := check.ExpiryWarningDays
	if warningDays <= 0 {
		warningDays = 30 // default warning threshold
	}

	// Look up WHOIS server for the domain's TLD
	tld := extractTLD(check.Domain)
	whoisServer, ok := whoisServers[tld]
	if !ok {
		return time.Since(start), fmt.Errorf(ErrWhoisServerNotFound, tld)
	}

	// Query WHOIS
	response, err := queryWhois(check.Domain, whoisServer, timeout)
	if err != nil {
		return time.Since(start), err
	}

	check.Logger.Debugf("WHOIS response for %s (%d bytes)", check.Domain, len(response))

	// Parse expiry date
	expiryDate, err := ParseExpiryDate(response)
	if err != nil {
		return time.Since(start), err
	}

	// Calculate days remaining
	daysRemaining := int(time.Until(expiryDate).Hours() / 24)

	if daysRemaining < warningDays {
		return time.Since(start), fmt.Errorf(ErrDomainExpiringSoon, check.Domain, daysRemaining, warningDays)
	}

	check.Logger.Infof(InfoDomainDaysRemaining, check.Domain, daysRemaining)
	return time.Since(start), nil
}

// extractTLD returns the top-level domain from a domain name.
// For example, "example.com" returns "com", "example.co.uk" returns "uk".
func extractTLD(domain string) string {
	parts := strings.Split(strings.TrimRight(domain, "."), ".")
	if len(parts) == 0 {
		return ""
	}
	return strings.ToLower(parts[len(parts)-1])
}

// queryWhois connects to a WHOIS server and queries for the given domain.
func queryWhois(domain, whoisServer string, timeout time.Duration) (string, error) {
	conn, err := net.DialTimeout("tcp", whoisServer+":43", timeout)
	if err != nil {
		return "", fmt.Errorf(ErrWhoisConnect, err)
	}
	defer conn.Close()

	// Set read/write deadlines
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return "", fmt.Errorf(ErrWhoisConnect, err)
	}

	// Send domain query
	_, err = fmt.Fprintf(conn, "%s\r\n", domain)
	if err != nil {
		return "", fmt.Errorf(ErrWhoisConnect, err)
	}

	// Read response
	var sb strings.Builder
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		sb.WriteString(scanner.Text())
		sb.WriteString("\n")
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf(ErrWhoisRead, err)
	}

	return sb.String(), nil
}

// ParseExpiryDate extracts and parses the expiry date from a WHOIS response string.
// It tries multiple common date formats.
func ParseExpiryDate(whoisResponse string) (time.Time, error) {
	for _, ep := range expiryPatterns {
		matches := ep.pattern.FindStringSubmatch(whoisResponse)
		if len(matches) < 2 {
			continue
		}
		dateStr := strings.TrimSpace(matches[1])
		for _, layout := range ep.layouts {
			t, err := time.Parse(layout, dateStr)
			if err == nil {
				return t, nil
			}
		}
	}
	return time.Time{}, fmt.Errorf(ErrExpiryDateNotFound)
}

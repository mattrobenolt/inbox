package links

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"go.withmatt.com/inbox/internal/config"
)

const (
	defaultMaxWorkers = 10
	defaultTimeout    = 5 * time.Second
	defaultDNSTimeout = 2 * time.Second
)

const (
	dnsModeSystem   = "system"
	dnsModeCustom   = "custom"
	dnsModeExplicit = "explicit"
)

var defaultDNSServers = []string{"1.1.1.1", "1.0.0.1"}

var urlRe = regexp.MustCompile(`https?://[^\s<>()\[\]"']+`)

type Resolver struct {
	domains     []string
	domainSet   map[string]struct{}
	domainsMu   sync.RWMutex
	denyDomains []string
	denySet     map[string]struct{}
	cache       *Cache
	client      *http.Client
	maxWorkers  int
	logf        func(string, ...any)
	dnsMode     string
	dnsServers  []string
}

func NewResolver(cfg config.LinkConfig, logf func(string, ...any)) *Resolver {
	cache, err := LoadCache()
	if err != nil {
		logf("link cache load error: %v", err)
		cache = &Cache{}
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	dnsMode := strings.ToLower(strings.TrimSpace(cfg.DNSMode))
	dnsServers := normalizeDNSServers(cfg.DNSServers)
	useCustomDNS := shouldUseCustomDNS(dnsMode, dnsServers)
	resolverDNSMode := dnsModeSystem
	var resolverDNSServers []string
	if dnsMode == dnsModeSystem && len(dnsServers) > 0 {
		logf("link resolver dns_mode=%q ignores dns_servers", cfg.DNSMode)
	}
	if useCustomDNS {
		if len(dnsServers) == 0 {
			dnsServers = defaultDNSServers
		}
		if dnsMode != "" && dnsMode != dnsModeCustom && dnsMode != dnsModeExplicit &&
			dnsMode != dnsModeSystem {
			logf("link resolver dns_mode=%q unrecognized; using dns_servers", cfg.DNSMode)
		}
		resolverDNSMode = dnsModeCustom
		resolverDNSServers = append([]string(nil), dnsServers...)
		dnsResolver := &net.Resolver{
			PreferGo: true,
			Dial:     dialDNS(dnsServers),
		}
		dialer := &net.Dialer{
			Timeout:  defaultTimeout,
			Resolver: dnsResolver,
		}
		transport.DialContext = dialer.DialContext
	} else if dnsMode != "" && dnsMode != dnsModeSystem {
		logf("link resolver dns_mode=%q unrecognized; using system resolver", cfg.DNSMode)
	}

	client := &http.Client{
		Timeout: defaultTimeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: transport,
	}

	resolver := &Resolver{
		cache:      cache,
		client:     client,
		maxWorkers: defaultMaxWorkers,
		logf:       logf,
		domainSet:  make(map[string]struct{}),
		denySet:    make(map[string]struct{}),
		dnsMode:    resolverDNSMode,
		dnsServers: resolverDNSServers,
	}
	logf("link resolver dns=%s servers=%v", resolver.dnsMode, resolver.dnsServers)
	resolver.addDenied(cfg.DoNotResolve)
	resolver.addDomains(cfg.UnwrapDomains)
	if cache != nil {
		learnedDomains, err := cache.ListDomains()
		if err != nil {
			logf("link cache domains error: %v", err)
		} else {
			resolver.addDomains(learnedDomains)
		}
	}
	if len(resolver.domains) == 0 && !cfg.AutoScan {
		return nil
	}
	return resolver
}

func (r *Resolver) ResolveText(ctx context.Context, text string) string {
	if r == nil || text == "" || !strings.Contains(text, "http") {
		return text
	}

	matches := extractURLMatches(text)
	if len(matches) == 0 {
		return text
	}

	resolved := make(map[string]string)

	for i := range matches {
		match := &matches[i]
		if match.trimmed == "" {
			continue
		}
		if !r.shouldResolve(match.trimmed) {
			continue
		}
		if r.cache != nil {
			if entry, ok := r.cache.Get(match.trimmed); ok {
				if !entry.NoChange && entry.Resolved != "" {
					resolved[match.trimmed] = entry.Resolved
				}
			}
		}
	}

	var b strings.Builder
	b.Grow(len(text))

	last := 0
	for _, match := range matches {
		b.WriteString(text[last:match.start])
		replacement := match.trimmed
		if resolvedURL, ok := resolved[match.trimmed]; ok && resolvedURL != "" {
			replacement = resolvedURL
		}
		b.WriteString(replacement)
		b.WriteString(match.suffix)
		last = match.end
	}
	b.WriteString(text[last:])

	return b.String()
}

type ScanMode int

const (
	ScanModeKnown ScanMode = iota
	ScanModeLearn
)

func (r *Resolver) ScanText(ctx context.Context, text string, mode ScanMode) {
	if r == nil || text == "" || !strings.Contains(text, "http") {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	matches := extractURLMatches(text)
	if len(matches) == 0 {
		return
	}

	toResolve := make(map[string]struct{})
	for _, match := range matches {
		if match.trimmed == "" {
			continue
		}
		if r.isDeniedURL(match.trimmed) {
			continue
		}
		if mode == ScanModeKnown && !r.shouldResolve(match.trimmed) {
			continue
		}
		if r.cache != nil {
			if _, ok := r.cache.Get(match.trimmed); ok {
				continue
			}
		}
		if _, ok := toResolve[match.trimmed]; ok {
			continue
		}
		toResolve[match.trimmed] = struct{}{}
	}
	if len(toResolve) == 0 {
		return
	}

	results := r.resolveURLs(ctx, toResolve)
	if r.cache != nil {
		if err := r.cache.SetMany(results); err != nil {
			r.logf("link cache save error: %v", err)
		}
	}

	if mode == ScanModeLearn {
		redirectDomains := r.redirectDomains(results)
		if len(redirectDomains) == 0 {
			return
		}
		if r.cache != nil {
			if err := r.cache.RecordDomains(redirectDomains); err != nil {
				r.logf("link cache domains save error: %v", err)
			}
		}
		r.addDomains(redirectDomains)
	}
}

type urlMatch struct {
	start   int
	end     int
	trimmed string
	suffix  string
}

func extractURLMatches(text string) []urlMatch {
	indices := urlRe.FindAllStringIndex(text, -1)
	if len(indices) == 0 {
		return nil
	}

	matches := make([]urlMatch, 0, len(indices))
	for _, index := range indices {
		raw := text[index[0]:index[1]]
		trimmed, suffix := trimURLSuffix(raw)
		if trimmed == "" {
			continue
		}
		matches = append(matches, urlMatch{
			start:   index[0],
			end:     index[1],
			trimmed: trimmed,
			suffix:  suffix,
		})
	}
	return matches
}

func trimURLSuffix(raw string) (string, string) {
	trimmed := raw
	suffix := ""
	for len(trimmed) > 0 {
		last := trimmed[len(trimmed)-1]
		switch last {
		case '.', ',', ';', ':', '!', '?', ')', ']', '}', '"', '\'':
			trimmed = trimmed[:len(trimmed)-1]
			suffix = string(last) + suffix
		default:
			return trimmed, suffix
		}
	}
	return raw, ""
}

func normalizeDomains(domains []string) []string {
	normalized := make([]string, 0, len(domains))
	for _, domain := range domains {
		normalizedDomain := normalizeDomain(domain)
		if normalizedDomain == "" {
			continue
		}
		normalized = append(normalized, normalizedDomain)
	}
	return normalized
}

func normalizeDomain(domain string) string {
	trimmed := strings.ToLower(strings.TrimSpace(domain))
	if trimmed == "" {
		return ""
	}
	if strings.Contains(trimmed, "://") {
		if parsed, err := url.Parse(trimmed); err == nil && parsed.Hostname() != "" {
			trimmed = parsed.Hostname()
		}
	}
	trimmed = strings.TrimPrefix(trimmed, ".")
	if idx := strings.Index(trimmed, "/"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, ":"); idx != -1 {
		trimmed = trimmed[:idx]
	}
	return trimmed
}

func NormalizeDomain(domain string) string {
	return normalizeDomain(domain)
}

func (r *Resolver) addDomains(domains []string) {
	if len(domains) == 0 {
		return
	}
	normalized := normalizeDomains(domains)
	if len(normalized) == 0 {
		return
	}
	r.domainsMu.Lock()
	defer r.domainsMu.Unlock()
	for _, domain := range normalized {
		if r.isDeniedHost(domain) {
			continue
		}
		if _, ok := r.domainSet[domain]; ok {
			continue
		}
		r.domainSet[domain] = struct{}{}
		r.domains = append(r.domains, domain)
	}
}

func (r *Resolver) addDenied(domains []string) {
	if len(domains) == 0 {
		return
	}
	normalized := normalizeDomains(domains)
	if len(normalized) == 0 {
		return
	}
	for _, domain := range normalized {
		if _, ok := r.denySet[domain]; ok {
			continue
		}
		r.denySet[domain] = struct{}{}
		r.denyDomains = append(r.denyDomains, domain)
	}
}

func (r *Resolver) isDeniedURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return false
	}
	return r.isDeniedHost(host)
}

func (r *Resolver) isDeniedHost(host string) bool {
	for _, domain := range r.denyDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func (r *Resolver) redirectDomains(results map[string]cacheEntry) []string {
	if len(results) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(results))
	domains := make([]string, 0, len(results))
	for rawURL, entry := range results {
		if entry.NoChange || entry.Resolved == "" {
			continue
		}
		domain := normalizeDomain(rawURL)
		if domain == "" {
			continue
		}
		if r.isDeniedHost(domain) {
			continue
		}
		if _, ok := seen[domain]; ok {
			continue
		}
		seen[domain] = struct{}{}
		domains = append(domains, domain)
	}
	return domains
}

func (r *Resolver) shouldResolve(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return false
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return false
	}
	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return false
	}
	if r.isDeniedHost(host) {
		return false
	}
	r.domainsMu.RLock()
	defer r.domainsMu.RUnlock()
	for _, domain := range r.domains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return true
		}
	}
	return false
}

func (r *Resolver) resolveURLs(
	ctx context.Context,
	urls map[string]struct{},
) map[string]cacheEntry {
	results := make(map[string]cacheEntry, len(urls))
	if len(urls) == 0 {
		return results
	}

	g, groupCtx := errgroup.WithContext(ctx)
	if r.maxWorkers > 0 {
		g.SetLimit(r.maxWorkers)
	}

	var mu sync.Mutex
	for rawURL := range urls {
		urlToResolve := rawURL
		g.Go(func() error {
			entry, err := r.resolveURL(groupCtx, urlToResolve)
			if err != nil {
				return nil
			}
			mu.Lock()
			results[urlToResolve] = entry
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()
	return results
}

func (r *Resolver) resolveURL(ctx context.Context, rawURL string) (cacheEntry, error) {
	r.logf("link resolve attempt url=%q", rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		r.logf(
			"link resolve error url=%q err=%v dns=%s servers=%v",
			rawURL,
			err,
			r.dnsMode,
			r.dnsServers,
		)

		return cacheEntry{}, err
	}
	req.Header.Set("User-Agent", "inbox-link-resolver/1")

	resp, err := r.client.Do(req)
	if err != nil {
		r.logf(
			"link resolve error url=%q err=%v dns=%s servers=%v",
			rawURL,
			err,
			r.dnsMode,
			r.dnsServers,
		)

		return cacheEntry{}, err
	}
	defer resp.Body.Close()
	if _, err := io.Copy(io.Discard, resp.Body); err != nil {
		r.logf(
			"link resolve read error url=%q err=%v dns=%s servers=%v",
			rawURL,
			err,
			r.dnsMode,
			r.dnsServers,
		)
		return cacheEntry{}, err
	}

	entry := cacheEntry{
		CheckedAt: time.Now().UTC(),
		NoChange:  true,
	}

	if isRedirectStatus(resp.StatusCode) {
		if location := resp.Header.Get("Location"); location != "" {
			resolved := resolveLocation(rawURL, location)
			if resolved != "" && resolved != rawURL {
				entry.Resolved = resolved
				entry.NoChange = false
				r.logf(
					"link resolve success url=%q status=%d resolved=%q",
					rawURL,
					resp.StatusCode,
					resolved,
				)
			} else {
				r.logf(
					"link resolve no-change url=%q status=%d location=%q",
					rawURL,
					resp.StatusCode,
					location,
				)
			}
		} else {
			r.logf("link resolve empty-location url=%q status=%d", rawURL, resp.StatusCode)
		}
	} else {
		r.logf("link resolve no-redirect url=%q status=%d", rawURL, resp.StatusCode)
	}

	return entry, nil
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

func resolveLocation(baseURL, location string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	target, err := url.Parse(location)
	if err != nil {
		return ""
	}
	return base.ResolveReference(target).String()
}

func shouldUseCustomDNS(mode string, servers []string) bool {
	switch mode {
	case "", dnsModeCustom, dnsModeExplicit:
		return true
	case dnsModeSystem:
		return false
	default:
		return len(servers) > 0
	}
}

func normalizeDNSServers(servers []string) []string {
	normalized := make([]string, 0, len(servers))
	for _, server := range servers {
		trimmed := strings.TrimSpace(server)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "://") {
			if parsed, err := url.Parse(trimmed); err == nil && parsed.Hostname() != "" {
				trimmed = parsed.Hostname()
			}
		}
		trimmed = strings.TrimPrefix(trimmed, "[")
		trimmed = strings.TrimSuffix(trimmed, "]")
		if host, _, err := net.SplitHostPort(trimmed); err == nil && host != "" {
			trimmed = host
		}
		if idx := strings.Index(trimmed, "/"); idx != -1 {
			trimmed = trimmed[:idx]
		}
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	return normalized
}

func dialDNS(servers []string) func(context.Context, string, string) (net.Conn, error) {
	var index uint32
	return func(ctx context.Context, network, _ string) (net.Conn, error) {
		start := int(atomic.AddUint32(&index, 1)-1) % len(servers)
		dialer := net.Dialer{Timeout: defaultDNSTimeout}
		var lastErr error
		for i := range servers {
			server := servers[(start+i)%len(servers)]
			conn, err := dialer.DialContext(ctx, network, net.JoinHostPort(server, "53"))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		return nil, lastErr
	}
}

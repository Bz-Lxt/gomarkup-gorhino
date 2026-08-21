package validate

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"gorhino/internal/shared/model"
)

var (
	ErrInvalid        = errors.New("validation failed")
	blockedMetadata   = []string{"metadata.google.internal", "metadata.goog", "169.254.169.254"}
	linkLocal         = mustCIDR("169.254.0.0/16")
	linkLocal6        = mustCIDR("fe80::/10")
	v4MappedLinkLocal = mustCIDR("::ffff:169.254.0.0/112")
)

func mustCIDR(s string) *net.IPNet {
	_, n, err := net.ParseCIDR(s)
	if err != nil {
		panic(err)
	}
	return n
}

type Issue struct {
	Field   string
	Message string
}

func (i Issue) Error() string {
	if i.Field == "" {
		return i.Message
	}
	return i.Field + ": " + i.Message
}

func Join(issues []Issue) error {
	if len(issues) == 0 {
		return nil
	}
	parts := make([]string, 0, len(issues))
	for _, is := range issues {
		parts = append(parts, is.Error())
	}
	return fmt.Errorf("%w: %s", ErrInvalid, strings.Join(parts, "; "))
}

func TaskSpec(spec model.TaskSpec) []Issue {
	var issues []Issue
	method := strings.ToUpper(strings.TrimSpace(spec.Method))
	switch method {
	case "GET", "POST", "PUT", "DELETE", "PATCH", "HEAD":
	default:
		issues = append(issues, Issue{"method", "must be GET/POST/PUT/DELETE/PATCH/HEAD"})
	}
	if strings.TrimSpace(spec.URL) == "" {
		issues = append(issues, Issue{"url", "required"})
	} else if utf8.RuneCountInString(spec.URL) > model.MaxURLLen {
		issues = append(issues, Issue{"url", "too long"})
	}
	if spec.VU < model.MinVU || spec.VU > model.MaxVU {
		issues = append(issues, Issue{"vu", fmt.Sprintf("must be %d..%d", model.MinVU, model.MaxVU)})
	}
	if spec.DurationSec < model.MinDuration || spec.DurationSec > model.MaxDuration {
		issues = append(issues, Issue{"duration_sec", fmt.Sprintf("must be %d..%d", model.MinDuration, model.MaxDuration)})
	}
	if spec.QPS < 0 || spec.QPS > model.MaxQPS {
		issues = append(issues, Issue{"qps", fmt.Sprintf("must be 0..%d", model.MaxQPS)})
	}
	tag := strings.TrimSpace(spec.VersionTag)
	if tag == "" {
		issues = append(issues, Issue{"version_tag", "required"})
	} else if utf8.RuneCountInString(tag) > model.MaxTagLen {
		issues = append(issues, Issue{"version_tag", "too long"})
	}
	if spec.Headers != nil {
		if len(spec.Headers) > model.MaxHeaderPairs {
			issues = append(issues, Issue{"headers", "too many pairs"})
		}
		n := 0
		for k, v := range spec.Headers {
			if strings.TrimSpace(k) == "" {
				issues = append(issues, Issue{"headers", "empty key"})
			}
			n += len(k) + len(v)
		}
		if n > model.MaxHeaderBytes {
			issues = append(issues, Issue{"headers", "payload too large"})
		}
	}
	if len(spec.Body) > model.MaxBodyBytes {
		issues = append(issues, Issue{"body", "exceeds 1MB"})
	}
	return issues
}

func NormalizeSpec(spec model.TaskSpec) model.TaskSpec {
	spec.Method = strings.ToUpper(strings.TrimSpace(spec.Method))
	spec.URL = strings.TrimSpace(spec.URL)
	spec.VersionTag = strings.TrimSpace(spec.VersionTag)
	if spec.Headers == nil {
		spec.Headers = map[string]string{}
	}
	return spec
}

func CheckURL(raw string, patterns []string) error {
	return CheckURLContext(context.Background(), raw, patterns)
}

func CheckURLContext(ctx context.Context, raw string, patterns []string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return Issue{"url", "not a URL"}
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return Issue{"url", "only http/https allowed"}
	}
	if u.User != nil {
		return Issue{"url", "userinfo not allowed"}
	}
	if u.Host == "" {
		return Issue{"url", "host required"}
	}
	host := strings.ToLower(u.Hostname())
	for _, bad := range blockedMetadata {
		if host == bad {
			return Issue{"url", "metadata host blocked"}
		}
	}
	if !MatchWhitelist(raw, patterns) {
		return Issue{"url", "not on whitelist"}
	}
	if net.ParseIP(host) != nil {
		if blockedIP(net.ParseIP(host)) {
			return Issue{"url", "destination IP is SSRF-sensitive"}
		}
		return nil
	}
	rctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIPAddr(rctx, host)
	if err != nil {
		return Issue{"url", "dns lookup failed"}
	}
	if len(ips) == 0 {
		return Issue{"url", "dns returned no addresses"}
	}
	for _, ipa := range ips {
		if blockedIP(ipa.IP) {
			return Issue{"url", "resolved IP is SSRF-sensitive"}
		}
	}
	return nil
}

func MatchWhitelist(raw string, patterns []string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	for _, p := range patterns {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		pl := strings.ToLower(p)
		if strings.HasPrefix(pl, "http://") || strings.HasPrefix(pl, "https://") {
			if strings.HasPrefix(strings.ToLower(raw), pl) {
				return true
			}
			continue
		}
		if strings.Contains(pl, "/") {
			continue
		}
		if strings.Contains(pl, ":") {
			if strings.ToLower(u.Host) == pl || host+":"+port == pl {
				return true
			}
			continue
		}
		if host == pl {
			return true
		}
	}
	return false
}

func blockedIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsUnspecified() || ip.IsMulticast() {
		return true
	}
	if linkLocal.Contains(ip) || linkLocal6.Contains(ip) || v4MappedLinkLocal.Contains(ip) {
		return true
	}
	return false
}

func Patterns(in []string) ([]string, error) {
	out := make([]string, 0, len(in))
	seen := map[string]struct{}{}
	for _, p := range in {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if len(p) > 512 {
			return nil, Issue{"patterns", "pattern too long"}
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	if len(out) == 0 {
		return nil, Issue{"patterns", "at least one pattern required"}
	}
	return out, nil
}

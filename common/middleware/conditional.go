package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// PathSkipRule describes a request that should bypass a global middleware.
// Empty Methods matches all HTTP methods. PathExact, when set, takes precedence
// over PathPrefix.
type PathSkipRule struct {
	Methods    []string
	PathPrefix string
	PathExact  string
}

// MatchesSkipRule reports whether the current request matches any skip rule.
func MatchesSkipRule(c *gin.Context, rules []PathSkipRule) bool {
	if c == nil || c.Request == nil || len(rules) == 0 {
		return false
	}

	method := c.Request.Method
	path := c.Request.URL.Path

	for _, rule := range rules {
		if !methodMatches(method, rule.Methods) {
			continue
		}
		if rule.PathExact != "" {
			if path == rule.PathExact {
				return true
			}
			continue
		}
		if rule.PathPrefix != "" && strings.HasPrefix(path, rule.PathPrefix) {
			return true
		}
	}
	return false
}

func methodMatches(method string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, m := range allowed {
		if strings.EqualFold(m, method) {
			return true
		}
	}
	return false
}

// UnlessSkipped runs mw for requests that do not match rules.
// Matching requests run fallback instead; if fallback is nil they proceed with c.Next().
func UnlessSkipped(
	rules []PathSkipRule,
	mw gin.HandlerFunc,
	fallback gin.HandlerFunc,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		if MatchesSkipRule(c, rules) {
			if fallback != nil {
				fallback(c)
				return
			}
			c.Next()
			return
		}
		mw(c)
	}
}

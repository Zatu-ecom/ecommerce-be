package payment_test

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestOrchestratorsHaveNoProviderLeaks is the executable Open/Closed boundary
// for the payment gateway platform (010-payment-gateway-platform, US5).
//
// Base services (payment/service/*.go, top level only) must never name a
// provider: no Razorpay event constants, no Razorpay credential helpers, and
// no hardcoded gateway codes. Provider specifics live in
// payment/service/payment_gateway/{code}/; wiring lives in the payment
// factory singleton. Adding Stripe later must not edit orchestrators — this
// test fails the build if anyone reintroduces a leak.
//
// Comments are stripped before matching so design docs in comments never trip
// the gate; only real code references count.
func TestOrchestratorsHaveNoProviderLeaks(t *testing.T) {
	root := repoRoot(t)
	files, err := filepath.Glob(filepath.Join(root, "payment", "service", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no payment/service/*.go files found; repo root resolution broken?")
	}

	forbidden := []string{
		"RAZORPAY_EVENT",
		"ParseRazorpay",
		"DecryptSensitive",
		"GATEWAY_CODE_RAZORPAY",
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		code := stripGoComments(string(content))
		for _, token := range forbidden {
			for i, line := range strings.Split(code, "\n") {
				if strings.Contains(line, token) {
					t.Errorf("%s:%d: orchestrator references provider token %q",
						filepath.Base(file), i+1, token)
				}
			}
		}
	}
}

// repoRoot resolves the repository root from this test file's location
// (test/payment/ → two levels up). It does not depend on the working
// directory the test binary runs from.
func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(file)))
}

// stripGoComments removes // line comments and /* */ block comments,
// respecting double-quoted, backquoted, and rune literals so comment markers
// inside strings survive.
func stripGoComments(src string) string {
	var out strings.Builder
	out.Grow(len(src))

	const (
		code = iota
		lineComment
		blockComment
		dqString
		rawString
		runeLit
	)
	state := code

	for i := 0; i < len(src); i++ {
		c := src[i]
		next := byte(0)
		if i+1 < len(src) {
			next = src[i+1]
		}

		switch state {
		case code:
			switch {
			case c == '/' && next == '/':
				state = lineComment
				i++
			case c == '/' && next == '*':
				state = blockComment
				i++
				out.WriteByte('\n') // keep line numbers stable
			case c == '"':
				state = dqString
				out.WriteByte(c)
			case c == '`':
				state = rawString
				out.WriteByte(c)
			case c == '\'':
				state = runeLit
				out.WriteByte(c)
			default:
				out.WriteByte(c)
			}
		case lineComment:
			if c == '\n' {
				state = code
				out.WriteByte(c)
			}
		case blockComment:
			if c == '*' && next == '/' {
				state = code
				i++
			} else if c == '\n' {
				out.WriteByte(c)
			} else {
				out.WriteByte(' ')
			}
		case dqString:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
			} else if c == '"' {
				state = code
			}
		case rawString:
			out.WriteByte(c)
			if c == '`' {
				state = code
			}
		case runeLit:
			out.WriteByte(c)
			if c == '\\' && i+1 < len(src) {
				i++
				out.WriteByte(src[i])
			} else if c == '\'' {
				state = code
			}
		}
	}
	return out.String()
}

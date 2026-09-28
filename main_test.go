package main

import (
	"errors"
	"regexp"
	"testing"

	"github.com/opentelekomcloud/gophertelekomcloud/openstack/elb/v3/certificates"
)

func TestKeepCertPattern(t *testing.T) {
	t.Setenv(EnvKeepCertRegex, "")
	if got := keepCertPattern(); got != "dummy" {
		t.Fatalf("unset pattern: got %q, want dummy", got)
	}

	t.Setenv(EnvKeepCertRegex, "^prod-")
	pattern, err := regexp.Compile(keepCertPattern())
	if err != nil {
		t.Fatal(err)
	}
	if !pattern.MatchString("prod-certificate") || pattern.MatchString("test-certificate") {
		t.Fatal("pattern did not select only matching certificate names")
	}
}

func TestShouldDeleteOldCertificate(t *testing.T) {
	pattern := regexp.MustCompile("dummy")
	cases := []struct {
		name      string
		cert      *certificates.Certificate
		lookupErr error
		want      bool
	}{
		{name: "lookup failed", lookupErr: errors.New("unavailable")},
		{name: "missing certificate"},
		{name: "protected", cert: &certificates.Certificate{Name: "k8s-dummy"}},
		{name: "unprotected", cert: &certificates.Certificate{Name: "k8s-production"}, want: true},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			if got := shouldDeleteOldCertificate(testCase.cert, testCase.lookupErr, pattern); got != testCase.want {
				t.Fatalf("shouldDeleteOldCertificate() = %t, want %t", got, testCase.want)
			}
		})
	}
}

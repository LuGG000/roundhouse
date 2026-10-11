package main

import "testing"

func TestModuleVersionAcceptsOnlyReleaseTags(t *testing.T) {
	for in, want := range map[string]string{
		"v0.4.1":                               "0.4.1",
		"v1.20.3":                              "1.20.3",
		"(devel)":                              "dev",
		"":                                     "dev",
		"v0.4.2-0.20261011101010-abcdef123456": "dev",
		"v0.4.1+dirty":                         "dev",
		"v0.5.0-rc.1":                          "dev",
		"0.4.1":                                "dev",
	} {
		if got := moduleVersion(in); got != want {
			t.Errorf("moduleVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

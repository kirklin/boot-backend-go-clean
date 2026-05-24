package geoip

import (
	"testing"

	"github.com/kirklin/boot-backend-go-clean/internal/domain/gateway"
)

func TestParseRegion(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect gateway.GeoLocation
	}{
		{
			name:  "typical Chinese IP",
			input: "中国|0|北京|北京市|电信",
			expect: gateway.GeoLocation{
				Country:  "中国",
				Region:   "",
				Province: "北京",
				City:     "北京市",
				ISP:      "电信",
			},
		},
		{
			name:  "all fields populated",
			input: "中国|华北|山东|济南|联通",
			expect: gateway.GeoLocation{
				Country:  "中国",
				Region:   "华北",
				Province: "山东",
				City:     "济南",
				ISP:      "联通",
			},
		},
		{
			name:  "foreign IP with zeros",
			input: "美国|0|加利福尼亚|旧金山|0",
			expect: gateway.GeoLocation{
				Country:  "美国",
				Region:   "",
				Province: "加利福尼亚",
				City:     "旧金山",
				ISP:      "",
			},
		},
		{
			name:  "all zeros",
			input: "0|0|0|0|0",
			expect: gateway.GeoLocation{
				Country:  "",
				Region:   "",
				Province: "",
				City:     "",
				ISP:      "",
			},
		},
		{
			name:   "too few fields",
			input:  "中国|北京",
			expect: gateway.GeoLocation{},
		},
		{
			name:   "empty string",
			input:  "",
			expect: gateway.GeoLocation{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseRegion(tt.input)
			if got != tt.expect {
				t.Errorf("parseRegion(%q)\n  got  = %+v\n  want = %+v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestClean(t *testing.T) {
	if clean("0") != "" {
		t.Error("clean(\"0\") should return empty string")
	}
	if clean("北京") != "北京" {
		t.Error("clean(\"北京\") should return 北京")
	}
	if clean("") != "" {
		t.Error("clean(\"\") should return empty string")
	}
}

func TestResolver_Resolve_NilSafety(t *testing.T) {
	// A nil resolver should return empty GeoLocation without panicking.
	var r *Resolver
	got := r.Resolve("1.2.3.4")
	if got != (gateway.GeoLocation{}) {
		t.Errorf("nil resolver should return empty GeoLocation, got %+v", got)
	}
}

func TestResolver_Close_NilSafety(t *testing.T) {
	// Close on nil resolver should not panic.
	var r *Resolver
	if err := r.Close(); err != nil {
		t.Errorf("nil resolver Close() should return nil, got %v", err)
	}
}

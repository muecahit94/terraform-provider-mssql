// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import "testing"

func args(step configurationStep) string {
	if len(step.Args) == 0 {
		return ""
	}
	return step.Args[0].(string)
}

func TestPlanConfigurationChange(t *testing.T) {
	basic := ServerConfiguration{Name: "clr enabled", Minimum: 0, Maximum: 1, IsDynamic: true}
	advanced := ServerConfiguration{Name: "max degree of parallelism", Minimum: 0, Maximum: 32767, IsDynamic: true, IsAdvanced: true}

	t.Run("a basic option is one sp_configure and RECONFIGURE", func(t *testing.T) {
		steps, err := planConfigurationChange(basic, 1, false)
		if err != nil || len(steps) != 2 {
			t.Fatalf("steps = %v, err = %v", steps, err)
		}
		if steps[0].Statement != spConfigure || args(steps[0]) != "clr enabled" || steps[0].Args[1].(int64) != 1 {
			t.Errorf("first step = %+v", steps[0])
		}
		if steps[1].Statement != reconfigure {
			t.Errorf("second step = %+v", steps[1])
		}
	})

	t.Run("an advanced option switches show advanced options on and back off", func(t *testing.T) {
		steps, err := planConfigurationChange(advanced, 4, false)
		if err != nil || len(steps) != 6 {
			t.Fatalf("steps = %v, err = %v", steps, err)
		}
		want := []string{"show advanced options", "", "max degree of parallelism", "", "show advanced options", ""}
		for i, w := range want {
			if args(steps[i]) != w {
				t.Errorf("step %d arg = %q, want %q", i, args(steps[i]), w)
			}
		}
		if steps[0].Args[1].(int64) != 1 || steps[4].Args[1].(int64) != 0 {
			t.Error("show advanced options must go 1 then back to 0")
		}
	})

	t.Run("show advanced options already on stays on", func(t *testing.T) {
		steps, err := planConfigurationChange(advanced, 4, true)
		if err != nil || len(steps) != 2 {
			t.Fatalf("steps = %v, err = %v", steps, err)
		}
	})

	t.Run("show advanced options itself is a normal option", func(t *testing.T) {
		show := ServerConfiguration{Name: "show advanced options", Minimum: 0, Maximum: 1, IsDynamic: true}
		steps, err := planConfigurationChange(show, 0, false)
		if err != nil || len(steps) != 2 {
			t.Fatalf("steps = %v, err = %v", steps, err)
		}
	})

	t.Run("a value outside the range is rejected before anything runs", func(t *testing.T) {
		for _, v := range []int64{-1, 2, 99} {
			if steps, err := planConfigurationChange(basic, v, false); err == nil || steps != nil {
				t.Errorf("value %d must be rejected, got %v", v, steps)
			}
		}
	})
}

func TestRestartRequired(t *testing.T) {
	tests := []struct {
		name string
		cfg  ServerConfiguration
		want bool
	}{
		{"dynamic, applied", ServerConfiguration{IsDynamic: true, Value: 1, ValueInUse: 1}, false},
		{"dynamic, pending RECONFIGURE is not a restart", ServerConfiguration{IsDynamic: true, Value: 2, ValueInUse: 1}, false},
		{"not dynamic, applied", ServerConfiguration{Value: 1, ValueInUse: 1}, false},
		{"not dynamic, waiting for a restart", ServerConfiguration{Value: 2, ValueInUse: 1}, true},
	}
	for _, tt := range tests {
		if got := tt.cfg.RestartRequired(); got != tt.want {
			t.Errorf("%s: RestartRequired() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

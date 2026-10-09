// Copyright (c) 2024 muecahit94
// SPDX-License-Identifier: MIT

package mssql

import "testing"

func TestAddJobStepArgs(t *testing.T) {
	step := AgentJobStep{Name: "load", Subsystem: "TSQL", Command: "SELECT 1", DatabaseName: "app", OnSuccessAction: 3, OnFailAction: 2, RetryAttempts: 2, RetryInterval: 5}
	got := addJobStepArgs("nightly", 2, step)
	if len(got) != 12 {
		t.Fatalf("the arguments must match the 12 parameters of addJobStepSQL, got %d", len(got))
	}
	if got[0] != "nightly" || got[1] != 2 || got[2] != "load" || got[3] != "TSQL" || got[4] != "SELECT 1" || got[5] != "app" {
		t.Errorf("addJobStepArgs() = %v", got)
	}

	// A step without a database leaves it to SQL Server (NULL): an empty string would name a database.
	if got := addJobStepArgs("j", 1, AgentJobStep{Name: "s", Subsystem: "CmdExec", Command: "dir"}); got[5] != nil {
		t.Errorf("an empty database name must be NULL, got %v", got[5])
	}
}

func TestAddJobScheduleArgs(t *testing.T) {
	s := AgentJobSchedule{Name: "daily", Enabled: true, FreqType: 4, FreqInterval: 1, FreqSubdayType: 1, ActiveStartDate: 20260101, ActiveEndDate: 99991231, ActiveStartTime: 20000, ActiveEndTime: 235959}
	got := addJobScheduleArgs("nightly", s)
	if len(got) != 13 {
		t.Fatalf("the arguments must match the 13 parameters of addJobScheduleSQL, got %d", len(got))
	}
	if got[0] != "nightly" || got[1] != "daily" || got[2] != 1 || got[3] != 4 || got[9] != 20260101 || got[11] != 20000 {
		t.Errorf("addJobScheduleArgs() = %v", got)
	}
	s.Enabled = false
	if got := addJobScheduleArgs("nightly", s); got[2] != 0 {
		t.Errorf("a disabled schedule must give 0, got %v", got[2])
	}
}

func TestNullIfEmptyString(t *testing.T) {
	if nullIfEmptyString("") != nil {
		t.Error("an empty string must become NULL")
	}
	if nullIfEmptyString("x") != "x" {
		t.Error("a value must stay")
	}
}

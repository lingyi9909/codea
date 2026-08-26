package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"codea/tui/internal/doctor"
)

type fakeDoctorRunner struct{}

func (fakeDoctorRunner) RunWithProgress(_ context.Context, onStart func(string, doctor.Category), onResult func(doctor.Result)) doctor.Report {
	onStart("企业 Agent", doctor.CategoryConnection)
	res := doctor.Result{Name: "企业 Agent", Category: doctor.CategoryConnection, Status: doctor.Pass, Detail: "ok"}
	if onResult != nil {
		onResult(res)
	}
	return doctor.Report{Results: []doctor.Result{res}}
}

func TestRunDoctorServicePrintsProgressBeforeCompletion(t *testing.T) {
	var out bytes.Buffer
	report := runDoctorService(context.Background(), fakeDoctorRunner{}, &out)
	if len(report.Results) != 1 {
		t.Fatalf("results = %d", len(report.Results))
	}
	text := out.String()
	if !strings.Contains(text, "正在检查：企业 Agent") {
		t.Fatalf("missing live progress output: %q", text)
	}
}

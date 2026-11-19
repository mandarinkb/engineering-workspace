package workflow_test

import (
	"testing"

	"bop/internal/domain/bot"
	"bop/internal/domain/workflow"
)

func TestStepClone_DoesNotShareK8sJob(t *testing.T) {
	orig := workflow.Step{
		BotName: "b",
		Config:  bot.Config{"url": "a"},
		K8sJob: &workflow.K8sJobSpec{
			Image:   "img",
			Command: []string{"run"},
			Args:    []string{"--x"},
			Env:     map[string]string{"K": "v"},
		},
	}

	cp := orig.Clone()
	cp.Config["url"] = "changed"
	cp.K8sJob.Image = "changed"
	cp.K8sJob.Command[0] = "changed"
	cp.K8sJob.Args[0] = "changed"
	cp.K8sJob.Env["K"] = "changed"

	if orig.Config["url"] != "a" || orig.K8sJob.Image != "img" || orig.K8sJob.Command[0] != "run" ||
		orig.K8sJob.Args[0] != "--x" || orig.K8sJob.Env["K"] != "v" {
		t.Errorf("original step changed through clone: %+v %+v", orig, *orig.K8sJob)
	}
}

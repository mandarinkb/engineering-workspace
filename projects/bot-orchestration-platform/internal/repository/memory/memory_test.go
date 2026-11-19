package memory_test

import (
	"context"
	"testing"
	"time"

	"bop/internal/domain/bot"
	"bop/internal/domain/job"
	"bop/internal/domain/schedule"
	"bop/internal/domain/workflow"
	"bop/internal/repository/memory"
)

// test ชุดนี้ยืนยันว่า repository copy ข้อมูลเข้า-ออกแบบลึก (รวม map ข้างใน) ผู้เรียกที่แก้
// ค่าที่ส่งเข้าไปหรือได้คืนมา ต้องไม่ไปเปลี่ยนข้อมูลที่ repository เก็บไว้

func TestJobRepository_CreateKeepsOwnConfig(t *testing.T) {
	repo := memory.NewJobRepository()
	ctx := context.Background()
	j := &job.Job{ID: "j1", Config: map[string]string{"url": "a"}, CreatedAt: time.Now()}
	if err := repo.Create(ctx, j); err != nil {
		t.Fatalf("Create: %v", err)
	}

	j.Config["url"] = "changed"

	got, err := repo.Get(ctx, "j1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Config["url"] != "a" {
		t.Errorf("stored Config changed through caller's map: url = %q, want a", got.Config["url"])
	}
}

func TestJobRepository_GetReturnsOwnConfig(t *testing.T) {
	repo := memory.NewJobRepository()
	ctx := context.Background()
	if err := repo.Create(ctx, &job.Job{ID: "j1", Config: map[string]string{"url": "a"}}); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, _ := repo.Get(ctx, "j1")
	got.Config["url"] = "changed"
	list, _ := repo.List(ctx)
	list[0].Config["url"] = "changed"

	again, _ := repo.Get(ctx, "j1")
	if again.Config["url"] != "a" {
		t.Errorf("stored Config changed through returned map: url = %q, want a", again.Config["url"])
	}
}

func TestScheduleRepository_KeepsOwnStepConfig(t *testing.T) {
	repo := memory.NewScheduleRepository()
	ctx := context.Background()
	d := &schedule.Definition{
		ID:    "s1",
		Steps: []workflow.Step{{BotName: "b", Config: bot.Config{"url": "a"}}},
	}
	if err := repo.Create(ctx, d); err != nil {
		t.Fatalf("Create: %v", err)
	}

	d.Steps[0].Config["url"] = "changed"
	got, _ := repo.Get(ctx, "s1")
	got.Steps[0].Config["url"] = "changed"

	again, _ := repo.Get(ctx, "s1")
	if again.Steps[0].Config["url"] != "a" {
		t.Errorf("stored step Config changed: url = %q, want a", again.Steps[0].Config["url"])
	}
}

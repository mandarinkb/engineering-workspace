// Package repository_test รัน test ชุดเดียวกัน (contract ของ job.Repository) กับ implementation
// ทุกตัว เพื่อยืนยันว่าสลับ memory กับ postgres ใน cmd/bop ได้โดยพฤติกรรมไม่เปลี่ยน
//
// ส่วนของ PostgreSQL รันเฉพาะเมื่อตั้ง BOP_TEST_DATABASE_URL ไว้ (ชี้ไปที่ database ทิ้งได้
// เพราะ test ล้างตารางทุกครั้ง) ไม่ตั้ง = skip เพื่อให้ `go test ./...` บนเครื่องที่ไม่มี
// PostgreSQL ยังผ่าน
package repository_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"bop/internal/domain/job"
	"bop/internal/repository/memory"
	"bop/internal/repository/postgres"
)

func TestJobRepositoryContract_Memory(t *testing.T) {
	runJobRepositoryContract(t, func(*testing.T) job.Repository {
		return memory.NewJobRepository()
	})
}

func TestJobRepositoryContract_Postgres(t *testing.T) {
	url := testDatabaseURL(t)
	runJobRepositoryContract(t, func(t *testing.T) job.Repository {
		return postgres.NewJobRepository(openCleanDatabase(t, url))
	})
}

// TestPostgresJobRepository_SurvivesReconnect จำลองการ restart ของ bop-api: เขียน Job ผ่าน
// connection pool ชุดหนึ่ง ปิดทิ้ง แล้วเปิดชุดใหม่ (รวมรัน migration ซ้ำแบบตอน start) ต้องยัง
// อ่าน Job และสถานะล่าสุดได้
func TestPostgresJobRepository_SurvivesReconnect(t *testing.T) {
	url := testDatabaseURL(t)
	ctx := context.Background()

	before := openCleanDatabase(t, url)
	j := &job.Job{ID: "j-restart", BotName: "b", Status: job.StatusPending, CreatedAt: time.Now()}
	repo := postgres.NewJobRepository(before)
	if err := repo.Create(ctx, j); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := j.Start(time.Now()); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := j.Succeed("ok", time.Now()); err != nil {
		t.Fatalf("Succeed: %v", err)
	}
	if err := repo.Update(ctx, j); err != nil {
		t.Fatalf("Update: %v", err)
	}
	before.Close()

	if err := postgres.Migrate(url); err != nil {
		t.Fatalf("Migrate again: %v", err)
	}
	after, err := postgres.Connect(ctx, url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(after.Close)

	got, err := postgres.NewJobRepository(after).Get(ctx, "j-restart")
	if err != nil {
		t.Fatalf("Get after reconnect: %v", err)
	}
	if got.Status != job.StatusSucceeded || got.Summary != "ok" {
		t.Errorf("after reconnect: status=%q summary=%q, want succeeded/ok", got.Status, got.Summary)
	}
}

func runJobRepositoryContract(t *testing.T, newRepo func(*testing.T) job.Repository) {
	t.Run("Get returns every field that Create stored", func(t *testing.T) {
		repo := newRepo(t)
		ctx := context.Background()
		created := time.Date(2026, 11, 1, 10, 0, 0, 123456000, time.UTC)
		want := &job.Job{
			ID:             "j1",
			BotName:        "http-status-checker",
			Config:         map[string]string{"url": "https://example.com"},
			TimeoutSeconds: 45,
			Status:         job.StatusPending,
			CreatedAt:      created,
		}
		if err := repo.Create(ctx, want); err != nil {
			t.Fatalf("Create: %v", err)
		}

		got, err := repo.Get(ctx, "j1")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		assertJobEqual(t, got, want)
	})

	t.Run("Create rejects a duplicate ID", func(t *testing.T) {
		repo := newRepo(t)
		ctx := context.Background()
		j := &job.Job{ID: "dup", BotName: "b", Status: job.StatusPending, CreatedAt: time.Now()}
		if err := repo.Create(ctx, j); err != nil {
			t.Fatalf("first Create: %v", err)
		}
		if err := repo.Create(ctx, j); err == nil {
			t.Error("second Create with the same ID: want error, got nil")
		}
	})

	t.Run("Get of a missing ID is an error", func(t *testing.T) {
		repo := newRepo(t)
		if _, err := repo.Get(context.Background(), "missing"); err == nil {
			t.Error("Get missing: want error, got nil")
		}
	})

	t.Run("Update stores the finished Job", func(t *testing.T) {
		repo := newRepo(t)
		ctx := context.Background()
		created := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
		j := &job.Job{ID: "j2", BotName: "b", Status: job.StatusPending, CreatedAt: created}
		if err := repo.Create(ctx, j); err != nil {
			t.Fatalf("Create: %v", err)
		}
		if err := j.Start(created.Add(time.Second)); err != nil {
			t.Fatalf("Start: %v", err)
		}
		if err := j.Fail("boom", created.Add(2*time.Second)); err != nil {
			t.Fatalf("Fail: %v", err)
		}
		if err := repo.Update(ctx, j); err != nil {
			t.Fatalf("Update: %v", err)
		}

		got, err := repo.Get(ctx, "j2")
		if err != nil {
			t.Fatalf("Get: %v", err)
		}
		assertJobEqual(t, got, j)
	})

	t.Run("List of an empty repository is an empty slice, not nil", func(t *testing.T) {
		// GET /jobs ส่งผลของ List เป็น JSON ตรงๆ — nil กลายเป็น null ที่ dashboard ไม่รับ
		list, err := newRepo(t).List(context.Background())
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if list == nil || len(list) != 0 {
			t.Errorf("List = %#v, want empty non-nil slice", list)
		}
	})

	t.Run("List returns newest first", func(t *testing.T) {
		repo := newRepo(t)
		ctx := context.Background()
		base := time.Date(2026, 11, 1, 10, 0, 0, 0, time.UTC)
		// สร้างสลับลำดับกับเวลา เพื่อพิสูจน์ว่าเรียงตาม CreatedAt ไม่ใช่ตามลำดับที่ insert
		for _, c := range []struct {
			id     string
			offset time.Duration
		}{{"old", 0}, {"new", 2 * time.Minute}, {"mid", time.Minute}} {
			j := &job.Job{ID: c.id, BotName: "b", Status: job.StatusPending, CreatedAt: base.Add(c.offset)}
			if err := repo.Create(ctx, j); err != nil {
				t.Fatalf("Create %s: %v", c.id, err)
			}
		}

		list, err := repo.List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		var got []string
		for _, j := range list {
			got = append(got, j.ID)
		}
		want := []string{"new", "mid", "old"}
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] || got[2] != want[2] {
			t.Errorf("List order = %v, want %v", got, want)
		}
	})
}

// assertJobEqual เทียบทุก field ที่ API ส่งออก เวลาเทียบด้วย Equal เพราะ PostgreSQL คืนเวลา
// เป็น time zone ของ session ไม่ใช่ UTC ตัวเดิม
func assertJobEqual(t *testing.T, got, want *job.Job) {
	t.Helper()
	if got.ID != want.ID || got.BotName != want.BotName || got.TimeoutSeconds != want.TimeoutSeconds ||
		got.Status != want.Status || got.Summary != want.Summary || got.Error != want.Error {
		t.Errorf("job = %+v, want %+v", got, want)
	}
	if len(got.Config) != len(want.Config) {
		t.Errorf("Config = %v, want %v", got.Config, want.Config)
	}
	for k, v := range want.Config {
		if got.Config[k] != v {
			t.Errorf("Config[%q] = %q, want %q", k, got.Config[k], v)
		}
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, want.CreatedAt)
	}
	assertTimePtrEqual(t, "StartedAt", got.StartedAt, want.StartedAt)
	assertTimePtrEqual(t, "EndedAt", got.EndedAt, want.EndedAt)
}

func assertTimePtrEqual(t *testing.T, name string, got, want *time.Time) {
	t.Helper()
	if (got == nil) != (want == nil) || (got != nil && !got.Equal(*want)) {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

// openCleanDatabase รัน migration แบบเดียวกับตอน bop-api start แล้วล้างตาราง Job ให้แต่ละ
// test เริ่มจากศูนย์ (test ใน package นี้ไม่รันขนานกันจึงล้างร่วม database เดียวกันได้)
func openCleanDatabase(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	if err := postgres.Migrate(url); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	db, err := postgres.Connect(context.Background(), url)
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	t.Cleanup(db.Close)
	if _, err := db.Exec(context.Background(), "TRUNCATE jobs"); err != nil {
		t.Fatalf("truncate jobs: %v", err)
	}
	return db
}

func testDatabaseURL(t *testing.T) string {
	t.Helper()
	url := os.Getenv("BOP_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("BOP_TEST_DATABASE_URL not set: skipping PostgreSQL test")
	}
	return url
}

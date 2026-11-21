// Package postgres คือ implementation ของ port การเก็บข้อมูลบน PostgreSQL (database `bop`
// แยกจากของ Temporal แต่อยู่บน instance เดียวกัน) — ไฟล์นี้รวมส่วนที่ใช้ร่วมกันทุก
// repository/logger ใน package: การเปิด connection pool และ migration ที่ฝังอยู่ใน binary
//
// ลำดับที่ composition root ใช้: bop-api เรียก Migrate ก่อนแล้วค่อย Connect ส่วน bop-worker
// เรียกแค่ Connect (ห้ามรัน migration เพื่อไม่ให้สอง process แข่งกันแก้ schema)
package postgres

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib" // register driver "pgx" ให้ database/sql ที่ Migrate ใช้
)

// connectTimeout จำกัดเวลา ping ตอนเปิด pool — ถ้า database ยังไม่พร้อม ให้ process จบด้วย
// error ให้ restart policy ของ compose/K8s เริ่มใหม่ แทนที่จะค้างรอไม่รู้จบ
const connectTimeout = 10 * time.Second

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Connect เปิด connection pool ไปที่ databaseURL แล้ว ping หนึ่งครั้งเพื่อให้ config ผิดหรือ
// database ไม่พร้อมพังตั้งแต่ตอน start ไม่ใช่ตอน request แรก ผู้เรียกต้อง Close pool เอง
func Connect(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("postgres: open pool: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}

// Migrate ปรับ schema ให้ถึง migration ล่าสุดที่ฝังอยู่ใน binary — เรียกซ้ำได้ (schema ล่าสุด
// อยู่แล้ว = ไม่ทำอะไร) ใช้ connection ของตัวเองแยกจาก pool เพราะ golang-migrate ปิด
// connection ที่ได้รับไปตอนจบ
func Migrate(databaseURL string) error {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return fmt.Errorf("postgres: open for migrate: %w", err)
	}
	driver, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		db.Close()
		return fmt.Errorf("postgres: migrate driver: %w", err)
	}
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		driver.Close()
		return fmt.Errorf("postgres: migrate source: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "pgx5", driver)
	if err != nil {
		src.Close()
		driver.Close()
		return fmt.Errorf("postgres: migrate: %w", err)
	}
	defer m.Close()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}

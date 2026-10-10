// Package apperr กำหนดประเภท error กลางที่ชั้น usecase ใช้บอกชั้น adapter (เช่น HTTP)
// ว่า "error นี้เป็นความผิดพลาดแบบไหน" โดยที่ usecase ไม่ต้องรู้จัก HTTP status code และ
// adapter ไม่ต้องรู้จัก error type ของ Temporal SDK เลย — adapter ฝั่ง Temporal เป็นคนแปลง
// error ของ SDK มาเป็นประเภทในนี้ (เช่น serviceerror.NotFound -> ErrNotFound) แล้ว HTTP
// handler ค่อยแปลงประเภทในนี้เป็น status code อีกทีหนึ่ง
//
// ทุก wrapper ในไฟล์นี้คืนข้อความ (Error()) เดิมของ error ที่ห่อไว้ทุกตัวอักษร เพราะ HTTP
// response body เดิมส่ง err.Error() กลับไปให้ client ตรงๆ — การ refactor ต้องไม่เปลี่ยน
// ข้อความที่ dashboard เห็น
package apperr

import "errors"

// sentinel ของแต่ละประเภท — เช็คด้วย errors.Is(err, apperr.ErrNotFound) เป็นต้น
var (
	ErrInvalid  = errors.New("invalid input") // input จาก client ผิด/ขาด (HTTP 400)
	ErrNotFound = errors.New("not found")     // ไม่พบสิ่งที่อ้างถึง (HTTP 404)
	ErrConflict = errors.New("conflict")      // สร้างซ้ำกับของที่มีอยู่แล้ว (HTTP 409)
)

// kindError ห่อ error ตัวจริงไว้พร้อมระบุประเภท — Error() คืนข้อความของ error ตัวจริง
// Is() ตอบ true กับ sentinel ของประเภทนั้น และ Unwrap() ให้ errors.As มองทะลุไปถึง error
// ตัวจริงข้างในได้ตามปกติ
type kindError struct {
	kind error
	err  error
}

func (e *kindError) Error() string        { return e.err.Error() }
func (e *kindError) Is(target error) bool { return target == e.kind }
func (e *kindError) Unwrap() error        { return e.err }

// Invalid สร้าง error ประเภท ErrInvalid จากข้อความ
func Invalid(msg string) error { return &kindError{kind: ErrInvalid, err: errors.New(msg)} }

// WrapInvalid ห่อ error ที่มีอยู่แล้วให้เป็นประเภท ErrInvalid (ข้อความเดิม)
func WrapInvalid(err error) error { return &kindError{kind: ErrInvalid, err: err} }

// WrapNotFound ห่อ error ที่มีอยู่แล้วให้เป็นประเภท ErrNotFound (ข้อความเดิม)
func WrapNotFound(err error) error { return &kindError{kind: ErrNotFound, err: err} }

// Conflict สร้าง error ประเภท ErrConflict จากข้อความ
func Conflict(msg string) error { return &kindError{kind: ErrConflict, err: errors.New(msg)} }

// plainError ตัดประเภทของ error ทิ้ง เหลือแค่ข้อความ — ใช้กับขั้นตอนที่ API เดิมตอบ 500
// เสมอไม่ว่า error ข้างในจะเป็นประเภทไหน (เช่น describe หลังสร้าง schedule สำเร็จแล้ว)
type plainError struct{ msg string }

func (e *plainError) Error() string { return e.msg }

// Plain คืน error ที่มีข้อความเดิมแต่ไม่ตรงกับ sentinel ใดเลย
func Plain(err error) error {
	if err == nil {
		return nil
	}
	return &plainError{msg: err.Error()}
}

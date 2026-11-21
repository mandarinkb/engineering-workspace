import { describe, expect, it } from "vitest";

import { initialCatchupWindow, validateCatchupWindow } from "./catchup-window";

describe("initialCatchupWindow", () => {
  it("สร้างใหม่ได้ค่าเริ่มต้น 3600", () => {
    expect(initialCatchupWindow(undefined)).toBe("3600");
  });

  it("แก้ไขได้ค่าที่ Schedule ตั้งอยู่", () => {
    expect(initialCatchupWindow(120)).toBe("120");
  });

  it("Schedule เดิมที่ไม่เคยตั้ง (0) ได้ค่าเริ่มต้น 3600", () => {
    expect(initialCatchupWindow(0)).toBe("3600");
  });
});

describe("validateCatchupWindow", () => {
  it("ยอมรับ 10 วินาทีขึ้นไป", () => {
    expect(validateCatchupWindow("10")).toBeNull();
    expect(validateCatchupWindow("3600")).toBeNull();
  });

  it("ปฏิเสธค่าที่น้อยกว่า 10 วินาที", () => {
    expect(validateCatchupWindow("5")).toBe(
      "Catch-up Window ต้องเป็นจำนวนเต็ม 10 ถึง 31536000 วินาที",
    );
  });

  it("ยอมรับ 1 ปีพอดี และปฏิเสธค่าที่เกิน", () => {
    expect(validateCatchupWindow("31536000")).toBeNull();
    expect(validateCatchupWindow("31536001")).not.toBeNull();
  });

  it("ปฏิเสธค่าว่างและค่าที่ไม่ใช่จำนวนเต็ม", () => {
    expect(validateCatchupWindow("")).not.toBeNull();
    expect(validateCatchupWindow("abc")).not.toBeNull();
    expect(validateCatchupWindow("12.5")).not.toBeNull();
  });
});

import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { JobStatusBadge } from "./job-status-badge";

// ทดสอบตาม roadmap ข้อ 1.9 ตรงๆ: "test ว่า status badge render สีถูกต้องตาม status" —
// เช็คทั้ง label ที่แสดง (ต้องตรงกับ JobStatus) และ CSS class ที่บ่งบอกสี (running=เหลือง,
// succeeded=เขียว, failed=แดง ตามที่ roadmap ข้อ 1.7 กำหนดไว้)
describe("JobStatusBadge", () => {
  it("แสดงสีเหลืองสำหรับสถานะ running", () => {
    render(<JobStatusBadge status="running" />);
    const badge = screen.getByText("running");
    expect(badge).toHaveClass("bg-yellow-100");
  });

  it("แสดงสีเขียวสำหรับสถานะ succeeded", () => {
    render(<JobStatusBadge status="succeeded" />);
    const badge = screen.getByText("succeeded");
    expect(badge).toHaveClass("bg-green-100");
  });

  it("แสดงสีแดงสำหรับสถานะ failed", () => {
    render(<JobStatusBadge status="failed" />);
    const badge = screen.getByText("failed");
    expect(badge).toHaveClass("bg-red-100");
  });

  it("แสดงสีเทาสำหรับสถานะ pending และ cancelled", () => {
    render(<JobStatusBadge status="pending" />);
    expect(screen.getByText("pending")).toHaveClass("bg-gray-100");

    render(<JobStatusBadge status="cancelled" />);
    expect(screen.getByText("cancelled")).toHaveClass("bg-gray-100");
  });
});

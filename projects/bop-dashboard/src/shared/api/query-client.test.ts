import { describe, expect, it, vi } from "vitest";

import { ApiError } from "./api-error";
import { createQueryClient } from "./query-client";

function failWith(status: number) {
  return () => Promise.reject(new ApiError("boom", status));
}

describe("createQueryClient onUnauthorized", () => {
  it("เรียก onUnauthorized เมื่อ query ได้ 401 (เช่น session หมดอายุระหว่างใช้งาน)", async () => {
    const onUnauthorized = vi.fn();
    const client = createQueryClient({ onUnauthorized });

    await client.fetchQuery({ queryKey: ["jobs"], queryFn: failWith(401) }).catch(() => {});

    expect(onUnauthorized).toHaveBeenCalledTimes(1);
  });

  it("เรียก onUnauthorized เมื่อ mutation ได้ 401", async () => {
    const onUnauthorized = vi.fn();
    const client = createQueryClient({ onUnauthorized });

    await client
      .getMutationCache()
      .build(client, { mutationFn: failWith(401) })
      .execute(undefined)
      .catch(() => {});

    expect(onUnauthorized).toHaveBeenCalledTimes(1);
  });

  it("ไม่เรียก onUnauthorized กับ error อื่นที่ไม่ใช่ 401", async () => {
    const onUnauthorized = vi.fn();
    const client = createQueryClient({ onUnauthorized });

    await client.fetchQuery({ queryKey: ["job", "x"], queryFn: failWith(404) }).catch(() => {});
    await client.fetchQuery({ queryKey: ["jobs"], queryFn: failWith(403) }).catch(() => {});

    expect(onUnauthorized).not.toHaveBeenCalled();
  });
});

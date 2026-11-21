"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";

import { useBots } from "@/entities/bot";
import type { ScheduleStep } from "@/entities/schedule";

import { useCreateWorkflowRun } from "../api/use-create-workflow-run";
import { EMPTY_STEP, validateSteps } from "../lib/steps";
import { StepsEditor } from "./steps-editor";

// RunWorkflowForm รัน steps ทันทีครั้งเดียว (ad-hoc) ผ่าน POST /workflow-runs — อยู่ใน slice
// schedule-form เดียวกับ ScheduleForm เพราะใช้ StepsEditor/validateSteps ร่วมกัน (กฎ FSD ห้าม
// feature หนึ่ง import อีก feature และ StepsEditor ลงไปอยู่ entities ไม่ได้เพราะต้องรู้จักทั้ง
// bot และ schedule) — ฟอร์มนี้มีแค่ชื่อ + steps ไม่มีตารางเวลา/overlap เพราะรันครั้งเดียว
export function RunWorkflowForm() {
  const router = useRouter();
  const { data: bots } = useBots();
  const createRun = useCreateWorkflowRun();

  const [workflowName, setWorkflowName] = useState("");
  const [steps, setSteps] = useState<ScheduleStep[]>([EMPTY_STEP]);
  const [formError, setFormError] = useState<string | null>(null);

  function onSubmit(event: React.FormEvent) {
    event.preventDefault();
    setFormError(null);

    // backend ใช้ชื่อนี้เป็น prefix ของ workflow ID (เช่น "demo-fail-<hex>")
    const name = workflowName.trim();
    if (!name) {
      setFormError("กรอกชื่อ workflow");
      return;
    }
    const stepsError = validateSteps(steps, bots);
    if (stepsError) {
      setFormError(stepsError);
      return;
    }

    createRun.mutate(
      { workflow_name: name, steps },
      { onSuccess: (run) => router.push(`/workflow-runs/${encodeURIComponent(run.workflow_id)}`) },
    );
  }

  return (
    <form onSubmit={onSubmit} className="flex max-w-xl flex-col gap-6">
      <div>
        <label htmlFor="run_workflow_name" className="block text-sm font-medium">
          Workflow Name
        </label>
        <input
          id="run_workflow_name"
          value={workflowName}
          onChange={(event) => setWorkflowName(event.target.value)}
          placeholder="manual-healthcheck"
          className="mt-1 w-full rounded border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none"
        />
        <p className="mt-1 text-xs text-gray-500">ใช้เป็นคำนำหน้าของ workflow ID เพื่อให้ค้นหาเจอในหน้า Workflow Runs</p>
      </div>

      <StepsEditor steps={steps} onChange={setSteps} bots={bots} />

      {formError && <p className="text-sm text-red-600">{formError}</p>}
      {createRun.isError && <p className="text-sm text-red-600">{createRun.error.message}</p>}

      <button
        type="submit"
        disabled={createRun.isPending}
        className="self-start rounded bg-blue-600 px-4 py-2 font-medium text-white hover:bg-blue-700 disabled:opacity-50"
      >
        {createRun.isPending ? "กำลังเริ่ม..." : "รันเลย"}
      </button>
    </form>
  );
}

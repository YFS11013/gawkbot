/**
 * The "done means delivered" refusal must reach the human as a dialog, not as
 * a dead sentence: the broker's 409 tells the caller to pass
 * verification=not_required, and before J2-5 no screen could send it. This
 * suite pins the retry contract — the SAME verb fires again, now carrying the
 * artifact path or the declared empty completion.
 */
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ApiError } from "../../api/client";
import { TaskActionToolbar } from "./TaskActionToolbar";

const tasksApi = vi.hoisted(() => ({
  updateTaskStatus: vi.fn(),
}));

vi.mock("../../api/tasks", async () => {
  const actual =
    await vi.importActual<typeof import("../../api/tasks")>("../../api/tasks");
  return {
    ...actual,
    updateTaskStatus: tasksApi.updateTaskStatus,
  };
});

const REFUSAL =
  'task OFFICE-9 cannot reach done with no artifact. Publish the deliverable to the wiki, then retry this complete with artifact_path set to the wiki-relative path (e.g. "team/playbooks/launch.md") or the visual-artifact id. If this work genuinely ships no file, pass verification=not_required with verification_reason.';

function refusalError() {
  return new ApiError({
    status: 409,
    statusText: "Conflict",
    bodyText: JSON.stringify({ error: REFUSAL }),
  });
}

function renderToolbar(onAfterAction: () => void = () => {}) {
  const client = new QueryClient({
    defaultOptions: { mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <TaskActionToolbar
        taskId="OFFICE-9"
        channel="cos__human"
        lifecycleState="running"
        onAfterAction={onAfterAction}
      />
    </QueryClientProvider>,
  );
}

describe("TaskActionToolbar delivery gate", () => {
  beforeEach(() => {
    tasksApi.updateTaskStatus.mockReset();
  });

  it("asks for a deliverable when the broker refuses a bare complete", async () => {
    tasksApi.updateTaskStatus
      .mockRejectedValueOnce(refusalError())
      .mockResolvedValueOnce({ task: {} });

    renderToolbar();
    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));

    const dialog = await screen.findByRole("dialog");
    expect(dialog).toBeTruthy();
    // The broker's own sentence is shown verbatim — it names the reason.
    expect(screen.getByText(/cannot reach done with no artifact/)).toBeTruthy();

    fireEvent.change(screen.getByLabelText("Wiki path"), {
      target: { value: " team/playbooks/launch.md " },
    });
    fireEvent.click(screen.getByRole("button", { name: "Mark done anyway" }));

    await waitFor(() =>
      expect(tasksApi.updateTaskStatus).toHaveBeenCalledTimes(2),
    );
    expect(tasksApi.updateTaskStatus).toHaveBeenLastCalledWith(
      "OFFICE-9",
      "complete",
      "cos__human",
      "human",
      expect.objectContaining({ artifactPath: "team/playbooks/launch.md" }),
    );
  });

  it("refuses a declaration with no reason, then sends the stamped reason", async () => {
    tasksApi.updateTaskStatus.mockRejectedValueOnce(refusalError());

    renderToolbar();
    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));
    await screen.findByRole("dialog");

    fireEvent.click(screen.getByLabelText("This ships no file"));
    fireEvent.click(screen.getByRole("button", { name: "Mark done anyway" }));
    expect(screen.getByRole("alert").textContent).toContain("ships no file");
    expect(tasksApi.updateTaskStatus).toHaveBeenCalledTimes(1);

    tasksApi.updateTaskStatus.mockResolvedValueOnce({ task: {} });
    fireEvent.change(screen.getByLabelText("Why nothing ships"), {
      target: { value: "the answer lived in chat" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Mark done anyway" }));

    await waitFor(() =>
      expect(tasksApi.updateTaskStatus).toHaveBeenCalledTimes(2),
    );
    expect(tasksApi.updateTaskStatus).toHaveBeenLastCalledWith(
      "OFFICE-9",
      "complete",
      "cos__human",
      "human",
      expect.objectContaining({
        verification: "not_required",
        verificationReason: "the answer lived in chat",
      }),
    );
  });

  it("leaves other errors on the inline banner", async () => {
    tasksApi.updateTaskStatus.mockRejectedValueOnce(
      new ApiError({
        status: 403,
        statusText: "Forbidden",
        bodyText: JSON.stringify({ error: "forbidden" }),
      }),
    );

    renderToolbar();
    fireEvent.click(screen.getByRole("button", { name: "Mark done" }));

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toContain("forbidden"),
    );
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});

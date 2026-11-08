import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { TaskNewForm } from "./TaskNewForm";

const tasksApi = vi.hoisted(() => ({
  createTasks: vi.fn(),
}));

const routerApi = vi.hoisted(() => ({
  navigate: vi.fn(),
}));

const clientApi = vi.hoisted(() => ({
  postMessage: vi.fn(),
}));

const platformApi = vi.hoisted(() => ({
  getHealth: vi.fn(),
}));

vi.mock("../../api/tasks", () => tasksApi);
vi.mock("../../api/client", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../api/client")>();
  return { ...actual, postMessage: clientApi.postMessage };
});
vi.mock("../../api/platform", () => platformApi);
vi.mock("../../lib/router", () => ({ router: routerApi }));

function renderForm() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <TaskNewForm />
    </QueryClientProvider>,
  );
}

describe("<TaskNewForm>", () => {
  beforeEach(() => {
    tasksApi.createTasks.mockReset();
    clientApi.postMessage.mockReset();
    routerApi.navigate.mockReset();
    platformApi.getHealth.mockReset();
  });

  it("creates manual issues as spec-level issue tasks", async () => {
    tasksApi.createTasks.mockResolvedValueOnce({
      tasks: [{ id: "task-issue-1", title: "Spec onboarding", status: "open" }],
    });

    renderForm();

    fireEvent.change(screen.getByTestId("issue-new-title"), {
      target: { value: "Spec onboarding handoff" },
    });
    fireEvent.change(screen.getByTestId("issue-new-details"), {
      target: { value: "Goal, context, and acceptance criteria." },
    });
    const form = screen.getByTestId("issue-new-submit").closest("form");
    expect(form).not.toBeNull();
    if (!form) return;
    fireEvent.submit(form);

    await waitFor(() => expect(tasksApi.createTasks).toHaveBeenCalled());
    expect(tasksApi.createTasks).toHaveBeenCalledWith(
      [
        {
          title: "Spec onboarding handoff",
          assignee: "human",
          details: "Goal, context, and acceptance criteria.",
          task_type: "issue",
        },
      ],
      // No channel: the form no longer pre-fills one, so the broker routes the
      // Issue to its assignee's DM. It used to send "general", a retired room.
      { channel: "", createdBy: "human" },
    );
  });

  it("holds a bare question and offers chat instead of filing an Issue", async () => {
    platformApi.getHealth.mockResolvedValue({
      status: "ok",
      one_on_one_agent: "cos",
    });
    tasksApi.createTasks.mockClear();
    renderForm();

    fireEvent.change(screen.getByTestId("issue-new-title"), {
      target: { value: "What is our Q3 renewal rate?" },
    });
    const form = screen.getByTestId("issue-new-submit").closest("form");
    expect(form).not.toBeNull();
    if (!form) return;
    fireEvent.submit(form);

    await waitFor(() =>
      expect(
        screen.getByTestId("issue-new-question-triage"),
      ).toBeInTheDocument(),
    );
    expect(tasksApi.createTasks).not.toHaveBeenCalled();
  });

  it("still files the question when the human insists", async () => {
    platformApi.getHealth.mockResolvedValue({
      status: "ok",
      one_on_one_agent: "cos",
    });
    tasksApi.createTasks.mockClear();
    tasksApi.createTasks.mockResolvedValueOnce({
      tasks: [
        {
          id: "task-q-1",
          title: "What is our Q3 renewal rate?",
          status: "open",
        },
      ],
    });
    renderForm();

    fireEvent.change(screen.getByTestId("issue-new-title"), {
      target: { value: "What is our Q3 renewal rate?" },
    });
    const form = screen.getByTestId("issue-new-submit").closest("form");
    if (!form) return;
    fireEvent.submit(form);
    fireEvent.click(await screen.findByTestId("issue-new-file-anyway"));

    await waitFor(() => expect(tasksApi.createTasks).toHaveBeenCalled());
    expect(tasksApi.createTasks.mock.calls[0][0][0].title).toBe(
      "What is our Q3 renewal rate?",
    );
  });

  it("asks the question in the office DM instead", async () => {
    platformApi.getHealth.mockResolvedValue({
      status: "ok",
      one_on_one_agent: "cos",
    });
    clientApi.postMessage.mockResolvedValueOnce({ id: "msg-1" });
    renderForm();

    fireEvent.change(screen.getByTestId("issue-new-title"), {
      target: { value: "How do we price the renewal?" },
    });
    const form = screen.getByTestId("issue-new-submit").closest("form");
    if (!form) return;
    fireEvent.submit(form);
    fireEvent.click(await screen.findByTestId("issue-new-ask-chat"));

    await waitFor(() =>
      expect(clientApi.postMessage).toHaveBeenCalledWith(
        "How do we price the renewal?",
        "cos__human",
      ),
    );
    expect(routerApi.navigate).toHaveBeenCalledWith({
      to: "/channels/$channelSlug",
      params: { channelSlug: "cos__human" },
    });
    expect(tasksApi.createTasks).not.toHaveBeenCalled();
  });
});

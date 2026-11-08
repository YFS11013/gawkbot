import { describe, expect, it } from "vitest";

import { looksLikeQuestion } from "./questionIntent";

describe("looksLikeQuestion", () => {
  it("triages punctuation-marked questions", () => {
    expect(looksLikeQuestion("What is our renewal rate?")).toBe(true);
    expect(looksLikeQuestion("续约率是多少？")).toBe(true);
  });

  it("triages CJK interrogative openers and final particles", () => {
    expect(looksLikeQuestion("如何提高续约率")).toBe(true);
    expect(looksLikeQuestion("请问三客户续约进展")).toBe(true);
    expect(looksLikeQuestion("Brightline 的 usage 涨了吗")).toBe(true);
  });

  it("triages Latin interrogative openers", () => {
    expect(looksLikeQuestion("how do we price the renewal")).toBe(true);
    expect(looksLikeQuestion("Can you draft the Q3 brief")).toBe(true);
    expect(looksLikeQuestion("is the wiki worker up")).toBe(true);
  });

  it("does not triage work specs", () => {
    expect(looksLikeQuestion("Ship the welcome email")).toBe(false);
    expect(looksLikeQuestion("Draft the Acme renewal email")).toBe(false);
    expect(looksLikeQuestion("整理三客户续约账户简报")).toBe(false);
    expect(looksLikeQuestion("How-to guide for the onboarding flow")).toBe(
      false,
    );
    expect(looksLikeQuestion("   ")).toBe(false);
  });
});

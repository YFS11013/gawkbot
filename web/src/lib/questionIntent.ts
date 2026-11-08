/**
 * Question-intent triage for the manual task form (P3 7g ①).
 *
 * A question typed into "File a task spec" used to become an Issue. That Issue
 * has no deliverable, so it can only be closed empty (the hollow-completion
 * chain: ACMER-2757 / 2764 / 2785, then Journey 1/2 repeats). The question
 * belongs in chat, where the office answers it and the answer IS the delivery.
 *
 * The check is deliberately narrow and title-only:
 *   - punctuation: the text ends with "?" / "？"
 *   - CJK interrogative openers or sentence-final particles (吗 / 呢)
 *   - Latin interrogative openers (who/what/why/how/is/can …)
 * Details text is never inspected: acceptance criteria legitimately contain
 * questions, and the form only triages when the details field is empty.
 */

const CJK_OPENERS = [
  "请问",
  "请教",
  "是否",
  "如何",
  "怎么",
  "怎样",
  "为什么",
  "为啥",
  "能不能",
  "能否",
  "可不可以",
  "多少",
  "几个",
  "哪些",
  "哪个",
  "谁",
  "何时",
  "什么时候",
  "有没有",
  "是不是",
  "要不要",
  "会不会",
  "行不行",
];

const LATIN_OPENERS = new Set([
  "who",
  "what",
  "when",
  "where",
  "why",
  "how",
  "which",
  "whose",
  "is",
  "are",
  "am",
  "was",
  "were",
  "do",
  "does",
  "did",
  "can",
  "could",
  "should",
  "would",
  "will",
  "shall",
  "may",
  "might",
  "have",
  "has",
]);

const CJK_FINAL_PARTICLES = ["吗", "呢"];

/**
 * looksLikeQuestion reports whether `text` reads as a question rather than a
 * work spec. Empty input is not a question.
 */
export function looksLikeQuestion(text: string): boolean {
  const trimmed = text.trim();
  if (!trimmed) return false;

  if (trimmed.endsWith("?") || trimmed.endsWith("？")) return true;

  for (const opener of CJK_OPENERS) {
    if (trimmed.startsWith(opener)) return true;
  }
  for (const particle of CJK_FINAL_PARTICLES) {
    if (trimmed.endsWith(particle)) return true;
  }

  // Latin openers only count as the FIRST word, so a title such as
  // "How-to guide for onboarding" (hyphenated, one word) is not triaged.
  const firstWord = trimmed.split(/[\s,，。:：;；]+/)[0]?.toLowerCase() ?? "";
  return LATIN_OPENERS.has(firstWord);
}

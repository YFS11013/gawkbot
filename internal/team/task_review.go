package team

// task_review.go — clean-context review lane.
//
// Why it exists: external evidence (DESIGN-DECISIONS.md §10) says a writer
// reviewing its own turn catches little, while a reviewer that never saw the
// writer's reasoning catches roughly two bugs per PR (Cognition 2026). This
// lane turns a delivered artifact into a second, independent read: the broker
// opens a review task for a bot that is NOT the author and hands it only the
// artifact path plus the original success criteria — no author chat, no
// author reasoning.
//
// What it is NOT: the reviewer never edits the author's artifact, never
// reopens the author's task, and never blocks the author's completion. Writes
// stay single-threaded (§10 judgement 1) — the findings are input for the
// human/CoS decision, not an automatic veto. The review task carries its own
// wiki_artifact gate (plus the freshness rule), so a review cannot close
// without a real findings file.

import (
	"fmt"
	"log"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/nex-crm/wuphf/internal/config"
)

// taskReviewSignalPrefix dedupes review tasks and marks them so a review is
// never itself reviewed.
const taskReviewSignalPrefix = "review:"

// queueTaskReview schedules the clean-context review of a delivered artifact.
// Called from the done branch of MutateTask, after b.mu is released.
func (b *Broker) queueTaskReview(taskID string) {
	go func() {
		defer recoverPanicTo("reviewCompletedTask", "task="+taskID)
		b.reviewCompletedTask(taskID)
	}()
}

// reviewCompletedTask opens one review task per delivered task. Idempotent:
// an existing task carrying the review signal short-circuits, so
// approve-after-complete and watchdog replays cannot fan out a second review.
func (b *Broker) reviewCompletedTask(taskID string) {
	taskID = strings.TrimSpace(taskID)
	if taskID == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()

	task := b.taskByIDLocked(taskID)
	if task == nil || !strings.EqualFold(strings.TrimSpace(task.status), "done") {
		return
	}
	// Only a task with a real contract gets a review: the review judges the
	// delivered artifact against stated success criteria, and a task without
	// a Definition has none. This is the same boundary the artifact gate uses.
	if task.Definition == nil || strings.TrimSpace(task.Artifact) == "" {
		return
	}
	if strings.HasPrefix(strings.TrimSpace(task.SourceSignalID), taskReviewSignalPrefix) {
		return
	}
	// A failed check means the delivery did not meet its own definition of
	// done; reviewing it would grade work the gate already refused.
	if v := task.Verification; v != nil && v.Required {
		if res := task.VerificationResult; res == nil || !res.Pass {
			return
		}
	}
	signal := taskReviewSignalPrefix + task.ID
	for i := range b.tasks {
		if strings.TrimSpace(b.tasks[i].SourceSignalID) == signal {
			return
		}
	}
	// WUPHF_TASK_REVIEWER=off turns the lane off for an office that does not
	// want to pay a second turn per delivery.
	switch strings.ToLower(strings.TrimSpace(config.ResolveTaskReviewer())) {
	case "off", "none", "disabled":
		return
	}
	reviewer := b.reviewerForTaskLocked(task)
	if reviewer == "" || reviewer == strings.TrimSpace(task.Owner) {
		log.Printf("task review: no reviewer available for %s (owner %q)", task.ID, task.Owner)
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	channel := b.preferredTaskChannelLocked("", "system", reviewer, "", "")
	b.counter++
	review := teamTask{
		ID:             b.allocateIssueIDLocked(),
		Channel:        channel,
		Title:          "审阅：" + truncateSummary(strings.TrimSpace(task.Title), 80),
		Details:        reviewTaskDetails(*task),
		Owner:          reviewer,
		status:         "in_progress",
		CreatedBy:      "system",
		TaskType:       "research",
		PipelineID:     "review",
		ExecutionMode:  "office",
		SourceSignalID: signal,
		DependsOn:      []string{task.ID},
		Definition: &TaskDefinition{
			Goal: fmt.Sprintf("独立审阅 %s 的交付，逐条对照验收标准给出可执行的发现。", task.ID),
			Deliverables: []TaskDeliverable{{
				Name:   "审阅结论",
				Format: "markdown，路径 " + reviewArtifactPath(task.ID),
			}},
			SuccessCriteria: []string{
				"文件存在且非空",
				"每条验收标准都有通过/不通过判断，并引用交付物原文位置",
				"发现的问题标明严重度与修改建议",
			},
			AccessNeeded: []string{"team_wiki_read", "team_wiki_write"},
			// The declared path is what /wiki/write-task accepts from this
			// task's owner — the reviewer needs no human go-ahead to write
			// the one file the broker already pinned for it.
			ArtifactPath: reviewArtifactPath(task.ID),
			DefinedAt:    now,
		},
		Verification: &TaskVerification{
			Kind:     taskVerificationKindWikiArtifact,
			Spec:     "artifact_path declared on complete",
			Required: true,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if channel != "" {
		b.ensureTaskOwnerChannelMembershipLocked(channel, reviewer)
	}
	b.queueTaskBehindActiveOwnerLaneLocked(&review)
	b.scheduleTaskLifecycleLocked(&review)
	b.reindexTaskLifecycleFromLegacyLocked(&review)
	b.tasks = append(b.tasks, review)
	b.appendActionLocked("task_created", "office", channel, "system", truncateSummary(review.Title, 140), review.ID)
	// The wake. A bare task_created action is only an audit line; bots wake on
	// a tagged chat message (the same reason the Issue card carries Tagged).
	// Target the reviewer alone — tagging the lead here would wake cos for
	// every delivery, which is the noise this lane exists to avoid.
	if id := b.postReviewRequestLocked(review); id != "" {
		if i := len(b.tasks) - 1; i >= 0 {
			b.tasks[i].ThreadID = id
		}
	}
	if err := b.saveLocked(); err != nil {
		log.Printf("task review: save after creating %s for %s: %v", review.ID, task.ID, err)
	}
}

// postReviewRequestLocked posts the one message that wakes the reviewer, and
// returns its id (empty when the channel has no home for it).
func (b *Broker) postReviewRequestLocked(review teamTask) string {
	channel := normalizeChannelSlug(review.Channel)
	if strings.TrimSpace(review.Channel) == "" || taskCardHasNoHome("task_review_requested", &review) {
		return ""
	}
	b.counter++
	id := fmt.Sprintf("msg-%d", b.counter)
	parent := ""
	if len(review.DependsOn) > 0 {
		parent = strings.TrimSpace(review.DependsOn[0])
	}
	b.appendMessageLocked(channelMessage{
		ID:        id,
		From:      "system",
		Channel:   channel,
		Kind:      "task_review_requested",
		Title:     review.Title,
		Content:   fmt.Sprintf("独立审阅请求：%s 的交付已通过机器校验，请以干净上下文读一遍并给出结论（写 %s）。", parent, reviewArtifactPath(parent)),
		Tagged:    []string{strings.TrimSpace(review.Owner)},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
	return id
}

// reviewArtifactPath is where a review of taskID must land. Deterministic so
// a re-run overwrites rather than piling up near-duplicate files.
func reviewArtifactPath(taskID string) string {
	// Under team/ because that is the only root the wiki worker will accept a
	// write for (wiki_fs.go / wiki_git.go both refuse anything else).
	return path.Join("team", "reviews", strings.TrimSpace(taskID)+".md")
}

// reviewerForTaskLocked picks the bot that reads the delivery. Preference:
// WUPHF_TASK_REVIEWER, then the office lead, then the first member by slug —
// always someone other than the author. "" means no review is possible (a
// one-bot office, or the only member wrote the task).
func (b *Broker) reviewerForTaskLocked(task *teamTask) string {
	if task == nil {
		return ""
	}
	owner := strings.TrimSpace(task.Owner)
	if want := strings.TrimSpace(strings.ToLower(config.ResolveTaskReviewer())); want != "" && want != owner && b.memberSlugExistsLocked(want) {
		return want
	}
	if lead := strings.TrimSpace(officeLeadSlugFrom(b.members)); lead != "" && lead != owner {
		return lead
	}
	slugs := make([]string, 0, len(b.members))
	for _, m := range b.members {
		if slug := strings.TrimSpace(m.Slug); slug != "" && slug != owner {
			slugs = append(slugs, slug)
		}
	}
	if len(slugs) == 0 {
		return ""
	}
	sort.Strings(slugs)
	return slugs[0]
}

func (b *Broker) memberSlugExistsLocked(slug string) bool {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return false
	}
	for _, m := range b.members {
		if strings.EqualFold(strings.TrimSpace(m.Slug), slug) {
			return true
		}
	}
	return false
}

// reviewTaskDetails is the reviewer's entire brief — clean context by
// construction: the author's chat, plan, and reasoning are deliberately not
// included, only the contract and where the deliverable lives.
func reviewTaskDetails(task teamTask) string {
	var sb strings.Builder
	sb.WriteString("独立审阅（干净上下文）：你只会看到交付物与验收标准，看不到作者的推理过程。不要假设作者做对了，也不要假设作者做错了——以交付物本身为准。\n\n")
	sb.WriteString(fmt.Sprintf("被审阅的任务：%s（作者 @%s）\n", task.ID, strings.TrimSpace(task.Owner)))
	if task.Definition != nil {
		if goal := strings.TrimSpace(task.Definition.Goal); goal != "" {
			sb.WriteString("原始目标：" + goal + "\n")
		}
		if len(task.Definition.SuccessCriteria) > 0 {
			sb.WriteString("验收标准：\n")
			for _, c := range task.Definition.SuccessCriteria {
				if c = strings.TrimSpace(c); c != "" {
					sb.WriteString("- " + c + "\n")
				}
			}
		}
	}
	sb.WriteString("交付物：" + strings.TrimSpace(task.Artifact) + "（wiki 相对路径）\n\n")
	sb.WriteString("要求：\n")
	sb.WriteString("1. 先完整读一遍交付物，再下判断。没有证据的判断不要写。\n")
	sb.WriteString("2. 逐条对照上面的验收标准，每条给「通过 / 不通过」，并引用交付物的章节或原文片段作为依据。\n")
	sb.WriteString("3. 列出发现的问题，标严重度（阻断 / 一般 / 提示），每条写清：在哪里、为什么算问题、怎么改。\n")
	sb.WriteString("4. 不要修改交付物，不要重开被审阅的任务——你的产出就是这份结论。\n")
	sb.WriteString(fmt.Sprintf("5. 用 team_wiki_write 写 %s，并带上本任务的 task_id（任务已声明该路径，不需要人类再批准）；然后用 artifact_path=%s 完成本任务。\n", reviewArtifactPath(task.ID), reviewArtifactPath(task.ID)))
	return sb.String()
}

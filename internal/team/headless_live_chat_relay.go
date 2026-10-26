package team

import (
	"strings"
)

const (
	// Flush only on paragraph boundaries with enough content: sentence-end
	// flushing (16 chars + trailing ./!/?) shattered streamed replies into
	// one-or-two-sentence fragments (data-keeper DM msg-393..397). Live
	// relay is a progress heartbeat — the turn's final post carries the
	// full text — so batching to paragraphs is strictly better UX.
	headlessLiveChatMinFlushChars = 200
	headlessLiveChatMaxFlushChars = 2000
)

type headlessLiveChatRelay struct {
	l            *Launcher
	slug         string
	channel      string
	replyTo      string
	logf         func(string)
	buf          strings.Builder
	lastPosted   string
	postFailures int
}

// resolveHeadlessTargetChannel normalizes the turn's target channel and
// repairs it when it names a channel that no longer exists (e.g.
// onboarding-era tasks bound to #general after the DM-first retirement).
// Dropping the output on "channel not found" burns the whole turn, so a
// missing non-DM channel falls back to the bot's own DM — PostMessage
// auto-creates it, and the message keeps an owner instead of vanishing.
func resolveHeadlessTargetChannel(l *Launcher, slug string, targetChannel string) string {
	targetChannel = normalizeChannelSlug(targetChannel)
	if targetChannel == "" {
		targetChannel = "general"
	}
	if IsDMSlug(targetChannel) {
		if targetBot := DMTargetBot(targetChannel); targetBot != "" {
			targetChannel = DMSlugFor(targetBot)
		}
		return targetChannel
	}
	if l == nil || l.broker == nil {
		return targetChannel
	}
	if l.broker.findChannelLocked(targetChannel) == nil {
		if dm := DMSlugFor(slug); dm != "" {
			targetChannel = dm
		}
	}
	return targetChannel
}
func newHeadlessLiveChatRelay(l *Launcher, slug string, targetChannel string, notification string, logf func(string)) *headlessLiveChatRelay {
	if l == nil || l.broker == nil {
		return nil
	}
	return &headlessLiveChatRelay{
		l:       l,
		slug:    strings.TrimSpace(slug),
		channel: resolveHeadlessTargetChannel(l, slug, targetChannel),
		replyTo: headlessReplyToID(notification),
		logf:    logf,
	}
}

func (r *headlessLiveChatRelay) OnText(chunk string) {
	if r == nil || chunk == "" {
		return
	}
	r.buf.WriteString(chunk)
	text := r.buf.String()
	if len(strings.TrimSpace(text)) >= headlessLiveChatMaxFlushChars || headlessLiveChatShouldFlush(text) {
		r.Flush()
	}
}

func (r *headlessLiveChatRelay) Flush() {
	if r == nil || r.l == nil || r.l.broker == nil {
		return
	}
	text := strings.TrimSpace(r.buf.String())
	r.buf.Reset()
	r.postText(text)
}

func (r *headlessLiveChatRelay) ReportIssue(detail string) {
	if r == nil || r.l == nil || r.l.broker == nil {
		return
	}
	detail = strings.TrimSpace(detail)
	if detail == "" {
		return
	}
	r.Flush()
	msg, _, posted, err := r.l.broker.ReportIncident(r.slug, r.channel, r.replyTo, detail)
	if err != nil {
		r.postFailures++
		if r.logf != nil && r.postFailures <= 3 {
			r.logf("live-chat-relay-issue-error: " + err.Error())
		}
		return
	}
	if posted {
		r.lastPosted = msg.Content
		if r.logf != nil {
			r.logf("live-chat-relay-post: posted streamed issue to #" + msg.Channel + " as " + msg.ID)
		}
	}
}

func (r *headlessLiveChatRelay) postText(text string) {
	if r == nil || r.l == nil || r.l.broker == nil {
		return
	}
	text = strings.TrimSpace(text)
	if text == "" || text == r.lastPosted || looksUnparsedToolCall(text) {
		return
	}
	msg, err := r.l.broker.PostMessage(r.slug, r.channel, text, nil, r.replyTo)
	if err != nil {
		r.postFailures++
		if r.logf != nil && r.postFailures <= 3 {
			r.logf("live-chat-relay-post-error: " + err.Error())
		}
		return
	}
	r.lastPosted = text
	if r.logf != nil {
		r.logf("live-chat-relay-post: posted streamed output to #" + msg.Channel + " as " + msg.ID)
	}
}

func headlessLiveChatShouldFlush(text string) bool {
	// Paragraph-boundary only: a double newline with enough buffered content
	// ends one post. Sentence-end flushing is gone (see const comment) —
	// short sentences accumulate until the paragraph completes or the max
	// cap forces a flush.
	if !strings.Contains(text, "\n\n") {
		return false
	}
	return len(strings.TrimSpace(text)) >= headlessLiveChatMinFlushChars
}

func headlessLiveChatLooksIssue(text string) bool {
	return classifyIncident(text).Visible
}

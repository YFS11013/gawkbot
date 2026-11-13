package teammcp

import "testing"

// Journey 28: the human's authorization was Chinese ("我现在就授权你把这两页
// 写进 wiki"), and the English-only verb list scored it false — the door that
// exists for exactly this case refused the person it was written for.
func TestDirectWikiWriteIntentReadsChinese(t *testing.T) {
	cases := []struct {
		text string
		want bool
	}{
		{"我现在就授权你把这两页写进 wiki：team/reviews/j28-internal.md、team/reviews/j28-public.md。", true},
		{"请把这篇更新发布到知识库，路径 team/accounts/acme.md。", true},
		{"把续约清单写入 wiki 吧。", true},
		{"不要把内部哈希写进 wiki，别发布。", false},
		{"这份东西记录得不错，wiki 上已经有了。", false},
		{"内容很好，不需要改动。", false},
	}
	for _, tc := range cases {
		if got := hasDirectWikiWriteIntent(tc.text); got != tc.want {
			t.Errorf("hasDirectWikiWriteIntent(%q) = %v, want %v", tc.text, got, tc.want)
		}
	}
}

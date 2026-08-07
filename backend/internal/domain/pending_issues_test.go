package domain

import "testing"

func TestPendingIssueFilterMatchesGroupAndDetail(t *testing.T) {
	t.Parallel()

	if !PendingIssueFilterMatches(string(PendingIssueMissingAssets), PendingIssueDetailMissingCover) {
		t.Fatalf("group filter should match detail in same group")
	}
	if !PendingIssueFilterMatches(string(PendingIssueDetailMissingCover), PendingIssueDetailMissingCover) {
		t.Fatalf("detail filter should match itself")
	}
	if PendingIssueFilterMatches(string(PendingIssueMissingWiki), PendingIssueDetailMissingCover) {
		t.Fatalf("group filter should not match unrelated detail")
	}
	if PendingIssueFilterMatches("", PendingIssueDetailMissingCover) {
		t.Fatalf("empty filter should not match any detail")
	}
}

func TestIsPendingIssueSevereUsesSharedPolicy(t *testing.T) {
	t.Parallel()

	if !IsPendingIssueSevere([]PendingIssueKey{PendingIssueMissingFiles}, 1) {
		t.Fatalf("missing files should always be severe")
	}
	if !IsPendingIssueSevere([]PendingIssueKey{PendingIssueMissingAssets, PendingIssueMissingWiki}, 2) {
		t.Fatalf("assets plus wiki should be severe")
	}
	if !IsPendingIssueSevere([]PendingIssueKey{PendingIssueMissingAssets, PendingIssueMissingMetadata}, 3) {
		t.Fatalf("three visible details should be severe")
	}
	if IsPendingIssueSevere([]PendingIssueKey{PendingIssueMissingAssets}, 1) {
		t.Fatalf("single non-files group should not be severe")
	}
}

func TestEvaluatePendingIssuesUsesDomainSeverityRules(t *testing.T) {
	t.Parallel()

	ready := "ready"
	game := Game{
		Title:             "Pending Rule Test",
		CoverImage:        &ready,
		BannerImage:       &ready,
		Summary:           &ready,
		PrimaryScreenshot: &ready,
		WikiContent:       &ready,
	}

	evaluation := EvaluatePendingIssues(game, 1, 1, 1, 0, 1, 1, nil)
	if !evaluation.Severe {
		t.Fatalf("evaluation.severe = false, want true when missing files is visible")
	}
	if len(evaluation.Groups) != 1 || evaluation.Groups[0] != PendingIssueMissingFiles {
		t.Fatalf("evaluation.groups = %#v, want only missing-files", evaluation.Groups)
	}
}

func TestEvaluatePendingIssuesMissingVideo(t *testing.T) {
	t.Parallel()

	ready := "ready"
	game := Game{
		Title:             "Missing Video Test",
		CoverImage:        &ready,
		BannerImage:       &ready,
		Summary:           &ready,
		PrimaryScreenshot: &ready,
		WikiContent:       &ready,
	}

	// 素材齐全但无视频：应报"缺预告片"，归属 missing-assets 组
	evaluation := EvaluatePendingIssues(game, 1, 1, 0, 1, 1, 1, nil)
	if !hasDetail(evaluation, PendingIssueDetailMissingVideo) {
		t.Fatalf("evaluation should include missing-video detail, got %#v", evaluation.Details)
	}
	if !hasGroup(evaluation, PendingIssueMissingAssets) {
		t.Fatalf("evaluation should include missing-assets group, got %#v", evaluation.Groups)
	}

	// 有视频：不应报缺预告片
	evaluation = EvaluatePendingIssues(game, 1, 1, 1, 1, 1, 1, nil)
	if hasDetail(evaluation, PendingIssueDetailMissingVideo) {
		t.Fatalf("evaluation should not include missing-video detail, got %#v", evaluation.Details)
	}
}

func hasDetail(evaluation PendingIssueEvaluation, key PendingIssueDetailKey) bool {
	for _, detail := range evaluation.Details {
		if detail.Key == key {
			return true
		}
	}
	return false
}

func hasGroup(evaluation PendingIssueEvaluation, key PendingIssueKey) bool {
	for _, group := range evaluation.Groups {
		if group == key {
			return true
		}
	}
	return false
}

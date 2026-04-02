package domain

type PendingIssueKey string
type PendingIssueDetailKey string

const (
	PendingIssueMissingAssets   PendingIssueKey = "missing-assets"
	PendingIssueMissingWiki     PendingIssueKey = "missing-wiki"
	PendingIssueMissingFiles    PendingIssueKey = "missing-files"
	PendingIssueMissingMetadata PendingIssueKey = "missing-metadata"
)

const (
	PendingIssueDetailMissingCover       PendingIssueDetailKey = "missing-cover"
	PendingIssueDetailMissingBanner      PendingIssueDetailKey = "missing-banner"
	PendingIssueDetailMissingScreenshots PendingIssueDetailKey = "missing-screenshots"
	PendingIssueDetailMissingWikiContent PendingIssueDetailKey = "missing-wiki-content"
	PendingIssueDetailMissingFilesList   PendingIssueDetailKey = "missing-files-list"
	PendingIssueDetailMissingDeveloper   PendingIssueDetailKey = "missing-developer"
	PendingIssueDetailMissingPublisher   PendingIssueDetailKey = "missing-publisher"
	PendingIssueDetailMissingPlatform    PendingIssueDetailKey = "missing-platform"
	PendingIssueDetailMissingSummary     PendingIssueDetailKey = "missing-summary"
)

type PendingIssueDefinition struct {
	Key         PendingIssueKey `json:"key"`
	Label       string          `json:"label"`
	Description string          `json:"description"`
}

type PendingIssueDetailDefinition struct {
	Key   PendingIssueDetailKey `json:"key"`
	Label string                `json:"label"`
	Group PendingIssueKey       `json:"group"`
}

type PendingIssueDetailState struct {
	Key     PendingIssueDetailKey `json:"key"`
	Group   PendingIssueKey       `json:"group"`
	Ignored bool                  `json:"ignored"`
	Reason  *string               `json:"reason,omitempty"`
}

type PendingIssueEvaluation struct {
	Groups  []PendingIssueKey         `json:"groups"`
	Details []PendingIssueDetailState `json:"details"`
	Severe  bool                      `json:"severe"`
}

type PendingIssueCatalog struct {
	Groups  []PendingIssueDefinition       `json:"groups"`
	Details []PendingIssueDetailDefinition `json:"details"`
}

type PendingIssueCountSummary struct {
	Groups       map[PendingIssueKey]int `json:"groups"`
	IgnoredTotal int                     `json:"ignored_total"`
}

var pendingIssueDefinitions = []PendingIssueDefinition{
	{Key: PendingIssueMissingAssets, Label: "缺少图片", Description: "封面、横幅或截图未补齐"},
	{Key: PendingIssueMissingWiki, Label: "缺少 Wiki", Description: "还没有游戏介绍内容"},
	{Key: PendingIssueMissingFiles, Label: "缺少文件", Description: "还没有可下载文件条目"},
	{Key: PendingIssueMissingMetadata, Label: "基础信息不完整", Description: "开发商、发行商、平台或简介缺失"},
}

var pendingIssueDetailDefinitions = []PendingIssueDetailDefinition{
	{Key: PendingIssueDetailMissingCover, Label: "缺封面", Group: PendingIssueMissingAssets},
	{Key: PendingIssueDetailMissingBanner, Label: "缺横幅", Group: PendingIssueMissingAssets},
	{Key: PendingIssueDetailMissingScreenshots, Label: "缺截图", Group: PendingIssueMissingAssets},
	{Key: PendingIssueDetailMissingWikiContent, Label: "缺 Wiki 内容", Group: PendingIssueMissingWiki},
	{Key: PendingIssueDetailMissingFilesList, Label: "缺下载文件", Group: PendingIssueMissingFiles},
	{Key: PendingIssueDetailMissingDeveloper, Label: "缺开发商", Group: PendingIssueMissingMetadata},
	{Key: PendingIssueDetailMissingPublisher, Label: "缺发行商", Group: PendingIssueMissingMetadata},
	{Key: PendingIssueDetailMissingPlatform, Label: "缺平台", Group: PendingIssueMissingMetadata},
	{Key: PendingIssueDetailMissingSummary, Label: "缺简介", Group: PendingIssueMissingMetadata},
}

var pendingIssueDefinitionMap = func() map[PendingIssueKey]PendingIssueDefinition {
	items := make(map[PendingIssueKey]PendingIssueDefinition, len(pendingIssueDefinitions))
	for _, item := range pendingIssueDefinitions {
		items[item.Key] = item
	}
	return items
}()

var pendingIssueDetailDefinitionMap = func() map[PendingIssueDetailKey]PendingIssueDetailDefinition {
	items := make(map[PendingIssueDetailKey]PendingIssueDetailDefinition, len(pendingIssueDetailDefinitions))
	for _, item := range pendingIssueDetailDefinitions {
		items[item.Key] = item
	}
	return items
}()

func PendingIssueCatalogDefinitions() PendingIssueCatalog {
	return PendingIssueCatalog{
		Groups:  append([]PendingIssueDefinition(nil), pendingIssueDefinitions...),
		Details: append([]PendingIssueDetailDefinition(nil), pendingIssueDetailDefinitions...),
	}
}

func IsAllowedPendingIssueFilter(value string) bool {
	if value == "" {
		return false
	}
	if _, ok := pendingIssueDefinitionMap[PendingIssueKey(value)]; ok {
		return true
	}
	_, ok := pendingIssueDetailDefinitionMap[PendingIssueDetailKey(value)]
	return ok
}

func IsAllowedPendingIssueDetail(value string) bool {
	if value == "" {
		return false
	}
	_, ok := pendingIssueDetailDefinitionMap[PendingIssueDetailKey(value)]
	return ok
}

func EvaluatePendingIssues(game Game, ignoredReasons map[PendingIssueDetailKey]*string) PendingIssueEvaluation {
	details := make([]PendingIssueDetailState, 0, len(pendingIssueDetailDefinitions))
	visibleGroups := make(map[PendingIssueKey]struct{}, len(pendingIssueDefinitions))
	visibleDetailCount := 0

	appendDetail := func(key PendingIssueDetailKey) {
		definition, ok := pendingIssueDetailDefinitionMap[key]
		if !ok {
			return
		}
		reason, ignored := ignoredReasons[key]
		details = append(details, PendingIssueDetailState{
			Key:     key,
			Group:   definition.Group,
			Ignored: ignored,
			Reason:  reason,
		})
		if !ignored {
			visibleGroups[definition.Group] = struct{}{}
			visibleDetailCount += 1
		}
	}

	hasMeaningfulWikiContent := game.WikiContent != nil && *game.WikiContent != ""

	if game.CoverImage == nil || *game.CoverImage == "" {
		appendDetail(PendingIssueDetailMissingCover)
	}
	if game.BannerImage == nil || *game.BannerImage == "" {
		appendDetail(PendingIssueDetailMissingBanner)
	}
	if game.ScreenshotCount <= 0 && (game.PrimaryScreenshot == nil || *game.PrimaryScreenshot == "") {
		appendDetail(PendingIssueDetailMissingScreenshots)
	}
	if !hasMeaningfulWikiContent {
		appendDetail(PendingIssueDetailMissingWikiContent)
	}
	if game.FileCount <= 0 {
		appendDetail(PendingIssueDetailMissingFilesList)
	}
	if game.DeveloperCount <= 0 {
		appendDetail(PendingIssueDetailMissingDeveloper)
	}
	if game.PublisherCount <= 0 {
		appendDetail(PendingIssueDetailMissingPublisher)
	}
	if game.PlatformCount <= 0 {
		appendDetail(PendingIssueDetailMissingPlatform)
	}
	if game.Summary == nil || *game.Summary == "" {
		appendDetail(PendingIssueDetailMissingSummary)
	}

	groups := make([]PendingIssueKey, 0, len(pendingIssueDefinitions))
	for _, definition := range pendingIssueDefinitions {
		if _, ok := visibleGroups[definition.Key]; ok {
			groups = append(groups, definition.Key)
		}
	}

	severe := visibleDetailCount >= 3
	if !severe {
		_, hasFiles := visibleGroups[PendingIssueMissingFiles]
		_, hasAssets := visibleGroups[PendingIssueMissingAssets]
		_, hasWiki := visibleGroups[PendingIssueMissingWiki]
		severe = hasFiles || (hasAssets && hasWiki)
	}

	return PendingIssueEvaluation{
		Groups:  groups,
		Details: details,
		Severe:  severe,
	}
}

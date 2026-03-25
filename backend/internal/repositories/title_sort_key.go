package repositories

import (
	"encoding/hex"
	"strings"
	"sync"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

var (
	zhPinyinCollator   = collate.New(language.MustParse("zh-u-co-pinyin"))
	zhPinyinCollatorMu sync.Mutex
)

func buildTitleSortKey(title string, titleAlt *string) string {
	target := strings.TrimSpace(title)
	if target == "" && titleAlt != nil {
		target = strings.TrimSpace(*titleAlt)
	}
	if target == "" {
		return ""
	}

	zhPinyinCollatorMu.Lock()
	buffer := &collate.Buffer{}
	key := zhPinyinCollator.Key(buffer, []byte(target))
	zhPinyinCollatorMu.Unlock()

	return hex.EncodeToString(key)
}

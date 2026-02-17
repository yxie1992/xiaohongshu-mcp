package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/errors"
)

const (
	defaultSavedFeedsLimit       = 20
	maxSavedFeedsScrollRounds    = 20
	savedFeedsScrollStableRounds = 3
)

// SavedFeedsAction 负责获取当前账号的收藏笔记列表。
type SavedFeedsAction struct {
	page *rod.Page
}

func NewSavedFeedsAction(page *rod.Page) *SavedFeedsAction {
	pp := page.Timeout(20 * time.Second)
	return &SavedFeedsAction{page: pp}
}

// ListSavedFeeds 获取当前账号的收藏笔记，默认返回前 20 条。
func (a *SavedFeedsAction) ListSavedFeeds(ctx context.Context, limit int) ([]Feed, error) {
	if limit <= 0 {
		limit = defaultSavedFeedsLimit
	}

	page := a.page.Context(ctx)
	if err := a.navigateToSavedPage(ctx, page); err != nil {
		return nil, err
	}

	a.waitStable(page, 1200*time.Millisecond)

	if clicked := a.clickSavedTab(page); !clicked && !a.isOnSavedTab(page) {
		return nil, fmt.Errorf("failed to switch to saved tab")
	}

	a.waitStable(page, 1200*time.Millisecond)
	time.Sleep(1200 * time.Millisecond)

	feeds, err := a.extractSavedFeeds(page)
	if err != nil {
		return nil, err
	}
	feeds = a.filterOwnerPostsOnSavedTab(page, feeds)

	lastCount := -1
	stableRounds := 0

	for i := 0; i < maxSavedFeedsScrollRounds && len(feeds) < limit; i++ {
		if len(feeds) == lastCount {
			stableRounds++
		} else {
			stableRounds = 0
		}
		if stableRounds >= savedFeedsScrollStableRounds {
			break
		}
		lastCount = len(feeds)

		a.safeEvalBool(page, `() => { window.scrollBy(0, Math.max(window.innerHeight * 1.6, 1200)); return true; }`)
		time.Sleep(1 * time.Second)
		a.waitStable(page, 1200*time.Millisecond)

		updatedFeeds, extractErr := a.extractSavedFeeds(page)
		if extractErr != nil {
			continue
		}
		updatedFeeds = a.filterOwnerPostsOnSavedTab(page, updatedFeeds)
		if len(updatedFeeds) > len(feeds) {
			feeds = updatedFeeds
		}
	}

	if len(feeds) == 0 {
		return nil, errors.ErrNoFeeds
	}

	if len(feeds) > limit {
		feeds = feeds[:limit]
	}

	return feeds, nil
}

func (a *SavedFeedsAction) navigateToSavedPage(ctx context.Context, page *rod.Page) error {
	pp := page.Context(ctx)

	selectors := []string{
		`div.main-container li.user.side-bar-component a.link-wrapper span.channel`,
		`div.main-container li.user.side-bar-component a.link-wrapper`,
		`li.user.side-bar-component a`,
	}

	for attempt := 0; attempt < 3; attempt++ {
		logrus.Infof("saved_feeds: navigate attempt %d", attempt+1)
		if err := pp.Navigate("https://www.xiaohongshu.com/explore"); err != nil {
			logrus.Warnf("saved_feeds: navigate explore failed: %v", err)
			continue
		}
		_ = pp.WaitLoad()
		a.waitStable(pp, 1200*time.Millisecond)

		clicked := false
		for _, selector := range selectors {
			exists, el, err := pp.Timeout(5 * time.Second).Has(selector)
			if err != nil || !exists || el == nil {
				continue
			}
			if clickErr := el.Click(proto.InputMouseButtonLeft, 1); clickErr == nil {
				clicked = true
				break
			}
		}

		if !clicked {
			clicked = a.safeEvalBool(pp, `() => {
				const target = Array.from(document.querySelectorAll('li.side-bar-component span.channel, li.side-bar-component a')).find(el => {
					const t = (el.textContent || '').trim();
					return t === '我' || t === 'Me' || t === 'Mine';
				});
				if (!target) return false;
				target.click();
				return true;
			}`)
		}

		if !clicked {
			logrus.Infof("saved_feeds: profile click not found on attempt %d", attempt+1)
			time.Sleep(800 * time.Millisecond)
			continue
		}

		_ = pp.WaitLoad()
		a.waitStable(pp, 1200*time.Millisecond)
		if strings.Contains(a.currentURL(pp), "/user/profile/") {
			logrus.Infof("saved_feeds: profile page reached on attempt %d", attempt+1)
			return nil
		}
	}

	if a.navigateViaCookieCandidates(pp) {
		logrus.Info("saved_feeds: profile page reached via cookie candidates")
		return nil
	}

	return fmt.Errorf("failed to navigate to profile page")
}

func (a *SavedFeedsAction) clickSavedTab(page *rod.Page) bool {
	return a.safeEvalBool(page, `() => {
		const roots = [
			document.querySelector('#userPostedFeeds'),
			document.querySelector('.user-page'),
			document.body,
		].filter(Boolean);

		const isSavedText = (text) => {
			const t = (text || '').trim().toLowerCase();
			if (!t) return false;
			if (t === '收藏' || t.startsWith('收藏')) return true;
			if (t === 'saved' || t.startsWith('saved')) return true;
			return false;
		};

		for (const root of roots) {
			const target = Array.from(root.querySelectorAll('button,div,span,a')).find(el => {
				if (!isSavedText(el.textContent)) return false;
				const rect = el.getBoundingClientRect();
				return rect.width > 0 && rect.height > 0;
			});
			if (target) {
				target.click();
				return true;
			}
		}

		return false;
	}`)
}

func (a *SavedFeedsAction) extractSavedFeeds(page *rod.Page) ([]Feed, error) {
	result := a.safeEvalString(page, `() => {
		const state = window.__INITIAL_STATE__ || {};
		const user = state.user || {};

		function unwrap(v) {
			if (v && typeof v === 'object') {
				if ('value' in v) return v.value;
				if ('_value' in v) return v._value;
			}
			return v;
		}

		function normalize(arr) {
			if (!Array.isArray(arr)) return [];
			if (arr.length > 0 && Array.isArray(arr[0])) return arr.flat();
			return arr;
		}

		const notesFeeds = normalize(unwrap(user.notes));
		if (notesFeeds.length > 0) {
			return JSON.stringify(notesFeeds);
		}

		const savedKey = Object.keys(user).find(k => /collect|collec|favor|favo|saved|bookmark/i.test(k));
		if (savedKey) {
			const savedFeeds = normalize(unwrap(user[savedKey]));
			if (savedFeeds.length > 0) {
				return JSON.stringify(savedFeeds);
			}
		}

		return '';
	}`)

	if strings.TrimSpace(result) == "" {
		return nil, errors.ErrNoFeeds
	}

	var feeds []Feed
	if err := json.Unmarshal([]byte(result), &feeds); err != nil {
		return nil, fmt.Errorf("failed to unmarshal saved feeds: %w", err)
	}

	return feeds, nil
}

func (a *SavedFeedsAction) filterOwnerPostsOnSavedTab(page *rod.Page, feeds []Feed) []Feed {
	if len(feeds) == 0 {
		return feeds
	}

	if !a.isOnSavedTab(page) {
		return feeds
	}

	currentUserID := a.profileUserID(page)
	if currentUserID == "" {
		currentUserID = a.currentUserID(page)
	}
	if currentUserID == "" {
		return feeds
	}

	return trimLeadingFeedsByAuthor(feeds, currentUserID)
}

func (a *SavedFeedsAction) profileUserID(page *rod.Page) string {
	currentURL := strings.TrimSpace(a.currentURL(page))
	if currentURL == "" {
		return ""
	}

	parsed, err := url.Parse(currentURL)
	if err != nil {
		return ""
	}

	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) >= 3 && parts[0] == "user" && parts[1] == "profile" {
		return strings.TrimSpace(parts[2])
	}

	return ""
}

func (a *SavedFeedsAction) currentUserID(page *rod.Page) string {
	return strings.TrimSpace(a.safeEvalString(page, `() => {
		const userInfo = (window.__INITIAL_STATE__ || {}).user?.userInfo || {};
		return userInfo.userId || userInfo.userid || '';
	}`))
}

func (a *SavedFeedsAction) isOnSavedTab(page *rod.Page) bool {
	currentURL := a.currentURL(page)
	if strings.Contains(currentURL, "tab=fav") {
		return true
	}

	return a.safeEvalBool(page, `() => {
		const user = (window.__INITIAL_STATE__ || {}).user || {};
		function unwrap(v) {
			if (v && typeof v === 'object') {
				if ('value' in v) return v.value;
				if ('_value' in v) return v._value;
			}
			return v;
		}
		const activeTab = unwrap(user.activeTab);
		return String(activeTab || '').toLowerCase() === 'fav';
	}`)
}

func (a *SavedFeedsAction) currentURL(page *rod.Page) string {
	info, err := page.Info()
	if err != nil || info == nil {
		return ""
	}
	return info.URL
}

func (a *SavedFeedsAction) safeEvalString(page *rod.Page, js string) (result string) {
	obj, err := page.Eval(js)
	if err != nil || obj == nil {
		return ""
	}
	return obj.Value.String()
}

func (a *SavedFeedsAction) safeEvalBool(page *rod.Page, js string) (result bool) {
	obj, err := page.Eval(js)
	if err != nil || obj == nil {
		return false
	}
	return obj.Value.Bool()
}

func (a *SavedFeedsAction) waitStable(page *rod.Page, stableFor time.Duration) {
	_ = page.Timeout(5 * time.Second).WaitStable(stableFor)
}

func (a *SavedFeedsAction) navigateViaCookieCandidates(page *rod.Page) bool {
	idsRaw := a.safeEvalString(page, `() => {
		const cookie = document.cookie || '';
		const matches = cookie.match(/[0-9a-f]{24}/ig) || [];
		const uniq = Array.from(new Set(matches));
		return uniq.slice(0, 12).join(',');
	}`)
	if strings.TrimSpace(idsRaw) == "" {
		logrus.Warn("saved_feeds: cookie candidates empty")
		return false
	}

	logrus.Infof("saved_feeds: cookie candidates=%s", idsRaw)

	bestURL := ""
	bestScore := 0

	for _, id := range strings.Split(idsRaw, ",") {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		targetURL := fmt.Sprintf("https://www.xiaohongshu.com/user/profile/%s?tab=fav&subTab=note", id)
		if err := page.Navigate(targetURL); err != nil {
			logrus.Warnf("saved_feeds: candidate navigate failed id=%s err=%v", id, err)
			continue
		}
		_ = page.WaitLoad()
		a.waitStable(page, 900*time.Millisecond)

		feeds, err := a.extractSavedFeeds(page)
		if err != nil || len(feeds) == 0 {
			logrus.Infof("saved_feeds: candidate id=%s no feeds err=%v", id, err)
			continue
		}

		score := distinctAuthorCount(feeds)
		logrus.Infof("saved_feeds: candidate id=%s feeds=%d distinct_authors=%d", id, len(feeds), score)
		if score > bestScore {
			bestScore = score
			bestURL = targetURL
		}
	}

	if bestURL == "" || bestScore == 0 {
		logrus.Warn("saved_feeds: no valid cookie candidate profile")
		return false
	}

	if err := page.Navigate(bestURL); err != nil {
		return false
	}
	_ = page.WaitLoad()
	a.waitStable(page, 900*time.Millisecond)
	return true
}

func distinctAuthorCount(feeds []Feed) int {
	authors := make(map[string]struct{})
	for _, feed := range feeds {
		uid := strings.TrimSpace(feed.NoteCard.User.UserID)
		if uid == "" {
			continue
		}
		authors[uid] = struct{}{}
	}
	return len(authors)
}

func trimLeadingFeedsByAuthor(feeds []Feed, authorID string) []Feed {
	authorID = strings.TrimSpace(authorID)
	if len(feeds) == 0 || authorID == "" {
		return feeds
	}

	trimIndex := 0
	for trimIndex < len(feeds) {
		currentAuthorID := strings.TrimSpace(feeds[trimIndex].NoteCard.User.UserID)
		if currentAuthorID != authorID {
			break
		}
		trimIndex++
	}

	// 只裁剪开头的本人笔记，避免全部过滤为空导致误判。
	if trimIndex > 0 && trimIndex < len(feeds) {
		return feeds[trimIndex:]
	}

	return feeds
}

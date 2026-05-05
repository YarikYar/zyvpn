package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"sync"
	"time"

	tele "gopkg.in/telebot.v3"

	"github.com/zyvpn/backend/internal/model"
)

// appSpec describes which GitHub repo and which release asset to ship for a
// given app_key. The pattern is matched against asset.name; first matching
// asset wins. Each spec also carries a human label and a platform icon for
// the inline button.
type appSpec struct {
	key         string
	label       string
	repo        string         // owner/name on GitHub
	pattern     *regexp.Regexp // matches asset.name on the release
	maxFileSize int64          // safety cap (bytes); 0 = no check
}

const telegramBotMaxUpload = 50 * 1024 * 1024 // 50 MB sendDocument limit

// appCatalog is the curated list of executable client apps we redistribute.
// Universal v2rayNG (~61 MB) doesn't fit Telegram's 50 MB cap, so we ship
// the arm64-v8a build which covers the vast majority of Android phones.
var appCatalog = []appSpec{
	{
		key:         "v2rayng_arm64",
		label:       "🤖 v2rayNG (Android arm64)",
		repo:        "2dust/v2rayNG",
		pattern:     regexp.MustCompile(`^v2rayNG_[\d.]+_arm64-v8a\.apk$`),
		maxFileSize: telegramBotMaxUpload,
	},
	{
		key:         "throne_win",
		label:       "💻 Throne (Windows installer)",
		repo:        "throneproj/Throne",
		pattern:     regexp.MustCompile(`^Throne-[\d.]+-windows64-installer\.exe$`),
		maxFileSize: telegramBotMaxUpload,
	},
	{
		key:         "throne_mac",
		label:       "🍏 Throne (macOS arm64)",
		repo:        "throneproj/Throne",
		pattern:     regexp.MustCompile(`^Throne-[\d.]+-macos-arm64\.zip$`),
		maxFileSize: telegramBotMaxUpload,
	},
	{
		key:         "throne_linux",
		label:       "🐧 Throne (Linux .deb)",
		repo:        "throneproj/Throne",
		pattern:     regexp.MustCompile(`^Throne-[\d.]+-debian-amd64\.deb$`),
		maxFileSize: telegramBotMaxUpload,
	},
}

func appSpecByKey(key string) *appSpec {
	for i := range appCatalog {
		if appCatalog[i].key == key {
			return &appCatalog[i]
		}
	}
	return nil
}

// In-flight deduplication: when several users tap «Скачать» at once, only
// one goroutine actually downloads/uploads, others reuse the cached
// file_id afterwards.
var (
	sendInFlight   = map[string]*sync.Mutex{}
	sendInFlightMu sync.Mutex
)

func keyMutex(appKey string) *sync.Mutex {
	sendInFlightMu.Lock()
	defer sendInFlightMu.Unlock()
	mu, ok := sendInFlight[appKey]
	if !ok {
		mu = &sync.Mutex{}
		sendInFlight[appKey] = mu
	}
	return mu
}

type ghAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
	Size int64  `json:"size"`
}

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

func fetchLatestRelease(ctx context.Context, repo string) (*ghRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "zyvpn-bot")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", repo, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("github %s status %d", repo, resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release: %w", err)
	}
	return &rel, nil
}

// sendAppDocument resolves the latest release for `appKey`, sends the
// matching asset to the chat as a Document, and caches the resulting
// Telegram file_id so subsequent sends are instant. Safe to call
// concurrently — per-key in-flight deduplication serialises the heavy path.
func (b *Bot) sendAppDocument(ctx context.Context, c tele.Context, appKey string) error {
	spec := appSpecByKey(appKey)
	if spec == nil {
		return c.Respond(&tele.CallbackResponse{Text: "Неизвестное приложение", ShowAlert: true})
	}

	mu := keyMutex(appKey)
	mu.Lock()
	defer mu.Unlock()

	// Acknowledge the callback so the user sees the spinner stop.
	_ = c.Respond(&tele.CallbackResponse{Text: "Качаю..."})

	rel, err := fetchLatestRelease(ctx, spec.repo)
	if err != nil {
		log.Printf("[apps] fetch release %s failed: %v", appKey, err)
		return c.Send(fmt.Sprintf("⚠️ Не удалось получить релиз с GitHub: %v", err))
	}

	var asset *ghAsset
	for i := range rel.Assets {
		if spec.pattern.MatchString(rel.Assets[i].Name) {
			asset = &rel.Assets[i]
			break
		}
	}
	if asset == nil {
		return c.Send(fmt.Sprintf("⚠️ В релизе %s нет файла под шаблон `%s`", rel.TagName, spec.pattern))
	}

	if spec.maxFileSize > 0 && asset.Size > spec.maxFileSize {
		return c.Send(fmt.Sprintf(
			"Файл %s весит %d МБ — Telegram-боту нельзя отправлять >50 МБ. Скачай вручную:\n%s",
			asset.Name, asset.Size/1024/1024, asset.URL,
		))
	}

	// 1) Try cache.
	cached, err := b.repo.GetAppRelease(ctx, spec.key, rel.TagName)
	if err != nil {
		log.Printf("[apps] cache lookup %s/%s failed: %v", spec.key, rel.TagName, err)
	}
	if cached != nil && cached.TelegramFileID != nil && *cached.TelegramFileID != "" {
		doc := &tele.Document{
			File:     tele.File{FileID: *cached.TelegramFileID},
			FileName: asset.Name,
			Caption:  fmt.Sprintf("%s — %s", spec.label, rel.TagName),
		}
		return c.Send(doc)
	}

	// 2) Download + upload.
	row := &model.AppRelease{
		AppKey:    spec.key,
		Tag:       rel.TagName,
		AssetName: asset.Name,
		AssetURL:  asset.URL,
		FileSize:  &asset.Size,
	}
	if err := b.repo.UpsertAppRelease(ctx, row); err != nil {
		log.Printf("[apps] cache upsert pre-send %s failed: %v", spec.key, err)
	}

	body, err := downloadFile(ctx, asset.URL)
	if err != nil {
		log.Printf("[apps] download %s failed: %v", asset.URL, err)
		return c.Send(fmt.Sprintf("⚠️ Не удалось скачать %s: %v", asset.Name, err))
	}
	defer body.Close()

	doc := &tele.Document{
		File:     tele.FromReader(body),
		FileName: asset.Name,
		Caption:  fmt.Sprintf("%s — %s", spec.label, rel.TagName),
	}
	msg, err := b.bot.Send(c.Recipient(), doc)
	if err != nil {
		log.Printf("[apps] sendDocument %s failed: %v", asset.Name, err)
		return c.Send(fmt.Sprintf("⚠️ Не удалось отправить файл: %v", err))
	}

	if msg.Document != nil && msg.Document.FileID != "" {
		if err := b.repo.SetAppReleaseFileID(ctx, spec.key, rel.TagName, msg.Document.FileID); err != nil {
			log.Printf("[apps] cache file_id %s/%s failed: %v", spec.key, rel.TagName, err)
		} else {
			log.Printf("[apps] cached %s/%s file_id=%s", spec.key, rel.TagName, msg.Document.FileID)
		}
	}
	return nil
}

func downloadFile(ctx context.Context, url string) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "zyvpn-bot")

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download status %d", resp.StatusCode)
	}
	return resp.Body, nil
}

package i18n

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Locale struct {
	App      map[string]string `json:"app"`
	Commands map[string]string `json:"commands"`
	Usage    map[string]string `json:"usage"`
	Options  map[string]string `json:"options"`
	Errors   map[string]string `json:"errors"`
	Tunnel   map[string]string `json:"tunnel"`
	Status   map[string]string `json:"status"`
	Messages map[string]string `json:"messages"`
	CLI      map[string]string `json:"cli"`
}

var (
	locales     = make(map[string]*Locale)
	currentLang string
	mu          sync.RWMutex
)

func init() {
	LoadLocales()
}

func LoadLocales() {
	mu.Lock()
	defer mu.Unlock()

	langs := []string{"en", "fr", "es"}
	for _, lang := range langs {
		loadLocale(lang)
	}
}

func loadLocale(lang string) {
	data, err := os.ReadFile(filepath.Join("internal", "i18n", "locales", lang+".json"))
	if err != nil {
		data, err = os.ReadFile(filepath.Join("locales", lang+".json"))
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not load locale %s: %v\n", lang, err)
		return
	}

	var locale Locale
	if err := json.Unmarshal(data, &locale); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not parse locale %s: %v\n", lang, err)
		return
	}

	locales[lang] = &locale
}

func SetLang(lang string) {
	mu.Lock()
	defer mu.Unlock()

	supported := map[string]bool{"en": true, "fr": true, "es": true}
	if !supported[lang] {
		lang = "en"
	}

	currentLang = lang
}

func GetLang() string {
	mu.RLock()
	defer mu.RUnlock()
	return currentLang
}

func DetectLang() string {
	langEnv := strings.ToLower(os.Getenv("LANG"))
	if strings.HasPrefix(langEnv, "fr") {
		return "fr"
	}
	if strings.HasPrefix(langEnv, "es") {
		return "es"
	}
	return "en"
}

func TServer(key string) string {
	return T("server." + key)
}

func T(key string) string {
	mu.RLock()
	defer mu.RUnlock()

	if currentLang == "" {
		currentLang = DetectLang()
	}

	locale, ok := locales[currentLang]
	if !ok {
		locale = locales["en"]
	}

	keys := parseKey(key)

	switch keys[0] {
	case "app":
		return locale.App[keys[1]]
	case "commands":
		return locale.Commands[keys[1]]
	case "usage":
		return locale.Usage[keys[1]]
	case "options":
		return locale.Options[keys[1]]
	case "errors":
		return locale.Errors[keys[1]]
	case "tunnel":
		return locale.Tunnel[keys[1]]
	case "status":
		return locale.Status[keys[1]]
	case "messages":
		return locale.Messages[keys[1]]
	case "cli":
		return locale.CLI[keys[1]]
	}

	return key
}

func TStatus(status string) string {
	return T("status." + status)
}

func TError(key string) string {
	return T("errors." + key)
}

func TCommand(cmd string) string {
	return T("commands." + cmd)
}

func TOption(opt string) string {
	return T("options." + opt)
}

func parseKey(key string) []string {
	var keys []string
	var current string
	for _, c := range key {
		if c == '.' {
			keys = append(keys, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	keys = append(keys, current)
	return keys
}

var once sync.Once

func Init() {
	once.Do(func() {
		lang := DetectLang()
		if envLang := os.Getenv("ATUNNELS_LANG"); envLang != "" {
			lang = envLang
		}
		SetLang(lang)
	})
}

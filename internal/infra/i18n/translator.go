package i18n

import (
	"fmt"
	"strings"
)

// Translator provides internationalization support for the application.
type Translator struct {
	locales        map[string]map[string]string
	locale         string
	fallbackLocale string
}

// Option is a functional option for configuring Translator.
type Option func(*Translator)

// WithFallbackLocale sets the fallback locale.
func WithFallbackLocale(locale string) Option {
	return func(t *Translator) {
		t.fallbackLocale = locale
	}
}

// NewTranslator creates a new Translator instance.
func NewTranslator(locales map[string]map[string]string, opts ...Option) *Translator {
	t := &Translator{
		locales:        locales,
		locale:         "en",
		fallbackLocale: "en",
	}

	for _, opt := range opts {
		opt(t)
	}

	return t
}

// Get returns the translation for the given key.
// If parameters are provided, they are used for formatting.
func (t *Translator) Get(key string, args ...interface{}) string {
	value := t.getTranslation(t.locale, key)

	// Fallback to default locale if key not found in current locale
	if value == "" && t.locale != t.fallbackLocale {
		value = t.getTranslation(t.fallbackLocale, key)
	}

	// Return key if no translation found
	if value == "" {
		return key
	}

	// Format with parameters if provided
	if len(args) > 0 {
		return fmt.Sprintf(value, args...)
	}

	return value
}

// getTranslation retrieves translation for a specific locale and key.
func (t *Translator) getTranslation(locale, key string) string {
	localeMap, ok := t.locales[locale]
	if !ok {
		return ""
	}

	value, ok := localeMap[key]
	if !ok {
		return ""
	}

	return value
}

// SetLocale sets the current locale.
func (t *Translator) SetLocale(locale string) {
	// Validate locale exists
	if _, ok := t.locales[locale]; !ok {
		locale = t.fallbackLocale
	}
	t.locale = locale
}

// GetLocale returns the current locale.
func (t *Translator) GetLocale() string {
	return t.locale
}

// GetAvailableLocales returns a list of available locales.
func (t *Translator) GetAvailableLocales() []string {
	var locales []string
	for locale := range t.locales {
		locales = append(locales, locale)
	}
	return locales
}

// DetectLocale detects the appropriate locale based on HTTP headers or query parameters.
// Priority: 1. Query parameter 2. Accept-Language header 3. Default locale
func (t *Translator) DetectLocale(queryLocale, acceptLanguage string) string {
	// 1. Check query parameter first
	if queryLocale != "" {
		if _, ok := t.locales[queryLocale]; ok {
			return queryLocale
		}
	}

	// 2. Parse Accept-Language header
	if acceptLanguage != "" {
		detected := t.matchAcceptLanguage(acceptLanguage)
		if detected != "" {
			return detected
		}
	}

	// 3. Return default locale
	return t.fallbackLocale
}

// matchAcceptLanguage matches the best locale from Accept-Language header.
func (t *Translator) matchAcceptLanguage(header string) string {
	// Parse Accept-Language header: "en-US,en;q=0.9,pt-BR;q=0.8"
	parts := strings.Split(header, ",")

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Handle quality value: "en;q=0.9"
		locale := part
		if idx := strings.Index(part, ";"); idx != -1 {
			locale = strings.TrimSpace(part[:idx])
		}

		// Try exact match
		if _, ok := t.locales[locale]; ok {
			return locale
		}

		// Try language only (e.g., "en-US" -> "en")
		if idx := strings.Index(locale, "-"); idx != -1 {
			lang := locale[:idx]
			if _, ok := t.locales[lang]; ok {
				return lang
			}
		}
	}

	return ""
}

// HasLocale checks if a locale is available.
func (t *Translator) HasLocale(locale string) bool {
	_, ok := t.locales[locale]
	return ok
}

// GetAllTranslations returns all translations for the current locale.
func (t *Translator) GetAllTranslations() map[string]string {
	if localeMap, ok := t.locales[t.locale]; ok {
		return localeMap
	}
	return make(map[string]string)
}

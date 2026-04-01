/**
 * Nivoxis i18n — frontend translation helper.
 *
 * Usage:
 *   window.t("nav.dashboard")  // returns translated string
 *   window.setLocale("nl")     // switches language, reloads translations
 *
 * Translations are fetched from /api/i18n/{locale} on page load.
 * The current locale is stored in localStorage("nivoxis-locale") and
 * falls back to the server-rendered value from the <html lang="..."> tag.
 */
(function () {
    var _translations = {};
    var _locale = document.documentElement.lang || "en";
    var _loading = null;

    function loadTranslations(locale, cb) {
        _loading = $.ajax({
            url: "/api/i18n/" + locale,
            method: "GET",
            dataType: "json",
            beforeSend: function (xhr) {
                if (window.user && window.user.api_key) {
                    xhr.setRequestHeader("Authorization", "Bearer " + window.user.api_key);
                }
            }
        }).done(function (data) {
            _translations = data || {};
            _locale = locale;
            localStorage.setItem("nivoxis-locale", locale);
            if (typeof cb === "function") cb();
        }).fail(function () {
            console.warn("i18n: failed to load translations for locale:", locale);
        });
        return _loading;
    }

    /**
     * t(key) — translate a key using the current locale's translations.
     * Returns the key itself if no translation is found.
     */
    function t(key) {
        if (_translations && _translations[key]) {
            return _translations[key];
        }
        return key;
    }

    /**
     * setLocale(locale) — switch to a new language.
     * Updates the user's preference via API, reloads translations, refreshes the page.
     */
    function setLocale(locale) {
        // Update server-side preference
        if (window.user && window.user.api_key) {
            $.ajax({
                url: "/api/user/language",
                method: "PUT",
                data: JSON.stringify({ preferred_language: locale }),
                dataType: "json",
                contentType: "application/json",
                beforeSend: function (xhr) {
                    xhr.setRequestHeader("Authorization", "Bearer " + window.user.api_key);
                }
            }).always(function () {
                localStorage.setItem("nivoxis-locale", locale);
                window.location.reload();
            });
        } else {
            localStorage.setItem("nivoxis-locale", locale);
            window.location.reload();
        }
    }

    // Expose globally
    window.t = t;
    window.setLocale = setLocale;
    window.loadTranslations = loadTranslations;
    window._nivoxisLocale = _locale;

    // Auto-load translations on DOM ready
    $(document).ready(function () {
        var storedLocale = localStorage.getItem("nivoxis-locale");
        var pageLocale = document.documentElement.lang || "en";
        var locale = storedLocale || pageLocale;
        loadTranslations(locale);
    });
})();

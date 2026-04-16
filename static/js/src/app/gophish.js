function errorFlash(message) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function successFlash(message) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-success\">\
        <i class=\"fa fa-check-circle\"></i> " + message + "</div>")
}

// Fade message after n seconds
function errorFlashFade(message, fade) {
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
    setTimeout(function(){ 
        $("#flashes").empty() 
    }, fade * 1000);
}
// Fade message after n seconds
function successFlashFade(message, fade) {  
    $("#flashes").empty()
    $("#flashes").append("<div style=\"text-align:center\" class=\"alert alert-success\">\
        <i class=\"fa fa-check-circle\"></i> " + message + "</div>")
    setTimeout(function(){ 
        $("#flashes").empty() 
    }, fade * 1000);

}

function modalError(message) {
    $("#modal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function query(endpoint, method, data, async) {
    return $.ajax({
        url: "/api" + endpoint,
        async: async,
        method: method,
        data: JSON.stringify(data),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    })
}

function escapeHtml(text) {
    return $("<div/>").text(text).html()
}
window.escapeHtml = escapeHtml

function unescapeHtml(html) {
    return $("<div/>").html(html).text()
}

/**
 * 
 * @param {string} string - The input string to capitalize
 * 
 */
var capitalize = function (string) {
    return string.charAt(0).toUpperCase() + string.slice(1);
}

/*
Define our API Endpoints
*/
var api = {
    // campaigns contains the endpoints for /campaigns
    campaigns: {
        // get() - Queries the API for GET /campaigns
        get: function () {
            return query("/campaigns/", "GET", {}, false)
        },
        // post() - Posts a campaign to POST /campaigns
        post: function (data) {
            return query("/campaigns/", "POST", data, false)
        },
        // summary() - Queries the API for GET /campaigns/summary
        summary: function () {
            return query("/campaigns/summary", "GET", {}, false)
        }
    },
    // campaignId contains the endpoints for /campaigns/:id
    campaignId: {
        // get() - Queries the API for GET /campaigns/:id
        get: function (id) {
            return query("/campaigns/" + id, "GET", {}, true)
        },
        // delete() - Deletes a campaign at DELETE /campaigns/:id
        delete: function (id) {
            return query("/campaigns/" + id, "DELETE", {}, false)
        },
        // results() - Queries the API for GET /campaigns/:id/results
        results: function (id) {
            return query("/campaigns/" + id + "/results", "GET", {}, true)
        },
        // complete() - Completes a campaign at POST /campaigns/:id/complete
        complete: function (id) {
            return query("/campaigns/" + id + "/complete", "GET", {}, true)
        },
        // summary() - Queries the API for GET /campaigns/summary
        summary: function (id) {
            return query("/campaigns/" + id + "/summary", "GET", {}, true)
        },
        // ── Campaign Advanced Analytics ──
        // funnel() - GET /campaigns/:id/analytics/funnel
        funnel: function (id) {
            return query("/campaigns/" + id + "/analytics/funnel", "GET", {}, true)
        },
        // timeToClick() - GET /campaigns/:id/analytics/time-to-click
        timeToClick: function (id) {
            return query("/campaigns/" + id + "/analytics/time-to-click", "GET", {}, true)
        },
        // repeatOffenders() - GET /campaigns/:id/analytics/repeat-offenders
        repeatOffenders: function (id) {
            return query("/campaigns/" + id + "/analytics/repeat-offenders", "GET", {}, true)
        },
        // deviceBreakdown() - GET /campaigns/:id/analytics/devices
        deviceBreakdown: function (id) {
            return query("/campaigns/" + id + "/analytics/devices", "GET", {}, true)
        }
    },
    // groups contains the endpoints for /groups
    groups: {
        // get() - Queries the API for GET /groups
        get: function () {
            return query("/groups/", "GET", {}, false)
        },
        // post() - Posts a group to POST /groups
        post: function (group) {
            return query("/groups/", "POST", group, false)
        },
        // summary() - Queries the API for GET /groups/summary
        summary: function () {
            return query("/groups/summary", "GET", {}, true)
        }
    },
    // groupId contains the endpoints for /groups/:id
    groupId: {
        // get() - Queries the API for GET /groups/:id
        get: function (id) {
            return query("/groups/" + id, "GET", {}, false)
        },
        // put() - Puts a group to PUT /groups/:id
        put: function (group) {
            return query("/groups/" + group.id, "PUT", group, false)
        },
        // delete() - Deletes a group at DELETE /groups/:id
        delete: function (id) {
            return query("/groups/" + id, "DELETE", {}, false)
        }
    },
    // templates contains the endpoints for /templates
    templates: {
        // get() - Queries the API for GET /templates
        get: function () {
            return query("/templates/", "GET", {}, false)
        },
        // post() - Posts a template to POST /templates
        post: function (template) {
            return query("/templates/", "POST", template, false)
        }
    },
    // templateId contains the endpoints for /templates/:id
    templateId: {
        // get() - Queries the API for GET /templates/:id
        get: function (id) {
            return query("/templates/" + id, "GET", {}, false)
        },
        // put() - Puts a template to PUT /templates/:id
        put: function (template) {
            return query("/templates/" + template.id, "PUT", template, false)
        },
        // delete() - Deletes a template at DELETE /templates/:id
        delete: function (id) {
            return query("/templates/" + id, "DELETE", {}, false)
        }
    },
    // pages contains the endpoints for /pages
    pages: {
        // get() - Queries the API for GET /pages
        get: function () {
            return query("/pages/", "GET", {}, false)
        },
        // post() - Posts a page to POST /pages
        post: function (page) {
            return query("/pages/", "POST", page, false)
        }
    },
    // pageId contains the endpoints for /pages/:id
    pageId: {
        // get() - Queries the API for GET /pages/:id
        get: function (id) {
            return query("/pages/" + id, "GET", {}, false)
        },
        // put() - Puts a page to PUT /pages/:id
        put: function (page) {
            return query("/pages/" + page.id, "PUT", page, false)
        },
        // delete() - Deletes a page at DELETE /pages/:id
        delete: function (id) {
            return query("/pages/" + id, "DELETE", {}, false)
        }
    },
    // feedbackPages contains the endpoints for /feedback_pages
    feedbackPages: {
        get: function () {
            return query("/feedback_pages/", "GET", {}, false)
        },
        post: function (fp) {
            return query("/feedback_pages/", "POST", fp, false)
        },
        getDefault: function (lang) {
            return query("/feedback_pages/default?lang=" + (lang || "en"), "GET", {}, false)
        }
    },
    feedbackPageId: {
        get: function (id) {
            return query("/feedback_pages/" + id, "GET", {}, false)
        },
        put: function (fp) {
            return query("/feedback_pages/" + fp.id, "PUT", fp, false)
        },
        delete: function (id) {
            return query("/feedback_pages/" + id, "DELETE", {}, false)
        }
    },
    // SMTP contains the endpoints for /smtp
    SMTP: {
        // get() - Queries the API for GET /smtp
        get: function () {
            return query("/smtp/", "GET", {}, false)
        },
        // post() - Posts a SMTP to POST /smtp
        post: function (smtp) {
            return query("/smtp/", "POST", smtp, false)
        }
    },
    // SMTPId contains the endpoints for /smtp/:id
    SMTPId: {
        // get() - Queries the API for GET /smtp/:id
        get: function (id) {
            return query("/smtp/" + id, "GET", {}, false)
        },
        // put() - Puts a SMTP to PUT /smtp/:id
        put: function (smtp) {
            return query("/smtp/" + smtp.id, "PUT", smtp, false)
        },
        // delete() - Deletes a SMTP at DELETE /smtp/:id
        delete: function (id) {
            return query("/smtp/" + id, "DELETE", {}, false)
        }
    },
    // SMSProviders contains the endpoints for /sms
    SMSProviders: {
        get: function () {
            return query("/sms/", "GET", {}, false)
        },
        post: function (sp) {
            return query("/sms/", "POST", sp, false)
        }
    },
    // SMSProviderId contains the endpoints for /sms/:id
    SMSProviderId: {
        get: function (id) {
            return query("/sms/" + id, "GET", {}, false)
        },
        put: function (sp) {
            return query("/sms/" + sp.id, "PUT", sp, false)
        },
        delete: function (id) {
            return query("/sms/" + id, "DELETE", {}, false)
        }
    },
    // TemplateLibrary contains the endpoints for /template-library
    TemplateLibrary: {
        get: function (category, difficulty) {
            var params = []
            if (category) params.push("category=" + encodeURIComponent(category))
            if (difficulty) params.push("difficulty=" + difficulty)
            var qs = params.length ? "?" + params.join("&") : ""
            return query("/template-library/" + qs, "GET", {}, false)
        },
        categories: function () {
            return query("/template-library/categories", "GET", {}, false)
        },
        import: function (slug, name) {
            var data = {}
            if (name) data.name = name
            return query("/template-library/" + slug + "/import", "POST", data, false)
        }
    },
    // Compliance contains endpoints for /compliance
    Compliance: {
        frameworks: function () {
            return query("/compliance/frameworks", "GET", {}, false)
        },
        orgFrameworks: function () {
            return query("/compliance/org-frameworks", "GET", {}, false)
        },
        enableFramework: function (frameworkId) {
            return query("/compliance/org-frameworks", "POST", { framework_id: frameworkId }, false)
        },
        disableFramework: function (frameworkId) {
            return query("/compliance/org-frameworks/" + frameworkId + "/disable", "POST", {}, false)
        },
        dashboard: function () {
            return query("/compliance/dashboard", "GET", {}, false)
        },
        frameworkDetail: function (id) {
            return query("/compliance/frameworks/" + id + "/detail", "GET", {}, false)
        },
        assess: function (frameworkId) {
            return query("/compliance/frameworks/" + frameworkId + "/assess", "POST", {}, false)
        },
        manualAssess: function (controlId, data) {
            return query("/compliance/controls/" + controlId + "/assess", "POST", data, false)
        }
    },
    // PowerBI contains endpoints for /powerbi
    PowerBI: {
        feed: function (dataset, params) {
            var qs = "?dataset=" + encodeURIComponent(dataset)
            if (params) {
                for (var k in params) {
                    qs += "&" + k + "=" + encodeURIComponent(params[k])
                }
            }
            return query("/powerbi/" + qs, "GET", {}, false)
        }
    },
    // IMAP containts the endpoints for /imap/
    IMAP: {
        get: function() {
            return query("/imap/", "GET", {}, !1)
        },
        post: function(e) {
            return query("/imap/", "POST", e, !1)
        },
        validate: function(e) {
            return query("/imap/validate", "POST", e, true)
        }
    },
    // users contains the endpoints for /users
    users: {
        // get() - Queries the API for GET /users
        get: function () {
            return query("/users/", "GET", {}, true)
        },
        // post() - Posts a user to POST /users
        post: function (user) {
            return query("/users/", "POST", user, true)
        }
    },
    // roles contains the endpoints for /roles
    roles: {
        // get() - Queries the API for GET /roles
        get: function () {
            return query("/roles/", "GET", {}, true)
        }
    },
    // userId contains the endpoints for /users/:id
    userId: {
        // get() - Queries the API for GET /users/:id
        get: function (id) {
            return query("/users/" + id, "GET", {}, true)
        },
        // put() - Puts a user to PUT /users/:id
        put: function (user) {
            return query("/users/" + user.id, "PUT", user, true)
        },
        // delete() - Deletes a user at DELETE /users/:id
        delete: function (id) {
            return query("/users/" + id, "DELETE", {}, true)
        }
    },
    webhooks: {
        get: function() {
            return query("/webhooks/", "GET", {}, false)
        },
        post: function(webhook) {
            return query("/webhooks/", "POST", webhook, false)
        },
    },
    webhookId: {
        get: function(id) {
            return query("/webhooks/" + id, "GET", {}, false)
        },
        put: function(webhook) {
            return query("/webhooks/" + webhook.id, "PUT", webhook, true)
        },
        delete: function(id) {
            return query("/webhooks/" + id, "DELETE", {}, false)
        },
        ping: function(id) {
            return query("/webhooks/" + id + "/validate", "POST", {}, true)
        },
    },
    // import handles all of the "import" functions in the api
    import_email: function (req) {
        return query("/import/email", "POST", req, false)
    },
    // clone_site handles importing a site by url
    clone_site: function (req) {
        return query("/import/site", "POST", req, false)
    },
    // send_test_email sends an email to the specified email address
    send_test_email: function (req) {
        return query("/util/send_test_email", "POST", req, true)
    },
    reset: function () {
        return query("/reset", "POST", {}, true)
    },
    // Training presentations
    trainingPresentations: {
        get: function () {
            return query("/training/", "GET", {}, true)
        }
    },
    // Quiz endpoints
    quiz: {
        get: function (presentationId) {
            return query("/training/" + presentationId + "/quiz", "GET", {}, true)
        },
        post: function (presentationId, data) {
            return query("/training/" + presentationId + "/quiz", "POST", data, false)
        },
        delete: function (presentationId) {
            return query("/training/" + presentationId + "/quiz", "DELETE", {}, false)
        }
    },
    quizAttempt: {
        get: function (presentationId) {
            return query("/training/" + presentationId + "/quiz/attempt", "GET", {}, true)
        },
        post: function (presentationId, data) {
            return query("/training/" + presentationId + "/quiz/attempt", "POST", data, false)
        }
    },
    // Assignment endpoints
    assignments: {
        get: function () {
            return query("/training/assignments/", "GET", {}, true)
        },
        post: function (assignment) {
            return query("/training/assignments/", "POST", assignment, false)
        },
        delete: function (id) {
            return query("/training/assignments/" + id, "DELETE", {}, false)
        },
        assignGroup: function (data) {
            return query("/training/assignments/group", "POST", data, false)
        },
        mine: function () {
            return query("/training/my-assignments", "GET", {}, true)
        }
    },
    // Certificate endpoints
    certificates: {
        verify: function (code) {
            return query("/training/certificates/verify/" + code, "GET", {}, true)
        },
        mine: function () {
            return query("/training/my-certificates", "GET", {}, true)
        }
    },
    // Real-time dashboard endpoints
    dashboard: {
        metrics: function (window) {
            return query("/dashboard/metrics?window=" + (window || "30d"), "GET", {}, true)
        },
        sparkline: function (metric, window) {
            return query("/dashboard/sparkline?metric=" + metric + "&window=" + (window || "7d"), "GET", {}, true)
        },
        preference: function () {
            return query("/dashboard/preference", "GET", {}, true)
        },
        setPreference: function (timeWindow) {
            return query("/dashboard/preference", "PUT", { time_window: timeWindow }, true)
        },
        liveCounts: function () {
            return query("/dashboard/live-counts", "GET", {}, true)
        },
        wsStatus: function () {
            return query("/dashboard/ws-status", "GET", {}, true)
        }
    },
    // Report endpoints
    reports: {
        overview: function () {
            return query("/reports/overview", "GET", {}, true)
        },
        trend: function (days) {
            return query("/reports/trend?days=" + (days || 30), "GET", {}, true)
        },
        riskScores: function () {
            return query("/reports/risk-scores", "GET", {}, true)
        },
        trainingSummary: function () {
            return query("/reports/training-summary", "GET", {}, true)
        },
        groupComparison: function () {
            return query("/reports/group-comparison", "GET", {}, true)
        }
    },
    // BRS (Behavioral Risk Score) endpoints
    brs: {
        user: function (id) {
            return query("/reports/brs/user/" + id, "GET", {}, true)
        },
        department: function () {
            return query("/reports/brs/department", "GET", {}, true)
        },
        benchmark: function () {
            return query("/reports/brs/benchmark", "GET", {}, true)
        },
        trend: function (userId, days) {
            return query("/reports/brs/trend?user_id=" + userId + "&days=" + (days || 90), "GET", {}, true)
        },
        leaderboard: function (limit) {
            return query("/reports/brs/leaderboard?limit=" + (limit || 25), "GET", {}, true)
        },
        recalculate: function () {
            return query("/reports/brs/recalculate", "POST", {}, true)
        }
    },
    // Board-ready report endpoints
    boardReports: {
        get: function () {
            return query("/board-reports/", "GET", {}, true)
        },
        getOne: function (id) {
            return query("/board-reports/" + id, "GET", {}, true)
        },
        getFull: function (id) {
            return query("/board-reports/" + id + "/full", "GET", {}, true)
        },
        create: function (data) {
            return query("/board-reports/", "POST", data, false)
        },
        update: function (id, data) {
            return query("/board-reports/" + id, "PUT", data, false)
        },
        remove: function (id) {
            return query("/board-reports/" + id, "DELETE", {}, false)
        },
        generate: function (data) {
            return query("/board-reports/generate", "POST", data, false)
        },
        generateNarrative: function (id) {
            return query("/board-reports/" + id + "/generate-narrative", "POST", {}, false)
        },
        editNarrative: function (id, data) {
            return query("/board-reports/" + id + "/narrative-edit", "PUT", data, false)
        },
        transition: function (id, data) {
            return query("/board-reports/" + id + "/transition", "POST", data, false)
        },
        getApprovals: function (id) {
            return query("/board-reports/" + id + "/approvals", "GET", {}, true)
        },
        getHeatmap: function () {
            return query("/board-reports/heatmap", "GET", {}, true)
        },
        getDeltas: function (data) {
            return query("/board-reports/deltas", "POST", data, false)
        },
        exportUrl: function (id, format) {
            return "/api/board-reports/" + id + "/export?format=" + (format || "pdf")
        }
    },
    // Audit log endpoint
    auditLog: {
        get: function (params) {
            var qs = $.param(params || {});
            return query("/audit-log?" + qs, "GET", {}, true)
        }
    },
    // Organization endpoints
    orgs: {
        get: function (id) {
            return query("/orgs/" + id, "GET", {}, true)
        },
        getAll: function () {
            return query("/orgs/", "GET", {}, true)
        },
        post: function (data) {
            return query("/orgs/", "POST", data, true)
        },
        put: function (id, data) {
            return query("/orgs/" + id, "PUT", data, true)
        },
        delete: function (id) {
            return query("/orgs/" + id, "DELETE", {}, true)
        },
        members: function (id) {
            return query("/orgs/" + id + "/members", "GET", {}, true)
        },
        addMember: function (id, data) {
            return query("/orgs/" + id + "/members", "POST", data, true)
        },
        removeMember: function (id, uid) {
            return query("/orgs/" + id + "/members/" + uid, "DELETE", {}, true)
        }
    },
    // tiers contains the endpoints for /tiers
    tiers: {
        get: function () {
            return query("/tiers/", "GET", {}, true)
        }
    },
    // orgFeatures returns the feature flags for the current org
    orgFeatures: {
        get: function () {
            return query("/org/features", "GET", {}, true)
        }
    },
    // AI template generation endpoints
    ai: {
        generateTemplate: function (data) {
            return query("/ai/generate-template", "POST", data, true)
        },
        usage: function () {
            return query("/ai/usage", "GET", {}, true)
        }
    },
    // Autopilot endpoints
    autopilot: {
        getConfig: function () {
            return query("/autopilot/config", "GET", {}, true)
        },
        saveConfig: function (data) {
            return query("/autopilot/config", "PUT", data, false)
        },
        enable: function () {
            return query("/autopilot/enable", "POST", {}, false)
        },
        disable: function () {
            return query("/autopilot/disable", "POST", {}, false)
        },
        getSchedule: function (limit) {
            return query("/autopilot/schedule?limit=" + (limit || 50), "GET", {}, true)
        },
        getBlackouts: function () {
            return query("/autopilot/blackout", "GET", {}, true)
        },
        addBlackout: function (data) {
            return query("/autopilot/blackout", "POST", data, false)
        },
        deleteBlackout: function (id) {
            return query("/autopilot/blackout/" + id, "DELETE", {}, false)
        }
    },
    // Academy endpoints
    academy: {
        getTiers: function () {
            return query("/academy/tiers", "GET", {}, true)
        },
        getTierSessions: function (slug) {
            return query("/academy/tiers/" + slug + "/sessions", "GET", {}, true)
        },
        completeTier: function (slug) {
            return query("/academy/tiers/" + slug + "/complete", "POST", {}, false)
        },
        myProgress: function () {
            return query("/academy/my-progress", "GET", {}, true)
        },
        createSession: function (data) {
            return query("/academy/sessions", "POST", data, false)
        },
        updateSession: function (data) {
            return query("/academy/sessions", "PUT", data, false)
        },
        deleteSession: function (id) {
            return query("/academy/sessions/" + id, "DELETE", {}, false)
        },
        getComplianceCerts: function () {
            return query("/academy/compliance", "GET", {}, true)
        },
        completeCert: function (id) {
            return query("/academy/compliance/" + id + "/complete", "POST", {}, false)
        },
        myCerts: function () {
            return query("/academy/compliance/my-certs", "GET", {}, true)
        },
        verifyCert: function (code) {
            return query("/academy/compliance/verify/" + code, "GET", {}, true)
        }
    },
    // Gamification endpoints
    gamification: {
        getLeaderboard: function (period, department) {
            var url = "/gamification/leaderboard?period=" + (period || "all_time");
            if (department) url += "&department=" + encodeURIComponent(department);
            return query(url, "GET", {}, true)
        },
        myPosition: function (period) {
            return query("/gamification/my-position?period=" + (period || "all_time"), "GET", {}, true)
        },
        getBadges: function () {
            return query("/gamification/badges", "GET", {}, true)
        },
        myBadges: function () {
            return query("/gamification/my-badges", "GET", {}, true)
        },
        myStreak: function () {
            return query("/gamification/my-streak", "GET", {}, true)
        }
    },
    // i18n endpoints
    i18n: {
        getTranslations: function (locale) {
            return query("/i18n/" + (locale || "en"), "GET", {}, true)
        },
        getLanguages: function () {
            return query("/i18n/languages", "GET", {}, true)
        },
        setLanguage: function (lang) {
            return query("/user/language", "PUT", { preferred_language: lang }, false)
        }
    },
    // Report button endpoints
    reportButton: {
        getConfig: function () {
            return query("/report-button/config", "GET", {}, true)
        },
        saveConfig: function (data) {
            return query("/report-button/config", "PUT", data, false)
        },
        regenerateKey: function () {
            return query("/report-button/regenerate-key", "POST", {}, false)
        }
    },
    // Content Library endpoints
    contentLibrary: {
        browse: function (category, difficulty) {
            var url = "/training/content-library";
            var params = [];
            if (category) params.push("category=" + encodeURIComponent(category));
            if (difficulty) params.push("difficulty=" + encodeURIComponent(difficulty));
            if (params.length > 0) url += "?" + params.join("&");
            return query(url, "GET", {}, true)
        },
        detail: function (slug) {
            return query("/training/content-library/detail?slug=" + encodeURIComponent(slug), "GET", {}, true)
        },
        categories: function () {
            return query("/training/content-library/categories", "GET", {}, true)
        },
        seedAll: function () {
            return query("/training/content-library/seed", "POST", {}, false)
        },
        seedSingle: function (slug) {
            return query("/training/content-library/seed-single", "POST", { slug: slug }, false)
        }
    },
    // Training satisfaction / analytics endpoints
    trainingSatisfaction: {
        rate: function (presentationId, rating, feedback) {
            return query("/training/" + presentationId + "/rate", "POST", { rating: rating, feedback: feedback || "" }, false)
        },
        stats: function () {
            return query("/training/satisfaction", "GET", {}, true)
        },
        analytics: function () {
            return query("/training/analytics", "GET", {}, true)
        }
    },
    // Configurable praise / feedback messages
    praiseMessages: {
        get: function () {
            return query("/training/praise-messages", "GET", {}, true)
        },
        put: function (messages) {
            return query("/training/praise-messages", "PUT", messages, false)
        },
        reset: function () {
            return query("/training/praise-messages/reset", "DELETE", {}, false)
        }
    },
    // Reported emails endpoints
    reportedEmails: {
        get: function () {
            return query("/reported-emails", "GET", {}, true)
        },
        classify: function (id, data) {
            return query("/reported-emails/" + id + "/classify", "PUT", data, false)
        }
    },
    // Training (alias for remediation path course selection)
    training: {
        get: function () {
            return query("/training/", "GET", {}, true)
        }
    },
    // Remediation path endpoints
    remediation: {
        get: function () {
            return query("/remediation/paths", "GET", {}, true)
        },
        getOne: function (id) {
            return query("/remediation/paths/" + id, "GET", {}, true)
        },
        create: function (data) {
            return query("/remediation/paths", "POST", data, false)
        },
        cancel: function (id) {
            return query("/remediation/paths/" + id, "DELETE", {}, false)
        },
        myPaths: function () {
            return query("/remediation/my-paths", "GET", {}, true)
        },
        completeStep: function (pathId, data) {
            return query("/remediation/paths/" + pathId + "/complete-step", "POST", data, false)
        },
        evaluate: function () {
            return query("/remediation/evaluate", "POST", {}, false)
        },
        summary: function () {
            return query("/remediation/summary", "GET", {}, true)
        },
        markExpired: function () {
            return query("/remediation/mark-expired", "POST", {}, false)
        }
    },
    // Cyber Hygiene endpoints
    hygiene: {
        devices: {
            get: function () {
                return query("/hygiene/devices/", "GET", {}, true)
            },
            getOne: function (id) {
                return query("/hygiene/devices/" + id, "GET", {}, true)
            },
            create: function (data) {
                return query("/hygiene/devices/", "POST", data, false)
            },
            update: function (id, data) {
                return query("/hygiene/devices/" + id, "PUT", data, false)
            },
            remove: function (id) {
                return query("/hygiene/devices/" + id, "DELETE", {}, false)
            },
            upsertCheck: function (deviceId, data) {
                return query("/hygiene/devices/" + deviceId + "/checks", "POST", data, false)
            }
        },
        techStack: {
            get: function () {
                return query("/hygiene/tech-stack", "GET", {}, true)
            },
            save: function (data) {
                return query("/hygiene/tech-stack", "POST", data, false)
            }
        },
        personalizedChecks: function () {
            return query("/hygiene/personalized-checks", "GET", {}, true)
        },
        admin: {
            devices: function () {
                return query("/hygiene/admin/devices", "GET", {}, true)
            },
            devicesEnriched: function () {
                return query("/hygiene/admin/devices-enriched", "GET", {}, true)
            },
            summary: function () {
                return query("/hygiene/admin/summary", "GET", {}, true)
            }
        }
    },
    // Threat alerts endpoints
    threatAlerts: {
        get: function () {
            return query("/threat-alerts", "GET", {}, true)
        },
        create: function (data) {
            return query("/threat-alerts/create", "POST", data, false)
        },
        getOne: function (id) {
            return query("/threat-alerts/" + id, "GET", {}, true)
        },
        update: function (id, data) {
            return query("/threat-alerts/" + id, "PUT", data, false)
        },
        delete: function (id) {
            return query("/threat-alerts/" + id, "DELETE", {}, false)
        },
        unreadCount: function () {
            return query("/threat-alerts/unread-count", "GET", {}, true)
        }
    },
    // Email Security: Inbox Monitor
    inboxMonitor: {
        getConfig: function () {
            return query("/inbox-monitor/config", "GET", {}, true)
        },
        saveConfig: function (data) {
            return query("/inbox-monitor/config", "PUT", data, false)
        },
        getResults: function (limit) {
            var url = "/inbox-monitor/results";
            if (limit) url += "?limit=" + limit;
            return query(url, "GET", {}, true)
        },
        getResult: function (id) {
            return query("/inbox-monitor/results/" + id, "GET", {}, true)
        },
        getSummary: function () {
            return query("/inbox-monitor/summary", "GET", {}, true)
        }
    },
    // Email Security: BEC Detection
    bec: {
        getProfiles: function () {
            return query("/bec/profiles", "GET", {}, true)
        },
        createProfile: function (data) {
            return query("/bec/profiles", "POST", data, false)
        },
        updateProfile: function (id, data) {
            return query("/bec/profiles/" + id, "PUT", data, false)
        },
        deleteProfile: function (id) {
            return query("/bec/profiles/" + id, "DELETE", {}, false)
        },
        getDetections: function () {
            return query("/bec/detections", "GET", {}, true)
        },
        resolveDetection: function (id, data) {
            return query("/bec/detections/" + id + "/resolve", "PUT", data, false)
        },
        getSummary: function () {
            return query("/bec/summary", "GET", {}, true)
        },
        analyze: function (data) {
            return query("/bec/analyze", "POST", data, false)
        }
    },
    // Email Security: Graymail Classification
    graymail: {
        getRules: function () {
            return query("/graymail/rules", "GET", {}, true)
        },
        createRule: function (data) {
            return query("/graymail/rules", "POST", data, false)
        },
        updateRule: function (id, data) {
            return query("/graymail/rules/" + id, "PUT", data, false)
        },
        deleteRule: function (id) {
            return query("/graymail/rules/" + id, "DELETE", {}, false)
        },
        getClassifications: function (limit) {
            var url = "/graymail/classifications";
            if (limit) url += "?limit=" + limit;
            return query(url, "GET", {}, true)
        },
        getSummary: function () {
            return query("/graymail/summary", "GET", {}, true)
        },
        analyze: function (data) {
            return query("/graymail/analyze", "POST", data, false)
        }
    },
    // Email Security: Remediation Actions
    remediationActions: {
        get: function () {
            return query("/remediation-actions/", "GET", {}, true)
        },
        create: function (data) {
            return query("/remediation-actions/create", "POST", data, false)
        },
        approve: function (id) {
            return query("/remediation-actions/" + id + "/approve", "PUT", {}, false)
        },
        reject: function (id, data) {
            return query("/remediation-actions/" + id + "/reject", "PUT", data, false)
        },
        getSummary: function () {
            return query("/remediation-actions/summary", "GET", {}, true)
        }
    },
    // Email Security: Phishing Tickets
    phishingTickets: {
        get: function (status) {
            var url = "/phishing-tickets/";
            if (status && status !== "all") url += "?status=" + encodeURIComponent(status);
            return query(url, "GET", {}, true)
        },
        getOne: function (id) {
            return query("/phishing-tickets/" + id, "GET", {}, true)
        },
        resolve: function (id, data) {
            return query("/phishing-tickets/" + id + "/resolve", "PUT", data, false)
        },
        escalate: function (id, data) {
            return query("/phishing-tickets/" + id + "/escalate", "PUT", data, false)
        },
        getSummary: function () {
            return query("/phishing-tickets/summary", "GET", {}, true)
        },
        getAutoRules: function () {
            return query("/phishing-tickets/auto-rules", "GET", {}, true)
        },
        createAutoRule: function (data) {
            return query("/phishing-tickets/auto-rules", "POST", data, false)
        },
        deleteAutoRule: function (id) {
            return query("/phishing-tickets/auto-rules/" + id, "DELETE", {}, false)
        }
    },
    // Email Security: Unified Dashboard
    emailSecurity: {
        getDashboard: function () {
            return query("/email-security/dashboard", "GET", {}, true)
        }
    },
    // Network Events: Security event correlation & MITRE mapping
    networkEvents: {
        list: function (qs) {
            var url = "/network-events/";
            if (qs) url += "?" + qs;
            return query(url, "GET", {}, true)
        },
        get: function (id) {
            return query("/network-events/" + id, "GET", {}, true)
        },
        updateStatus: function (id, status) {
            return query("/network-events/" + id, "PUT", { status: status }, false)
        },
        ingest: function (data) {
            return query("/network-events/ingest", "POST", data, false)
        },
        bulkIngest: function (data) {
            return query("/network-events/bulk-ingest", "POST", data, false)
        },
        dashboard: function () {
            return query("/network-events/dashboard", "GET", {}, true)
        },
        trend: function (days) {
            return query("/network-events/trend?days=" + (days || 30), "GET", {}, true)
        },
        addNote: function (id, content) {
            return query("/network-events/" + id + "/notes", "POST", { content: content }, false)
        },
        mitreHeatmap: function () {
            return query("/network-events/mitre-heatmap", "GET", {}, true)
        },
        correlate: function () {
            return query("/network-events/correlate", "POST", {}, false)
        },
        incidents: function (status, limit) {
            var url = "/network-events/incidents";
            var params = [];
            if (status) params.push("status=" + encodeURIComponent(status));
            if (limit) params.push("limit=" + limit);
            if (params.length) url += "?" + params.join("&");
            return query(url, "GET", {}, true)
        },
        getIncident: function (id) {
            return query("/network-events/incidents/" + id, "GET", {}, true)
        },
        updateIncidentStatus: function (id, status) {
            return query("/network-events/incidents/" + id, "PUT", { status: status }, false)
        },
        playbookLogs: function (limit) {
            return query("/network-events/playbook-logs?limit=" + (limit || 50), "GET", {}, true)
        },
        getRules: function () {
            return query("/network-events/rules", "GET", {}, true)
        },
        createRule: function (data) {
            return query("/network-events/rules", "POST", data, false)
        },
        updateRule: function (id, data) {
            return query("/network-events/rules/" + id, "PUT", data, false)
        },
        deleteRule: function (id) {
            return query("/network-events/rules/" + id, "DELETE", {}, false)
        }
    },
    // ── ROI Enhanced: Benchmarks, Monte Carlo, History ──
    roi: {
        generate: function (data) {
            return query("/roi/generate", "POST", data, false)
        },
        generateAndSave: function (data) {
            return query("/roi/generate-and-save", "POST", data, false)
        },
        getConfig: function () {
            return query("/roi/config", "GET", {}, true)
        },
        saveConfig: function (data) {
            return query("/roi/config", "PUT", data, false)
        },
        getBenchmarks: function () {
            return query("/roi/benchmarks", "GET", {}, true)
        },
        saveBenchmark: function (data) {
            return query("/roi/benchmarks", "POST", data, false)
        },
        deleteBenchmark: function (id) {
            return query("/roi/benchmarks/" + id, "DELETE", {}, false)
        },
        seedBenchmarks: function () {
            return query("/roi/benchmarks/seed", "POST", {}, false)
        },
        compareBenchmarks: function (start, end) {
            return query("/roi/benchmarks/compare?start=" + start + "&end=" + end, "GET", {}, true)
        },
        monteCarlo: function (data) {
            return query("/roi/monte-carlo", "POST", data, false)
        },
        getHistory: function () {
            return query("/roi/history", "GET", {}, true)
        },
        deleteHistoryItem: function (id) {
            return query("/roi/history/" + id, "DELETE", {}, false)
        },
        getTrend: function () {
            return query("/roi/trend", "GET", {}, true)
        },
        exportPdf: function (start, end) {
            return "/api/roi/export-pdf?start=" + start + "&end=" + end
        },
        exportUrl: function (format, start, end) {
            return "/api/roi/export?format=" + format + "&start=" + start + "&end=" + end
        }
    },
    // ── AI Admin Assistant ──
    adminAssistant: {
        chat: function (data) {
            return query("/admin-assistant/chat", "POST", data, false)
        },
        getConversations: function () {
            return query("/admin-assistant/conversations", "GET", {}, true)
        },
        getConversation: function (id) {
            return query("/admin-assistant/conversations/" + id, "GET", {}, true)
        },
        getOnboarding: function () {
            return query("/admin-assistant/onboarding", "GET", {}, true)
        },
        completeOnboardingStep: function (step) {
            return query("/admin-assistant/onboarding/" + step + "/complete", "POST", {}, false)
        }
    },
    // ── Scheduled Reports ──
    scheduledReports: {
        getAll: function () {
            return query("/scheduled-reports/", "GET", {}, true)
        },
        get: function (id) {
            return query("/scheduled-reports/" + id, "GET", {}, true)
        },
        create: function (data) {
            return query("/scheduled-reports/", "POST", data, false)
        },
        update: function (id, data) {
            return query("/scheduled-reports/" + id, "PUT", data, false)
        },
        delete: function (id) {
            return query("/scheduled-reports/" + id, "DELETE", {}, false)
        },
        toggle: function (id, isActive) {
            return query("/scheduled-reports/" + id + "/toggle", "POST", { is_active: isActive }, false)
        },
        getSummary: function () {
            return query("/scheduled-reports/summary", "GET", {}, true)
        },
        getTypes: function () {
            return query("/scheduled-reports/types", "GET", {}, true)
        }
    },
    // ── Unified Export ──
    export: {
        url: function (reportType, format, start, end) {
            return "/api/export?type=" + reportType + "&format=" + (format || "pdf") + "&start=" + (start || "") + "&end=" + (end || "")
        }
    }
}
window.api = api

// Register our moment.js datatables listeners
$(document).ready(function () {
    // Setup nav highlighting
    var path = location.pathname;
    $('.nav-sidebar li').each(function () {
        var $this = $(this);
        // if the current path is like this link, make it active
        if ($this.find("a").attr('href') === path) {
            $this.addClass('active');
        }
    })
    $.fn.dataTable.moment('MMMM Do YYYY, h:mm:ss a');
    // Setup tooltips
    $('[data-toggle="tooltip"]').tooltip()
});
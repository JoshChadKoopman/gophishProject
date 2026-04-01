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
    // Reported emails endpoints
    reportedEmails: {
        get: function () {
            return query("/reported-emails", "GET", {}, true)
        },
        classify: function (id, data) {
            return query("/reported-emails/" + id + "/classify", "PUT", data, false)
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
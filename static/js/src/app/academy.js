$(document).ready(function () {
    loadAcademy();
});

var currentTierSlug = null;

function loadAcademy() {
    // Load tiers, badges, streak, compliance in parallel
    $.when(
        api.academy.getTiers(),
        api.gamification ? api.gamification.myBadges() : $.Deferred().resolve([]),
        api.gamification ? api.gamification.myStreak() : $.Deferred().resolve([]),
        api.academy.getComplianceCerts()
    ).done(function (tiersResp, badgesResp, streaksResp, complianceResp) {
        var tiers = tiersResp[0] || tiersResp;
        var badges = badgesResp[0] || badgesResp || [];
        var streaks = streaksResp[0] || streaksResp || [];
        var compliance = complianceResp[0] || complianceResp || [];

        $("#loading").hide();
        $("#academyContent").show();

        renderStats(tiers, badges, streaks, compliance);
        renderTierProgress(tiers);
        renderTierCards(tiers);
        renderBadges(badges);
        renderCompliance(compliance);
    }).fail(function () {
        $("#loading").hide();
        $("#academyContent").show();
    });
}

function renderStats(tiers, badges, streaks, compliance) {
    var completed = 0;
    $.each(tiers, function (i, t) {
        if (t.user_progress && t.user_progress.tier_completed) completed++;
    });
    $("#tiersCompleted").text(completed + " / " + tiers.length);
    $("#badgeCount").text(badges.length || 0);

    var weeklyStreak = 0;
    $.each(streaks, function (i, s) {
        if (s.streak_type === "weekly") weeklyStreak = s.current_streak;
    });
    $("#streakCount").text(weeklyStreak + " weeks");

    var earnedCerts = 0;
    $.each(compliance, function (i, c) {
        if (c.earned) earnedCerts++;
    });
    $("#complianceCount").text(earnedCerts);
}

function renderTierProgress(tiers) {
    var bar = $("#tierProgressBar");
    bar.empty();
    var tierIcons = { "bronze": "fa-shield", "silver": "fa-shield", "gold": "fa-shield", "platinum": "fa-diamond" };
    $.each(tiers, function (i, tier) {
        if (i > 0) {
            var prevCompleted = tiers[i - 1].user_progress && tiers[i - 1].user_progress.tier_completed;
            bar.append('<div class="tier-connector ' + (prevCompleted ? 'active' : '') + '"></div>');
        }
        var status = "locked";
        if (tier.user_progress && tier.user_progress.tier_completed) {
            status = "completed";
        } else if (tier.user_progress && tier.user_progress.tier_unlocked) {
            status = "unlocked";
        }
        var icon = tierIcons[tier.slug] || "fa-star";
        bar.append(
            '<div class="tier-node">' +
            '<div class="tier-circle ' + status + '" data-slug="' + escapeHtml(tier.slug) + '">' +
            '<i class="fa ' + icon + '"></i>' +
            '</div>' +
            '<div style="font-weight:bold;">' + escapeHtml(tier.name) + '</div>' +
            '<div style="color:#888;font-size:12px;">' +
            (status === "completed" ? "Completed" : (status === "unlocked" ? "In Progress" : "Locked")) +
            '</div>' +
            '</div>'
        );
    });
}

function renderTierCards(tiers) {
    var container = $("#tierCards");
    container.empty();
    $.each(tiers, function (i, tier) {
        var isLocked = !(tier.user_progress && tier.user_progress.tier_unlocked);
        var isCompleted = tier.user_progress && tier.user_progress.tier_completed;
        var sessionsCompleted = tier.user_progress ? tier.user_progress.sessions_completed : 0;
        var totalRequired = tier.required_sessions || 0;
        var pct = totalRequired > 0 ? Math.round((sessionsCompleted / totalRequired) * 100) : 0;

        var statusLabel = isCompleted ? '<span class="label label-success">Completed</span>' :
            (isLocked ? '<span class="label label-default">Locked</span>' :
                '<span class="label label-warning">In Progress</span>');

        container.append(
            '<div class="col-md-3">' +
            '<div class="panel panel-default" style="opacity:' + (isLocked ? '0.5' : '1') + ';">' +
            '<div class="panel-heading text-center">' +
            '<h4>' + escapeHtml(tier.name) + ' ' + statusLabel + '</h4>' +
            '</div>' +
            '<div class="panel-body text-center">' +
            '<p style="color:#666;">' + escapeHtml(tier.description) + '</p>' +
            '<p><strong>' + sessionsCompleted + ' / ' + totalRequired + '</strong> sessions completed</p>' +
            '<div class="progress" style="margin-bottom:10px;">' +
            '<div class="progress-bar progress-bar-' + (isCompleted ? 'success' : 'warning') + '" style="width:' + pct + '%;min-width:0%;">' + pct + '%</div>' +
            '</div>' +
            (isLocked ? '' : '<button class="btn btn-sm btn-primary btn-view-sessions" data-slug="' + escapeHtml(tier.slug) + '"><i class="fa fa-arrow-right"></i> View Sessions</button>') +
            '</div>' +
            '</div>' +
            '</div>'
        );
    });
}

function openTierSessions(slug) {
    currentTierSlug = slug;
    api.academy.getTierSessions(slug).done(function (sessions) {
        $("#sessionsTierName").text(slug.charAt(0).toUpperCase() + slug.slice(1) + " Tier — Sessions");
        var tbody = $("#sessionsBody");
        tbody.empty();

        var allRequiredDone = true;
        $.each(sessions, function (i, s) {
            var statusBadge = s.completed ?
                '<span class="label label-success">Completed</span>' :
                '<span class="label label-default">Not Started</span>';
            if (s.is_required && !s.completed) allRequiredDone = false;

            tbody.append(
                '<tr>' +
                '<td>' + (i + 1) + '</td>' +
                '<td>' + escapeHtml(s.presentation_name || 'Course #' + s.presentation_id) + '</td>' +
                '<td>' + s.estimated_minutes + ' min</td>' +
                '<td>' + (s.is_required ? '<i class="fa fa-check text-danger"></i> Yes' : 'Optional') + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td>' +
                (s.completed ?
                    '<button class="btn btn-xs btn-default" onclick="startCourse(' + s.presentation_id + ')"><i class="fa fa-refresh"></i> Review</button>' :
                    '<button class="btn btn-xs btn-success" onclick="startCourse(' + s.presentation_id + ')"><i class="fa fa-play"></i> Start</button>') +
                '</td>' +
                '</tr>'
            );
        });

        if (allRequiredDone && sessions.length > 0) {
            $("#btnCompleteTier").show();
        } else {
            $("#btnCompleteTier").hide();
        }

        $("#sessionsPanel").show();
        $("html, body").animate({ scrollTop: $("#sessionsPanel").offset().top - 70 }, 300);
    });
}

function closeSessions() {
    $("#sessionsPanel").hide();
    currentTierSlug = null;
}

function startCourse(presentationId) {
    // Redirect to training page — the course viewer will handle it
    window.location.href = "/training?open=" + presentationId;
}

// Delegated click handlers for tier circles and view-session buttons (avoid inline onclick XSS)
$(document).on("click", ".tier-circle[data-slug]", function () {
    openTierSessions($(this).data("slug"));
});
$(document).on("click", ".btn-view-sessions[data-slug]", function () {
    openTierSessions($(this).data("slug"));
});

// Complete tier button handler
$(document).on("click", "#btnCompleteTier", function () {
    if (!currentTierSlug) return;
    api.academy.completeTier(currentTierSlug).done(function (data) {
        if (data.new_badges && data.new_badges.length > 0) {
            var badgeNames = data.new_badges.map(function (b) { return b.badge_name; }).join(", ");
            successFlash("Tier completed! New badges earned: " + badgeNames);
        } else if (data.progress && data.progress.tier_completed) {
            successFlash("Tier completed! Well done!");
        } else {
            successFlash("Progress updated!");
        }
        loadAcademy();
        closeSessions();
    }).fail(function (data) {
        errorFlash(data.responseJSON ? data.responseJSON.message : "Failed to complete tier");
    });
});

function renderBadges(badges) {
    var container = $("#badgesContainer");
    if (!container.length) return;
    container.empty();

    if (!badges || badges.length === 0) {
        $("#noBadges").show();
        return;
    }
    $("#noBadges").hide();

    // Map of badge categories to icons
    var categoryIcons = {
        "training": "fa-book",
        "quiz": "fa-question-circle",
        "academy": "fa-graduation-cap",
        "streak": "fa-fire",
        "simulation": "fa-envelope",
        "compliance": "fa-certificate"
    };

    // First load all available badges, then mark earned ones
    api.gamification.getBadges().done(function (allBadges) {
        var earnedSlugs = {};
        $.each(badges, function (i, b) { earnedSlugs[b.badge_slug] = b; });

        $.each(allBadges, function (i, b) {
            var earned = earnedSlugs[b.slug];
            var icon = categoryIcons[b.category] || "fa-star";
            container.append(
                '<div class="badge-card ' + (earned ? 'earned' : 'unearned') + '" title="' + escapeHtml(b.description) + '">' +
                '<div class="badge-icon"><i class="fa ' + icon + '" style="color:' + (earned ? '#f0ad4e' : '#ccc') + ';"></i></div>' +
                '<div style="font-weight:bold;font-size:11px;">' + escapeHtml(b.name) + '</div>' +
                (earned ? '<div style="font-size:10px;color:#888;">' + moment(earned.earned_date).format("MMM D") + '</div>' : '') +
                '</div>'
            );
        });
    });
}

function renderCompliance(certifications) {
    var container = $("#complianceContainer");
    if (!container.length) return;
    container.empty();

    if (!certifications || certifications.length === 0) {
        $("#noCompliance").show();
        return;
    }
    $("#noCompliance").hide();

    $.each(certifications, function (i, cert) {
        var pct = cert.total_required > 0 ? Math.round((cert.user_completed / cert.total_required) * 100) : 0;
        container.append(
            '<div class="col-md-4">' +
            '<div class="compliance-card">' +
            '<h4>' + escapeHtml(cert.name) +
            (cert.earned ? ' <span class="label label-success"><i class="fa fa-check"></i> Earned</span>' : '') +
            '</h4>' +
            '<p style="color:#666;">' + escapeHtml(cert.description) + '</p>' +
            '<p>' + cert.user_completed + ' / ' + cert.total_required + ' sessions completed</p>' +
            '<div class="progress">' +
            '<div class="progress-bar progress-bar-info" style="width:' + pct + '%;min-width:0%;">' + pct + '%</div>' +
            '</div>' +
            (cert.earned ? '' :
                (pct >= 100 ?
                    '<button class="btn btn-sm btn-success" onclick="claimCert(' + cert.id + ')"><i class="fa fa-certificate"></i> Claim Certificate</button>' :
                    '<span style="color:#888;font-size:12px;">Complete all required sessions to earn this certificate.</span>')) +
            '</div>' +
            '</div>'
        );
    });
}

function claimCert(certId) {
    api.academy.completeCert(certId).done(function (data) {
        successFlash("Compliance certificate earned! Verification code: " + data.verification_code);
        loadAcademy();
    }).fail(function (data) {
        errorFlash(data.responseJSON ? data.responseJSON.message : "Failed to claim certificate");
    });
}

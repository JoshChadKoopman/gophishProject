/**
 * Email Security Dashboard — frontend logic.
 * Handles the unified email security view with tabs for:
 *   - Overview (stats, charts, recent threats)
 *   - Inbox Monitor (config, scan results)
 *   - BEC Detection (profiles, detections)
 *   - Graymail (rules, classifications)
 *   - Remediation Actions (create, approve/reject)
 *   - Phishing Tickets (list, resolve, escalate)
 */
$(document).ready(function () {
    // ========== OVERVIEW TAB ==========
    loadDashboard();

    function loadDashboard() {
        api.emailSecurity.getDashboard()
            .success(function (data) {
                if (!data) return;
                // Populate stat cards
                $("#statTotalScanned").text(data.total_scanned || 0);
                $("#statThreatsDetected").text(data.threats_detected || 0);
                $("#statBECAttempts").text(data.bec_attempts || 0);
                $("#statRemediated").text(data.remediated_count || 0);
                $("#mttrValue").text(data.mean_time_to_remediate || "N/A");
                $("#openTicketsCount").text(data.open_tickets || 0);
                $("#userReports7d").text(data.user_reports_7d || 0);

                // Render threat trend chart
                renderThreatTrend(data.threat_trend || []);
                // Render classification pie
                renderClassificationChart(data.classification_breakdown || {});
                // Render recent threats table
                renderRecentThreats(data.recent_threats || []);
            })
            .error(function () {
                console.log("Email Security Dashboard: could not load data (feature may not be enabled)");
                $("#statTotalScanned, #statThreatsDetected, #statBECAttempts, #statRemediated").text("0");
                $("#mttrValue, #openTicketsCount, #userReports7d").text("0");
                renderRecentThreats([]);
            });
    }

    function renderThreatTrend(data) {
        var el = document.getElementById("threatTrendChart");
        if (!el || typeof echarts === "undefined") {
            $("#threatTrendChart").html('<div class="text-center text-muted" style="padding:40px;">Chart library not loaded</div>');
            return;
        }
        var chart = echarts.init(el);
        var dates = data.map(function (d) { return d.date; });
        var safe = data.map(function (d) { return d.safe || 0; });
        var suspicious = data.map(function (d) { return d.suspicious || 0; });
        var phishing = data.map(function (d) { return d.phishing || 0; });
        chart.setOption({
            tooltip: { trigger: "axis" },
            legend: { data: ["Safe", "Suspicious", "Phishing"] },
            xAxis: { type: "category", data: dates },
            yAxis: { type: "value" },
            series: [
                { name: "Safe", type: "line", stack: "Total", areaStyle: {}, data: safe, itemStyle: { color: "#5cb85c" } },
                { name: "Suspicious", type: "line", stack: "Total", areaStyle: {}, data: suspicious, itemStyle: { color: "#f0ad4e" } },
                { name: "Phishing", type: "line", stack: "Total", areaStyle: {}, data: phishing, itemStyle: { color: "#d9534f" } }
            ]
        });
    }

    function renderClassificationChart(breakdown) {
        var el = document.getElementById("classificationChart");
        if (!el || typeof echarts === "undefined") {
            $("#classificationChart").html('<div class="text-center text-muted" style="padding:40px;">Chart library not loaded</div>');
            return;
        }
        var chart = echarts.init(el);
        var items = [];
        var colors = {
            "phishing": "#d9534f", "bec": "#a94442", "graymail": "#f0ad4e",
            "spam": "#999", "safe": "#5cb85c", "suspicious": "#e67e22"
        };
        for (var key in breakdown) {
            items.push({ name: key, value: breakdown[key] });
        }
        chart.setOption({
            tooltip: { trigger: "item", formatter: "{a} <br/>{b}: {c} ({d}%)" },
            series: [{
                name: "Classification",
                type: "pie",
                radius: ["40%", "70%"],
                data: items,
                color: items.map(function (i) { return colors[i.name] || "#337ab7"; })
            }]
        });
    }

    function renderRecentThreats(threats) {
        var tbody = $("#recentThreatsBody");
        tbody.empty();
        if (!threats || threats.length === 0) {
            tbody.append('<tr><td colspan="6" class="text-center text-muted">No threats detected recently</td></tr>');
            return;
        }
        $.each(threats, function (i, t) {
            var date = moment(t.detected_date || t.created_date).format("MMM D, h:mm a");
            tbody.append(
                '<tr>' +
                '<td>' + escapeHtml(date) + '</td>' +
                '<td>' + escapeHtml(t.sender_email || '') + '</td>' +
                '<td>' + escapeHtml(t.subject || '') + '</td>' +
                '<td><span class="threat-' + (t.threat_level || 'safe') + '">' + escapeHtml(t.threat_level || 'unknown') + '</span></td>' +
                '<td>' + escapeHtml(t.classification || '') + '</td>' +
                '<td>' + escapeHtml(t.action_taken || 'none') + '</td>' +
                '</tr>'
            );
        });
    }

    // ========== INBOX MONITOR TAB ==========
    $("#emailSecurityTabs a[href='#tabInboxMonitor']").on("shown.bs.tab", function () {
        loadMonitorConfig();
        loadScanResults();
    });

    function loadMonitorConfig() {
        api.inboxMonitor.getConfig()
            .success(function (cfg) {
                if (!cfg) return;
                $("#monitorEnabled").val(cfg.enabled ? "true" : "false");
                $("#monitorInterval").val(cfg.scan_interval_seconds || 300);
                $("#monitorThreshold").val(cfg.threat_threshold || "suspicious");
                $("#monitorMailboxes").val((cfg.monitored_mailboxes || "").replace(/,/g, "\n"));
                $("#monitorAutoQuarantine").prop("checked", cfg.auto_quarantine);
                $("#monitorAutoDelete").prop("checked", cfg.auto_delete);
                $("#monitorBEC").prop("checked", cfg.bec_detection_enabled);
                $("#monitorGraymail").prop("checked", cfg.graymail_classification_enabled);
            });
    }

    function loadScanResults() {
        api.inboxMonitor.getResults(50)
            .success(function (results) {
                var tbody = $("#scanResultsBody");
                tbody.empty();
                if (!results || results.length === 0) {
                    tbody.append('<tr><td colspan="8" class="text-center text-muted">No scan results yet</td></tr>');
                    return;
                }
                $.each(results, function (i, r) {
                    var date = moment(r.scanned_date || r.created_date).format("MMM D, h:mm a");
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(date) + '</td>' +
                        '<td>' + escapeHtml(r.mailbox_email || '') + '</td>' +
                        '<td>' + escapeHtml(r.sender_email || '') + '</td>' +
                        '<td>' + escapeHtml(r.subject || '') + '</td>' +
                        '<td><span class="threat-' + (r.threat_level || 'safe') + '">' + escapeHtml(r.threat_level || 'unknown') + '</span></td>' +
                        '<td>' + (r.is_bec ? '<i class="fa fa-check text-danger"></i>' : '') + '</td>' +
                        '<td>' + (r.is_graymail ? '<i class="fa fa-check text-warning"></i>' : '') + '</td>' +
                        '<td>' + escapeHtml(r.action_taken || 'none') + '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    $("#saveMonitorConfigBtn").on("click", function () {
        var data = {
            enabled: $("#monitorEnabled").val() === "true",
            scan_interval_seconds: parseInt($("#monitorInterval").val()) || 300,
            threat_threshold: $("#monitorThreshold").val(),
            monitored_mailboxes: $("#monitorMailboxes").val().trim().replace(/\n+/g, ","),
            auto_quarantine: $("#monitorAutoQuarantine").is(":checked"),
            auto_delete: $("#monitorAutoDelete").is(":checked"),
            bec_detection_enabled: $("#monitorBEC").is(":checked"),
            graymail_classification_enabled: $("#monitorGraymail").is(":checked")
        };
        api.inboxMonitor.saveConfig(data)
            .success(function () { flash("success", "Monitor configuration saved"); })
            .error(function () { flash("danger", "Failed to save configuration"); });
    });

    // ========== BEC DETECTION TAB ==========
    $("#emailSecurityTabs a[href='#tabBEC']").on("shown.bs.tab", function () {
        loadBECProfiles();
        loadBECDetections();
    });

    function loadBECProfiles() {
        api.bec.getProfiles()
            .success(function (profiles) {
                var tbody = $("#becProfilesBody");
                tbody.empty();
                if (!profiles || profiles.length === 0) {
                    tbody.append('<tr><td colspan="5" class="text-center text-muted">No executive profiles configured</td></tr>');
                    return;
                }
                $.each(profiles, function (i, p) {
                    var riskBadge = '<span class="priority-' + p.risk_level + '">' + escapeHtml(p.risk_level) + '</span>';
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(p.executive_name || '') + '</td>' +
                        '<td>' + escapeHtml(p.email || '') + '</td>' +
                        '<td>' + escapeHtml(p.title || '') + '</td>' +
                        '<td>' + riskBadge + '</td>' +
                        '<td>' +
                        '<button class="btn btn-xs btn-danger delete-bec-profile" data-id="' + p.id + '"><i class="fa fa-trash"></i></button>' +
                        '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    function loadBECDetections() {
        api.bec.getDetections()
            .success(function (detections) {
                var tbody = $("#becDetectionsBody");
                tbody.empty();
                if (!detections || detections.length === 0) {
                    tbody.append('<tr><td colspan="7" class="text-center text-muted">No BEC attempts detected</td></tr>');
                    return;
                }
                $.each(detections, function (i, d) {
                    var date = moment(d.detected_date || d.created_date).format("MMM D, h:mm a");
                    var statusBadge = '<span class="status-badge status-' + (d.resolved ? 'resolved' : 'open') + '">' + (d.resolved ? 'Resolved' : 'Open') + '</span>';
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(date) + '</td>' +
                        '<td>' + escapeHtml(d.impersonated_name || '') + '</td>' +
                        '<td>' + escapeHtml(d.actual_sender || '') + '</td>' +
                        '<td>' + escapeHtml(d.technique || '') + '</td>' +
                        '<td>' + (d.confidence_score ? d.confidence_score.toFixed(1) + '%' : '-') + '</td>' +
                        '<td>' + statusBadge + '</td>' +
                        '<td>' +
                        (!d.resolved ? '<button class="btn btn-xs btn-success resolve-bec" data-id="' + d.id + '"><i class="fa fa-check"></i> Resolve</button>' : '') +
                        '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    $("#addBECProfileBtn").on("click", function () {
        $("#becProfileForm")[0].reset();
        $("#becProfileModal").modal("show");
    });

    $("#saveBECProfileBtn").on("click", function () {
        var data = {
            executive_name: $("#becProfileName").val(),
            email: $("#becProfileEmail").val(),
            title: $("#becProfileTitle").val(),
            risk_level: $("#becProfileRisk").val(),
            known_aliases: $("#becProfileAliases").val()
        };
        api.bec.createProfile(data)
            .success(function () {
                $("#becProfileModal").modal("hide");
                flash("success", "BEC profile created");
                loadBECProfiles();
            })
            .error(function () { flash("danger", "Failed to create BEC profile"); });
    });

    $(document).on("click", ".delete-bec-profile", function () {
        var id = $(this).data("id");
        if (confirm("Delete this BEC profile?")) {
            api.bec.deleteProfile(id)
                .success(function () { loadBECProfiles(); flash("success", "Profile deleted"); })
                .error(function () { flash("danger", "Failed to delete profile"); });
        }
    });

    $(document).on("click", ".resolve-bec", function () {
        var id = $(this).data("id");
        api.bec.resolveDetection(id, { resolution: "confirmed_and_resolved" })
            .success(function () { loadBECDetections(); flash("success", "BEC detection resolved"); })
            .error(function () { flash("danger", "Failed to resolve detection"); });
    });

    // ========== GRAYMAIL TAB ==========
    $("#emailSecurityTabs a[href='#tabGraymail']").on("shown.bs.tab", function () {
        loadGraymailRules();
        loadGraymailClassifications();
        loadGraymailSummary();
    });

    function loadGraymailRules() {
        api.graymail.getRules()
            .success(function (rules) {
                var tbody = $("#graymailRulesBody");
                tbody.empty();
                if (!rules || rules.length === 0) {
                    tbody.append('<tr><td colspan="6" class="text-center text-muted">No graymail rules configured</td></tr>');
                    return;
                }
                $.each(rules, function (i, r) {
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(r.name || '') + '</td>' +
                        '<td>' + escapeHtml(r.category || '') + '</td>' +
                        '<td>' + escapeHtml(r.match_criteria || '') + '</td>' +
                        '<td>' + escapeHtml(r.action || '') + '</td>' +
                        '<td>' + (r.enabled ? '<i class="fa fa-check text-success"></i>' : '<i class="fa fa-times text-muted"></i>') + '</td>' +
                        '<td>' +
                        '<button class="btn btn-xs btn-danger delete-graymail-rule" data-id="' + r.id + '"><i class="fa fa-trash"></i></button>' +
                        '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    function loadGraymailClassifications() {
        api.graymail.getClassifications(20)
            .success(function (items) {
                var tbody = $("#graymailRecentBody");
                tbody.empty();
                if (!items || items.length === 0) {
                    tbody.append('<tr><td colspan="4" class="text-center text-muted">No classifications yet</td></tr>');
                    return;
                }
                $.each(items, function (i, c) {
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(c.sender_email || '') + '</td>' +
                        '<td>' + escapeHtml(c.email_subject || '') + '</td>' +
                        '<td>' + escapeHtml(c.category || '') + '</td>' +
                        '<td>' + (c.confidence_score ? c.confidence_score.toFixed(1) + '%' : '-') + '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    function loadGraymailSummary() {
        api.graymail.getSummary()
            .success(function (summary) {
                if (!summary || !summary.category_breakdown) return;
                renderGraymailCategoryChart(summary.category_breakdown);
            });
    }

    function renderGraymailCategoryChart(breakdown) {
        var el = document.getElementById("graymailCategoryChart");
        if (!el || typeof echarts === "undefined") {
            $("#graymailCategoryChart").html('<div class="text-center text-muted" style="padding:40px;">Chart library not loaded</div>');
            return;
        }
        var chart = echarts.init(el);
        var items = [];
        for (var key in breakdown) {
            items.push({ name: key, value: breakdown[key] });
        }
        chart.setOption({
            tooltip: { trigger: "item" },
            series: [{
                type: "pie",
                radius: "65%",
                data: items
            }]
        });
    }

    $(document).on("click", ".delete-graymail-rule", function () {
        var id = $(this).data("id");
        if (confirm("Delete this graymail rule?")) {
            api.graymail.deleteRule(id)
                .success(function () { loadGraymailRules(); })
                .error(function () { flash("danger", "Failed to delete rule"); });
        }
    });

    // ========== REMEDIATION TAB ==========
    $("#emailSecurityTabs a[href='#tabRemediation']").on("shown.bs.tab", loadRemediationActions);

    function loadRemediationActions() {
        api.remediationActions.get()
            .success(function (actions) {
                var tbody = $("#remediationActionsBody");
                tbody.empty();
                if (!actions || actions.length === 0) {
                    tbody.append('<tr><td colspan="8" class="text-center text-muted">No remediation actions</td></tr>');
                    return;
                }
                $.each(actions, function (i, a) {
                    var date = moment(a.created_date).format("MMM D, h:mm a");
                    var statusBadge = '<span class="status-badge status-' + (a.status || 'pending') + '">' + escapeHtml(a.status || 'pending') + '</span>';
                    var actionBtns = '';
                    if (a.status === 'pending' && a.requires_approval) {
                        actionBtns = '<button class="btn btn-xs btn-success approve-rem" data-id="' + a.id + '"><i class="fa fa-check"></i></button> ' +
                            '<button class="btn btn-xs btn-danger reject-rem" data-id="' + a.id + '"><i class="fa fa-times"></i></button>';
                    }
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(date) + '</td>' +
                        '<td>' + escapeHtml(a.action_type || '') + '</td>' +
                        '<td>' + escapeHtml(a.target_email || '') + '</td>' +
                        '<td>' + escapeHtml(a.subject || '') + '</td>' +
                        '<td>' + escapeHtml(a.scope || '') + '</td>' +
                        '<td>' + statusBadge + '</td>' +
                        '<td>' + (a.affected_count || 0) + '</td>' +
                        '<td>' + actionBtns + '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    $("#newRemediationBtn").on("click", function () {
        $("#remediationForm")[0].reset();
        $("#remediationModal").modal("show");
    });

    $("#executeRemediationBtn").on("click", function () {
        var data = {
            action_type: $("#remActionType").val(),
            target_email: $("#remTargetEmail").val(),
            sender_email: $("#remSenderEmail").val(),
            subject: $("#remSubject").val(),
            message_id: $("#remMessageId").val(),
            scope: $("#remScope").val(),
            justification: $("#remJustification").val()
        };
        api.remediationActions.create(data)
            .success(function () {
                $("#remediationModal").modal("hide");
                flash("success", "Remediation action created");
                loadRemediationActions();
            })
            .error(function () { flash("danger", "Failed to create remediation action"); });
    });

    $(document).on("click", ".approve-rem", function () {
        var id = $(this).data("id");
        api.remediationActions.approve(id)
            .success(function () { loadRemediationActions(); flash("success", "Action approved"); })
            .error(function () { flash("danger", "Failed to approve action"); });
    });

    $(document).on("click", ".reject-rem", function () {
        var id = $(this).data("id");
        api.remediationActions.reject(id, { reason: "Rejected by admin" })
            .success(function () { loadRemediationActions(); flash("success", "Action rejected"); })
            .error(function () { flash("danger", "Failed to reject action"); });
    });

    // ========== TICKETS TAB ==========
    var currentTicketFilter = "all";
    $("#emailSecurityTabs a[href='#tabTickets']").on("shown.bs.tab", function () {
        loadTickets(currentTicketFilter);
    });

    $("#ticketFilter button").on("click", function () {
        $("#ticketFilter button").removeClass("active");
        $(this).addClass("active");
        currentTicketFilter = $(this).data("status");
        loadTickets(currentTicketFilter);
    });

    var currentTicketId = null;

    function loadTickets(status) {
        api.phishingTickets.get(status)
            .success(function (tickets) {
                var tbody = $("#ticketsBody");
                tbody.empty();
                if (!tickets || tickets.length === 0) {
                    tbody.append('<tr><td colspan="8" class="text-center text-muted">No tickets found</td></tr>');
                    return;
                }
                $.each(tickets, function (i, t) {
                    var date = moment(t.created_date).format("MMM D, h:mm a");
                    var priorityBadge = '<span class="priority-' + (t.priority || 'medium') + '">' + escapeHtml(t.priority || 'medium') + '</span>';
                    var statusBadge = '<span class="status-badge status-' + (t.status || 'open') + '">' + escapeHtml(t.status || 'open') + '</span>';
                    var sla = t.sla_deadline ? moment(t.sla_deadline).fromNow() : '-';
                    if (t.sla_breached) sla = '<span class="text-danger"><i class="fa fa-warning"></i> ' + sla + '</span>';
                    tbody.append(
                        '<tr class="ticket-row" data-id="' + t.id + '" style="cursor:pointer;">' +
                        '<td>' + escapeHtml(t.ticket_number || '') + '</td>' +
                        '<td>' + escapeHtml(date) + '</td>' +
                        '<td>' + escapeHtml(t.subject || '') + '</td>' +
                        '<td>' + priorityBadge + '</td>' +
                        '<td>' + statusBadge + '</td>' +
                        '<td>' + escapeHtml(t.assigned_to_name || t.assigned_to_email || 'Unassigned') + '</td>' +
                        '<td>' + sla + '</td>' +
                        '<td>' +
                        '<button class="btn btn-xs btn-info view-ticket" data-id="' + t.id + '"><i class="fa fa-eye"></i></button>' +
                        '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    $(document).on("click", ".view-ticket, .ticket-row", function (e) {
        if ($(e.target).closest("button").length && !$(e.target).closest(".view-ticket").length) return;
        var id = $(this).data("id");
        currentTicketId = id;
        api.phishingTickets.getOne(id)
            .success(function (t) {
                if (!t) return;
                $("#ticketDetailNumber").text(t.ticket_number || "");
                var html = '<div class="row">' +
                    '<div class="col-md-6">' +
                    '<dl class="dl-horizontal">' +
                    '<dt>Subject</dt><dd>' + escapeHtml(t.subject || '') + '</dd>' +
                    '<dt>Reporter</dt><dd>' + escapeHtml(t.reporter_email || '') + '</dd>' +
                    '<dt>Sender</dt><dd>' + escapeHtml(t.sender_email || '') + '</dd>' +
                    '<dt>Priority</dt><dd><span class="priority-' + (t.priority || 'medium') + '">' + escapeHtml(t.priority || '') + '</span></dd>' +
                    '<dt>Status</dt><dd><span class="status-badge status-' + (t.status || 'open') + '">' + escapeHtml(t.status || '') + '</span></dd>' +
                    '</dl></div>' +
                    '<div class="col-md-6">' +
                    '<dl class="dl-horizontal">' +
                    '<dt>Threat Level</dt><dd><span class="threat-' + (t.threat_level || 'safe') + '">' + escapeHtml(t.threat_level || '') + '</span></dd>' +
                    '<dt>Classification</dt><dd>' + escapeHtml(t.classification || '') + '</dd>' +
                    '<dt>Created</dt><dd>' + moment(t.created_date).format("MMMM Do YYYY, h:mm a") + '</dd>' +
                    '<dt>SLA</dt><dd>' + (t.sla_deadline ? moment(t.sla_deadline).format("MMMM Do YYYY, h:mm a") : 'None') + '</dd>' +
                    '<dt>Escalated</dt><dd>' + (t.escalated ? '<span class="text-danger">Yes</span>' : 'No') + '</dd>' +
                    '</dl></div></div>';
                if (t.summary) {
                    html += '<div class="well well-sm"><strong>Summary:</strong> ' + escapeHtml(t.summary) + '</div>';
                }
                if (t.resolution_notes) {
                    html += '<div class="alert alert-success"><strong>Resolution:</strong> ' + escapeHtml(t.resolution_notes) + '</div>';
                }
                $("#ticketDetailBody").html(html);
                // Show/hide action buttons
                var isOpen = t.status === "open" || t.status === "in_progress";
                $("#ticketResolveBtn").toggle(isOpen);
                $("#ticketEscalateBtn").toggle(isOpen && !t.escalated);
                $("#ticketDetailModal").modal("show");
            });
    });

    $("#ticketResolveBtn").on("click", function () {
        if (!currentTicketId) return;
        var notes = prompt("Resolution notes:");
        if (notes === null) return;
        api.phishingTickets.resolve(currentTicketId, { resolution_notes: notes, classification: "resolved" })
            .success(function () {
                $("#ticketDetailModal").modal("hide");
                flash("success", "Ticket resolved");
                loadTickets(currentTicketFilter);
            })
            .error(function () { flash("danger", "Failed to resolve ticket"); });
    });

    $("#ticketEscalateBtn").on("click", function () {
        if (!currentTicketId) return;
        api.phishingTickets.escalate(currentTicketId, {})
            .success(function () {
                $("#ticketDetailModal").modal("hide");
                flash("success", "Ticket escalated");
                loadTickets(currentTicketFilter);
            })
            .error(function () { flash("danger", "Failed to escalate ticket"); });
    });

    // ========== HELPERS ==========
    function flash(type, message) {
        var el = $('<div class="alert alert-' + type + ' alert-dismissible" role="alert">' +
            '<button type="button" class="close" data-dismiss="alert"><span>&times;</span></button>' +
            message + '</div>');
        $("#flashes").append(el);
        setTimeout(function () { el.fadeOut(function () { el.remove(); }); }, 5000);
    }

    function escapeHtml(str) {
        if (!str) return "";
        return $("<div>").text(str).html();
    }
});

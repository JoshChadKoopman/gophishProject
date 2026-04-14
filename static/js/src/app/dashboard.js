var campaigns = []
// statuses is a helper map to point result statuses to ui classes
var statuses = {
    "Email Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "Emails Sent": {
        color: "#1abc9c",
        label: "label-success",
        icon: "fa-envelope",
        point: "ct-point-sent"
    },
    "In progress": {
        label: "label-primary"
    },
    "Queued": {
        label: "label-info"
    },
    "Completed": {
        label: "label-success"
    },
    "Email Opened": {
        color: "#f9bf3b",
        label: "label-warning",
        icon: "fa-envelope",
        point: "ct-point-opened"
    },
    "Email Reported": {
        color: "#45d6ef",
        label: "label-warning",
        icon: "fa-bullhorne",
        point: "ct-point-reported"
    },
    "Clicked Link": {
        color: "#F39C12",
        label: "label-clicked",
        icon: "fa-mouse-pointer",
        point: "ct-point-clicked"
    },
    "Success": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Error": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Error Sending Email": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-times",
        point: "ct-point-error"
    },
    "Submitted Data": {
        color: "#f05b4f",
        label: "label-danger",
        icon: "fa-exclamation",
        point: "ct-point-clicked"
    },
    "Unknown": {
        color: "#6c7a89",
        label: "label-default",
        icon: "fa-question",
        point: "ct-point-error"
    },
    "Sending": {
        color: "#428bca",
        label: "label-primary",
        icon: "fa-spinner",
        point: "ct-point-sending"
    },
    "Campaign Created": {
        label: "label-success",
        icon: "fa-rocket"
    }
}

var statsMapping = {
    "sent": "Email Sent",
    "opened": "Email Opened",
    "email_reported": "Email Reported",
    "clicked": "Clicked Link",
    "submitted_data": "Submitted Data",
}

// ═══════════════════════════════════════════════════════════════
// Real-Time Dashboard — WebSocket, Sparklines, Time Windows
// ═══════════════════════════════════════════════════════════════

var currentTimeWindow = "30d";
var dashboardWS = null;
var wsReconnectTimer = null;
var liveActivityEvents = [];
var MAX_LIVE_EVENTS = 50;

// ── WebSocket Connection Manager ──

function connectDashboardWS() {
    if (dashboardWS && dashboardWS.readyState <= 1) return; // already open/connecting

    var proto = (location.protocol === "https:") ? "wss:" : "ws:";
    var wsUrl = proto + "//" + location.host + "/api/dashboard/ws?api_key=" + user.api_key;

    setWSIndicator("connecting");
    dashboardWS = new WebSocket(wsUrl);

    dashboardWS.onopen = function () {
        setWSIndicator("connected");
        if (wsReconnectTimer) { clearTimeout(wsReconnectTimer); wsReconnectTimer = null; }
        // Tell server our preferred time window
        dashboardWS.send(JSON.stringify({ action: "set_window", window: currentTimeWindow }));
    };

    dashboardWS.onmessage = function (evt) {
        try {
            var msg = JSON.parse(evt.data);
            handleWSMessage(msg);
        } catch (e) { /* ignore malformed */ }
    };

    dashboardWS.onclose = function () {
        setWSIndicator("disconnected");
        // Auto-reconnect after 5 seconds
        wsReconnectTimer = setTimeout(connectDashboardWS, 5000);
    };

    dashboardWS.onerror = function () {
        setWSIndicator("disconnected");
    };
}

function setWSIndicator(state) {
    var el = $("#wsIndicator");
    el.removeClass("ws-connected ws-disconnected ws-connecting");
    switch (state) {
        case "connected":
            el.addClass("ws-connected").html('<i class="fa fa-circle"></i> Live');
            break;
        case "disconnected":
            el.addClass("ws-disconnected").html('<i class="fa fa-circle"></i> Offline');
            break;
        case "connecting":
            el.addClass("ws-connecting").html('<i class="fa fa-circle-o"></i> Connecting…');
            break;
    }
}

function handleWSMessage(msg) {
    // dashboard.pulse carries live counts snapshot
    if (msg.type === "dashboard.pulse" && msg.payload) {
        updateLiveCountCards(msg.payload);
        return;
    }

    // All other events go to the live activity feed and card flash
    addLiveActivityEvent(msg);

    // Increment/flash the relevant card
    switch (msg.type) {
        case "email.sent":
            flashCard("rtEmailsSent");
            break;
        case "email.opened":
            break;
        case "link.clicked":
            flashCard("rtClickRate");
            break;
        case "email.reported":
            flashCard("rtReportRate");
            break;
        case "ticket.created":
            flashCard("rtOpenTickets");
            break;
        case "training.completed":
            flashCard("rtTrainingCompletion");
            break;
        case "campaign.progress":
        case "campaign.completed":
            flashCard("rtActiveCampaigns");
            break;
    }
}

function flashCard(elemId) {
    var el = $("#" + elemId);
    el.css("color", "#e67e22");
    setTimeout(function () { el.css("color", ""); }, 800);
}

// ── Live Activity Feed ──

var eventLabels = {
    "email.sent": { icon: "fa-envelope", color: "#1abc9c", label: "Email Sent" },
    "email.opened": { icon: "fa-envelope-open-o", color: "#f9bf3b", label: "Opened" },
    "link.clicked": { icon: "fa-mouse-pointer", color: "#e74c3c", label: "Clicked" },
    "data.submitted": { icon: "fa-exclamation-circle", color: "#f05b4f", label: "Data Submitted" },
    "email.reported": { icon: "fa-flag", color: "#2ecc71", label: "Reported" },
    "campaign.progress": { icon: "fa-bullhorn", color: "#3498db", label: "Campaign Updated" },
    "campaign.completed": { icon: "fa-check-circle", color: "#27ae60", label: "Campaign Completed" },
    "ticket.created": { icon: "fa-ticket", color: "#9b59b6", label: "New Ticket" },
    "ticket.resolved": { icon: "fa-check", color: "#27ae60", label: "Ticket Resolved" },
    "training.completed": { icon: "fa-graduation-cap", color: "#f39c12", label: "Training Completed" },
};

function addLiveActivityEvent(msg) {
    var info = eventLabels[msg.type] || { icon: "fa-info-circle", color: "#888", label: msg.type };
    var time = msg.timestamp ? moment(msg.timestamp).format("HH:mm:ss") : moment().format("HH:mm:ss");
    var detail = "";
    if (msg.payload) {
        if (msg.payload.email) detail = escapeHtml(msg.payload.email);
        else if (msg.payload.name) detail = escapeHtml(msg.payload.name);
    }

    liveActivityEvents.unshift({ time: time, info: info, detail: detail });
    if (liveActivityEvents.length > MAX_LIVE_EVENTS) liveActivityEvents.pop();

    renderLiveActivityFeed();
}

function renderLiveActivityFeed() {
    var feed = $("#liveActivityFeed");
    if (liveActivityEvents.length === 0) {
        feed.html('<p class="text-muted" style="margin:0; font-size:12px;">Waiting for events…</p>');
        $("#liveActivityCount").text("0");
        return;
    }
    var html = "";
    liveActivityEvents.forEach(function (evt) {
        html += '<div class="live-event">' +
            '<span class="live-event-time">' + evt.time + '</span>' +
            '<i class="fa ' + evt.info.icon + '" style="color:' + evt.info.color + '; margin-right:4px;"></i> ' +
            '<strong>' + evt.info.label + '</strong>' +
            (evt.detail ? ' — <span style="color:#555;">' + evt.detail + '</span>' : '') +
            '</div>';
    });
    feed.html(html);
    $("#liveActivityCount").text(liveActivityEvents.length);
}

// ── Live Counts Update (from WS pulse or polling) ──

function updateLiveCountCards(counts) {
    if (counts.active_campaigns !== undefined) $("#rtActiveCampaigns").text(counts.active_campaigns);
    if (counts.emails_sent_today !== undefined) $("#rtEmailsSent").text(counts.emails_sent_today);
    if (counts.avg_click_rate !== undefined) $("#rtClickRate").text(counts.avg_click_rate.toFixed(1) + "%");
    if (counts.avg_report_rate !== undefined) $("#rtReportRate").text(counts.avg_report_rate.toFixed(1) + "%");
    if (counts.open_tickets !== undefined) $("#rtOpenTickets").text(counts.open_tickets);
}

// ── Full Metrics + Sparklines ──

function loadDashboardMetrics(window) {
    api.dashboard.metrics(window)
        .success(function (data) {
            if (!data || !data.cards) return;
            var c = data.cards;

            // Update card values
            $("#rtActiveCampaigns").text(c.campaigns.active_count);
            $("#rtEmailsSent").text(c.campaigns.emails_sent);
            $("#rtClickRate").text(c.click_rate.current_rate.toFixed(1) + "%");
            $("#rtReportRate").text(c.report_rate.current_rate.toFixed(1) + "%");
            if (c.tickets) $("#rtOpenTickets").text(c.tickets.open_count);
            if (c.training) $("#rtTrainingCompletion").text(c.training.completion_rate.toFixed(1) + "%");

            // Render sparklines
            renderMiniSparkline("sparkCampaigns", c.campaigns.sparkline, "#3498db");
            renderMiniSparkline("sparkEmailsSent", c.campaigns.sparkline, "#1abc9c");
            renderMiniSparkline("sparkClickRate", c.click_rate.sparkline, "#e74c3c");
            renderMiniSparkline("sparkReportRate", c.report_rate.sparkline, "#2ecc71");
            if (c.tickets) renderMiniSparkline("sparkTickets", c.tickets.sparkline, "#9b59b6");
            if (c.training) renderMiniSparkline("sparkTraining", c.training.sparkline, "#f39c12");

            // Render trends
            renderTrendBadge("trendCampaigns", c.campaigns.trend);
            renderTrendBadge("trendEmailsSent", c.campaigns.trend);
            renderTrendBadge("trendClickRate", c.click_rate.trend, true);  // click: down=good
            renderTrendBadge("trendReportRate", c.report_rate.trend, false, true);  // report: up=good
            if (c.tickets) renderTrendBadge("trendTickets", c.tickets.trend);
            if (c.training) renderTrendBadge("trendTraining", c.training.trend);

            // Timestamp
            if (data.generated_at) {
                $("#metricsTimestamp").text("Updated " + moment(data.generated_at).fromNow());
            }
        })
        .error(function () {
            // Fallback: use legacy summary cards
            $("#summaryCards").show();
            api.reports.overview()
                .success(function (overview) {
                    renderSummaryCards(overview);
                    renderRiskGauge(overview);
                });
        });
}

function renderMiniSparkline(containerId, points, color) {
    if (!points || points.length === 0) return;
    var el = document.getElementById(containerId);
    if (!el) return;

    var values = points.map(function (p) { return p.value; });
    var maxVal = Math.max.apply(null, values) || 1;

    // Simple SVG sparkline
    var width = 120, height = 30;
    var step = width / Math.max(values.length - 1, 1);
    var pathParts = [];
    values.forEach(function (v, i) {
        var x = i * step;
        var y = height - (v / maxVal) * (height - 2) - 1;
        pathParts.push((i === 0 ? "M" : "L") + x.toFixed(1) + "," + y.toFixed(1));
    });

    // Fill area
    var areaPath = pathParts.join(" ") +
        " L" + ((values.length - 1) * step).toFixed(1) + "," + height +
        " L0," + height + " Z";

    var svg = '<svg width="' + width + '" height="' + height + '" style="display:block; margin:0 auto;">' +
        '<path d="' + areaPath + '" fill="' + color + '" fill-opacity="0.12" />' +
        '<path d="' + pathParts.join(" ") + '" fill="none" stroke="' + color + '" stroke-width="1.5" />' +
        '<circle cx="' + ((values.length - 1) * step).toFixed(1) + '" cy="' +
        (height - (values[values.length - 1] / maxVal) * (height - 2) - 1).toFixed(1) +
        '" r="2.5" fill="' + color + '" />' +
        '</svg>';
    el.innerHTML = svg;
}

function renderTrendBadge(elemId, trend, clickInvert, reportMode) {
    var el = $("#" + elemId);
    if (!trend || trend === "flat") {
        el.html('<span class="trend-flat">— Flat</span>');
        return;
    }
    if (trend === "up") {
        if (clickInvert) {
            el.html('<span class="trend-up"><i class="fa fa-arrow-up"></i> Up</span>');
        } else if (reportMode) {
            el.html('<span class="trend-report-up"><i class="fa fa-arrow-up"></i> Up</span>');
        } else {
            el.html('<span class="trend-up"><i class="fa fa-arrow-up"></i> Up</span>');
        }
    } else {
        if (clickInvert) {
            el.html('<span class="trend-down"><i class="fa fa-arrow-down"></i> Down</span>');
        } else if (reportMode) {
            el.html('<span class="trend-report-down"><i class="fa fa-arrow-down"></i> Down</span>');
        } else {
            el.html('<span class="trend-down"><i class="fa fa-arrow-down"></i> Down</span>');
        }
    }
}

// ── Time Window Selector ──

function switchTimeWindow(window) {
    currentTimeWindow = window;
    // Save preference
    api.dashboard.setPreference(window);
    // Reload metrics (no page reload)
    loadDashboardMetrics(window);
    // Sync the trend chart to the nearest matching day range
    var dayMap = { "7d": 7, "30d": 30, "90d": 90, "ytd": 365 };
    var days = dayMap[window] || 30;
    renderTrendChart(days);
    // Update the trend range buttons to match
    $("#trendRange button").removeClass("active");
    $("#trendRange button[data-days='" + days + "']").addClass("active");
    // Tell WS server
    if (dashboardWS && dashboardWS.readyState === 1) {
        dashboardWS.send(JSON.stringify({ action: "set_window", window: window }));
    }
}

// ── Polling Fallback ──

var pollingInterval = null;

function startPollingFallback() {
    if (pollingInterval) return;
    pollingInterval = setInterval(function () {
        api.dashboard.liveCounts()
            .success(function (counts) {
                updateLiveCountCards(counts);
            });
    }, 15000);
}

function stopPollingFallback() {
    if (pollingInterval) { clearInterval(pollingInterval); pollingInterval = null; }
}

function deleteCampaign(idx) {
    if (confirm("Delete " + campaigns[idx].name + "?")) {
        api.campaignId.delete(campaigns[idx].id)
            .success(function (data) {
                successFlash(data.message)
                location.reload()
            })
    }
}

/* Renders a pie chart using the provided chartops */
function renderPieChart(chartopts) {
    return Highcharts.chart(chartopts['elemId'], {
        chart: {
            type: 'pie',
            events: {
                load: function () {
                    var chart = this,
                        rend = chart.renderer,
                        pie = chart.series[0],
                        left = chart.plotLeft + pie.center[0],
                        top = chart.plotTop + pie.center[1];
                    this.innerText = rend.text(chartopts['data'][0].count, left, top).
                    attr({
                        'text-anchor': 'middle',
                        'font-size': '16px',
                        'font-weight': 'bold',
                        'fill': chartopts['colors'][0],
                        'font-family': 'Helvetica,Arial,sans-serif'
                    }).add();
                },
                render: function () {
                    this.innerText.attr({
                        text: chartopts['data'][0].count
                    })
                }
            }
        },
        title: {
            text: chartopts['title']
        },
        plotOptions: {
            pie: {
                innerSize: '80%',
                dataLabels: {
                    enabled: false
                }
            }
        },
        credits: {
            enabled: false
        },
        tooltip: {
            formatter: function () {
                if (this.key == undefined) {
                    return false
                }
                return '<span style="color:' + this.color + '">\u25CF</span>' + this.point.name + ': <b>' + this.y + '%</b><br/>'
            }
        },
        series: [{
            data: chartopts['data'],
            colors: chartopts['colors'],
        }]
    })
}

function generateStatsPieCharts(campaigns) {
    var stats_data = []
    var stats_series_data = {}
    var total = 0

    $.each(campaigns, function (i, campaign) {
        $.each(campaign.stats, function (status, count) {
            if (status == "total") {
                total += count
                return true
            }
            if (!stats_series_data[status]) {
                stats_series_data[status] = count;
            } else {
                stats_series_data[status] += count;
            }
        })
    })
    $.each(stats_series_data, function (status, count) {
        if (!(status in statsMapping)) {
            return true
        }
        status_label = statsMapping[status]
        stats_data.push({
            name: status_label,
            y: Math.floor((count / total) * 100),
            count: count
        })
        stats_data.push({
            name: '',
            y: 100 - Math.floor((count / total) * 100)
        })
        var stats_chart = renderPieChart({
            elemId: status + '_chart',
            title: status_label,
            name: status,
            data: stats_data,
            colors: [statuses[status_label].color, "#dddddd"]
        })

        stats_data = []
    });
}

// ---- Enhanced dashboard widgets ----

function renderSummaryCards(overview) {
    var html = '';
    html += '<div class="col-md-3"><div class="well text-center" style="margin-bottom:10px;">' +
        '<i class="fa fa-bullhorn fa-2x" style="color:#3498db; margin-bottom:8px;"></i>' +
        '<h3 style="margin:0; font-weight:700;">' + overview.total_campaigns + '</h3>' +
        '<p style="margin:0; font-size:13px; color:#888;">Total Campaigns</p></div></div>';
    html += '<div class="col-md-3"><div class="well text-center" style="margin-bottom:10px;">' +
        '<i class="fa fa-bolt fa-2x" style="color:#e67e22; margin-bottom:8px;"></i>' +
        '<h3 style="margin:0; font-weight:700;">' + overview.active_campaigns + '</h3>' +
        '<p style="margin:0; font-size:13px; color:#888;">Active Campaigns</p></div></div>';
    html += '<div class="col-md-3"><div class="well text-center" style="margin-bottom:10px;">' +
        '<i class="fa fa-mouse-pointer fa-2x" style="color:#e74c3c; margin-bottom:8px;"></i>' +
        '<h3 style="margin:0; font-weight:700;">' + overview.avg_click_rate + '%</h3>' +
        '<p style="margin:0; font-size:13px; color:#888;">Avg Click Rate</p></div></div>';
    html += '<div class="col-md-3"><div class="well text-center" style="margin-bottom:10px;">' +
        '<i class="fa fa-flag fa-2x" style="color:#2ecc71; margin-bottom:8px;"></i>' +
        '<h3 style="margin:0; font-weight:700;">' + overview.avg_report_rate + '%</h3>' +
        '<p style="margin:0; font-size:13px; color:#888;">Avg Report Rate</p></div></div>';
    $("#summaryCards").html(html);
}

function renderTrendChart(days) {
    api.reports.trend(days)
        .success(function (data) {
            var sentData = [], clickedData = [], reportedData = [];
            data.forEach(function (pt) {
                var ts = moment(pt.date).valueOf();
                sentData.push([ts, pt.sent]);
                clickedData.push([ts, pt.clicked]);
                reportedData.push([ts, pt.reported]);
            });
            Highcharts.chart('trendChart', {
                chart: { zoomType: 'x', type: 'areaspline' },
                title: { text: 'Phishing Event Trends (' + days + ' days)' },
                xAxis: { type: 'datetime' },
                yAxis: { title: { text: 'Events' }, min: 0 },
                tooltip: { shared: true },
                legend: { enabled: true },
                credits: { enabled: false },
                plotOptions: { areaspline: { fillOpacity: 0.15 } },
                series: [
                    { name: 'Sent', data: sentData, color: '#3498db' },
                    { name: 'Clicked', data: clickedData, color: '#e74c3c' },
                    { name: 'Reported', data: reportedData, color: '#2ecc71' }
                ]
            });
        });
}

function renderRiskGauge(overview) {
    var riskValue = overview.avg_click_rate || 0;
    var gaugeColor = '#2ecc71';
    if (riskValue >= 40) gaugeColor = '#e74c3c';
    else if (riskValue >= 20) gaugeColor = '#f39c12';

    Highcharts.chart('riskGauge', {
        chart: { type: 'solidgauge', height: 200 },
        title: null,
        pane: {
            center: ['50%', '75%'],
            size: '130%',
            startAngle: -90,
            endAngle: 90,
            background: {
                backgroundColor: '#eee',
                innerRadius: '60%',
                outerRadius: '100%',
                shape: 'arc',
                borderWidth: 0
            }
        },
        yAxis: { min: 0, max: 100, stops: [[0.3, '#2ecc71'], [0.6, '#f39c12'], [0.9, '#e74c3c']], lineWidth: 0, tickWidth: 0, minorTickInterval: null, labels: { y: 16, style: { fontSize: '11px' } } },
        credits: { enabled: false },
        series: [{ name: 'Risk', data: [Math.round(riskValue)], dataLabels: { format: '<span style="font-size:24px;font-weight:bold;">{y}%</span>', y: -30 } }],
        tooltip: { enabled: false }
    });
}

function renderTrainingWidget() {
    api.reports.trainingSummary()
        .success(function (data) {
            var pct = data.completion_rate || 0;
            var barColor = pct >= 80 ? '#2ecc71' : (pct >= 50 ? '#f39c12' : '#e74c3c');
            var html = '<div style="margin-bottom:12px;">' +
                '<p style="margin:0 0 4px; font-size:13px;"><strong>Completion Rate</strong></p>' +
                '<div class="progress" style="margin-bottom:8px;"><div class="progress-bar" style="width:' + pct + '%; background:' + barColor + ';">' + pct + '%</div></div>' +
                '</div>' +
                '<p style="margin:2px 0; font-size:13px;"><i class="fa fa-book"></i> <strong>' + data.total_courses + '</strong> courses</p>' +
                '<p style="margin:2px 0; font-size:13px;"><i class="fa fa-tasks"></i> <strong>' + data.completed_count + '/' + data.total_assignments + '</strong> assignments done</p>' +
                '<p style="margin:2px 0; font-size:13px;"><i class="fa fa-certificate"></i> <strong>' + data.certificates_issued + '</strong> certificates issued</p>';
            if (data.overdue_count > 0) {
                html += '<p style="margin:2px 0; font-size:13px; color:#e74c3c;"><i class="fa fa-exclamation-triangle"></i> <strong>' + data.overdue_count + '</strong> overdue</p>';
            }
            $("#trainingWidget").html(html);
        })
        .error(function () {
            $("#trainingWidget").html('<p class="text-muted">Unable to load training data.</p>');
        });
}

function renderTopVulnerableUsers() {
    api.reports.riskScores()
        .success(function (data) {
            var tbody = $("#topRiskBody");
            tbody.empty();
            var top5 = data.slice(0, 5);
            if (top5.length === 0) {
                tbody.append('<tr><td colspan="2" class="text-center text-muted">No data</td></tr>');
                return;
            }
            top5.forEach(function (u) {
                var scoreColor = '#2ecc71';
                if (u.risk_score >= 60) scoreColor = '#e74c3c';
                else if (u.risk_score >= 30) scoreColor = '#f39c12';
                tbody.append('<tr><td style="font-size:12px;">' + escapeHtml(u.email) + '</td>' +
                    '<td><span style="color:' + scoreColor + '; font-weight:700;">' + u.risk_score.toFixed(1) + '</span></td></tr>');
            });
        })
        .error(function () {
            $("#topRiskBody").html('<tr><td colspan="2" class="text-muted">Unable to load</td></tr>');
        });
}

$(document).ready(function () {
    Highcharts.setOptions({
        global: {
            useUTC: false
        }
    })

    // ── Load user's saved time window preference ──
    if (api.dashboard && api.dashboard.preference) {
        api.dashboard.preference()
            .success(function (pref) {
                if (pref && pref.time_window && pref.time_window !== currentTimeWindow) {
                    currentTimeWindow = pref.time_window;
                    $("#timeWindowSelector button").removeClass("active");
                    $("#timeWindowSelector button[data-window='" + currentTimeWindow + "']").addClass("active");
                }
                initDashboard();
            })
            .error(function () {
                initDashboard();
            });
    } else {
        initDashboard();
    }

    function initDashboard() {
        // ── Real-time metrics (sparklines + cards) ──
        loadDashboardMetrics(currentTimeWindow);

        // ── WebSocket connection for live updates ──
        try {
            connectDashboardWS();
        } catch (e) {
            // WebSocket not available, use polling fallback
            startPollingFallback();
        }

        // Legacy report widgets (still useful)
        api.reports.overview()
            .success(function (overview) {
                renderSummaryCards(overview);
                renderRiskGauge(overview);
            })
            .error(function () {
                // Widgets fail gracefully
            });

        renderTrendChart(currentTimeWindow === "7d" ? 7 : currentTimeWindow === "90d" ? 90 : 30);
        renderTrainingWidget();
        renderTopVulnerableUsers();
    }

    // ── Time window selector (global, no page reload) ──
    $("#timeWindowSelector button").on("click", function () {
        $("#timeWindowSelector button").removeClass("active");
        $(this).addClass("active");
        switchTimeWindow($(this).data("window"));
    });

    // Trend range buttons (legacy, synced with time window)
    $("#trendRange button").on("click", function () {
        $("#trendRange button").removeClass("active");
        $(this).addClass("active");
        renderTrendChart(parseInt($(this).data("days")));
    });

    // Clear live activity feed
    $("#clearActivity").on("click", function () {
        liveActivityEvents = [];
        renderLiveActivityFeed();
    });

    // Load campaigns table (existing functionality)
    api.campaigns.summary()
        .success(function (data) {
            $("#loading").hide()
            campaigns = data.campaigns
            if (campaigns.length > 0) {
                $("#dashboard").show()
                campaignTable = $("#campaignTable").DataTable({
                    columnDefs: [{
                            orderable: false,
                            targets: "no-sort"
                        },
                        {
                            className: "color-sent",
                            targets: [2]
                        },
                        {
                            className: "color-opened",
                            targets: [3]
                        },
                        {
                            className: "color-clicked",
                            targets: [4]
                        },
                        {
                            className: "color-success",
                            targets: [5]
                        },
                        {
                            className: "color-reported",
                            targets: [6]
                        }
                    ],
                    order: [
                        [1, "desc"]
                    ]
                });
                campaignRows = []
                $.each(campaigns, function (i, campaign) {
                    var campaign_date = moment(campaign.created_date).format('MMMM Do YYYY, h:mm:ss a')
                    var label = statuses[campaign.status].label || "label-default";
                    var launchDate;
                    if (moment(campaign.launch_date).isAfter(moment())) {
                        launchDate = "Scheduled to start: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a')
                        var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total
                    } else {
                        launchDate = "Launch Date: " + moment(campaign.launch_date).format('MMMM Do YYYY, h:mm:ss a')
                        var quickStats = launchDate + "<br><br>" + "Number of recipients: " + campaign.stats.total + "<br><br>" + "Emails opened: " + campaign.stats.opened + "<br><br>" + "Emails clicked: " + campaign.stats.clicked + "<br><br>" + "Submitted Credentials: " + campaign.stats.submitted_data + "<br><br>" + "Errors : " + campaign.stats.error + "<br><br>" + "Reported : " + campaign.stats.email_reported
                    }
                    campaignRows.push([
                        escapeHtml(campaign.name),
                        campaign_date,
                        campaign.stats.sent,
                        campaign.stats.opened,
                        campaign.stats.clicked,
                        campaign.stats.submitted_data,
                        campaign.stats.email_reported,
                        "<span class=\"label " + label + "\" data-toggle=\"tooltip\" data-placement=\"right\" data-html=\"true\" title=\"" + quickStats + "\">" + campaign.status + "</span>",
                        "<div class='pull-right'><a class='btn btn-primary' href='/campaigns/" + campaign.id + "' data-toggle='tooltip' data-placement='left' title='View Results'>\
                    <i class='fa fa-bar-chart'></i>\
                    </a>\
                    <button class='btn btn-danger' onclick='deleteCampaign(" + i + ")' data-toggle='tooltip' data-placement='left' title='Delete Campaign'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                    $('[data-toggle="tooltip"]').tooltip()
                })
                campaignTable.rows.add(campaignRows).draw()
                generateStatsPieCharts(campaigns)
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            errorFlash("Error fetching campaigns")
        })
})

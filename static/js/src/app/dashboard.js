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

    // Load enhanced dashboard widgets
    api.reports.overview()
        .success(function (overview) {
            renderSummaryCards(overview);
            renderRiskGauge(overview);
        })
        .error(function () {
            // Widgets fail gracefully
        });

    renderTrendChart(30);
    renderTrainingWidget();
    renderTopVulnerableUsers();

    // Trend range buttons
    $("#trendRange button").on("click", function () {
        $("#trendRange button").removeClass("active");
        $(this).addClass("active");
        renderTrendChart(parseInt($(this).data("days")));
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

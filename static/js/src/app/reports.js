var tabsLoaded = {};

function renderSummaryCard(icon, label, value, color) {
    return '<div class="col-md-3">' +
        '<div class="well text-center" style="margin-bottom:10px;">' +
        '<i class="fa ' + icon + ' fa-2x" style="color:' + color + '; margin-bottom:8px;"></i>' +
        '<h3 style="margin:0; font-weight:700;">' + value + '</h3>' +
        '<p style="margin:0; font-size:13px; color:#888;">' + label + '</p>' +
        '</div></div>';
}

function renderPieChart(elemId, title, data, colors) {
    if (!document.getElementById(elemId)) return;
    new Highcharts.Chart({
        chart: { renderTo: elemId, type: 'pie' },
        title: { text: title, style: { fontSize: '14px', fontWeight: '600' } },
        plotOptions: {
            pie: {
                innerSize: '60%',
                dataLabels: { enabled: true, format: '{point.name}: {point.y:.1f}%' }
            }
        },
        credits: { enabled: false },
        series: [{ name: 'Percentage', colorByPoint: true, data: data }],
        colors: colors || ['#2ecc71', '#e74c3c', '#3498db', '#f39c12', '#9b59b6']
    });
}

// ---- Overview tab ----
function loadOverview() {
    if (tabsLoaded['overview']) return;
    $("#overviewLoading").show();
    $("#overviewContent").hide();

    api.reports.overview()
        .done(function (data) {
            tabsLoaded['overview'] = true;
            var cards = renderSummaryCard('fa-bullhorn', 'Total Campaigns', data.total_campaigns, '#3498db') +
                renderSummaryCard('fa-bolt', 'Active Campaigns', data.active_campaigns, '#e67e22') +
                renderSummaryCard('fa-mouse-pointer', 'Avg Click Rate', data.avg_click_rate + '%', '#e74c3c') +
                renderSummaryCard('fa-flag', 'Avg Report Rate', data.avg_report_rate + '%', '#2ecc71');
            $("#overviewCards").html(cards);

            var total = data.stats.total || 1;
            renderPieChart('overviewSentChart', 'Emails Sent', [
                { name: 'Sent', y: (data.stats.sent / total) * 100 },
                { name: 'Not Sent', y: ((total - data.stats.sent) / total) * 100 }
            ], ['#3498db', '#ecf0f1']);

            renderPieChart('overviewClickedChart', 'Links Clicked', [
                { name: 'Clicked', y: (data.stats.clicked / total) * 100 },
                { name: 'Not Clicked', y: ((total - data.stats.clicked) / total) * 100 }
            ], ['#e74c3c', '#ecf0f1']);

            renderPieChart('overviewReportedChart', 'Emails Reported', [
                { name: 'Reported', y: (data.stats.email_reported / total) * 100 },
                { name: 'Not Reported', y: ((total - data.stats.email_reported) / total) * 100 }
            ], ['#2ecc71', '#ecf0f1']);

            $("#overviewLoading").hide();
            $("#overviewContent").show();
        })
        .fail(function () {
            $("#overviewLoading").hide();
            errorFlash("Failed to load overview report.");
        });
}

// ---- Groups tab ----
function loadGroups() {
    if (tabsLoaded['groups']) return;
    $("#groupsLoading").show();
    $("#groupsContent").hide();

    api.reports.groupComparison()
        .done(function (data) {
            tabsLoaded['groups'] = true;
            var tbody = $("#groupsTableBody");
            tbody.empty();

            var chartCategories = [];
            var chartClickData = [];
            var chartSubmitData = [];

            data.forEach(function (g) {
                chartCategories.push(g.group_name);
                chartClickData.push(g.click_rate);
                chartSubmitData.push(g.submit_rate);

                tbody.append('<tr>' +
                    '<td>' + escapeHtml(g.group_name) + '</td>' +
                    '<td>' + g.stats.total + '</td>' +
                    '<td>' + g.stats.sent + '</td>' +
                    '<td>' + g.stats.opened + '</td>' +
                    '<td>' + g.stats.clicked + '</td>' +
                    '<td>' + g.stats.submitted_data + '</td>' +
                    '<td>' + g.stats.email_reported + '</td>' +
                    '<td><strong>' + g.click_rate + '%</strong></td>' +
                    '</tr>');
            });

            if (data.length > 0) {
                new Highcharts.Chart({
                    chart: { renderTo: 'groupsChart', type: 'bar' },
                    title: { text: 'Group Comparison — Click & Submit Rates', style: { fontSize: '14px' } },
                    xAxis: { categories: chartCategories },
                    yAxis: { title: { text: 'Rate (%)' }, max: 100 },
                    credits: { enabled: false },
                    series: [
                        { name: 'Click Rate', data: chartClickData, color: '#e74c3c' },
                        { name: 'Submit Rate', data: chartSubmitData, color: '#e67e22' }
                    ]
                });
            }

            $("#groupsLoading").hide();
            $("#groupsContent").show();
        })
        .fail(function () {
            $("#groupsLoading").hide();
            errorFlash("Failed to load group comparison.");
        });
}

// ---- Training tab ----
function loadTraining() {
    if (tabsLoaded['training']) return;
    $("#trainingLoading").show();
    $("#trainingContent").hide();

    api.reports.trainingSummary()
        .done(function (data) {
            tabsLoaded['training'] = true;
            var cards = renderSummaryCard('fa-graduation-cap', 'Total Courses', data.total_courses, '#9b59b6') +
                renderSummaryCard('fa-tasks', 'Assignments', data.total_assignments, '#3498db') +
                renderSummaryCard('fa-check-circle', 'Completed', data.completed_count, '#2ecc71') +
                renderSummaryCard('fa-certificate', 'Certificates', data.certificates_issued, '#f39c12');
            $("#trainingCards").html(cards);

            // Status donut
            renderPieChart('trainingStatusChart', 'Assignment Status', [
                { name: 'Completed', y: data.completed_count },
                { name: 'In Progress', y: data.in_progress_count },
                { name: 'Not Started', y: data.not_started_count },
                { name: 'Overdue', y: data.overdue_count }
            ], ['#2ecc71', '#3498db', '#95a5a6', '#e74c3c']);

            // Metrics panel
            var panel = '<div class="panel panel-default" style="margin-top:20px;">' +
                '<div class="panel-heading"><strong>Training Metrics</strong></div>' +
                '<div class="panel-body">' +
                '<p><strong>Completion Rate:</strong> ' + data.completion_rate + '%</p>' +
                '<div class="progress"><div class="progress-bar progress-bar-success" style="width:' + data.completion_rate + '%;"></div></div>' +
                '<p><strong>Avg Quiz Score:</strong> ' + data.avg_quiz_score + '%</p>' +
                '<p><strong>Overdue Assignments:</strong> <span style="color:' + (data.overdue_count > 0 ? '#e74c3c' : '#2ecc71') + '; font-weight:700;">' + data.overdue_count + '</span></p>' +
                '</div></div>';
            $("#trainingMetricsPanel").html(panel);

            $("#trainingLoading").hide();
            $("#trainingContent").show();
        })
        .fail(function () {
            $("#trainingLoading").hide();
            errorFlash("Failed to load training summary.");
        });
}

// ---- Risk Assessment tab ----
function loadRisk() {
    if (tabsLoaded['risk']) return;
    $("#riskLoading").show();
    $("#riskContent").hide();

    api.reports.riskScores()
        .done(function (data) {
            tabsLoaded['risk'] = true;
            var tbody = $("#riskTableBody");
            tbody.empty();

            // Histogram buckets
            var buckets = [0, 0, 0, 0, 0]; // 0-20, 20-40, 40-60, 60-80, 80-100

            data.forEach(function (u) {
                var scoreColor = '#2ecc71';
                if (u.risk_score >= 60) scoreColor = '#e74c3c';
                else if (u.risk_score >= 30) scoreColor = '#f39c12';

                tbody.append('<tr>' +
                    '<td>' + escapeHtml(u.email) + '</td>' +
                    '<td>' + escapeHtml(u.first_name + ' ' + u.last_name) + '</td>' +
                    '<td>' + u.total_emails + '</td>' +
                    '<td>' + u.clicked + '</td>' +
                    '<td>' + u.submitted + '</td>' +
                    '<td>' + u.reported + '</td>' +
                    '<td><span style="color:' + scoreColor + '; font-weight:700; font-size:15px;">' + u.risk_score.toFixed(1) + '</span></td>' +
                    '</tr>');

                var idx = Math.min(Math.floor(u.risk_score / 20), 4);
                buckets[idx]++;
            });

            // Histogram chart
            if (data.length > 0) {
                new Highcharts.Chart({
                    chart: { renderTo: 'riskHistogram', type: 'column' },
                    title: { text: 'Risk Score Distribution', style: { fontSize: '14px' } },
                    xAxis: { categories: ['0-20 (Low)', '20-40', '40-60 (Medium)', '60-80', '80-100 (High)'] },
                    yAxis: { title: { text: 'Number of Users' }, allowDecimals: false },
                    credits: { enabled: false },
                    legend: { enabled: false },
                    series: [{
                        name: 'Users',
                        data: [
                            { y: buckets[0], color: '#2ecc71' },
                            { y: buckets[1], color: '#27ae60' },
                            { y: buckets[2], color: '#f39c12' },
                            { y: buckets[3], color: '#e67e22' },
                            { y: buckets[4], color: '#e74c3c' }
                        ]
                    }]
                });
            }

            $("#riskLoading").hide();
            $("#riskContent").show();
        })
        .fail(function () {
            $("#riskLoading").hide();
            errorFlash("Failed to load risk scores.");
        });
}

// ---- BRS (Behavioral Risk Score) tab ----
function scoreColor(score) {
    if (score >= 70) return '#2ecc71';
    if (score >= 40) return '#f39c12';
    return '#e74c3c';
}

function loadBRS() {
    if (tabsLoaded['brs']) return;
    $("#brsLoading").show();
    $("#brsContent").hide();

    api.brs.leaderboard(50)
        .done(function (data) {
            tabsLoaded['brs'] = true;

            // Leaderboard table
            var tbody = $("#brsTableBody");
            tbody.empty();
            var buckets = [0, 0, 0, 0, 0];

            data.forEach(function (u, idx) {
                var c = scoreColor(u.composite_score);
                tbody.append('<tr>' +
                    '<td>' + (idx + 1) + '</td>' +
                    '<td>' + escapeHtml(u.first_name + ' ' + u.last_name) + '</td>' +
                    '<td>' + escapeHtml(u.email) + '</td>' +
                    '<td>' + escapeHtml(u.department || '—') + '</td>' +
                    '<td>' + u.simulation_score.toFixed(1) + '</td>' +
                    '<td>' + u.academy_score.toFixed(1) + '</td>' +
                    '<td>' + u.quiz_score.toFixed(1) + '</td>' +
                    '<td>' + u.trend_score.toFixed(1) + '</td>' +
                    '<td>' + u.consistency_score.toFixed(1) + '</td>' +
                    '<td><span style="color:' + c + '; font-weight:700; font-size:15px;">' + u.composite_score.toFixed(1) + '</span></td>' +
                    '<td>' + u.percentile.toFixed(0) + '%</td>' +
                    '</tr>');

                var bi = Math.min(Math.floor(u.composite_score / 20), 4);
                buckets[bi]++;
            });

            // Summary cards
            var avgScore = 0;
            if (data.length > 0) {
                var sum = 0;
                data.forEach(function (u) { sum += u.composite_score; });
                avgScore = (sum / data.length).toFixed(1);
            }
            var cards = renderSummaryCard('fa-shield', 'Users Scored', data.length, '#3498db') +
                renderSummaryCard('fa-line-chart', 'Avg BRS', avgScore, scoreColor(parseFloat(avgScore))) +
                renderSummaryCard('fa-arrow-up', 'Best Score', data.length > 0 ? data[0].composite_score.toFixed(1) : '—', '#2ecc71') +
                renderSummaryCard('fa-arrow-down', 'Worst Score', data.length > 0 ? data[data.length - 1].composite_score.toFixed(1) : '—', '#e74c3c');
            $("#brsCards").html(cards);

            // Distribution chart
            if (data.length > 0) {
                new Highcharts.Chart({
                    chart: { renderTo: 'brsDistributionChart', type: 'column' },
                    title: { text: 'BRS Distribution', style: { fontSize: '14px' } },
                    xAxis: { categories: ['0-20 (High Risk)', '20-40', '40-60 (Medium)', '60-80', '80-100 (Low Risk)'] },
                    yAxis: { title: { text: 'Number of Users' }, allowDecimals: false },
                    credits: { enabled: false },
                    legend: { enabled: false },
                    series: [{
                        name: 'Users',
                        data: [
                            { y: buckets[0], color: '#e74c3c' },
                            { y: buckets[1], color: '#e67e22' },
                            { y: buckets[2], color: '#f39c12' },
                            { y: buckets[3], color: '#27ae60' },
                            { y: buckets[4], color: '#2ecc71' }
                        ]
                    }]
                });
            }

            $("#brsLoading").hide();
            $("#brsContent").show();

            // Load department chart (may fail if feature not available)
            loadBRSDepartment();
            loadBRSBenchmark();
        })
        .fail(function () {
            $("#brsLoading").hide();
            errorFlash("Failed to load BRS data.");
        });
}

function loadBRSDepartment() {
    api.brs.department()
        .done(function (data) {
            if (!data || data.length === 0) return;
            var cats = [];
            var scores = [];
            data.forEach(function (d) {
                cats.push(d.department);
                scores.push(d.composite_score);
            });
            new Highcharts.Chart({
                chart: { renderTo: 'brsDepartmentChart', type: 'bar' },
                title: { text: 'Department BRS', style: { fontSize: '14px' } },
                xAxis: { categories: cats },
                yAxis: { title: { text: 'Composite Score' }, min: 0, max: 100 },
                credits: { enabled: false },
                legend: { enabled: false },
                series: [{
                    name: 'BRS',
                    data: scores,
                    color: '#3498db'
                }]
            });
        });
}

function loadBRSBenchmark() {
    api.brs.benchmark()
        .done(function (data) {
            if (!data) return;
            var html = '<div class="row">' +
                '<div class="col-md-3 text-center"><h4>' + data.org_avg_score.toFixed(1) + '</h4><p class="text-muted">Org Average</p></div>' +
                '<div class="col-md-3 text-center"><h4>' + data.org_median_score.toFixed(1) + '</h4><p class="text-muted">Org Median</p></div>' +
                '<div class="col-md-3 text-center"><h4>' + data.global_avg_score.toFixed(1) + '</h4><p class="text-muted">Global Average</p></div>' +
                '<div class="col-md-3 text-center"><h4>' + data.global_median_score.toFixed(1) + '</h4><p class="text-muted">Global Median</p></div>' +
                '</div>';
            $("#brsBenchmarkContent").html(html);
            $("#brsBenchmarkRow").show();
        });
}

$(document).ready(function () {
    // Load overview immediately (active tab)
    loadOverview();

    // Lazy-load other tabs on first activation
    $('a[data-toggle="tab"]').on('shown.bs.tab', function (e) {
        var target = $(e.target).attr('href');
        if (target === '#groupsTab') loadGroups();
        else if (target === '#trainingTab') loadTraining();
        else if (target === '#riskTab') loadRisk();
        else if (target === '#brsTab') loadBRS();
    });

    // Recalculate button
    $("#brsRecalcBtn").on('click', function () {
        var btn = $(this);
        btn.prop('disabled', true).find('i').addClass('fa-spin');
        api.brs.recalculate()
            .done(function () {
                successFlash("BRS recalculation started. Refresh in a few minutes to see updated scores.");
                tabsLoaded['brs'] = false;
                setTimeout(function () {
                    btn.prop('disabled', false).find('i').removeClass('fa-spin');
                }, 3000);
            })
            .fail(function () {
                errorFlash("Failed to trigger BRS recalculation.");
                btn.prop('disabled', false).find('i').removeClass('fa-spin');
            });
    });
});

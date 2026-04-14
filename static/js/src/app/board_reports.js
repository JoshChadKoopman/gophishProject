$(document).ready(function () {

    loadReports();

    // ---- Load saved reports ----
    function loadReports() {
        api.boardReports.get()
            .success(function (reports) {
                renderReports(reports);
            })
            .error(function () {
                errorFlash("Failed to load board reports.");
            });
    }

    function renderReports(reports) {
        var body = $("#reportsBody");
        body.empty();

        if (!reports || reports.length === 0) {
            $("#reportsTable").closest(".panel").hide();
            $("#reportsEmpty").show();
            return;
        }
        $("#reportsTable").closest(".panel").show();
        $("#reportsEmpty").hide();

        $.each(reports, function (i, r) {
            var statusBadge = r.status === "published"
                ? '<span class="label label-success">Published</span>'
                : '<span class="label label-default">Draft</span>';
            var periodStart = r.period_start ? moment(r.period_start).format("MMM D, YYYY") : "—";
            var periodEnd = r.period_end ? moment(r.period_end).format("MMM D, YYYY") : "—";
            var created = r.created_date ? moment(r.created_date).format("MMM D, YYYY") : "—";

            body.append(
                '<tr>' +
                '<td><strong>' + escapeHtml(r.title) + '</strong></td>' +
                '<td>' + periodStart + ' — ' + periodEnd + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td>' + created + '</td>' +
                '<td>' +
                '<button class="btn btn-xs btn-info view-report" data-id="' + r.id + '"><i class="fa fa-eye"></i> View</button> ' +
                '<button class="btn btn-xs btn-success publish-report" data-id="' + r.id + '" data-status="' + r.status + '">' +
                (r.status === 'draft' ? '<i class="fa fa-check"></i> Publish' : '<i class="fa fa-pencil"></i> Draft') + '</button> ' +
                '<button class="btn btn-xs btn-danger delete-report" data-id="' + r.id + '"><i class="fa fa-trash"></i></button>' +
                '</td>' +
                '</tr>'
            );
        });
    }

    // ---- View Report Detail ----
    $(document).on("click", ".view-report", function () {
        var id = $(this).data("id");
        $("#reportDetailBody").html('<p class="text-muted"><i class="fa fa-spinner fa-spin"></i> Loading report data...</p>');
        $("#reportDetailModal").modal("show");

        api.boardReports.getOne(id)
            .success(function (br) {
                $("#reportDetailTitle").text(br.title);
                $("#exportPDF").attr("href", api.boardReports.exportUrl(id, "pdf"));
                $("#exportXLSX").attr("href", api.boardReports.exportUrl(id, "xlsx"));
                $("#exportCSV").attr("href", api.boardReports.exportUrl(id, "csv"));

                if (br.snapshot) {
                    renderSnapshot($("#reportDetailBody"), br.snapshot);
                } else {
                    $("#reportDetailBody").html('<p class="text-muted">No data available.</p>');
                }
            })
            .error(function () {
                $("#reportDetailBody").html('<p class="text-danger">Failed to load report.</p>');
            });
    });

    // ---- Render Snapshot ----
    function renderSnapshot(container, snap) {
        var trendIcon = snap.risk_trend === 'improving' ? '<i class="fa fa-arrow-up text-success"></i>' :
            snap.risk_trend === 'declining' ? '<i class="fa fa-arrow-down text-danger"></i>' :
                '<i class="fa fa-minus text-muted"></i>';

        var scoreColor = snap.security_posture_score >= 70 ? '#2ecc71' :
            snap.security_posture_score >= 40 ? '#f39c12' : '#e74c3c';

        var html = '';

        // Security Posture
        html += '<div class="text-center" style="margin-bottom:20px;">' +
            '<h2 style="margin:0;"><span style="color:' + scoreColor + '; font-weight:800; font-size:48px;">' +
            Math.round(snap.security_posture_score) + '</span><span style="font-size:20px; color:#888;">/100</span></h2>' +
            '<p style="font-size:16px; margin:4px 0;">Security Posture Score ' + trendIcon + ' ' + snap.risk_trend + '</p>' +
            '<p class="text-muted">' + escapeHtml(snap.period_label) + '</p>' +
            '</div>';

        // Summary cards row
        html += '<div class="row">';
        html += summaryCard('fa-crosshairs', 'Avg Click Rate', snap.phishing.avg_click_rate.toFixed(1) + '%', snap.phishing.avg_click_rate > 25 ? '#e74c3c' : '#2ecc71');
        html += summaryCard('fa-graduation-cap', 'Training Completion', snap.training.completion_rate.toFixed(0) + '%', snap.training.completion_rate >= 70 ? '#2ecc71' : '#f39c12');
        html += summaryCard('fa-shield', 'Compliance Score', snap.compliance.overall_score.toFixed(0) + '%', snap.compliance.overall_score >= 70 ? '#2ecc71' : '#e74c3c');
        html += summaryCard('fa-laptop', 'Hygiene Score', snap.hygiene.avg_score.toFixed(0) + '%', snap.hygiene.avg_score >= 70 ? '#2ecc71' : '#f39c12');
        html += '</div>';

        // Phishing section
        html += sectionHeader('1. Phishing Simulations');
        html += metricRow([
            ['Total Campaigns', snap.phishing.total_campaigns],
            ['Total Recipients', snap.phishing.total_recipients],
            ['Avg Click Rate', snap.phishing.avg_click_rate.toFixed(1) + '%'],
            ['Avg Submit Rate', snap.phishing.avg_submit_rate.toFixed(1) + '%'],
            ['Avg Report Rate', snap.phishing.avg_report_rate.toFixed(1) + '%']
        ]);

        // Training section
        html += sectionHeader('2. Training & Awareness');
        html += metricRow([
            ['Completion Rate', snap.training.completion_rate.toFixed(0) + '%'],
            ['Total Courses', snap.training.total_courses],
            ['Overdue', snap.training.overdue_count],
            ['Avg Quiz Score', snap.training.avg_quiz_score.toFixed(0) + '%'],
            ['Certificates', snap.training.certificates_issued]
        ]);

        // Risk section
        html += sectionHeader('3. Risk Assessment');
        html += metricRow([
            ['High Risk Users', snap.risk.high_risk_users],
            ['Medium Risk Users', snap.risk.medium_risk_users],
            ['Low Risk Users', snap.risk.low_risk_users],
            ['Avg Risk Score', snap.risk.avg_risk_score.toFixed(1)]
        ]);

        // Compliance section
        html += sectionHeader('4. Compliance Posture');
        html += metricRow([
            ['Frameworks', snap.compliance.framework_count],
            ['Overall Score', snap.compliance.overall_score.toFixed(0) + '%'],
            ['Compliant', snap.compliance.compliant],
            ['Partial', snap.compliance.partial],
            ['Non-Compliant', snap.compliance.non_compliant]
        ]);

        // Remediation section
        html += sectionHeader('5. Remediation Progress');
        html += metricRow([
            ['Total Paths', snap.remediation.total_paths],
            ['Active', snap.remediation.active_paths],
            ['Completed', snap.remediation.completed_paths],
            ['Critical', snap.remediation.critical_count],
            ['Avg Completion', Math.round(snap.remediation.avg_completion_pct) + '%']
        ]);

        // Hygiene section
        html += sectionHeader('6. Cyber Hygiene');
        html += metricRow([
            ['Total Devices', snap.hygiene.total_devices],
            ['Avg Score', snap.hygiene.avg_score.toFixed(0) + '%'],
            ['Fully Compliant', snap.hygiene.fully_compliant],
            ['At Risk Devices', snap.hygiene.at_risk_devices]
        ]);

        // Recommendations
        html += sectionHeader('Key Recommendations');
        if (snap.recommendations && snap.recommendations.length > 0) {
            html += '<ol style="padding-left:20px;">';
            $.each(snap.recommendations, function (i, rec) {
                html += '<li style="margin-bottom:6px;">' + escapeHtml(rec) + '</li>';
            });
            html += '</ol>';
        } else {
            html += '<p class="text-success"><i class="fa fa-check-circle"></i> Security posture is strong.</p>';
        }

        container.html(html);
    }

    function summaryCard(icon, label, value, color) {
        return '<div class="col-md-3">' +
            '<div class="well text-center" style="margin-bottom:10px;">' +
            '<i class="fa ' + icon + ' fa-2x" style="color:' + color + '; margin-bottom:8px;"></i>' +
            '<h3 style="margin:0; font-weight:700; color:' + color + ';">' + value + '</h3>' +
            '<p style="margin:0; font-size:13px; color:#888;">' + label + '</p>' +
            '</div></div>';
    }

    function sectionHeader(title) {
        return '<h4 style="margin-top:20px; border-bottom:2px solid #3498db; padding-bottom:6px; color:#2c3e50;">' + title + '</h4>';
    }

    function metricRow(metrics) {
        var html = '<div class="row" style="margin-bottom:10px;">';
        var colSize = metrics.length <= 4 ? 3 : 2;
        $.each(metrics, function (i, m) {
            html += '<div class="col-md-' + colSize + ' text-center" style="margin-bottom:6px;">' +
                '<strong style="font-size:18px;">' + m[1] + '</strong>' +
                '<br><span style="font-size:12px; color:#888;">' + m[0] + '</span>' +
                '</div>';
        });
        html += '</div>';
        return html;
    }

    // ---- New Report ----
    $("#newReportBtn").on("click", function () {
        // Default period: last 3 months
        var end = new Date();
        var start = new Date();
        start.setMonth(start.getMonth() - 3);
        $("#reportStart").val(start.toISOString().slice(0, 10));
        $("#reportEnd").val(end.toISOString().slice(0, 10));
        $("#reportTitle").val("");
        $("#newReportModal").modal("show");
    });

    $("#saveReportBtn").on("click", function () {
        var title = $("#reportTitle").val().trim();
        if (!title) {
            errorFlash("Report title is required.");
            return;
        }
        var data = {
            title: title,
            period_start: $("#reportStart").val(),
            period_end: $("#reportEnd").val()
        };
        api.boardReports.create(data)
            .success(function () {
                successFlash("Board report created.");
                $("#newReportModal").modal("hide");
                loadReports();
            })
            .error(function (resp) {
                errorFlash(resp.responseJSON ? resp.responseJSON.message : "Failed to create report.");
            });
    });

    // ---- Quick Preview ----
    $("#previewBtn").on("click", function () {
        var end = new Date();
        var start = new Date();
        start.setMonth(start.getMonth() - 3);
        var data = {
            period_start: start.toISOString().slice(0, 10),
            period_end: end.toISOString().slice(0, 10)
        };
        $("#previewPanel").show();
        $("#previewContent").html('<p class="text-muted"><i class="fa fa-spinner fa-spin"></i> Generating executive summary...</p>');

        api.boardReports.generate(data)
            .success(function (snap) {
                renderSnapshot($("#previewContent"), snap);
            })
            .error(function () {
                $("#previewContent").html('<p class="text-danger">Failed to generate preview.</p>');
            });
    });

    $("#closePreview").on("click", function () {
        $("#previewPanel").hide();
    });

    // ---- Toggle Publish / Draft ----
    $(document).on("click", ".publish-report", function () {
        var id = $(this).data("id");
        var currentStatus = $(this).data("status");
        var newStatus = currentStatus === "draft" ? "published" : "draft";
        api.boardReports.update(id, { status: newStatus })
            .success(function () {
                successFlash("Report status updated.");
                loadReports();
            })
            .error(function () {
                errorFlash("Failed to update report status.");
            });
    });

    // ---- Delete Report ----
    $(document).on("click", ".delete-report", function () {
        var id = $(this).data("id");
        if (!confirm("Delete this board report?")) return;
        api.boardReports.remove(id)
            .success(function () {
                successFlash("Report deleted.");
                loadReports();
            })
            .error(function () {
                errorFlash("Failed to delete report.");
            });
    });
});

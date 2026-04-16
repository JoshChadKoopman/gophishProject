$(document).ready(function () {

    // ── State ──
    var currentReportId = null;
    var currentReportStatus = null;
    var statusTransitions = {
        "draft": ["review"],
        "review": ["approved", "draft"],
        "approved": ["published", "draft"],
        "published": ["draft"]
    };

    loadReports();
    loadHeatmap();

    // ── Helpers ──
    function statusBadge(status) {
        var map = {
            "draft": '<span class="label label-default">Draft</span>',
            "review": '<span class="label label-warning">In Review</span>',
            "approved": '<span class="label label-info">Approved</span>',
            "published": '<span class="label label-success">Published</span>'
        };
        return map[status] || '<span class="label label-default">' + escapeHtml(status) + '</span>';
    }

    // ── Load saved reports ──
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
            var periodStart = r.period_start ? moment(r.period_start).format("MMM D, YYYY") : "—";
            var periodEnd = r.period_end ? moment(r.period_end).format("MMM D, YYYY") : "—";
            var created = r.created_date ? moment(r.created_date).format("MMM D, YYYY") : "—";

            body.append(
                '<tr>' +
                '<td><strong>' + escapeHtml(r.title) + '</strong></td>' +
                '<td>' + periodStart + ' — ' + periodEnd + '</td>' +
                '<td>' + statusBadge(r.status) + '</td>' +
                '<td>' + created + '</td>' +
                '<td>' +
                '<button class="btn btn-xs btn-info view-report" data-id="' + r.id + '"><i class="fa fa-eye"></i> View</button> ' +
                '<button class="btn btn-xs btn-warning workflow-btn" data-id="' + r.id + '" data-status="' + r.status + '"><i class="fa fa-exchange"></i> Status</button> ' +
                '<button class="btn btn-xs btn-danger delete-report" data-id="' + r.id + '"><i class="fa fa-trash"></i></button>' +
                '</td>' +
                '</tr>'
            );
        });
    }

    // ── Department Risk Heatmap ──
    function loadHeatmap() {
        api.boardReports.getHeatmap()
            .success(function (heatmap) {
                renderHeatmap(heatmap);
            })
            .error(function () {
                $("#heatmapContent").html('<p class="text-muted text-center" style="padding:20px;">Unable to load heatmap. Ensure departments are configured on user accounts.</p>');
            });
    }

    function renderHeatmap(heatmap) {
        if (!heatmap || !heatmap.rows || heatmap.rows.length === 0) {
            $("#heatmapContent").html('<p class="text-muted text-center" style="padding:20px;">No department data available. Assign departments to users to enable the risk heatmap.</p>');
            return;
        }
        var cols = heatmap.columns || [];
        var rows = heatmap.rows || [];

        var html = '<table class="table table-bordered" style="margin:0; font-size:13px;">';
        html += '<thead><tr style="background:#34495e; color:#fff;">';
        html += '<th style="min-width:140px;">Department</th>';
        html += '<th style="width:60px; text-align:center;">Users</th>';
        $.each(cols, function (i, col) {
            html += '<th style="text-align:center; min-width:100px;">' + escapeHtml(col.label) + '</th>';
        });
        html += '</tr></thead><tbody>';

        $.each(rows, function (i, row) {
            html += '<tr>';
            html += '<td><strong>' + escapeHtml(row.department) + '</strong></td>';
            html += '<td style="text-align:center;">' + row.user_count + '</td>';
            $.each(cols, function (j, col) {
                var cell = row.cells[col.key];
                if (cell) {
                    var textColor = (cell.level === "critical" || cell.level === "high") ? "#fff" : "#333";
                    html += '<td style="text-align:center; background:' + cell.color + '; color:' + textColor + '; font-weight:600;">' +
                        cell.value.toFixed(1) + '%' +
                        '<br><small style="font-weight:400; opacity:0.8;">' + cell.level + '</small></td>';
                } else {
                    html += '<td style="text-align:center; color:#999;">—</td>';
                }
            });
            html += '</tr>';
        });

        html += '</tbody></table>';
        // Legend
        html += '<div style="padding:8px 12px; font-size:11px; color:#666; border-top:1px solid #eee;">' +
            '<i class="fa fa-square" style="color:#2ecc71;"></i> Low Risk &nbsp;&nbsp;' +
            '<i class="fa fa-square" style="color:#f1c40f;"></i> Medium &nbsp;&nbsp;' +
            '<i class="fa fa-square" style="color:#e67e22;"></i> High &nbsp;&nbsp;' +
            '<i class="fa fa-square" style="color:#e74c3c;"></i> Critical' +
            '</div>';
        $("#heatmapContent").html(html);
    }

    $("#refreshHeatmap").on("click", function () {
        $("#heatmapContent").html('<p class="text-muted text-center" style="padding:30px;"><i class="fa fa-spinner fa-spin"></i> Refreshing...</p>');
        loadHeatmap();
    });

    // ── View Full Report (Enhanced) ──
    $(document).on("click", ".view-report", function () {
        var id = $(this).data("id");
        currentReportId = id;
        $("#reportDetailBody").html('<p class="text-muted"><i class="fa fa-spinner fa-spin"></i> Loading enhanced report...</p>');
        $("#reportDetailModal").modal("show");

        api.boardReports.getFull(id)
            .success(function (payload) {
                currentReportStatus = payload.status;
                $("#reportDetailTitle").text(payload.title);
                $("#reportStatusBadge").html(statusBadge(payload.status));
                $("#exportPDF").attr("href", api.boardReports.exportUrl(id, "pdf"));
                $("#exportXLSX").attr("href", api.boardReports.exportUrl(id, "xlsx"));
                $("#exportCSV").attr("href", api.boardReports.exportUrl(id, "csv"));
                renderFullReport($("#reportDetailBody"), payload);
            })
            .error(function () {
                // Fallback to basic view
                api.boardReports.getOne(id)
                    .success(function (br) {
                        $("#reportDetailTitle").text(br.title);
                        $("#reportStatusBadge").html(statusBadge(br.status));
                        currentReportStatus = br.status;
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
    });

    // ── Render Full Enhanced Report ──
    function renderFullReport(container, payload) {
        var html = '';

        // ─── AI Narrative Section ───
        html += '<div style="background:#f0f7ff; border-radius:6px; padding:16px 20px; margin-bottom:20px; border-left:4px solid #3498db;">';
        html += '<h4 style="margin-top:0; color:#2c3e50;"><i class="fa fa-file-text-o"></i> Executive Summary';
        html += ' <button class="btn btn-xs btn-default pull-right edit-narrative-btn" data-id="' + payload.id + '"><i class="fa fa-pencil"></i> Edit</button>';
        html += ' <button class="btn btn-xs btn-primary pull-right generate-narrative-btn" data-id="' + payload.id + '" style="margin-right:6px;"><i class="fa fa-magic"></i> Generate AI Narrative</button>';
        html += '</h4>';
        if (payload.narrative && payload.narrative.full_narrative) {
            var paras = [payload.narrative.paragraph1, payload.narrative.paragraph2, payload.narrative.paragraph3];
            $.each(paras, function (i, p) {
                if (p) {
                    html += '<p style="margin-bottom:10px; line-height:1.6; color:#333;">' + escapeHtml(p) + '</p>';
                }
            });
            if (payload.narrative.ai_generated) {
                html += '<small class="text-muted"><i class="fa fa-robot"></i> AI-generated narrative — review before publishing</small>';
            }
        } else {
            html += '<p class="text-muted"><i class="fa fa-info-circle"></i> No executive narrative generated yet. Click "Generate AI Narrative" to create one.</p>';
        }
        html += '</div>';

        // ─── Period-Over-Period Deltas ───
        if (payload.deltas && payload.deltas.length > 0) {
            html += '<h4 style="border-bottom:2px solid #3498db; padding-bottom:6px; color:#2c3e50;"><i class="fa fa-exchange"></i> Period-Over-Period Changes</h4>';
            html += '<div class="row" style="margin-bottom:16px;">';
            $.each(payload.deltas, function (i, d) {
                var color = d.favorable ? '#2ecc71' : (d.direction === 'flat' ? '#95a5a6' : '#e74c3c');
                var bg = d.favorable ? 'rgba(46,204,113,0.08)' : (d.direction === 'flat' ? 'rgba(149,165,166,0.08)' : 'rgba(231,76,60,0.08)');
                html += '<div class="col-md-4 col-sm-6" style="margin-bottom:8px;">' +
                    '<div style="background:' + bg + '; border-radius:6px; padding:10px 14px; border-left:3px solid ' + color + ';">' +
                    '<strong style="color:' + color + '; font-size:16px;">' + escapeHtml(d.arrow) + ' ' + Math.abs(d.abs_change).toFixed(1) + '</strong>' +
                    '<br><span style="font-size:12px; color:#666;">' + escapeHtml(d.label) + '</span>' +
                    '<br><span style="font-size:11px; color:#999;">' + d.prior_value.toFixed(1) + ' → ' + d.current_value.toFixed(1) + '</span>' +
                    '</div></div>';
            });
            html += '</div>';
        }

        // ─── Snapshot Data ───
        if (payload.snapshot) {
            html += renderSnapshotHTML(payload.snapshot);
        }

        // ─── Workflow & Approval Trail ───
        html += '<h4 style="border-bottom:2px solid #9b59b6; padding-bottom:6px; color:#2c3e50; margin-top:24px;"><i class="fa fa-shield"></i> Approval Workflow</h4>';
        html += '<div style="margin-bottom:10px;">';
        html += '<span style="font-weight:600;">Current Status:</span> ' + statusBadge(payload.status) + ' &nbsp;';
        html += '<button class="btn btn-xs btn-warning workflow-modal-btn" data-id="' + payload.id + '" data-status="' + payload.status + '"><i class="fa fa-exchange"></i> Change Status</button>';
        html += '</div>';

        // Approval history
        if (payload.approvals && payload.approvals.length > 0) {
            html += '<table class="table table-condensed" style="font-size:12px;">';
            html += '<thead><tr><th>Date</th><th>Transition</th><th>By</th><th>Comment</th></tr></thead><tbody>';
            $.each(payload.approvals, function (i, a) {
                html += '<tr>';
                html += '<td>' + moment(a.created_date).format("MMM D, YYYY HH:mm") + '</td>';
                html += '<td>' + statusBadge(a.from_status) + ' → ' + statusBadge(a.to_status) + '</td>';
                html += '<td>' + escapeHtml(a.username) + '</td>';
                html += '<td>' + escapeHtml(a.comment || '—') + '</td>';
                html += '</tr>';
            });
            html += '</tbody></table>';
        } else {
            html += '<p class="text-muted" style="font-size:12px;">No status changes recorded yet.</p>';
        }

        container.html(html);
    }

    // ── Render Snapshot (reusable) ──
    function renderSnapshotHTML(snap) {
        var trendIcon = snap.risk_trend === 'improving' ? '<i class="fa fa-arrow-up text-success"></i>' :
            snap.risk_trend === 'declining' ? '<i class="fa fa-arrow-down text-danger"></i>' :
                '<i class="fa fa-minus text-muted"></i>';
        var scoreColor = snap.security_posture_score >= 70 ? '#2ecc71' :
            snap.security_posture_score >= 40 ? '#f39c12' : '#e74c3c';

        var html = '';

        // Security Posture
        html += '<div class="text-center" style="margin-bottom:20px;">';
        html += '<h2 style="margin:0;"><span style="color:' + scoreColor + '; font-weight:800; font-size:48px;">' +
            Math.round(snap.security_posture_score) + '</span><span style="font-size:20px; color:#888;">/100</span></h2>';
        html += '<p style="font-size:16px; margin:4px 0;">Security Posture Score ' + trendIcon + ' ' + snap.risk_trend + '</p>';
        html += '<p class="text-muted">' + escapeHtml(snap.period_label) + '</p>';
        html += '</div>';

        // Summary cards
        html += '<div class="row">';
        html += summaryCard('fa-crosshairs', 'Avg Click Rate', snap.phishing.avg_click_rate.toFixed(1) + '%', snap.phishing.avg_click_rate > 25 ? '#e74c3c' : '#2ecc71');
        html += summaryCard('fa-graduation-cap', 'Training Completion', snap.training.completion_rate.toFixed(0) + '%', snap.training.completion_rate >= 70 ? '#2ecc71' : '#f39c12');
        html += summaryCard('fa-shield', 'Compliance Score', snap.compliance.overall_score.toFixed(0) + '%', snap.compliance.overall_score >= 70 ? '#2ecc71' : '#e74c3c');
        html += summaryCard('fa-laptop', 'Hygiene Score', snap.hygiene.avg_score.toFixed(0) + '%', snap.hygiene.avg_score >= 70 ? '#2ecc71' : '#f39c12');
        html += '</div>';

        // Sections
        html += sectionHeader('Phishing Simulations');
        html += metricRow([
            ['Total Campaigns', snap.phishing.total_campaigns],
            ['Total Recipients', snap.phishing.total_recipients],
            ['Avg Click Rate', snap.phishing.avg_click_rate.toFixed(1) + '%'],
            ['Avg Submit Rate', snap.phishing.avg_submit_rate.toFixed(1) + '%'],
            ['Avg Report Rate', snap.phishing.avg_report_rate.toFixed(1) + '%']
        ]);
        html += sectionHeader('Training & Awareness');
        html += metricRow([
            ['Completion Rate', snap.training.completion_rate.toFixed(0) + '%'],
            ['Total Courses', snap.training.total_courses],
            ['Overdue', snap.training.overdue_count],
            ['Avg Quiz Score', snap.training.avg_quiz_score.toFixed(0) + '%'],
            ['Certificates', snap.training.certificates_issued]
        ]);
        html += sectionHeader('Risk Assessment');
        html += metricRow([
            ['High Risk Users', snap.risk.high_risk_users],
            ['Medium Risk Users', snap.risk.medium_risk_users],
            ['Low Risk Users', snap.risk.low_risk_users],
            ['Avg Risk Score', snap.risk.avg_risk_score.toFixed(1)]
        ]);
        html += sectionHeader('Compliance Posture');
        html += metricRow([
            ['Frameworks', snap.compliance.framework_count],
            ['Overall Score', snap.compliance.overall_score.toFixed(0) + '%'],
            ['Compliant', snap.compliance.compliant],
            ['Non-Compliant', snap.compliance.non_compliant]
        ]);
        html += sectionHeader('Remediation Progress');
        html += metricRow([
            ['Total Paths', snap.remediation.total_paths],
            ['Active', snap.remediation.active_paths],
            ['Completed', snap.remediation.completed_paths],
            ['Critical', snap.remediation.critical_count]
        ]);
        html += sectionHeader('Cyber Hygiene');
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

        return html;
    }

    // ── Render Snapshot (for Quick Preview — standalone) ──
    function renderSnapshot(container, snap) {
        container.html(renderSnapshotHTML(snap));
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

    // ── Generate AI Narrative ──
    $(document).on("click", ".generate-narrative-btn", function () {
        var id = $(this).data("id");
        var btn = $(this);
        btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Generating...');

        api.boardReports.generateNarrative(id)
            .success(function (narrative) {
                successFlash("Executive narrative generated successfully.");
                // Refresh the report view
                $(".view-report[data-id='" + id + "']").click();
            })
            .error(function (resp) {
                errorFlash(resp.responseJSON ? resp.responseJSON.message : "Failed to generate narrative.");
                btn.prop("disabled", false).html('<i class="fa fa-magic"></i> Generate AI Narrative');
            });
    });

    // ── Edit Narrative ──
    $(document).on("click", ".edit-narrative-btn", function () {
        var id = $(this).data("id");
        currentReportId = id;

        // Fetch current narrative
        api.boardReports.getFull(id)
            .success(function (payload) {
                if (payload.narrative) {
                    $("#editParagraph1").val(payload.narrative.paragraph1 || "");
                    $("#editParagraph2").val(payload.narrative.paragraph2 || "");
                    $("#editParagraph3").val(payload.narrative.paragraph3 || "");
                } else {
                    $("#editParagraph1").val("");
                    $("#editParagraph2").val("");
                    $("#editParagraph3").val("");
                }
                $("#narrativeModal").modal("show");
            })
            .error(function () {
                errorFlash("Failed to load narrative for editing.");
            });
    });

    $("#saveNarrativeBtn").on("click", function () {
        if (!currentReportId) return;
        var data = {
            paragraph1: $("#editParagraph1").val(),
            paragraph2: $("#editParagraph2").val(),
            paragraph3: $("#editParagraph3").val()
        };
        api.boardReports.editNarrative(currentReportId, data)
            .success(function () {
                successFlash("Narrative updated.");
                $("#narrativeModal").modal("hide");
                $(".view-report[data-id='" + currentReportId + "']").click();
            })
            .error(function () {
                errorFlash("Failed to save narrative.");
            });
    });

    // ── Approval Workflow ──
    $(document).on("click", ".workflow-btn, .workflow-modal-btn", function () {
        var id = $(this).data("id");
        var status = $(this).data("status");
        currentReportId = id;
        currentReportStatus = status;

        $("#approvalCurrentStatus").html(statusBadge(status));
        var select = $("#approvalNewStatus");
        select.empty();
        var transitions = statusTransitions[status] || [];
        $.each(transitions, function (i, s) {
            select.append('<option value="' + s + '">' + s.charAt(0).toUpperCase() + s.slice(1) + '</option>');
        });
        if (transitions.length === 0) {
            select.append('<option disabled>No transitions available</option>');
        }
        $("#approvalComment").val("");

        // Load approval history
        api.boardReports.getApprovals(id)
            .success(function (approvals) {
                if (approvals && approvals.length > 0) {
                    var hist = '<table class="table table-condensed" style="font-size:11px;">';
                    hist += '<thead><tr><th>Date</th><th>Change</th><th>By</th><th>Comment</th></tr></thead><tbody>';
                    $.each(approvals, function (i, a) {
                        hist += '<tr>';
                        hist += '<td>' + moment(a.created_date).format("MMM D HH:mm") + '</td>';
                        hist += '<td>' + escapeHtml(a.from_status) + ' → ' + escapeHtml(a.to_status) + '</td>';
                        hist += '<td>' + escapeHtml(a.username) + '</td>';
                        hist += '<td>' + escapeHtml(a.comment || '—') + '</td>';
                        hist += '</tr>';
                    });
                    hist += '</tbody></table>';
                    $("#approvalHistory").html(hist);
                } else {
                    $("#approvalHistory").html('<p class="text-muted">No history yet.</p>');
                }
            });

        $("#approvalModal").modal("show");
    });

    $("#submitTransitionBtn").on("click", function () {
        if (!currentReportId) return;
        var data = {
            status: $("#approvalNewStatus").val(),
            comment: $("#approvalComment").val()
        };
        api.boardReports.transition(currentReportId, data)
            .success(function () {
                successFlash("Report status updated to " + data.status + ".");
                $("#approvalModal").modal("hide");
                loadReports();
                // If detail modal is open, refresh it
                if ($("#reportDetailModal").is(":visible")) {
                    $(".view-report[data-id='" + currentReportId + "']").click();
                }
            })
            .error(function (resp) {
                errorFlash(resp.responseJSON ? resp.responseJSON.message : "Failed to transition status.");
            });
    });

    // ── New Report ──
    $("#newReportBtn").on("click", function () {
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

    // ── Quick Preview ──
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

    // ── Delete Report ──
    $(document).on("click", ".delete-report", function () {
        var id = $(this).data("id");
        if (!confirm("Delete this board report? This cannot be undone.")) return;
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

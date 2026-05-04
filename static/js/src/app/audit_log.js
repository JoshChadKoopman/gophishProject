var currentOffset = 0;
var pageSize = 25;
var totalEntries = 0;

var actionLabels = {
    'role_change': '<span class="label label-warning">Role Change</span>',
    'user_created': '<span class="label label-success">User Created</span>',
    'user_deleted': '<span class="label label-danger">User Deleted</span>',
    'user_locked': '<span class="label label-danger">User Locked</span>',
    'user_unlocked': '<span class="label label-info">User Unlocked</span>',
    'login_success': '<span class="label label-success">Login Success</span>',
    'login_failed': '<span class="label label-danger">Login Failed</span>',
    'mfa_enrolled': '<span class="label label-info">MFA Enrolled</span>',
    'mfa_verified': '<span class="label label-success">MFA Verified</span>',
    'mfa_failed': '<span class="label label-danger">MFA Failed</span>',
    'mfa_lockout': '<span class="label label-danger">MFA Lockout</span>',
    'training_assigned': '<span class="label label-info">Training Assigned</span>',
    'training_auto_assigned': '<span class="label label-info">Training Auto-Assigned</span>',
    'training_completed': '<span class="label label-success">Training Completed</span>',
    'certificate_issued': '<span class="label label-warning">Certificate Issued</span>'
};

function getFilters() {
    return {
        limit: pageSize,
        offset: currentOffset,
        action: $("#filterAction").val(),
        actor: $("#filterActor").val().trim(),
        date_from: $("#filterDateFrom").val().trim(),
        date_to: $("#filterDateTo").val().trim()
    };
}

function formatDetailsSummary(details) {
    if (!details) return '<span class="text-muted">—</span>';
    try {
        var obj = JSON.parse(details);
        var parts = [];
        for (var key in obj) {
            if (obj.hasOwnProperty(key)) {
                parts.push('<strong>' + escapeHtml(key) + '</strong>: ' + escapeHtml(String(obj[key])));
            }
        }
        if (parts.length === 0) return '<span class="text-muted">—</span>';
        return '<small class="text-muted"><i class="fa fa-info-circle"></i> Click to expand</small>';
    } catch (e) {
        if (!details) return '<span class="text-muted">—</span>';
        return '<small class="text-muted"><i class="fa fa-info-circle"></i> Click to expand</small>';
    }
}

function formatDetailsExpanded(details) {
    if (!details) return '<em class="text-muted">No details</em>';
    try {
        var obj = JSON.parse(details);
        var rows = [];
        for (var key in obj) {
            if (obj.hasOwnProperty(key)) {
                rows.push('<tr><td style="padding:2px 8px; font-weight:600; white-space:nowrap;">' +
                    escapeHtml(key) + '</td><td style="padding:2px 8px;">' +
                    escapeHtml(String(obj[key])) + '</td></tr>');
            }
        }
        if (rows.length === 0) return '<em class="text-muted">No details</em>';
        return '<table style="font-size:0.9em;">' + rows.join('') + '</table>';
    } catch (e) {
        return '<pre style="font-size:0.85em; margin:0;">' + escapeHtml(details) + '</pre>';
    }
}

function renderAuditLogs(data) {
    var logs = data.logs || [];
    totalEntries = data.total || 0;

    if (totalEntries === 0) {
        $("#auditLoading").hide();
        $("#auditContent").hide();
        $("#auditEmpty").show();
        return;
    }

    $("#auditEmpty").hide();
    var tbody = $("#auditTableBody");
    tbody.empty();

    logs.forEach(function (entry) {
        var actionHtml = actionLabels[entry.action] || '<span class="label label-default">' + escapeHtml(entry.action) + '</span>';
        var target = '';
        if (entry.target_username) {
            target = escapeHtml(entry.target_username);
            if (entry.target_type) {
                target += ' <small class="text-muted">(' + escapeHtml(entry.target_type) + ')</small>';
            }
        }
        var hasDetails = entry.details && entry.details !== '{}' && entry.details !== 'null';
        var detailsAttr = hasDetails ? ' data-details="' + escapeHtml(entry.details) + '" style="cursor:pointer;"' : '';
        var detailsCell = hasDetails
            ? '<td class="detail-cell"' + detailsAttr + '>' + formatDetailsSummary(entry.details) + '</td>'
            : '<td><span class="text-muted">—</span></td>';

        var row = '<tr>' +
            '<td style="white-space:nowrap;">' + moment(entry.timestamp).format('MMM D, YYYY h:mm:ss A') + '</td>' +
            '<td>' + escapeHtml(entry.actor_username || '') + '</td>' +
            '<td>' + actionHtml + '</td>' +
            '<td>' + target + '</td>' +
            detailsCell +
            '<td><small>' + escapeHtml(entry.ip_address || '') + '</small></td>' +
            '</tr>';
        tbody.append(row);
    });

    // Pagination info
    var totalPages = Math.ceil(totalEntries / pageSize);
    var currentPage = Math.floor(currentOffset / pageSize) + 1;
    var start = currentOffset + 1;
    var end = Math.min(currentOffset + pageSize, totalEntries);
    $("#paginationInfo").text('Showing ' + start + '–' + end + ' of ' + totalEntries);
    $("#prevPage").prop("disabled", currentOffset === 0);
    $("#nextPage").prop("disabled", currentOffset + pageSize >= totalEntries);
    $("#pageJump").val(currentPage);
    $("#pageTotal").text(' of ' + totalPages);

    $("#auditLoading").hide();
    $("#auditContent").show();
}

function loadAuditLogs() {
    $("#auditLoading").show();
    $("#auditContent").hide();
    $("#auditEmpty").hide();

    api.auditLog.get(getFilters())
        .success(function (data) {
            renderAuditLogs(data);
        })
        .error(function () {
            $("#auditLoading").hide();
            errorFlash("Failed to load audit log entries.");
        });
}

// Export the currently visible page to CSV
function exportToCsv() {
    var rows = [['Timestamp', 'Actor', 'Action', 'Target', 'Details', 'IP Address']];
    $("#auditTableBody tr").each(function () {
        var cells = $(this).find("td");
        var detailsRaw = cells.eq(4).attr("data-details") || cells.eq(4).text().trim();
        rows.push([
            cells.eq(0).text().trim(),
            cells.eq(1).text().trim(),
            cells.eq(2).text().trim(),
            cells.eq(3).text().trim(),
            detailsRaw,
            cells.eq(5).text().trim()
        ]);
    });
    var csv = rows.map(function(row) {
        return row.map(function(val) {
            return '"' + String(val).replace(/"/g, '""') + '"';
        }).join(',');
    }).join('\n');
    var blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' });
    var a = document.createElement('a');
    a.href = URL.createObjectURL(blob);
    a.download = 'audit-log-' + moment().format('YYYY-MM-DD') + '.csv';
    a.click();
}

$(document).ready(function () {
    // Date pickers
    if ($.fn.datetimepicker) {
        $("#filterDateFrom").datetimepicker({
            widgetPositioning: { vertical: "bottom" },
            format: "YYYY-MM-DD",
            useCurrent: false
        });
        $("#filterDateTo").datetimepicker({
            widgetPositioning: { vertical: "bottom" },
            format: "YYYY-MM-DD",
            useCurrent: false
        });
    }

    // Date range presets
    $(".date-preset").on("click", function () {
        var preset = $(this).data("preset");
        var today = moment().format("YYYY-MM-DD");
        if (preset === "today") {
            $("#filterDateFrom").val(today);
            $("#filterDateTo").val(today);
        } else if (preset === "7d") {
            $("#filterDateFrom").val(moment().subtract(6, "days").format("YYYY-MM-DD"));
            $("#filterDateTo").val(today);
        } else if (preset === "30d") {
            $("#filterDateFrom").val(moment().subtract(29, "days").format("YYYY-MM-DD"));
            $("#filterDateTo").val(today);
        } else if (preset === "month") {
            $("#filterDateFrom").val(moment().startOf("month").format("YYYY-MM-DD"));
            $("#filterDateTo").val(today);
        } else if (preset === "clear") {
            $("#filterDateFrom").val("");
            $("#filterDateTo").val("");
        }
        currentOffset = 0;
        loadAuditLogs();
    });

    // Export CSV
    $("#exportCsv").on("click", exportToCsv);

    // Filter apply
    $("#applyFilters").on("click", function () {
        currentOffset = 0;
        loadAuditLogs();
    });

    // Pagination
    $("#prevPage").on("click", function () {
        if (currentOffset > 0) {
            currentOffset = Math.max(0, currentOffset - pageSize);
            loadAuditLogs();
        }
    });
    $("#nextPage").on("click", function () {
        if (currentOffset + pageSize < totalEntries) {
            currentOffset += pageSize;
            loadAuditLogs();
        }
    });

    // Page jump
    $("#pageJump").on("change keypress", function (e) {
        if (e.type === "keypress" && e.which !== 13) return;
        var page = parseInt($(this).val(), 10);
        var totalPages = Math.ceil(totalEntries / pageSize);
        if (page >= 1 && page <= totalPages) {
            currentOffset = (page - 1) * pageSize;
            loadAuditLogs();
        }
    });

    // Expandable detail rows — click a detail cell to show formatted JSON
    $("#auditTableBody").on("click", ".detail-cell", function () {
        var details = $(this).attr("data-details");
        if (!details) return;
        var expanded = formatDetailsExpanded(details);
        var $tr = $(this).closest("tr");
        var $next = $tr.next(".detail-expanded-row");
        if ($next.length) {
            $next.remove();
        } else {
            $tr.after('<tr class="detail-expanded-row"><td colspan="6" style="background:#f9f9f9; padding:10px 20px;">' +
                expanded + '</td></tr>');
        }
    });

    // Enter key triggers filter
    $("#filterActor, #filterDateFrom, #filterDateTo").on("keypress", function (e) {
        if (e.which === 13) {
            currentOffset = 0;
            loadAuditLogs();
        }
    });

    loadAuditLogs();
});

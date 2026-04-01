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

function formatDetails(details) {
    if (!details) return '';
    try {
        var obj = JSON.parse(details);
        var parts = [];
        for (var key in obj) {
            if (obj.hasOwnProperty(key)) {
                parts.push(escapeHtml(key) + ': ' + escapeHtml(obj[key]));
            }
        }
        return '<small>' + parts.join(', ') + '</small>';
    } catch (e) {
        return escapeHtml(details);
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
        var row = '<tr>' +
            '<td style="white-space:nowrap;">' + moment(entry.timestamp).format('MMM D, YYYY h:mm:ss A') + '</td>' +
            '<td>' + escapeHtml(entry.actor_username || '') + '</td>' +
            '<td>' + actionHtml + '</td>' +
            '<td>' + target + '</td>' +
            '<td style="max-width:250px; overflow:hidden; text-overflow:ellipsis;">' + formatDetails(entry.details) + '</td>' +
            '<td><small>' + escapeHtml(entry.ip_address || '') + '</small></td>' +
            '</tr>';
        tbody.append(row);
    });

    // Pagination info
    var start = currentOffset + 1;
    var end = Math.min(currentOffset + pageSize, totalEntries);
    $("#paginationInfo").text('Showing ' + start + '-' + end + ' of ' + totalEntries);
    $("#prevPage").prop("disabled", currentOffset === 0);
    $("#nextPage").prop("disabled", currentOffset + pageSize >= totalEntries);

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

    // Enter key triggers filter
    $("#filterActor, #filterDateFrom, #filterDateTo").on("keypress", function (e) {
        if (e.which === 13) {
            currentOffset = 0;
            loadAuditLogs();
        }
    });

    loadAuditLogs();
});

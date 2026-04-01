$(document).ready(function () {
    var editingAlertId = null;
    var currentFilter = "all";

    function loadAlerts() {
        api.threatAlerts.get()
            .success(function (alerts) {
                renderAlerts(alerts);
            })
            .error(function () {
                errorFlash("Failed to load threat alerts.");
            });
    }

    function renderAlerts(alerts) {
        var container = $("#alertsList");
        container.empty();
        var filtered = alerts;
        if (currentFilter !== "all") {
            filtered = alerts.filter(function (a) { return a.severity === currentFilter; });
        }
        if (!filtered || filtered.length === 0) {
            $("#alertsEmpty").show();
            return;
        }
        $("#alertsEmpty").hide();
        $.each(filtered, function (i, alert) {
            var severityClass = {
                'info': 'panel-info',
                'warning': 'panel-warning',
                'critical': 'panel-danger'
            }[alert.severity] || 'panel-default';
            var severityIcon = {
                'info': 'fa-info-circle',
                'warning': 'fa-exclamation-triangle',
                'critical': 'fa-bolt'
            }[alert.severity] || 'fa-bell';
            var readBadge = '';
            if (alert.is_read === false && alert.published) {
                readBadge = ' <span class="label label-primary">New</span>';
            }
            var date = alert.published_date && alert.published
                ? moment(alert.published_date).format("MMMM Do YYYY")
                : moment(alert.created_date).format("MMMM Do YYYY");
            var statusBadge = alert.published
                ? '<span class="label label-success">Published</span>'
                : '<span class="label label-default">Draft</span>';
            var adminActions = '';
            if (permissions && permissions.modify_objects) {
                adminActions =
                    '<div class="pull-right">' +
                    statusBadge + ' ' +
                    '<span class="label label-default">' + (alert.read_count || 0) + ' reads</span> ' +
                    '<button class="btn btn-xs btn-default edit-alert" data-id="' + alert.id + '"><i class="fa fa-pencil"></i></button> ' +
                    '<button class="btn btn-xs btn-danger delete-alert" data-id="' + alert.id + '"><i class="fa fa-trash"></i></button>' +
                    '</div>';
            }
            container.append(
                '<div class="panel ' + severityClass + ' alert-card" data-id="' + alert.id + '">' +
                '<div class="panel-heading">' +
                '<i class="fa ' + severityIcon + '"></i> ' +
                '<strong>' + escapeHtml(alert.title) + '</strong>' + readBadge +
                adminActions +
                '<span class="text-muted pull-right" style="margin-right:10px;">' + date + '</span>' +
                '</div>' +
                '<div class="panel-body" style="cursor:pointer;" onclick="viewAlert(' + alert.id + ')">' +
                '<p>' + escapeHtml(alert.body).substring(0, 200) + (alert.body.length > 200 ? '...' : '') + '</p>' +
                '</div>' +
                '</div>'
            );
        });
    }

    // View alert detail
    window.viewAlert = function (id) {
        api.threatAlerts.getOne(id)
            .success(function (alert) {
                $("#detailTitle").text(alert.title);
                var severityBadge = '<span class="label label-' +
                    ({ 'info': 'info', 'warning': 'warning', 'critical': 'danger' }[alert.severity] || 'default') +
                    '">' + alert.severity + '</span>';
                var date = alert.published_date
                    ? moment(alert.published_date).format("MMMM Do YYYY, h:mm a")
                    : moment(alert.created_date).format("MMMM Do YYYY, h:mm a");
                $("#detailMeta").html(severityBadge + ' <span class="text-muted">' + date + '</span>');
                // Render body as paragraphs (simple line-break to <p> conversion)
                var bodyHtml = escapeHtml(alert.body).replace(/\n\n/g, '</p><p>').replace(/\n/g, '<br>');
                $("#detailBody").html('<p>' + bodyHtml + '</p>');
                $("#alertDetailModal").modal("show");
                // Refresh list to update read status
                loadAlerts();
            });
    };

    // Severity filter
    $("#severityFilter button").click(function () {
        currentFilter = $(this).data("severity");
        $("#severityFilter button").removeClass("active");
        $(this).addClass("active");
        loadAlerts();
    });

    // Create alert button
    $("#createAlertBtn").click(function () {
        editingAlertId = null;
        $("#alertModalTitle").text("New Threat Alert");
        $("#alertTitle").val("");
        $("#alertBody").val("");
        $("#alertSeverity").val("info");
        $("#alertPublished").prop("checked", false);
        $("#alertTargetRoles").val("");
        $("#alertTargetDepts").val("");
        $("#alertModal").modal("show");
    });

    // Edit alert
    $(document).on("click", ".edit-alert", function (e) {
        e.stopPropagation();
        var id = $(this).data("id");
        api.threatAlerts.getOne(id)
            .success(function (alert) {
                editingAlertId = alert.id;
                $("#alertModalTitle").text("Edit Threat Alert");
                $("#alertTitle").val(alert.title);
                $("#alertBody").val(alert.body);
                $("#alertSeverity").val(alert.severity);
                $("#alertPublished").prop("checked", alert.published);
                $("#alertTargetRoles").val(alert.target_roles || "");
                $("#alertTargetDepts").val(alert.target_departments || "");
                $("#alertModal").modal("show");
            });
    });

    // Delete alert
    $(document).on("click", ".delete-alert", function (e) {
        e.stopPropagation();
        var id = $(this).data("id");
        if (!confirm("Are you sure you want to delete this alert?")) return;
        api.threatAlerts.delete(id)
            .success(function () {
                successFlash("Alert deleted.");
                loadAlerts();
            })
            .error(function (data) {
                errorFlash(data.responseJSON.message);
            });
    });

    // Save alert
    $("#saveAlertBtn").click(function () {
        var data = {
            title: $("#alertTitle").val(),
            body: $("#alertBody").val(),
            severity: $("#alertSeverity").val(),
            published: $("#alertPublished").is(":checked"),
            target_roles: $("#alertTargetRoles").val(),
            target_departments: $("#alertTargetDepts").val()
        };
        if (!data.title || !data.body) {
            modalError("Title and body are required.");
            return;
        }
        var promise;
        if (editingAlertId) {
            promise = api.threatAlerts.update(editingAlertId, data);
        } else {
            promise = api.threatAlerts.create(data);
        }
        promise
            .success(function () {
                $("#alertModal").modal("hide");
                successFlash(editingAlertId ? "Alert updated." : "Alert created.");
                loadAlerts();
            })
            .error(function (data) {
                modalError(data.responseJSON.message);
            });
    });

    loadAlerts();
});

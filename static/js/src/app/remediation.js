$(document).ready(function () {
    var currentFilter = "all";
    var currentPathId = null;
    var isAdmin = permissions && permissions.modify_objects;

    // Load initial data
    if (isAdmin) {
        loadSummary();
        loadPaths();
    } else {
        $("#remediationSummary").hide();
        $("#pathsTable").hide();
        $("#statusFilter").hide();
        $("#myPathsSection").show();
        loadMyPaths();
    }

    // ---- Summary ----
    function loadSummary() {
        api.remediation.summary()
            .success(function (s) {
                $("#totalPaths").text(s.total_paths);
                $("#activePaths").text(s.active_paths);
                $("#completedPaths").text(s.completed_paths);
                $("#criticalCount").text(s.critical_count);
                $("#highCount").text(s.high_count);
                $("#avgCompletion").text(Math.round(s.avg_completion_pct) + "%");
            });
    }

    // ---- All Paths (admin) ----
    function loadPaths() {
        api.remediation.get()
            .success(function (paths) {
                renderPaths(paths);
            })
            .error(function () {
                errorFlash("Failed to load remediation paths.");
            });
    }

    function renderPaths(paths) {
        var body = $("#pathsBody");
        body.empty();

        var filtered = paths;
        if (currentFilter !== "all") {
            filtered = paths.filter(function (p) { return p.status === currentFilter; });
        }

        if (!filtered || filtered.length === 0) {
            $("#pathsEmpty").show();
            $("#pathsTable").hide();
            return;
        }
        $("#pathsEmpty").hide();
        $("#pathsTable").show();

        $.each(filtered, function (i, p) {
            var pct = p.total_courses > 0 ? Math.round((p.completed_count / p.total_courses) * 100) : 0;
            var riskBadge = riskLabel(p.risk_level);
            var statusBadge = statusLabel(p.status);
            var dueDate = p.due_date ? moment(p.due_date).format("MMM D, YYYY") : "—";
            var progressBar =
                '<div class="progress" style="margin:0;height:18px;">' +
                '<div class="progress-bar ' + progressBarClass(pct) + '" style="width:' + pct + '%; min-width:2em;">' + pct + '%</div></div>';

            body.append(
                '<tr data-id="' + p.id + '" class="path-row" style="cursor:pointer;">' +
                '<td>' + escapeHtml(p.user_name || p.user_email || 'User #' + p.user_id) + '</td>' +
                '<td>' + escapeHtml(p.name) + '</td>' +
                '<td>' + riskBadge + '</td>' +
                '<td>' + statusBadge + '</td>' +
                '<td style="width:160px;">' + progressBar + '</td>' +
                '<td>' + dueDate + '</td>' +
                '<td>' +
                '<button class="btn btn-xs btn-info view-path" data-id="' + p.id + '"><i class="fa fa-eye"></i></button> ' +
                (p.status === 'active' ? '<button class="btn btn-xs btn-danger cancel-path" data-id="' + p.id + '"><i class="fa fa-ban"></i></button>' : '') +
                '</td>' +
                '</tr>'
            );
        });
    }

    // ---- My Paths (user) ----
    function loadMyPaths() {
        api.remediation.myPaths()
            .success(function (paths) {
                renderMyPaths(paths);
            })
            .error(function () {
                errorFlash("Failed to load your remediation paths.");
            });
    }

    function renderMyPaths(paths) {
        var container = $("#myPathsList");
        container.empty();

        if (!paths || paths.length === 0) {
            container.html('<div style="text-align:center; padding:30px; color:#999;">' +
                '<i class="fa fa-check-circle fa-3x" style="color:#5cb85c;"></i>' +
                '<h3>No remediation paths assigned.</h3>' +
                '<p>Great job! You have no outstanding remediation training.</p></div>');
            return;
        }

        $.each(paths, function (i, p) {
            var pct = p.total_courses > 0 ? Math.round((p.completed_count / p.total_courses) * 100) : 0;
            var stepsHtml = '';
            if (p.steps && p.steps.length > 0) {
                $.each(p.steps, function (j, s) {
                    var icon = s.status === 'completed' ? '<i class="fa fa-check-circle text-success"></i>' :
                        s.status === 'skipped' ? '<i class="fa fa-minus-circle text-muted"></i>' :
                            '<i class="fa fa-circle-o text-warning"></i>';
                    stepsHtml += '<li class="list-group-item">' + icon + ' ' +
                        escapeHtml(s.course_name || 'Course #' + s.presentation_id) +
                        (s.status === 'pending' ? ' <a href="/training" class="btn btn-xs btn-primary pull-right"><i class="fa fa-play"></i> Start</a>' : '') +
                        '</li>';
                });
            }

            container.append(
                '<div class="panel ' + (p.status === 'completed' ? 'panel-success' : p.status === 'active' ? 'panel-info' : 'panel-default') + '">' +
                '<div class="panel-heading">' +
                '<strong>' + escapeHtml(p.name) + '</strong> ' +
                riskLabel(p.risk_level) + ' ' + statusLabel(p.status) +
                '<span class="pull-right text-muted">Due: ' + (p.due_date ? moment(p.due_date).format("MMM D, YYYY") : "—") + '</span>' +
                '</div>' +
                '<div class="panel-body">' +
                '<div class="progress" style="height:20px;">' +
                '<div class="progress-bar ' + progressBarClass(pct) + ' progress-bar-striped" style="width:' + pct + '%; min-width:2em;">' + pct + '%</div>' +
                '</div>' +
                '<ul class="list-group" style="margin-top:10px;">' + stepsHtml + '</ul>' +
                '</div></div>'
            );
        });
    }

    // ---- Status Filter ----
    $("#statusFilter button").click(function () {
        $("#statusFilter button").removeClass("active");
        $(this).addClass("active");
        currentFilter = $(this).data("status");
        loadPaths();
    });

    // ---- View Path Detail ----
    $(document).on("click", ".view-path, .path-row", function (e) {
        if ($(e.target).hasClass("cancel-path") || $(e.target).parent().hasClass("cancel-path")) return;
        var id = $(this).data("id") || $(this).closest("tr").data("id");
        viewPathDetail(id);
    });

    function viewPathDetail(id) {
        currentPathId = id;
        api.remediation.getOne(id)
            .success(function (p) {
                $("#detailPathName").text(p.name);
                $("#detailUser").text(p.user_name || p.user_email || 'User #' + p.user_id);
                $("#detailRisk").html(riskLabel(p.risk_level));
                $("#detailStatus").html(statusLabel(p.status));
                $("#detailFailCount").text(p.fail_count);
                $("#detailDueDate").text(p.due_date ? moment(p.due_date).format("MMM D, YYYY") : "—");
                var pct = p.total_courses > 0 ? Math.round((p.completed_count / p.total_courses) * 100) : 0;
                $("#detailCompletion").text(p.completed_count + " / " + p.total_courses);
                $("#detailProgressBar").css("width", pct + "%").text(pct + "%").attr("class", "progress-bar progress-bar-striped " + progressBarClass(pct));

                var stepsBody = $("#detailSteps");
                stepsBody.empty();
                if (p.steps) {
                    $.each(p.steps, function (i, s) {
                        var statusIcon = s.status === 'completed' ? '<span class="label label-success">Completed</span>' :
                            s.status === 'skipped' ? '<span class="label label-default">Skipped</span>' :
                                '<span class="label label-warning">Pending</span>';
                        var completedDate = s.completed_date && s.status === 'completed' ? moment(s.completed_date).format("MMM D, YYYY") : "—";
                        var actionBtn = s.status === 'pending' && isAdmin ?
                            '<button class="btn btn-xs btn-success complete-step" data-path="' + p.id + '" data-pres="' + s.presentation_id + '"><i class="fa fa-check"></i> Mark Complete</button>' : '';
                        stepsBody.append(
                            '<tr><td>' + s.sort_order + '</td>' +
                            '<td>' + escapeHtml(s.course_name || 'Course #' + s.presentation_id) + '</td>' +
                            '<td>' + statusIcon + '</td>' +
                            '<td>' + completedDate + '</td>' +
                            '<td>' + actionBtn + '</td></tr>'
                        );
                    });
                }

                if (p.status === 'active' && isAdmin) {
                    $("#cancelPathBtn").show();
                } else {
                    $("#cancelPathBtn").hide();
                }

                $("#pathDetailModal").modal("show");
            })
            .error(function () {
                errorFlash("Failed to load path detail.");
            });
    }

    // ---- Complete Step ----
    $(document).on("click", ".complete-step", function () {
        var pathId = $(this).data("path");
        var presId = $(this).data("pres");
        api.remediation.completeStep(pathId, { presentation_id: presId })
            .success(function () {
                viewPathDetail(pathId);
                loadSummary();
            })
            .error(function () {
                errorFlash("Failed to complete step.");
            });
    });

    // ---- Cancel Path ----
    $(document).on("click", ".cancel-path", function (e) {
        e.stopPropagation();
        var id = $(this).data("id");
        if (!confirm("Cancel this remediation path? Pending steps will be marked as skipped.")) return;
        api.remediation.cancel(id)
            .success(function () {
                successFlash("Remediation path cancelled.");
                loadPaths();
                loadSummary();
            })
            .error(function () {
                errorFlash("Failed to cancel path.");
            });
    });

    $("#cancelPathBtn").click(function () {
        if (!confirm("Cancel this remediation path?")) return;
        api.remediation.cancel(currentPathId)
            .success(function () {
                $("#pathDetailModal").modal("hide");
                successFlash("Path cancelled.");
                loadPaths();
                loadSummary();
            });
    });

    // ---- Create Path ----
    $("#createPathBtn").click(function () {
        // Load courses for selection
        api.training.get()
            .success(function (courses) {
                var container = $("#courseCheckboxes");
                container.empty();
                if (!courses || courses.length === 0) {
                    container.html('<em>No training courses available.</em>');
                    return;
                }
                $.each(courses, function (i, c) {
                    container.append(
                        '<div class="checkbox"><label>' +
                        '<input type="checkbox" class="course-cb" value="' + c.id + '"> ' +
                        escapeHtml(c.name) +
                        '</label></div>'
                    );
                });
            });
        // Set default due date 14 days from now
        var defaultDue = moment().add(14, 'days').format('YYYY-MM-DD');
        $("#pathDueDate").val(defaultDue);
        $("#createPathModal").modal("show");
    });

    $("#savePathBtn").click(function () {
        var courseIds = [];
        $(".course-cb:checked").each(function () {
            courseIds.push(parseInt($(this).val()));
        });
        if (courseIds.length === 0) {
            modalError("Select at least one course.");
            return;
        }
        var name = $("#pathName").val().trim();
        if (!name) {
            modalError("Path name is required.");
            return;
        }
        var dueDate = $("#pathDueDate").val();
        var data = {
            name: name,
            user_email: $("#pathUserEmail").val().trim(),
            user_id: parseInt($("#pathUserId").val()) || 0,
            fail_count: parseInt($("#pathFailCount").val()) || 0,
            due_date: dueDate ? dueDate + "T23:59:59Z" : "",
            course_ids: courseIds
        };
        api.remediation.create(data)
            .success(function () {
                $("#createPathModal").modal("hide");
                successFlash("Remediation path created.");
                loadPaths();
                loadSummary();
            })
            .error(function (resp) {
                var msg = resp.responseJSON ? resp.responseJSON.message : "Failed to create path.";
                modalError(msg);
            });
    });

    // ---- Auto-Evaluate ----
    $("#evaluateBtn").click(function () {
        var btn = $(this);
        btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Evaluating...');
        api.remediation.evaluate()
            .success(function (result) {
                btn.prop("disabled", false).html('<i class="fa fa-refresh"></i> Auto-Evaluate');
                successFlash("Evaluation complete. " + (result.paths_created || 0) + " new paths created.");
                loadPaths();
                loadSummary();
            })
            .error(function () {
                btn.prop("disabled", false).html('<i class="fa fa-refresh"></i> Auto-Evaluate');
                errorFlash("Evaluation failed.");
            });
    });

    // ---- Helpers ----
    function riskLabel(level) {
        var cls = { critical: 'danger', high: 'warning', medium: 'info', low: 'success' }[level] || 'default';
        return '<span class="label label-' + cls + '">' + (level || 'N/A').toUpperCase() + '</span>';
    }

    function statusLabel(status) {
        var cls = { active: 'info', completed: 'success', expired: 'warning', cancelled: 'default' }[status] || 'default';
        return '<span class="label label-' + cls + '">' + capitalize(status || 'unknown') + '</span>';
    }

    function progressBarClass(pct) {
        if (pct >= 100) return 'progress-bar-success';
        if (pct >= 50) return 'progress-bar-info';
        if (pct >= 25) return 'progress-bar-warning';
        return 'progress-bar-danger';
    }

    function capitalize(s) {
        return s.charAt(0).toUpperCase() + s.slice(1);
    }
});

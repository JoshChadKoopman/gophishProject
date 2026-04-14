$(document).ready(function () {
    var isAdmin = permissions && permissions.modify_system;
    var editingDeviceId = null;

    // Load initial data
    loadDevices();

    // When tabs change, lazy-load data
    $('a[data-toggle="tab"]').on("shown.bs.tab", function (e) {
        var target = $(e.target).attr("href");
        if (target === "#checklistTab") loadChecklist();
        if (target === "#techStackTab") loadTechStack();
        if (target === "#adminTab") loadAdminDashboard();
    });

    // ===================== MY DEVICES TAB =====================
    function loadDevices() {
        api.hygiene.devices.get()
            .success(function (devices) {
                renderDevices(devices);
            })
            .error(function () {
                errorFlash("Failed to load devices.");
            });
    }

    function renderDevices(devices) {
        var container = $("#devicesList");
        container.empty();

        if (!devices || devices.length === 0) {
            $("#devicesEmpty").show();
            return;
        }
        $("#devicesEmpty").hide();

        $.each(devices, function (i, d) {
            var scoreClass = d.hygiene_score >= 80 ? 'success' : d.hygiene_score >= 50 ? 'warning' : 'danger';
            var typeIcon = {
                laptop: 'fa-laptop', desktop: 'fa-desktop',
                mobile: 'fa-mobile', tablet: 'fa-tablet', other: 'fa-cube'
            }[d.device_type] || 'fa-cube';

            var checksHtml = renderDeviceChecks(d);

            container.append(
                '<div class="panel panel-default device-card" data-id="' + d.id + '">' +
                '<div class="panel-heading">' +
                '<i class="fa ' + typeIcon + '"></i> ' +
                '<strong>' + escapeHtml(d.name) + '</strong>' +
                '<span class="label label-' + scoreClass + ' pull-right">' + d.hygiene_score + '%</span>' +
                '<span class="text-muted pull-right" style="margin-right:10px;">' + escapeHtml(d.os || '') + ' · ' + escapeHtml(d.device_type) + '</span>' +
                '</div>' +
                '<div class="panel-body">' +
                '<div class="progress" style="height:10px; margin-bottom:10px;">' +
                '<div class="progress-bar progress-bar-' + scoreClass + '" style="width:' + d.hygiene_score + '%;"></div>' +
                '</div>' +
                checksHtml +
                '<div style="margin-top:10px;">' +
                '<button class="btn btn-xs btn-default edit-device" data-id="' + d.id + '"><i class="fa fa-pencil"></i> Edit</button> ' +
                '<button class="btn btn-xs btn-danger delete-device" data-id="' + d.id + '"><i class="fa fa-trash"></i> Delete</button>' +
                '</div>' +
                '</div></div>'
            );
        });
    }

    function renderDeviceChecks(device) {
        var checkTypes = [
            { type: 'os_updated', label: 'OS Updated', icon: 'fa-download' },
            { type: 'antivirus_active', label: 'Antivirus', icon: 'fa-shield' },
            { type: 'disk_encrypted', label: 'Disk Encrypted', icon: 'fa-lock' },
            { type: 'screen_lock', label: 'Screen Lock', icon: 'fa-clock-o' },
            { type: 'password_manager', label: 'Password Mgr', icon: 'fa-key' },
            { type: 'vpn_enabled', label: 'VPN', icon: 'fa-globe' },
            { type: 'mfa_enabled', label: 'MFA', icon: 'fa-id-card-o' }
        ];

        var checksMap = {};
        if (device.checks) {
            $.each(device.checks, function (i, c) {
                checksMap[c.check_type] = c;
            });
        }

        var html = '<div class="row">';
        $.each(checkTypes, function (i, ct) {
            var check = checksMap[ct.type];
            var status = check ? check.status : 'unknown';
            var statusIcon = status === 'pass' ? '<i class="fa fa-check-circle text-success"></i>' :
                status === 'fail' ? '<i class="fa fa-times-circle text-danger"></i>' :
                    '<i class="fa fa-question-circle text-muted"></i>';
            var btnClass = status === 'pass' ? 'btn-success' : status === 'fail' ? 'btn-danger' : 'btn-default';

            html += '<div class="col-xs-6 col-md-3" style="margin-bottom:6px;">' +
                '<button class="btn btn-xs ' + btnClass + ' update-check" style="width:100%; text-align:left;" ' +
                'data-device="' + device.id + '" data-check="' + ct.type + '" data-label="' + ct.label + '">' +
                statusIcon + ' <i class="fa ' + ct.icon + '"></i> ' + ct.label +
                '</button></div>';
        });
        html += '</div>';
        return html;
    }

    // ---- Add Device ----
    $("#addDeviceBtn").click(function () {
        editingDeviceId = null;
        $("#deviceModalTitle").text("Register Device");
        $("#deviceName").val('');
        $("#deviceType").val('laptop');
        $("#deviceOS").val('');
        $("#deviceId").val('');
        $("#deviceModal").modal("show");
    });

    // ---- Edit Device ----
    $(document).on("click", ".edit-device", function () {
        var id = $(this).data("id");
        editingDeviceId = id;
        api.hygiene.devices.getOne(id)
            .success(function (d) {
                $("#deviceModalTitle").text("Edit Device");
                $("#deviceId").val(d.id);
                $("#deviceName").val(d.name);
                $("#deviceType").val(d.device_type);
                $("#deviceOS").val(d.os);
                $("#deviceModal").modal("show");
            })
            .error(function () {
                errorFlash("Failed to load device.");
            });
    });

    // ---- Save Device ----
    $("#saveDeviceBtn").click(function () {
        var name = $("#deviceName").val().trim();
        if (!name) {
            modalError("Device name is required.");
            return;
        }
        var data = {
            name: name,
            device_type: $("#deviceType").val(),
            os: $("#deviceOS").val()
        };

        if (editingDeviceId) {
            api.hygiene.devices.update(editingDeviceId, data)
                .success(function () {
                    $("#deviceModal").modal("hide");
                    successFlash("Device updated.");
                    loadDevices();
                })
                .error(function () { modalError("Failed to update device."); });
        } else {
            api.hygiene.devices.create(data)
                .success(function () {
                    $("#deviceModal").modal("hide");
                    successFlash("Device registered.");
                    loadDevices();
                })
                .error(function () { modalError("Failed to register device."); });
        }
    });

    // ---- Delete Device ----
    $(document).on("click", ".delete-device", function () {
        var id = $(this).data("id");
        if (!confirm("Delete this device? All hygiene check data will be lost.")) return;
        api.hygiene.devices.remove(id)
            .success(function () {
                successFlash("Device deleted.");
                loadDevices();
            })
            .error(function () {
                errorFlash("Failed to delete device.");
            });
    });

    // ---- Update Hygiene Check ----
    $(document).on("click", ".update-check", function () {
        var deviceId = $(this).data("device");
        var checkType = $(this).data("check");
        var label = $(this).data("label");
        $("#checkDeviceId").val(deviceId);
        $("#checkType").val(checkType);
        $("#checkLabel").text(label);
        $("#checkStatus").val("pass");
        $("#checkNote").val("");
        $("#checkModal").modal("show");
    });

    $("#saveCheckBtn").click(function () {
        var deviceId = parseInt($("#checkDeviceId").val());
        var data = {
            check_type: $("#checkType").val(),
            status: $("#checkStatus").val(),
            note: $("#checkNote").val()
        };
        api.hygiene.devices.upsertCheck(deviceId, data)
            .success(function () {
                $("#checkModal").modal("hide");
                successFlash("Check updated.");
                loadDevices();
            })
            .error(function () {
                modalError("Failed to save check.");
            });
    });

    // ===================== PERSONALIZED CHECKLIST TAB =====================
    function loadChecklist() {
        api.hygiene.personalizedChecks()
            .success(function (checks) {
                renderChecklist(checks);
            })
            .error(function () {
                $("#checklistContent").html('<p class="text-danger">Failed to load checklist.</p>');
            });
    }

    function renderChecklist(checks) {
        var container = $("#checklistContent");
        container.empty();

        if (!checks || checks.length === 0) {
            container.html('<p class="text-muted">No checks available.</p>');
            return;
        }

        var html = '<div class="list-group">';
        $.each(checks, function (i, c) {
            var icon = c.relevant ? 'fa-check-square-o text-primary' : 'fa-square-o text-muted';
            html += '<div class="list-group-item">' +
                '<div class="row">' +
                '<div class="col-md-1 text-center"><i class="fa ' + icon + ' fa-2x"></i></div>' +
                '<div class="col-md-4"><strong>' + escapeHtml(c.label) + '</strong><br>' +
                '<small class="text-muted">' + escapeHtml(c.check_type) + '</small></div>' +
                '<div class="col-md-4">' + escapeHtml(c.description) + '</div>' +
                '<div class="col-md-3"><em style="color:#888;">' + escapeHtml(c.reason) + '</em></div>' +
                '</div></div>';
        });
        html += '</div>';
        container.html(html);
    }

    // ===================== TECH STACK PROFILE TAB =====================
    function loadTechStack() {
        api.hygiene.techStack.get()
            .success(function (result) {
                if (result.has_profile && result.profile) {
                    var p = result.profile;
                    $("#tsOS").val(p.primary_os || '');
                    $("#tsBrowser").val(p.browser || '');
                    $("#tsEmail").val(p.email_client || '');
                    $("#tsRemoteAccess").val(p.remote_access || '');
                    $("#tsMobile").val(p.mobile_device || '');
                    $("#tsCloudApps").val(p.cloud_apps || '');
                    $("#tsDevTools").val(p.dev_tools || '');
                }
            });
    }

    $("#techStackForm").submit(function (e) {
        e.preventDefault();
        var data = {
            primary_os: $("#tsOS").val(),
            browser: $("#tsBrowser").val(),
            email_client: $("#tsEmail").val(),
            remote_access: $("#tsRemoteAccess").val(),
            mobile_device: $("#tsMobile").val(),
            cloud_apps: $("#tsCloudApps").val(),
            dev_tools: $("#tsDevTools").val()
        };
        api.hygiene.techStack.save(data)
            .success(function () {
                successFlash("Tech stack profile saved. Your personalized checklist will be updated.");
            })
            .error(function () {
                errorFlash("Failed to save tech stack profile.");
            });
    });

    // ===================== ADMIN DASHBOARD TAB =====================
    function loadAdminDashboard() {
        // Summary
        api.hygiene.admin.summary()
            .success(function (s) {
                $("#adminTotalDevices").text(s.total_devices);
                $("#adminFullyCompliant").text(s.fully_compliant);
                $("#adminAtRisk").text(s.at_risk_devices);
                $("#adminAvgScore").text(Math.round(s.avg_score) + "%");
                $("#adminProfileCount").text(s.profile_count);
                var totalChecks = s.pass_count + s.fail_count + s.unknown_count;
                var passRate = totalChecks > 0 ? Math.round((s.pass_count / totalChecks) * 100) : 0;
                $("#adminPassRate").text(passRate + "%");

                // Check breakdown
                renderCheckBreakdown(s.check_breakdown);
                renderOSBreakdown(s.os_breakdown);
                renderDeviceTypeBreakdown(s.device_type_breakdown);
            });

        // Enriched device table
        api.hygiene.admin.devicesEnriched()
            .success(function (devices) {
                renderAdminDevices(devices);
            });
    }

    function renderCheckBreakdown(breakdown) {
        var container = $("#checkBreakdown");
        container.empty();
        if (!breakdown || Object.keys(breakdown).length === 0) {
            container.html('<p class="text-muted">No data yet.</p>');
            return;
        }
        var html = '<table class="table table-condensed"><thead><tr><th>Check</th><th>Pass</th><th>Fail</th><th>Unknown</th></tr></thead><tbody>';
        var labels = {
            os_updated: 'OS Updated', antivirus_active: 'Antivirus', disk_encrypted: 'Disk Encrypted',
            screen_lock: 'Screen Lock', password_manager: 'Password Mgr', vpn_enabled: 'VPN', mfa_enabled: 'MFA'
        };
        $.each(breakdown, function (key, stat) {
            var total = stat.pass + stat.fail + stat.unknown;
            var passPct = total > 0 ? Math.round((stat.pass / total) * 100) : 0;
            html += '<tr><td>' + (labels[key] || key) + '</td>' +
                '<td><span class="text-success">' + stat.pass + '</span></td>' +
                '<td><span class="text-danger">' + stat.fail + '</span></td>' +
                '<td><span class="text-muted">' + stat.unknown + '</span></td></tr>';
        });
        html += '</tbody></table>';
        container.html(html);
    }

    function renderOSBreakdown(breakdown) {
        var container = $("#osBreakdown");
        container.empty();
        if (!breakdown || Object.keys(breakdown).length === 0) {
            container.html('<p class="text-muted">No data.</p>');
            return;
        }
        var html = '<ul class="list-group">';
        $.each(breakdown, function (os, count) {
            html += '<li class="list-group-item"><span class="badge">' + count + '</span>' + escapeHtml(os) + '</li>';
        });
        html += '</ul>';
        container.html(html);
    }

    function renderDeviceTypeBreakdown(breakdown) {
        var container = $("#deviceTypeBreakdown");
        container.empty();
        if (!breakdown || Object.keys(breakdown).length === 0) {
            container.html('<p class="text-muted">No data.</p>');
            return;
        }
        var icons = {
            laptop: 'fa-laptop', desktop: 'fa-desktop',
            mobile: 'fa-mobile', tablet: 'fa-tablet', other: 'fa-cube'
        };
        var html = '<ul class="list-group">';
        $.each(breakdown, function (dt, count) {
            html += '<li class="list-group-item"><span class="badge">' + count + '</span>' +
                '<i class="fa ' + (icons[dt] || 'fa-cube') + '"></i> ' + escapeHtml(dt) + '</li>';
        });
        html += '</ul>';
        container.html(html);
    }

    function renderAdminDevices(devices) {
        var body = $("#adminDevicesBody");
        body.empty();
        if (!devices || devices.length === 0) {
            body.append('<tr><td colspan="6" class="text-center text-muted">No devices registered.</td></tr>');
            return;
        }
        $.each(devices, function (i, d) {
            var scoreClass = d.hygiene_score >= 80 ? 'success' : d.hygiene_score >= 50 ? 'warning' : 'danger';
            body.append(
                '<tr>' +
                '<td>' + escapeHtml(d.user_name || '—') + '</td>' +
                '<td>' + escapeHtml(d.user_email || '—') + '</td>' +
                '<td>' + escapeHtml(d.name) + '</td>' +
                '<td>' + escapeHtml(d.device_type) + '</td>' +
                '<td>' + escapeHtml(d.os || '—') + '</td>' +
                '<td><span class="label label-' + scoreClass + '">' + d.hygiene_score + '%</span></td>' +
                '</tr>'
            );
        });
    }
});

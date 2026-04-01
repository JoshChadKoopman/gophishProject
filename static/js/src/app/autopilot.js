$(document).ready(function () {
    // Populate hour dropdowns
    var startSelect = $("#activeHoursStart");
    var endSelect = $("#activeHoursEnd");
    for (var h = 0; h < 24; h++) {
        var label = (h < 10 ? "0" : "") + h + ":00";
        startSelect.append('<option value="' + h + '">' + label + '</option>');
        endSelect.append('<option value="' + h + '">' + label + '</option>');
    }
    startSelect.val("9");
    endSelect.val("17");

    // Load dependent data (groups, sending profiles, landing pages) and config
    loadFormData();
    loadConfig();
    loadBlackoutDates();
    loadSchedule();

    // Save config
    $("#configForm").submit(function (e) {
        e.preventDefault();
        saveConfig();
    });

    // Enable/Disable
    $("#btnEnable").click(function () {
        api.autopilot.enable().done(function () {
            successFlash("Autopilot enabled");
            loadConfig();
        }).fail(function (data) {
            errorFlash(data.responseJSON.message);
        });
    });

    $("#btnDisable").click(function () {
        Swal.fire({
            title: "Disable Autopilot?",
            text: "No new automated campaigns will be created until re-enabled.",
            type: "warning",
            showCancelButton: true,
            confirmButtonText: "Disable",
            confirmButtonColor: "#d33"
        }).then(function (result) {
            if (result.value) {
                api.autopilot.disable().done(function () {
                    successFlash("Autopilot disabled");
                    loadConfig();
                }).fail(function (data) {
                    errorFlash(data.responseJSON.message);
                });
            }
        });
    });

    // Add blackout date
    $("#blackoutForm").submit(function (e) {
        e.preventDefault();
        var date = $("#blackoutDate").val();
        if (!date) {
            errorFlash("Please select a date");
            return;
        }
        api.autopilot.addBlackout({
            date: date,
            reason: $("#blackoutReason").val()
        }).done(function () {
            $("#blackoutDate").val("");
            $("#blackoutReason").val("");
            loadBlackoutDates();
        }).fail(function (data) {
            errorFlash(data.responseJSON.message);
        });
    });
});

function loadFormData() {
    // Load groups
    api.groups.get().done(function (groups) {
        var select = $("#targetGroups");
        select.empty();
        $.each(groups, function (i, group) {
            select.append('<option value="' + group.id + '">' + escapeHtml(group.name) + '</option>');
        });
    });

    // Load sending profiles
    api.SMTP.get().done(function (profiles) {
        var select = $("#sendingProfile");
        select.find("option:not(:first)").remove();
        $.each(profiles, function (i, p) {
            select.append('<option value="' + p.id + '">' + escapeHtml(p.name) + '</option>');
        });
    });

    // Load landing pages
    api.pages.get().done(function (pages) {
        var select = $("#landingPage");
        select.find("option:not(:first)").remove();
        $.each(pages, function (i, p) {
            select.append('<option value="' + p.id + '">' + escapeHtml(p.name) + '</option>');
        });
    });
}

function loadConfig() {
    api.autopilot.getConfig().done(function (config) {
        $("#loading").hide();
        $("#autopilotContent").show();

        // Fill form
        $("#cadenceDays").val(config.cadence_days || 15);
        $("#activeHoursStart").val(config.active_hours_start || 9);
        $("#activeHoursEnd").val(config.active_hours_end || 17);
        $("#timezone").val(config.timezone || "UTC");
        $("#phishUrl").val(config.phish_url || "");
        $("#sendingProfile").val(config.sending_profile_id || 0);
        $("#landingPage").val(config.landing_page_id || 0);

        // Parse and select target groups
        if (config.target_group_ids) {
            try {
                var groupIds = JSON.parse(config.target_group_ids);
                $("#targetGroups").val(groupIds);
            } catch (e) { /* ignore */ }
        }

        // Update status
        updateStatus(config.enabled, config.last_run, config.next_run);
    }).fail(function () {
        // No config yet — show empty form
        $("#loading").hide();
        $("#autopilotContent").show();
        updateStatus(false, null, null);
    });
}

function updateStatus(enabled, lastRun, nextRun) {
    if (enabled) {
        $("#statusBadge").text("Active").removeClass("label-default label-danger").addClass("label-success");
        $("#btnEnable").hide();
        $("#btnDisable").show();
    } else {
        $("#statusBadge").text("Disabled").removeClass("label-default label-success").addClass("label-danger");
        $("#btnEnable").show();
        $("#btnDisable").hide();
    }

    if (lastRun && lastRun !== "0001-01-01T00:00:00Z") {
        $("#lastRunInfo").text("Last run: " + moment(lastRun).format("MMMM Do YYYY, h:mm a"));
    } else {
        $("#lastRunInfo").text("Last run: Never");
    }

    if (nextRun && nextRun !== "0001-01-01T00:00:00Z" && enabled) {
        $("#nextRunInfo").text("Next run: " + moment(nextRun).format("MMMM Do YYYY, h:mm a"));
    } else {
        $("#nextRunInfo").text("");
    }
}

function saveConfig() {
    var selectedGroups = $("#targetGroups").val() || [];
    var groupIds = selectedGroups.map(function (v) { return parseInt(v); });

    var config = {
        cadence_days: parseInt($("#cadenceDays").val()) || 15,
        active_hours_start: parseInt($("#activeHoursStart").val()) || 9,
        active_hours_end: parseInt($("#activeHoursEnd").val()) || 17,
        timezone: $("#timezone").val() || "UTC",
        phish_url: $("#phishUrl").val(),
        sending_profile_id: parseInt($("#sendingProfile").val()) || 0,
        landing_page_id: parseInt($("#landingPage").val()) || 0,
        target_group_ids: JSON.stringify(groupIds)
    };

    api.autopilot.saveConfig(config).done(function () {
        successFlash("Configuration saved");
        loadConfig();
    }).fail(function (data) {
        errorFlash(data.responseJSON.message);
    });
}

function loadBlackoutDates() {
    api.autopilot.getBlackouts().done(function (dates) {
        var tbody = $("#blackoutBody");
        tbody.empty();
        if (!dates || dates.length === 0) {
            $("#blackoutTable").hide();
            $("#noBlackouts").show();
            return;
        }
        $("#blackoutTable").show();
        $("#noBlackouts").hide();
        $.each(dates, function (i, d) {
            tbody.append(
                '<tr>' +
                '<td>' + escapeHtml(d.date) + '</td>' +
                '<td>' + escapeHtml(d.reason || "") + '</td>' +
                '<td><button class="btn btn-danger btn-xs btn-delete-blackout" data-id="' + d.id + '"><i class="fa fa-trash"></i></button></td>' +
                '</tr>'
            );
        });

        // Bind delete buttons
        $(".btn-delete-blackout").click(function () {
            var id = $(this).data("id");
            api.autopilot.deleteBlackout(id).done(function () {
                loadBlackoutDates();
            }).fail(function (data) {
                errorFlash(data.responseJSON.message);
            });
        });
    });
}

function loadSchedule() {
    api.autopilot.getSchedule(50).done(function (entries) {
        var tbody = $("#scheduleBody");
        tbody.empty();
        if (!entries || entries.length === 0) {
            $("#scheduleTable").hide();
            $("#noSchedule").show();
            return;
        }
        $("#scheduleTable").show();
        $("#noSchedule").hide();

        var diffLabels = { 1: "Easy", 2: "Medium", 3: "Hard", 4: "Sophisticated" };
        $.each(entries, function (i, e) {
            tbody.append(
                '<tr>' +
                '<td>' + escapeHtml(e.user_email) + '</td>' +
                '<td>' + e.campaign_id + '</td>' +
                '<td>' + (diffLabels[e.difficulty_level] || e.difficulty_level) + '</td>' +
                '<td>' + moment(e.scheduled_date).format("MMMM Do YYYY, h:mm a") + '</td>' +
                '<td>' + (e.sent ? '<span class="label label-success">Sent</span>' : '<span class="label label-default">Pending</span>') + '</td>' +
                '</tr>'
            );
        });
    });
}

$(document).ready(function () {
    var classifyId = null;

    function loadConfig() {
        api.reportButton.getConfig()
            .success(function (config) {
                if (config && config.plugin_api_key) {
                    $("#pluginApiKey").val(config.plugin_api_key);
                    $("#feedbackSimulation").val(config.feedback_simulation);
                    $("#feedbackReal").val(config.feedback_real);
                    if (!config.enabled) {
                        $("input[name='enabled'][value='false']").prop("checked", true);
                    }
                }
            });
    }

    function loadEmails() {
        api.reportedEmails.get()
            .success(function (emails) {
                var tbody = $("#reportedEmailsBody");
                tbody.empty();
                if (!emails || emails.length === 0) {
                    tbody.append('<tr><td colspan="7" style="text-align:center; color:#999;">No reported emails yet</td></tr>');
                    return;
                }
                $.each(emails, function (i, email) {
                    var typeBadge = email.is_simulation
                        ? '<span class="label label-info">Simulation</span>'
                        : '<span class="label label-default">Real</span>';
                    var classBadge = classificationBadge(email.classification);
                    var date = moment(email.created_date).format("MMMM Do YYYY, h:mm a");
                    var actions = '';
                    if (email.classification === 'pending') {
                        actions = '<button class="btn btn-xs btn-primary classify-btn" data-id="' + email.id + '"><i class="fa fa-tag"></i> Classify</button>';
                    }
                    tbody.append(
                        '<tr>' +
                        '<td>' + escapeHtml(date) + '</td>' +
                        '<td>' + escapeHtml(email.reporter_email) + '</td>' +
                        '<td>' + escapeHtml(email.sender_email) + '</td>' +
                        '<td>' + escapeHtml(email.subject) + '</td>' +
                        '<td>' + typeBadge + '</td>' +
                        '<td>' + classBadge + '</td>' +
                        '<td>' + actions + '</td>' +
                        '</tr>'
                    );
                });
            });
    }

    function classificationBadge(classification) {
        var colors = {
            'pending': 'warning',
            'simulation': 'info',
            'safe': 'success',
            'phishing': 'danger',
            'spam': 'default',
            'suspicious': 'warning'
        };
        var color = colors[classification] || 'default';
        return '<span class="label label-' + color + '">' + escapeHtml(classification) + '</span>';
    }

    // Save config
    $("#reportButtonConfigForm").submit(function (e) {
        e.preventDefault();
        var data = {
            feedback_simulation: $("#feedbackSimulation").val(),
            feedback_real: $("#feedbackReal").val(),
            enabled: $("input[name='enabled']:checked").val() === "true"
        };
        api.reportButton.saveConfig(data)
            .success(function (config) {
                successFlash("Report button configuration saved.");
                if (config.plugin_api_key) {
                    $("#pluginApiKey").val(config.plugin_api_key);
                }
            })
            .error(function (data) {
                errorFlash(data.responseJSON.message);
            });
    });

    // Regenerate API key
    $("#regenerateKeyBtn").click(function () {
        if (!confirm("Are you sure? The current plugin API key will stop working.")) return;
        api.reportButton.regenerateKey()
            .success(function (config) {
                $("#pluginApiKey").val(config.plugin_api_key);
                successFlash("Plugin API key regenerated.");
            })
            .error(function (data) {
                errorFlash(data.responseJSON.message);
            });
    });

    // Open classify modal
    $(document).on("click", ".classify-btn", function () {
        classifyId = $(this).data("id");
        $("#classifySelect").val("safe");
        $("#classifyNotes").val("");
        $("#classifyModal").modal("show");
    });

    // Submit classification
    $("#classifySubmitBtn").click(function () {
        if (!classifyId) return;
        api.reportedEmails.classify(classifyId, {
            classification: $("#classifySelect").val(),
            admin_notes: $("#classifyNotes").val()
        })
            .success(function () {
                $("#classifyModal").modal("hide");
                successFlash("Email classified successfully.");
                loadEmails();
            })
            .error(function (data) {
                errorFlash(data.responseJSON.message);
            });
    });

    loadConfig();
    loadEmails();
});

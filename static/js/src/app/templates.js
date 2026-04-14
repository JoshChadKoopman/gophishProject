var templates = []
var icons = {
    "application/vnd.ms-excel": "fa-file-excel-o",
    "text/plain": "fa-file-text-o",
    "image/gif": "fa-file-image-o",
    "image/png": "fa-file-image-o",
    "application/pdf": "fa-file-pdf-o",
    "application/x-zip-compressed": "fa-file-archive-o",
    "application/x-gzip": "fa-file-archive-o",
    "application/vnd.openxmlformats-officedocument.presentationml.presentation": "fa-file-powerpoint-o",
    "application/vnd.openxmlformats-officedocument.wordprocessingml.document": "fa-file-word-o",
    "application/octet-stream": "fa-file-o",
    "application/x-msdownload": "fa-file-o"
}

// Save attempts to POST to /templates/
function save(idx) {
    var template = {
        attachments: []
    }
    template.name = $("#name").val()
    template.subject = $("#subject").val()
    template.envelope_sender = $("#envelope-sender").val()
    template.html = CKEDITOR.instances["html_editor"].getData();
    // Fix the URL Scheme added by CKEditor (until we can remove it from the plugin)
    template.html = template.html.replace(/https?:\/\/{{\.URL}}/gi, "{{.URL}}")
    // If the "Add Tracker Image" checkbox is checked, add the tracker
    if ($("#use_tracker_checkbox").prop("checked")) {
        if (template.html.indexOf("{{.Tracker}}") == -1 &&
            template.html.indexOf("{{.TrackingUrl}}") == -1) {
            template.html = template.html.replace("</body>", "{{.Tracker}}</body>")
        }
    } else {
        // Otherwise, remove the tracker
        template.html = template.html.replace("{{.Tracker}}</body>", "</body>")
    }
    template.text = $("#text_editor").val()
    // Add the attachments
    $.each($("#attachmentsTable").DataTable().rows().data(), function (i, target) {
        template.attachments.push({
            name: unescapeHtml(target[1]),
            content: target[3],
            type: target[4],
        })
    })

    if (idx != -1) {
        template.id = templates[idx].id
        api.templateId.put(template)
            .success(function (data) {
                successFlash("Template edited successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    } else {
        // Submit the template
        api.templates.post(template)
            .success(function (data) {
                successFlash("Template added successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    }
}

function dismiss() {
    $("#modal\\.flashes").empty()
    $("#attachmentsTable").dataTable().DataTable().clear().draw()
    $("#name").val("")
    $("#subject").val("")
    $("#text_editor").val("")
    $("#html_editor").val("")
    $("#modal").modal('hide')
}

var deleteTemplate = function (idx) {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the template. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(templates[idx].name),
        confirmButtonColor: "#E94560",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.templateId.delete(templates[idx].id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if(result.value) {
            Swal.fire(
                'Template Deleted!',
                'This template has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

function deleteTemplate(idx) {
    if (confirm("Delete " + templates[idx].name + "?")) {
        api.templateId.delete(templates[idx].id)
            .success(function (data) {
                successFlash(data.message)
                load()
            })
    }
}

function attach(files) {
    attachmentsTable = $("#attachmentsTable").DataTable({
        destroy: true,
        "order": [
            [1, "asc"]
        ],
        columnDefs: [{
            orderable: false,
            targets: "no-sort"
        }, {
            sClass: "datatable_hidden",
            targets: [3, 4]
        }]
    });
    $.each(files, function (i, file) {
        var reader = new FileReader();
        /* Make this a datatable */
        reader.onload = function (e) {
            var icon = icons[file.type] || "fa-file-o"
            // Add the record to the modal
            attachmentsTable.row.add([
                '<i class="fa ' + icon + '"></i>',
                escapeHtml(file.name),
                '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
                reader.result.split(",")[1],
                file.type || "application/octet-stream"
            ]).draw()
        }
        reader.onerror = function (e) {
            console.log(e)
        }
        reader.readAsDataURL(file)
    })
}

function edit(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(idx)
    })
    $("#attachmentUpload").unbind('click').click(function () {
        this.value = null
    })
    $("#html_editor").ckeditor()
    setupAutocomplete(CKEDITOR.instances["html_editor"])
    $("#attachmentsTable").show()
    attachmentsTable = $('#attachmentsTable').DataTable({
        destroy: true,
        "order": [
            [1, "asc"]
        ],
        columnDefs: [{
            orderable: false,
            targets: "no-sort"
        }, {
            sClass: "datatable_hidden",
            targets: [3, 4]
        }]
    });
    var template = {
        attachments: []
    }
    if (idx != -1) {
        $("#templateModalLabel").text("Edit Template")
        template = templates[idx]
        $("#name").val(template.name)
        $("#subject").val(template.subject)
        $("#envelope-sender").val(template.envelope_sender)
        $("#html_editor").val(template.html)
        $("#text_editor").val(template.text)
        attachmentRows = []
        $.each(template.attachments, function (i, file) {
            var icon = icons[file.type] || "fa-file-o"
            // Add the record to the modal
            attachmentRows.push([
                '<i class="fa ' + icon + '"></i>',
                escapeHtml(file.name),
                '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
                file.content,
                file.type || "application/octet-stream"
            ])
        })
        attachmentsTable.rows.add(attachmentRows).draw()
        if (template.html.indexOf("{{.Tracker}}") != -1) {
            $("#use_tracker_checkbox").prop("checked", true)
        } else {
            $("#use_tracker_checkbox").prop("checked", false)
        }

    } else {
        $("#templateModalLabel").text("New Template")
    }
    // Handle Deletion
    $("#attachmentsTable").unbind('click').on("click", "span>i.fa-trash-o", function () {
        attachmentsTable.row($(this).parents('tr'))
            .remove()
            .draw();
    })
}

function copy(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(-1)
    })
    $("#attachmentUpload").unbind('click').click(function () {
        this.value = null
    })
    $("#html_editor").ckeditor()
    $("#attachmentsTable").show()
    attachmentsTable = $('#attachmentsTable').DataTable({
        destroy: true,
        "order": [
            [1, "asc"]
        ],
        columnDefs: [{
            orderable: false,
            targets: "no-sort"
        }, {
            sClass: "datatable_hidden",
            targets: [3, 4]
        }]
    });
    var template = {
        attachments: []
    }
    template = templates[idx]
    $("#name").val("Copy of " + template.name)
    $("#subject").val(template.subject)
    $("#envelope-sender").val(template.envelope_sender)
    $("#html_editor").val(template.html)
    $("#text_editor").val(template.text)
    $.each(template.attachments, function (i, file) {
        var icon = icons[file.type] || "fa-file-o"
        // Add the record to the modal
        attachmentsTable.row.add([
            '<i class="fa ' + icon + '"></i>',
            escapeHtml(file.name),
            '<span class="remove-row"><i class="fa fa-trash-o"></i></span>',
            file.content,
            file.type || "application/octet-stream"
        ]).draw()
    })
    // Handle Deletion
    $("#attachmentsTable").unbind('click').on("click", "span>i.fa-trash-o", function () {
        attachmentsTable.row($(this).parents('tr'))
            .remove()
            .draw();
    })
    if (template.html.indexOf("{{.Tracker}}") != -1) {
        $("#use_tracker_checkbox").prop("checked", true)
    } else {
        $("#use_tracker_checkbox").prop("checked", false)
    }
}

function importEmail() {
    raw = $("#email_content").val()
    convert_links = $("#convert_links_checkbox").prop("checked")
    if (!raw) {
        modalError("No Content Specified!")
    } else {
        api.import_email({
                content: raw,
                convert_links: convert_links
            })
            .success(function (data) {
                $("#text_editor").val(data.text)
                $("#html_editor").val(data.html)
                $("#subject").val(data.subject)
                // If the HTML is provided, let's open that view in the editor
                if (data.html) {
                    CKEDITOR.instances["html_editor"].setMode('wysiwyg')
                    $('.nav-tabs a[href="#html"]').click()
                }
                $("#importEmailModal").modal("hide")
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    }
}

function load() {
    $("#templateTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    api.templates.get()
        .success(function (ts) {
            templates = ts
            $("#loading").hide()
            if (templates.length > 0) {
                $("#templateTable").show()
                templateTable = $("#templateTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                templateTable.clear()
                templateRows = []
                $.each(templates, function (i, template) {
                    templateRows.push([
                        escapeHtml(template.name),
                        moment(template.modified_date).format('MMMM Do YYYY, h:mm:ss a'),
                        "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit Template' onclick='edit(" + i + ")'>\
                    <i class='fa fa-pencil'></i>\
                    </button></span>\
		    <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy Template' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>\
                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete Template' onclick='deleteTemplate(" + i + ")'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                })
                templateTable.rows.add(templateRows).draw()
                $('[data-toggle="tooltip"]').tooltip()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching templates")
        })
}

$(document).ready(function () {
    // Setup multiple modals
    // Code based on http://miles-by-motorcycle.com/static/bootstrap-modal/index.html
    $('.modal').on('hidden.bs.modal', function (event) {
        $(this).removeClass('fv-modal-stack');
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') - 1);
    });
    $('.modal').on('shown.bs.modal', function (event) {
        // Keep track of the number of open modals
        if (typeof ($('body').data('fv_open_modals')) == 'undefined') {
            $('body').data('fv_open_modals', 0);
        }
        // if the z-index of this modal has been set, ignore.
        if ($(this).hasClass('fv-modal-stack')) {
            return;
        }
        $(this).addClass('fv-modal-stack');
        // Increment the number of open modals
        $('body').data('fv_open_modals', $('body').data('fv_open_modals') + 1);
        // Setup the appropriate z-index
        $(this).css('z-index', 1040 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('.fv-modal-stack').css('z-index', 1039 + (10 * $('body').data('fv_open_modals')));
        $('.modal-backdrop').not('fv-modal-stack').addClass('fv-modal-stack');
    });
    $.fn.modal.Constructor.prototype.enforceFocus = function () {
        $(document)
            .off('focusin.bs.modal') // guard against infinite focus loop
            .on('focusin.bs.modal', $.proxy(function (e) {
                if (
                    this.$element[0] !== e.target && !this.$element.has(e.target).length
                    // CKEditor compatibility fix start.
                    &&
                    !$(e.target).closest('.cke_dialog, .cke').length
                    // CKEditor compatibility fix end.
                ) {
                    this.$element.trigger('focus');
                }
            }, this));
    };
    // Scrollbar fix - https://stackoverflow.com/questions/19305821/multiple-modals-overlay
    $(document).on('hidden.bs.modal', '.modal', function () {
        $('.modal:visible').length && $(document.body).addClass('modal-open');
    });
    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    });
    $("#importEmailModal").on('hidden.bs.modal', function (event) {
        $("#email_content").val("")
    })
    CKEDITOR.on('dialogDefinition', function (ev) {
        // Take the dialog name and its definition from the event data.
        var dialogName = ev.data.name;
        var dialogDefinition = ev.data.definition;

        // Check if the definition is from the dialog window you are interested in (the "Link" dialog window).
        if (dialogName == 'link') {
            dialogDefinition.minWidth = 500
            dialogDefinition.minHeight = 100

            // Remove the linkType field
            var infoTab = dialogDefinition.getContents('info');
            infoTab.get('linkType').hidden = true;
        }
    });
    load()

    // --- AI Template Generation ---
    function getAIParams() {
        return {
            prompt: $("#ai-prompt").val(),
            difficulty_level: parseInt($("#ai-difficulty").val()),
            language: $("#ai-language").val(),
            target_role: $("#ai-target-role").val(),
            target_department: $("#ai-target-dept").val(),
            company_name: $("#ai-company").val(),
            sender_name: $("#ai-sender").val()
        }
    }

    function aiSetLoading(loading) {
        if (loading) {
            $("#aiGenerating").show();
            $("#aiGenerateAndEdit, #aiGenerateAndSave").prop("disabled", true);
        } else {
            $("#aiGenerating").hide();
            $("#aiGenerateAndEdit, #aiGenerateAndSave").prop("disabled", false);
        }
    }

    function aiError(msg) {
        $("#aiFlashes").html('<div class="alert alert-danger" style="text-align:center"><i class="fa fa-exclamation-circle"></i> ' + escapeHtml(msg) + '</div>');
    }

    // Generate & Edit: generate via AI, then populate the template editor modal
    $("#aiGenerateAndEdit").on("click", function () {
        var params = getAIParams();
        if (!params.prompt) { aiError("Please describe the phishing scenario."); return; }
        $("#aiFlashes").empty();
        aiSetLoading(true);
        api.ai.generateTemplate(params)
            .success(function (data) {
                aiSetLoading(false);
                $("#aiGenerateModal").modal("hide");
                // Open the template editor with generated content
                edit(-1);
                $("#name").val($("#ai-template-name").val() || "AI Generated - " + moment().format("YYYY-MM-DD HH:mm"));
                $("#subject").val(data.subject);
                $("#text_editor").val(data.text);
                $("#html_editor").val(data.html);
                if (data.html && CKEDITOR.instances["html_editor"]) {
                    CKEDITOR.instances["html_editor"].setData(data.html);
                    $('.nav-tabs a[href="#html"]').click();
                }
                successFlash("AI template generated successfully. Review and save when ready.");
            })
            .error(function (data) {
                aiSetLoading(false);
                var msg = (data.responseJSON && data.responseJSON.message) || "AI generation failed.";
                aiError(msg);
            });
    });

    // Generate & Save: generate via AI and save directly as a template
    $("#aiGenerateAndSave").on("click", function () {
        var params = getAIParams();
        if (!params.prompt) { aiError("Please describe the phishing scenario."); return; }
        $("#aiFlashes").empty();
        aiSetLoading(true);
        params.save_as_template = true;
        params.template_name = $("#ai-template-name").val();
        api.ai.generateTemplate(params)
            .success(function (data) {
                aiSetLoading(false);
                $("#aiGenerateModal").modal("hide");
                successFlash("AI template '" + escapeHtml(data.template.name) + "' generated and saved!");
                load();
            })
            .error(function (data) {
                aiSetLoading(false);
                var msg = (data.responseJSON && data.responseJSON.message) || "AI generation failed.";
                aiError(msg);
            });
    });

    // Clear AI modal on close
    $("#aiGenerateModal").on("hidden.bs.modal", function () {
        $("#aiFlashes").empty();
        $("#ai-prompt").val("");
        $("#ai-template-name").val("");
        $("#ai-target-role").val("");
        $("#ai-target-dept").val("");
        $("#ai-company").val("");
        $("#ai-sender").val("");
        aiSetLoading(false);
    });

    // --- Template Library ---
    var libTemplates = [];

    function renderLibrary(list) {
        var grid = $("#libraryGrid").empty();
        if (!list || list.length === 0) {
            grid.html('<div class="col-md-12"><p class="text-muted text-center">No templates match the selected filters.</p></div>');
            return;
        }
        var diffLabels = {1: "Easy", 2: "Medium", 3: "Hard"};
        var diffColors = {1: "#28a745", 2: "#ffc107", 3: "#dc3545"};
        $.each(list, function (i, t) {
            var card = '<div class="col-md-6" style="margin-bottom:15px">' +
                '<div style="border:1px solid #e0e0e0;border-radius:6px;padding:15px;height:100%">' +
                '<h5 style="margin-top:0">' + escapeHtml(t.name) + '</h5>' +
                '<p style="font-size:12px;color:#888;margin:4px 0"><span class="label label-default">' + escapeHtml(t.category) + '</span> ' +
                '<span class="label" style="background:' + (diffColors[t.difficulty_level] || '#999') + '">' + (diffLabels[t.difficulty_level] || t.difficulty_level) + '</span></p>' +
                '<p style="font-size:13px;color:#555">' + escapeHtml(t.description) + '</p>' +
                '<p style="font-size:12px;color:#999">Subject: <em>' + escapeHtml(t.subject || '(SMS — no subject)') + '</em></p>' +
                '<button class="btn btn-sm btn-primary lib-import-btn" data-slug="' + escapeHtml(t.slug) + '"><i class="fa fa-download"></i> Import</button>' +
                '</div></div>';
            grid.append(card);
        });
    }

    function filterLibrary() {
        var cat = $("#libCategoryFilter").val();
        var diff = parseInt($("#libDifficultyFilter").val()) || 0;
        var filtered = libTemplates;
        if (cat) filtered = filtered.filter(function (t) { return t.category === cat; });
        if (diff) filtered = filtered.filter(function (t) { return t.difficulty_level === diff; });
        renderLibrary(filtered);
    }

    $("#libraryModal").on("show.bs.modal", function () {
        $("#libraryLoading").show();
        $("#libraryGrid").hide();
        // Load categories
        api.TemplateLibrary.categories()
            .success(function (cats) {
                var sel = $("#libCategoryFilter").empty().append('<option value="">All Categories</option>');
                $.each(cats, function (i, c) {
                    sel.append('<option value="' + escapeHtml(c) + '">' + escapeHtml(c) + '</option>');
                });
            });
        // Load templates
        api.TemplateLibrary.get()
            .success(function (data) {
                libTemplates = data;
                $("#libraryLoading").hide();
                $("#libraryGrid").show();
                renderLibrary(libTemplates);
            })
            .error(function () {
                $("#libraryLoading").hide();
                $("#libraryFlashes").html('<div class="alert alert-danger">Error loading template library</div>');
            });
    });

    $("#libCategoryFilter, #libDifficultyFilter").on("change", filterLibrary);

    $(document).on("click", ".lib-import-btn", function () {
        var btn = $(this);
        var slug = btn.data("slug");
        btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Importing...');
        api.TemplateLibrary.import(slug)
            .success(function (data) {
                btn.prop("disabled", false).html('<i class="fa fa-check"></i> Imported!');
                setTimeout(function () {
                    btn.html('<i class="fa fa-download"></i> Import');
                }, 2000);
                load();
            })
            .error(function (data) {
                btn.prop("disabled", false).html('<i class="fa fa-download"></i> Import');
                var msg = (data.responseJSON && data.responseJSON.message) || "Import failed";
                $("#libraryFlashes").html('<div class="alert alert-danger">' + escapeHtml(msg) + '</div>');
            });
    });

})

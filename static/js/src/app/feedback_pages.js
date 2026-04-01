/*
    feedback_pages.js
    Handles the creation, editing, and deletion of feedback pages
    (educational interstitials shown after phishing link click)
*/
var feedbackPages = []

function save(idx) {
    var fp = {}
    fp.name = $("#name").val()
    fp.language = $("#language").val()
    fp.redirect_delay_seconds = parseInt($("#redirect_delay").val()) || 10
    fp.redirect_url = $("#redirect_url_input").val()
    var editor = CKEDITOR.instances["html_editor"]
    fp.html = editor.getData()
    if (idx != -1) {
        fp.id = feedbackPages[idx].id
        api.feedbackPageId.put(fp)
            .success(function (data) {
                successFlash("Feedback page edited successfully!")
                load()
                dismiss()
            })
            .error(function (data) {
                modalError(data.responseJSON.message)
            })
    } else {
        api.feedbackPages.post(fp)
            .success(function (data) {
                successFlash("Feedback page added successfully!")
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
    $("#name").val("")
    $("#html_editor").val("")
    $("#redirect_url_input").val("")
    $("#language").val("en")
    $("#redirect_delay").val("10")
    $("#modal").modal('hide')
}

var deleteFeedbackPage = function (idx) {
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the feedback page. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(feedbackPages[idx].name),
        confirmButtonColor: "#E94560",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.feedbackPageId.delete(feedbackPages[idx].id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            })
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                'Feedback Page Deleted!',
                'This feedback page has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

function loadDefault() {
    var lang = $("#language").val()
    api.feedbackPages.getDefault(lang)
        .success(function (data) {
            CKEDITOR.instances["html_editor"].setData(data.html)
            successFlash("Default template loaded for " + lang)
        })
        .error(function () {
            modalError("Error loading default template")
        })
}

function edit(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(idx)
    })
    $("#html_editor").ckeditor()
    if (idx != -1) {
        $("#modalLabel").text("Edit Feedback Page")
        var fp = feedbackPages[idx]
        $("#name").val(fp.name)
        $("#language").val(fp.language || "en")
        $("#redirect_delay").val(fp.redirect_delay_seconds || 10)
        $("#redirect_url_input").val(fp.redirect_url)
        $("#html_editor").val(fp.html)
    } else {
        $("#modalLabel").text("New Feedback Page")
    }
}

function copy(idx) {
    $("#modalSubmit").unbind('click').click(function () {
        save(-1)
    })
    $("#html_editor").ckeditor()
    var fp = feedbackPages[idx]
    $("#name").val("Copy of " + fp.name)
    $("#language").val(fp.language || "en")
    $("#redirect_delay").val(fp.redirect_delay_seconds || 10)
    $("#redirect_url_input").val(fp.redirect_url)
    $("#html_editor").val(fp.html)
}

function load() {
    $("#feedbackPagesTable").hide()
    $("#emptyMessage").hide()
    $("#loading").show()
    api.feedbackPages.get()
        .success(function (fps) {
            feedbackPages = fps
            $("#loading").hide()
            if (feedbackPages.length > 0) {
                $("#feedbackPagesTable").show()
                var table = $("#feedbackPagesTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                table.clear()
                var rows = []
                $.each(feedbackPages, function (i, fp) {
                    rows.push([
                        escapeHtml(fp.name),
                        escapeHtml(fp.language || "en"),
                        moment(fp.modified_date).format('MMMM Do YYYY, h:mm:ss a'),
                        "<div class='pull-right'><span data-toggle='modal' data-backdrop='static' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Edit' onclick='edit(" + i + ")'>\
                    <i class='fa fa-pencil'></i>\
                    </button></span>\
                    <span data-toggle='modal' data-target='#modal'><button class='btn btn-primary' data-toggle='tooltip' data-placement='left' title='Copy' onclick='copy(" + i + ")'>\
                    <i class='fa fa-copy'></i>\
                    </button></span>\
                    <button class='btn btn-danger' data-toggle='tooltip' data-placement='left' title='Delete' onclick='deleteFeedbackPage(" + i + ")'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                    ])
                })
                table.rows.add(rows).draw()
                $('[data-toggle="tooltip"]').tooltip()
            } else {
                $("#emptyMessage").show()
            }
        })
        .error(function () {
            $("#loading").hide()
            errorFlash("Error fetching feedback pages")
        })
}

$(document).ready(function () {
    // Preview tab handler
    $('a[href="#preview"]').on('shown.bs.tab', function () {
        var editor = CKEDITOR.instances["html_editor"]
        var html = editor.getData()
        var iframe = document.getElementById("preview_iframe")
        iframe.contentDocument.open()
        iframe.contentDocument.write(html)
        iframe.contentDocument.close()
    })

    $('#modal').on('hidden.bs.modal', function (event) {
        dismiss()
    })

    load()
})

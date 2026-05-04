let users = []

const roleDescriptions = {
    superadmin: "Super Admin — Nivoxis staff. Full platform access across all tenants: users, campaigns, training, system settings.",
    org_admin: "Org Admin — Client HR admin. Manages their organisation's campaigns, users, and training. Requires MFA.",
    campaign_manager: "Campaign Manager — Can create and launch phishing campaigns and view results.",
    trainer: "Trainer — Can upload and manage training presentations. Cannot access phishing campaign tools.",
    learner: "Learner — End user. Completes assigned training and views their own results only.",
    auditor: "Auditor — Read-only access to reports and audit logs. Cannot create or modify any objects."
}

// Save attempts to POST or PUT to /users/
const save = (id) => {
    var changingPassword = id == -1 || $("#password-fields").is(":visible")
    // Validate that the passwords match (only when password fields are shown)
    if (changingPassword && $("#password").val() !== $("#confirm_password").val()) {
        modalError("Passwords must match.")
        return
    }
    // Validate required fields
    var firstName = $("#first_name").val().trim()
    var lastName = $("#last_name").val().trim()
    var emailVal = $("#email").val().trim()
    var positionVal = $("#position").val().trim()

    if (!firstName) { modalError("First name is required."); return }
    if (!lastName) { modalError("Last name / surname is required."); return }
    if (!emailVal) { modalError("Email is required."); return }
    if (!positionVal) { modalError("Position is required."); return }

    let user = {
        username: emailVal,
        password: changingPassword ? $("#password").val() : undefined,
        first_name: firstName,
        last_name: lastName,
        email: emailVal,
        position: positionVal,
        role: $("#role").val(),
        password_change_required: $("#force_password_change_checkbox").prop('checked'),
        account_locked: $("#account_locked_checkbox").prop('checked')
    }
    // Submit the user
    if (id != -1) {
        // If we're just editing an existing user,
        // we need to PUT /user/:id
        user.id = id
        api.userId.put(user)
            .success((data) => {
                successFlash("User " + escapeHtml(user.first_name + " " + user.last_name) + " updated successfully!")
                load()
                dismiss()
                $("#modal").modal('hide')
            })
            .error((data) => {
                modalError(data.responseJSON.message)
            })
    } else {
        // Else, if this is a new user, POST it
        // to /user
        api.users.post(user)
            .success((data) => {
                successFlash("User " + escapeHtml(user.first_name + " " + user.last_name) + " registered successfully!")
                load()
                dismiss()
                $("#modal").modal('hide')
            })
            .error((data) => {
                modalError(data.responseJSON.message)
            })
    }
}

const dismiss = () => {
    $("#first_name").val("")
    $("#last_name").val("")
    $("#email").val("")
    $("#position").val("")
    $("#password").val("")
    $("#confirm_password").val("")
    $("#role").val("")
    $("#force_password_change_checkbox").prop('checked', true)
    $("#account_locked_checkbox").prop('checked', false)
    $("#modal\\.flashes").empty()
    // Reset password section for next open
    $("#change-password-toggle-row").hide()
    $("#password-fields").show()
    $("#change-password-toggle").html('<i class="fa fa-lock"></i> Change password')
}

function toggleChangePassword(e) {
    e.preventDefault()
    var $fields = $("#password-fields")
    $fields.slideToggle(150, function () {
        var visible = $fields.is(":visible")
        $("#change-password-toggle").html(visible
            ? '<i class="fa fa-lock-open"></i> Hide password fields'
            : '<i class="fa fa-lock"></i> Change password')
    })
}

const edit = (id) => {
    populateRoleDropdown();
    $("#modalSubmit").unbind('click').click(() => {
        save(id)
    })
    if (id == -1) {
        $("#userModalLabel").text("New User")
        $("#role").val("reader")
        $("#role").trigger("change")
        // New user: show password fields, hide toggle link
        $("#change-password-toggle-row").hide()
        $("#password-fields").show()
    } else {
        $("#userModalLabel").text("Edit User")
        // Edit: hide password fields behind toggle
        $("#change-password-toggle-row").show()
        $("#password-fields").hide()
        $("#change-password-toggle").html('<i class="fa fa-lock"></i> Change password')
        api.userId.get(id)
            .success((user) => {
                $("#first_name").val(user.first_name || '')
                $("#last_name").val(user.last_name || '')
                $("#email").val(user.email || '')
                $("#position").val(user.position || '')
                $("#role").val(user.role.slug)
                $("#role").trigger("change")
                $("#force_password_change_checkbox").prop('checked', user.password_change_required)
                $("#account_locked_checkbox").prop('checked', user.account_locked)
            })
            .error(function () {
                errorFlash("Error fetching user")
            })
    }
}

const populateRoleDropdown = () => {
    api.roles.get()
        .success((roles) => {
            var $select = $("#role");
            var currentVal = $select.val();
            $select.empty();
            $.each(roles, (i, role) => {
                $select.append($('<option>', {
                    value: role.slug,
                    text: role.name
                }));
            });
            if (currentVal) {
                $select.val(currentVal);
            }
            $select.trigger("change");
        })
        .error(() => {
            // Fallback: hardcode roles
            var $select = $("#role");
            $select.empty();
            $select.append('<option value="admin">Admin</option>');
            $select.append('<option value="user">User</option>');
            $select.append('<option value="contributor">Contributor</option>');
            $select.append('<option value="reader">Reader</option>');
        })
}

const deleteUser = (id) => {
    var user = users.find(x => x.id == id)
    if (!user) {
        return
    }
    var displayName = (user.first_name || '') + (user.last_name ? ' ' + user.last_name : '') + ' (' + (user.email || '') + ')'
    if (user.role && user.role.slug === "superadmin" && user.email === "admin") {
        Swal.fire({
            title: "Unable to Delete User",
            text: "The admin account cannot be deleted.",
            type: "info"
        });
        return
    }
    Swal.fire({
        title: "Delete " + escapeHtml(displayName) + "?",
        html: "<p>This will permanently delete the account for <strong>" + escapeHtml(displayName) + "</strong>.</p>" +
              "<p><strong>The following will also be deleted:</strong></p>" +
              "<ul style='text-align:left; margin-left:20px;'>" +
              "<li>Campaigns they created</li>" +
              "<li>Training records and assignments</li>" +
              "<li>Audit log entries</li>" +
              "</ul>" +
              "<p class='text-danger' style='margin-top:8px;'><strong>This cannot be undone.</strong></p>",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete User",
        confirmButtonColor: "#E94560",
        cancelButtonText: "Cancel",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise((resolve, reject) => {
                api.userId.delete(id)
                    .success((msg) => {
                        resolve()
                    })
                    .error((data) => {
                        reject(data.responseJSON.message)
                    })
            })
            .catch(error => {
                Swal.showValidationMessage(error)
              })
        }
    }).then(function (result) {
        if (result.value){
            Swal.fire(
                'User Deleted!',
                "The user account for " + escapeHtml(displayName) + " and all associated objects have been deleted!",
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

const impersonate = (id) => {
    var user = users.find(x => x.id == id)
    if (!user) {
        return
    }
    var displayName = escapeHtml((user.first_name || '') + (user.last_name ? ' ' + user.last_name : '') + ' (' + (user.email || '') + ')')
    Swal.fire({
        title: "Are you sure?",
        html: "You will be logged out of your account and logged in as <strong>" + displayName + "</strong>",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Swap User",
        confirmButtonColor: "#428bca",
        reverseButtons: true,
        allowOutsideClick: false,
    }).then((result) => {
        if (result.value) {

         fetch('/impersonate', {
                method: 'post',
                body: "username=" + user.email + "&csrf_token=" + encodeURIComponent(csrf_token),
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                  },
          }).then((response) => {
                if (response.status == 200) {
                    Swal.fire({
                        title: "Success!",
                        html: "Successfully changed to user <strong>" + displayName + "</strong>.",
                        type: "success",
                        showCancelButton: false,
                        confirmButtonText: "Home",
                        allowOutsideClick: false,
                    }).then((result) => {
                        if (result.value) {
                            window.location.href = "/"
                        }});
                } else {
                    Swal.fire({
                        title: "Error!",
                        type: "error",
                        html: "Failed to change to user <strong>" + displayName + "</strong>.",
                        showCancelButton: false,
                    })
                }
            })
        }
      })
}

const load = () => {
    $("#userTable").hide()
    $("#loading").show()
    api.users.get()
        .success((us) => {
            users = us
            $("#loading").hide()
            $("#userTable").show()
            let userTable = $("#userTable").DataTable({
                destroy: true,
                columnDefs: [{
                    orderable: false,
                    targets: "no-sort"
                }]
            });
            userTable.clear();
            let userRows = []
            $.each(users, (i, user) => {
                let lastlogin = ""
                if (user.last_login != "0001-01-01T00:00:00Z") {
                    lastlogin = moment(user.last_login).format('MMMM Do YYYY, h:mm:ss a')
                }
                var roleBadge = "";
                if (user.role.slug === "superadmin") {
                    roleBadge = " <span class='label label-danger'>Super Admin</span>";
                } else if (user.role.slug === "org_admin") {
                    roleBadge = " <span class='label label-warning'>Org Admin</span>";
                } else if (user.role.slug === "campaign_manager") {
                    roleBadge = " <span class='label label-warning'>Campaign Mgr</span>";
                } else if (user.role.slug === "trainer") {
                    roleBadge = " <span class='label label-default'>Trainer</span>";
                } else if (user.role.slug === "auditor") {
                    roleBadge = " <span class='label label-info'>Auditor</span>";
                } else {
                    roleBadge = " <span class='label label-default'>Learner</span>";
                }
                var fullName = (user.first_name || '') + (user.last_name ? ' ' + user.last_name : '')
                userRows.push([
                    escapeHtml(fullName),
                    escapeHtml(user.email || ''),
                    escapeHtml(user.position || ''),
                    escapeHtml(user.role.name) + roleBadge,
                    lastlogin,
                    "<div class='pull-right'>\
                    <button class='btn btn-warning impersonate_button' data-user-id='" + user.id + "'>\
                    <i class='fa fa-retweet'></i>\
                    </button>\
                    <button class='btn btn-primary edit_button' data-toggle='modal' data-backdrop='static' data-target='#modal' data-user-id='" + user.id + "'>\
                    <i class='fa fa-pencil'></i>\
                    </button>\
                    <button class='btn btn-danger delete_button' data-user-id='" + user.id + "'>\
                    <i class='fa fa-trash-o'></i>\
                    </button></div>"
                ])
            })
            userTable.rows.add(userRows).draw();
        })
        .error(() => {
            errorFlash("Error fetching users")
        })
}

$(document).ready(function () {
    load()
    // Setup the event listeners
    $("#modal").on("hide.bs.modal", function () {
        dismiss();
    });
    // Update role description when role changes
    $("#role").on("change", function () {
        var selectedRole = $(this).val();
        var desc = roleDescriptions[selectedRole] || "Select a role to see its permissions.";
        $("#role-desc-text").html("<small class='text-muted'>" + desc + "</small>");
    })
    $("#new_button").on("click", function () {
        edit(-1)
    })
    $("#userTable").on('click', '.edit_button', function (e) {
        edit($(this).attr('data-user-id'))
    })
    $("#userTable").on('click', '.delete_button', function (e) {
        deleteUser($(this).attr('data-user-id'))
    })
    $("#userTable").on('click', '.impersonate_button', function (e) {
        impersonate($(this).attr('data-user-id'))
    })
});

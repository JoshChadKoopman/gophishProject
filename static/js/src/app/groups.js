var groups = []
var dashboardUsers = []
var targets = null

var roleDescriptions = {
    admin: "Full access — Can manage all objects, users, webhooks, and system settings.",
    user: "Standard user — Can create and manage campaigns, templates, groups, landing pages, and sending profiles.",
    contributor: "Contributor — Can modify email templates, landing pages, sending profiles, campaigns, and groups. Cannot manage users or account settings.",
    reader: "Reader — Read-only access to the dashboard and campaign results. Cannot create, edit, or delete any objects."
}

// ============================================================
//  GROUP FUNCTIONS
// ============================================================

function saveGroup(id) {
    var targetRows = []
    $.each($("#targetsTable").DataTable().rows().data(), function (i, target) {
        // Columns: Name | Email | Position | Delete
        var nameParts = unescapeHtml(target[0]).split(' ')
        targetRows.push({
            first_name: nameParts[0] || '',
            last_name: nameParts.slice(1).join(' ') || '',
            email: unescapeHtml(target[1]),
            position: unescapeHtml(target[2])
        })
    })
    var group = {
        name: $("#groupName").val(),
        targets: targetRows
    }
    if (id != -1) {
        group.id = id
        api.groupId.put(group)
            .success(function (data) {
                successFlash("Group updated successfully!")
                loadAll()
                dismissGroup()
                $("#groupModal").modal('hide')
            })
            .error(function (data) {
                groupModalError(data.responseJSON.message)
            })
    } else {
        api.groups.post(group)
            .success(function (data) {
                successFlash("Group added successfully!")
                loadAll()
                dismissGroup()
                $("#groupModal").modal('hide')
            })
            .error(function (data) {
                groupModalError(data.responseJSON.message)
            })
    }
}

function groupModalError(message) {
    $("#groupModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function dismissGroup() {
    $("#targetsTable").dataTable().DataTable().clear().draw()
    $("#groupName").val("")
    $("#userSelect").val("")
    $("#groupModal\\.flashes").empty()
}

// Populate the user select dropdown in the group modal with registered users (by email/name)
function populateUserSelect() {
    var $select = $("#userSelect")
    $select.empty()
    $select.append('<option value="">-- Select a registered user --</option>')
    api.users.get()
        .success(function (us) {
            dashboardUsers = us
            $.each(us, function (i, u) {
                var fullName = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '')
                var label = fullName ? fullName + ' (' + (u.email || u.username) + ')' : (u.email || u.username)
                $select.append($('<option>', {
                    value: u.id,
                    text: label
                }))
            })
        })
        .error(function () {
            $select.append('<option value="" disabled>Unable to load users</option>')
        })
}

// Find a dashboard user by id
function findUserById(id) {
    return dashboardUsers.find(function (u) { return u.id == id })
}

function editGroup(id) {
    targets = $("#targetsTable").dataTable({
        destroy: true,
        columnDefs: [{
            orderable: false,
            targets: "no-sort"
        }]
    })
    populateUserSelect()
    $("#groupModalSubmit").unbind('click').click(function () {
        saveGroup(id)
    })
    if (id == -1) {
        $("#groupModalLabel").text("New Group")
    } else {
        $("#groupModalLabel").text("Edit Group")
        api.groupId.get(id)
            .success(function (group) {
                $("#groupName").val(group.name)
                var targetData = []
                $.each(group.targets, function (i, record) {
                    var fullName = (record.first_name || '') + (record.last_name ? ' ' + record.last_name : '')
                    targetData.push([
                        escapeHtml(fullName),
                        escapeHtml(record.email),
                        escapeHtml(record.position),
                        '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'
                    ])
                })
                targets.DataTable().rows.add(targetData).draw()
            })
            .error(function () {
                errorFlash("Error fetching group")
            })
    }
    // CSV upload for group members — validates against registered users by email
    $("#csvupload").fileupload({
        url: "/api/import/group",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        },
        add: function (e, data) {
            $("#groupModal\\.flashes").empty()
            var acceptFileTypes = /(csv|txt)$/i;
            var filename = data.originalFiles[0]['name']
            if (filename && !acceptFileTypes.test(filename.split(".").pop())) {
                groupModalError("Unsupported file extension (use .csv or .txt)")
                return false;
            }
            data.submit();
        },
        done: function (e, data) {
            var skipped = []
            $.each(data.result, function (i, record) {
                // Match by email to a registered user
                var matchedUser = dashboardUsers.find(function (u) {
                    return u.email && u.email.toLowerCase() === record.email.toLowerCase()
                })
                if (matchedUser) {
                    addGroupTarget(
                        (matchedUser.first_name || '') + ' ' + (matchedUser.last_name || ''),
                        matchedUser.email,
                        matchedUser.position || record.position
                    )
                } else {
                    skipped.push(record.email || record.first_name || 'unknown')
                }
            });
            targets.DataTable().draw();
            if (skipped.length > 0) {
                groupModalError("Skipped " + skipped.length + " non-registered user(s): " + skipped.join(', '))
            }
        }
    })
}

var downloadCSVTemplate = function () {
    var csvScope = [{
        'First Name': 'Example',
        'Last Name': 'User',
        'Email': 'foobar@example.com',
        'Position': 'Systems Administrator'
    }]
    var filename = 'group_template.csv'
    var csvString = Papa.unparse(csvScope, {})
    var csvData = new Blob([csvString], {
        type: 'text/csv;charset=utf-8;'
    });
    if (navigator.msSaveBlob) {
        navigator.msSaveBlob(csvData, filename);
    } else {
        var csvURL = window.URL.createObjectURL(csvData);
        var dlLink = document.createElement('a');
        dlLink.href = csvURL;
        dlLink.setAttribute('download', filename)
        document.body.appendChild(dlLink)
        dlLink.click();
        document.body.removeChild(dlLink)
    }
}

var downloadUserCSVTemplate = function () {
    var csvScope = [{
        'Password': 'SecureP@ss1',
        'First Name': 'John',
        'Last Name': 'Doe',
        'Email': 'jdoe@example.com',
        'Position': 'Systems Administrator',
        'Role': 'user'
    }]
    var filename = 'user_import_template.csv'
    var csvString = Papa.unparse(csvScope, {})
    var csvData = new Blob([csvString], {
        type: 'text/csv;charset=utf-8;'
    });
    if (navigator.msSaveBlob) {
        navigator.msSaveBlob(csvData, filename);
    } else {
        var csvURL = window.URL.createObjectURL(csvData);
        var dlLink = document.createElement('a');
        dlLink.href = csvURL;
        dlLink.setAttribute('download', filename)
        document.body.appendChild(dlLink)
        dlLink.click();
        document.body.removeChild(dlLink)
    }
}

var deleteGroup = function (id) {
    var group = groups.find(function (x) {
        return x.id === id
    })
    if (!group) {
        return
    }
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the group. This can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete " + escapeHtml(group.name),
        confirmButtonColor: "#E94560",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.groupId.delete(id)
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
                'Group Deleted!',
                'This group has been deleted!',
                'success'
            );
        }
        $('button:contains("OK")').on('click', function () {
            location.reload()
        })
    })
}

// Add a target row to the group targets table
// Columns: Name | Email | Position | Delete
function addGroupTarget(fullName, email, position) {
    var emailLower = escapeHtml(email).toLowerCase();
    var newRow = [
        escapeHtml(fullName),
        emailLower,
        escapeHtml(position),
        '<span style="cursor:pointer;"><i class="fa fa-trash-o"></i></span>'
    ];
    var targetsTable = targets.DataTable();
    // Deduplicate by email (column 1)
    var existingRowIndex = -1;
    targetsTable.column(1, { order: "index" }).data().each(function (val, idx) {
        if (val === emailLower) {
            existingRowIndex = idx;
        }
    });
    if (existingRowIndex >= 0) {
        targetsTable.row(existingRowIndex, { order: "index" }).data(newRow);
    } else {
        targetsTable.row.add(newRow);
    }
}

// Build group action buttons
function groupActionButtons(groupId) {
    if (typeof permissions !== 'undefined' && permissions.modify_objects) {
        return "<div class='pull-right'><button class='btn btn-primary btn-sm' data-toggle='modal' data-backdrop='static' data-target='#groupModal' onclick='editGroup(" + groupId + ")'>\
            <i class='fa fa-pencil'></i>\
            </button>\
            <button class='btn btn-danger btn-sm' onclick='deleteGroup(" + groupId + ")'>\
            <i class='fa fa-trash-o'></i>\
            </button></div>";
    }
    return "";
}

// Build group rows for DataTables
function buildGroupRows(groupList) {
    var rows = []
    $.each(groupList, function (i, group) {
        rows.push([
            escapeHtml(group.name),
            escapeHtml(group.num_targets),
            moment(group.modified_date).format('MMMM Do YYYY, h:mm:ss a'),
            groupActionButtons(group.id)
        ])
    })
    return rows
}

// Load groups into the Groups tab
function loadGroups() {
    $("#groupTable").hide()
    $("#groupEmptyMessage").hide()
    $("#groupLoading").show()
    api.groups.summary()
        .success(function (response) {
            $("#groupLoading").hide()
            if (response.total > 0) {
                groups = response.groups
                $("#groupEmptyMessage").hide()
                $("#groupTable").show()
                var groupTable = $("#groupTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                groupTable.clear();
                groupTable.rows.add(buildGroupRows(groups)).draw()
            } else {
                $("#groupEmptyMessage").show()
            }
        })
        .error(function () {
            $("#groupLoading").hide()
            errorFlash("Error fetching groups")
        })
}

// ============================================================
//  DASHBOARD USER FUNCTIONS
// ============================================================

function userModalError(message) {
    $("#userModal\\.flashes").empty().append("<div style=\"text-align:center\" class=\"alert alert-danger\">\
        <i class=\"fa fa-exclamation-circle\"></i> " + message + "</div>")
}

function saveUser(id) {
    if ($("#userPassword").val() !== $("#userConfirmPassword").val()) {
        userModalError("Passwords must match.")
        return
    }
    // Validate required fields
    var firstName = $("#userFirstName").val().trim()
    var lastName = $("#userLastName").val().trim()
    var emailVal = $("#userEmail").val().trim()
    var positionVal = $("#userPosition").val().trim()

    if (!firstName) { userModalError("First name is required."); return }
    if (!lastName) { userModalError("Surname is required."); return }
    if (!emailVal) { userModalError("Email is required."); return }
    if (!positionVal) { userModalError("Position is required."); return }

    var userData = {
        username: emailVal,
        password: $("#userPassword").val(),
        first_name: firstName,
        last_name: lastName,
        email: emailVal,
        position: positionVal,
        role: $("#role").val(),
        password_change_required: $("#force_password_change_checkbox").prop('checked'),
        account_locked: $("#account_locked_checkbox").prop('checked')
    }
    if (id != -1) {
        userData.id = id
        api.userId.put(userData)
            .success(function (data) {
                successFlash("User " + escapeHtml(firstName + " " + lastName) + " updated successfully!")
                loadAll()
                dismissUser()
                $("#userModal").modal('hide')
            })
            .error(function (data) {
                userModalError(data.responseJSON.message)
            })
    } else {
        api.users.post(userData)
            .success(function (data) {
                successFlash("User " + escapeHtml(firstName + " " + lastName) + " registered successfully!")
                loadAll()
                dismissUser()
                $("#userModal").modal('hide')
            })
            .error(function (data) {
                userModalError(data.responseJSON.message)
            })
    }
}

function dismissUser() {
    $("#userFirstName").val("")
    $("#userLastName").val("")
    $("#userEmail").val("")
    $("#userPosition").val("")
    $("#userPassword").val("")
    $("#userConfirmPassword").val("")
    $("#role").val("")
    $("#force_password_change_checkbox").prop('checked', true)
    $("#account_locked_checkbox").prop('checked', false)
    $("#userModal\\.flashes").empty()
}

function editUser(id) {
    populateRoleDropdown()
    $("#userModalSubmit").unbind('click').click(function () {
        saveUser(id)
    })
    if (id == -1) {
        $("#userModalLabel").text("New User")
        $("#role").val("user")
        $("#role").trigger("change")
    } else {
        $("#userModalLabel").text("Edit User")
        api.userId.get(id)
            .success(function (u) {
                $("#userFirstName").val(u.first_name || '')
                $("#userLastName").val(u.last_name || '')
                $("#userEmail").val(u.email || '')
                $("#userPosition").val(u.position || '')
                $("#role").val(u.role.slug)
                $("#role").trigger("change")
                $("#force_password_change_checkbox").prop('checked', u.password_change_required)
                $("#account_locked_checkbox").prop('checked', u.account_locked)
            })
            .error(function () {
                errorFlash("Error fetching user")
            })
    }
}

function populateRoleDropdown() {
    api.roles.get()
        .success(function (roles) {
            var $select = $("#role");
            var currentVal = $select.val();
            $select.empty();
            $.each(roles, function (i, role) {
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
        .error(function () {
            var $select = $("#role");
            $select.empty();
            $select.append('<option value="admin">Admin</option>');
            $select.append('<option value="user">User</option>');
            $select.append('<option value="contributor">Contributor</option>');
            $select.append('<option value="reader">Reader</option>');
        })
}

var deleteUser = function (id) {
    var u = dashboardUsers.find(function (x) { return x.id == id })
    if (!u) {
        return
    }
    var displayName = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '') || u.email || u.username
    if (u.username == "admin") {
        Swal.fire({
            title: "Unable to Delete User",
            text: "The admin account cannot be deleted.",
            type: "info"
        });
        return
    }
    Swal.fire({
        title: "Are you sure?",
        text: "This will delete the account for " + escapeHtml(displayName) + " as well as all of the objects they have created.\n\nThis can't be undone!",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete",
        confirmButtonColor: "#E94560",
        reverseButtons: true,
        allowOutsideClick: false,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                api.userId.delete(id)
                    .success(function (msg) {
                        resolve()
                    })
                    .error(function (data) {
                        reject(data.responseJSON.message)
                    })
            }).catch(function (error) {
                Swal.showValidationMessage(error)
            })
        }
    }).then(function (result) {
        if (result.value) {
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

var impersonateUser = function (id) {
    var u = dashboardUsers.find(function (x) { return x.id == id })
    if (!u) {
        return
    }
    var displayName = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '') || u.email || u.username
    Swal.fire({
        title: "Are you sure?",
        html: "You will be logged out of your account and logged in as <strong>" + escapeHtml(displayName) + "</strong>",
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Swap User",
        confirmButtonColor: "#E94560",
        reverseButtons: true,
        allowOutsideClick: false,
    }).then(function (result) {
        if (result.value) {
            fetch('/impersonate', {
                method: 'post',
                body: "username=" + u.username + "&csrf_token=" + encodeURIComponent(csrf_token),
                headers: {
                    'Content-Type': 'application/x-www-form-urlencoded',
                },
            }).then(function (response) {
                if (response.status == 200) {
                    Swal.fire({
                        title: "Success!",
                        html: "Successfully changed to user <strong>" + escapeHtml(displayName) + "</strong>.",
                        type: "success",
                        showCancelButton: false,
                        confirmButtonText: "Home",
                        allowOutsideClick: false,
                    }).then(function (result) {
                        if (result.value) {
                            window.location.href = "/"
                        }
                    });
                } else {
                    Swal.fire({
                        title: "Error!",
                        type: "error",
                        html: "Failed to change to user <strong>" + escapeHtml(displayName) + "</strong>.",
                        showCancelButton: false,
                    })
                }
            })
        }
    })
}

// Build user action buttons
function userActionButtons(userId) {
    return "<div class='pull-right'>\
        <button class='btn btn-warning btn-sm impersonate_button' data-user-id='" + userId + "' title='Impersonate'>\
        <i class='fa fa-retweet'></i>\
        </button>\
        <button class='btn btn-primary btn-sm edit_user_button' data-toggle='modal' data-backdrop='static' data-target='#userModal' data-user-id='" + userId + "' title='Edit'>\
        <i class='fa fa-pencil'></i>\
        </button>\
        <button class='btn btn-danger btn-sm delete_user_button' data-user-id='" + userId + "' title='Delete'>\
        <i class='fa fa-trash-o'></i>\
        </button></div>"
}

// Build user rows for DataTables — Name, Email, Position, Role, Last Login, Actions
function buildUserRows(userList) {
    var rows = []
    $.each(userList, function (i, u) {
        var lastlogin = ""
        if (u.last_login != "0001-01-01T00:00:00Z") {
            lastlogin = moment(u.last_login).format('MMMM Do YYYY, h:mm:ss a')
        }
        var roleBadge = "";
        if (u.role.slug === "reader") {
            roleBadge = " <span class='label label-info'>Read Only</span>";
        } else if (u.role.slug === "contributor") {
            roleBadge = " <span class='label label-warning'>Contributor</span>";
        } else if (u.role.slug === "admin") {
            roleBadge = " <span class='label label-danger'>Admin</span>";
        } else {
            roleBadge = " <span class='label label-default'>User</span>";
        }
        var fullName = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '')
        rows.push([
            escapeHtml(fullName),
            escapeHtml(u.email || ''),
            escapeHtml(u.position || ''),
            escapeHtml(u.role.name) + roleBadge,
            lastlogin,
            userActionButtons(u.id)
        ])
    })
    return rows
}

// Build user rows for the All tab (no Last Login column)
function buildAllTabUserRows(userList) {
    var rows = []
    $.each(userList, function (i, u) {
        var roleBadge = "";
        if (u.role.slug === "reader") {
            roleBadge = " <span class='label label-info'>Read Only</span>";
        } else if (u.role.slug === "contributor") {
            roleBadge = " <span class='label label-warning'>Contributor</span>";
        } else if (u.role.slug === "admin") {
            roleBadge = " <span class='label label-danger'>Admin</span>";
        } else {
            roleBadge = " <span class='label label-default'>User</span>";
        }
        var fullName = (u.first_name || '') + (u.last_name ? ' ' + u.last_name : '')
        rows.push([
            escapeHtml(fullName),
            escapeHtml(u.email || ''),
            escapeHtml(u.position || ''),
            escapeHtml(u.role.name) + roleBadge,
            userActionButtons(u.id)
        ])
    })
    return rows
}

// Load dashboard users into the Users tab
function loadDashboardUsers() {
    if (typeof permissions === 'undefined' || !permissions.modify_system) {
        return
    }
    $("#dashboardUserTable").hide()
    $("#userLoading").show()
    api.users.get()
        .success(function (us) {
            dashboardUsers = us
            $("#userLoading").hide()
            $("#dashboardUserTable").show()
            var userTable = $("#dashboardUserTable").DataTable({
                destroy: true,
                columnDefs: [{
                    orderable: false,
                    targets: "no-sort"
                }]
            });
            userTable.clear();
            userTable.rows.add(buildUserRows(dashboardUsers)).draw();
        })
        .error(function () {
            $("#userLoading").hide()
            errorFlash("Error fetching dashboard users")
        })
}

// ============================================================
//  BULK USER CSV IMPORT
// ============================================================

function handleUserCSVImport(fileInput) {
    var file = fileInput.files[0]
    if (!file) return

    var acceptFileTypes = /(csv|txt)$/i;
    if (!acceptFileTypes.test(file.name.split(".").pop())) {
        errorFlash("Unsupported file extension (use .csv or .txt)")
        fileInput.value = ''
        return
    }

    var reader = new FileReader()
    reader.onload = function (e) {
        var results = Papa.parse(e.target.result, { header: true, skipEmptyLines: true })
        if (results.errors.length > 0) {
            errorFlash("Error parsing CSV: " + results.errors[0].message)
            fileInput.value = ''
            return
        }

        var rows = results.data
        if (rows.length === 0) {
            errorFlash("CSV file is empty")
            fileInput.value = ''
            return
        }

        var created = 0
        var failed = []
        var total = rows.length
        var processed = 0

        $.each(rows, function (i, row) {
            var normalized = {}
            $.each(row, function (key, val) {
                normalized[key.toLowerCase().replace(/[\s_-]+/g, '_')] = (val || '').trim()
            })

            var password = normalized['password'] || ''
            var firstName = normalized['first_name'] || normalized['firstname'] || ''
            var lastName = normalized['last_name'] || normalized['lastname'] || normalized['surname'] || ''
            var email = normalized['email'] || ''
            var position = normalized['position'] || ''
            var role = normalized['role'] || 'user'

            if (!firstName || !lastName || !email || !position) {
                failed.push((email || 'row ' + (i + 1)) + ' (missing required fields: first name, surname, email, position)')
                processed++
                checkDone()
                return
            }
            if (!password) {
                failed.push(email + ' (missing password)')
                processed++
                checkDone()
                return
            }

            var userData = {
                username: email,
                password: password,
                first_name: firstName,
                last_name: lastName,
                email: email,
                position: position,
                role: role,
                password_change_required: true,
                account_locked: false
            }

            api.users.post(userData)
                .success(function () {
                    created++
                    processed++
                    checkDone()
                })
                .error(function (data) {
                    var msg = data.responseJSON ? data.responseJSON.message : 'unknown error'
                    failed.push(email + ' (' + msg + ')')
                    processed++
                    checkDone()
                })
        })

        function checkDone() {
            if (processed >= total) {
                if (created > 0) {
                    successFlash("Successfully imported " + created + " user(s).")
                }
                if (failed.length > 0) {
                    errorFlash("Failed to import: " + failed.join(', '))
                }
                loadAll()
                fileInput.value = ''
            }
        }
    }
    reader.readAsText(file)
}

// ============================================================
//  ALL TAB — loads both groups and users
// ============================================================

function loadAll() {
    loadGroups()
    loadDashboardUsers()

    $("#allGroupTable").hide()
    $("#allUserTable").hide()
    $("#allEmptyMessage").hide()
    $("#allLoading").show()

    api.groups.summary()
        .success(function (response) {
            groups = response.groups || []
            if (groups.length > 0) {
                $("#allGroupTable").show()
                var allGroupTable = $("#allGroupTable").DataTable({
                    destroy: true,
                    columnDefs: [{
                        orderable: false,
                        targets: "no-sort"
                    }]
                });
                allGroupTable.clear();
                allGroupTable.rows.add(buildGroupRows(groups)).draw()
            }
            if (typeof permissions !== 'undefined' && permissions.modify_system) {
                api.users.get()
                    .success(function (us) {
                        dashboardUsers = us
                        $("#allLoading").hide()
                        if (dashboardUsers.length > 0) {
                            $("#allUserTable").show()
                            var allUserTable = $("#allUserTable").DataTable({
                                destroy: true,
                                columnDefs: [{
                                    orderable: false,
                                    targets: "no-sort"
                                }]
                            });
                            allUserTable.clear();
                            allUserTable.rows.add(buildAllTabUserRows(dashboardUsers)).draw()
                        }
                        if (groups.length === 0 && dashboardUsers.length === 0) {
                            $("#allEmptyMessage").show()
                        }
                    })
                    .error(function () {
                        $("#allLoading").hide()
                        errorFlash("Error fetching users")
                    })
            } else {
                $("#allLoading").hide()
                if (groups.length === 0) {
                    $("#allEmptyMessage").show()
                }
            }
        })
        .error(function () {
            $("#allLoading").hide()
            errorFlash("Error fetching groups")
        })
}

// ============================================================
//  INIT
// ============================================================

$(document).ready(function () {
    loadAll()

    // --- GROUP EVENT LISTENERS ---
    $("#addUserToGroupBtn").on("click", function () {
        var selectedId = $("#userSelect").val()
        if (!selectedId) {
            groupModalError("Please select a registered user to add.")
            return
        }
        var u = findUserById(selectedId)
        if (!u) {
            groupModalError("User not found. Only registered users can be added to groups.")
            return
        }
        addGroupTarget(
            (u.first_name || '') + ' ' + (u.last_name || ''),
            u.email || '',
            u.position || ''
        )
        targets.DataTable().draw()
        $("#userSelect").val("")
    })

    $("#targetsTable").on("click", "span>i.fa-trash-o", function () {
        targets.DataTable()
            .row($(this).parents('tr'))
            .remove()
            .draw();
    });
    $("#groupModal").on("hide.bs.modal", function () {
        dismissGroup();
    });
    $("#csv-template").click(downloadCSVTemplate)

    // --- USER EVENT LISTENERS ---
    $("#new_user_button, #new_user_button_all").on("click", function () {
        editUser(-1)
    })
    $("#userModal").on("hide.bs.modal", function () {
        dismissUser();
    });

    $("#dashboardUserTable").on('click', '.edit_user_button', function (e) {
        editUser($(this).attr('data-user-id'))
    })
    $("#dashboardUserTable").on('click', '.delete_user_button', function (e) {
        deleteUser($(this).attr('data-user-id'))
    })
    $("#dashboardUserTable").on('click', '.impersonate_button', function (e) {
        impersonateUser($(this).attr('data-user-id'))
    })

    $("#allUserTable").on('click', '.edit_user_button', function (e) {
        editUser($(this).attr('data-user-id'))
    })
    $("#allUserTable").on('click', '.delete_user_button', function (e) {
        deleteUser($(this).attr('data-user-id'))
    })
    $("#allUserTable").on('click', '.impersonate_button', function (e) {
        impersonateUser($(this).attr('data-user-id'))
    })

    // Role description update
    $("#role").on("change", function () {
        var selectedRole = $(this).val();
        var desc = roleDescriptions[selectedRole] || "Select a role to see its permissions.";
        $("#role-desc-text").html("<small class='text-muted'>" + desc + "</small>");
    })

    // Bulk user CSV import
    $("#userCsvUpload").on("change", function () {
        handleUserCSVImport(this)
    })
    $("#user-csv-template").on("click", downloadUserCSVTemplate)
});

$(document).ready(function () {
    loadLeaderboard();

    $("#filterDepartment").change(function () {
        loadLeaderboard();
    });
});

function loadLeaderboard() {
    var department = $("#filterDepartment").val() || "";

    // Load leaderboard and my position in parallel
    $.when(
        api.gamification.getLeaderboard("all_time", department),
        api.gamification.myPosition("all_time")
    ).done(function (lbResp, posResp) {
        var entries = lbResp[0] || lbResp;
        var myPos = posResp[0] || posResp;

        $("#loading").hide();
        $("#leaderboardContent").show();

        // Update my position
        if (myPos && myPos.rank) {
            $("#myRank").text("#" + myPos.rank);
            $("#myScore").text(myPos.score);
            $("#myBadges").text(myPos.badge_count || 0);
        }

        renderLeaderboard(entries);
        populateDepartments(entries);
    }).fail(function () {
        $("#loading").hide();
        $("#leaderboardContent").show();
        // Try loading just the leaderboard without position
        api.gamification.getLeaderboard("all_time", department).done(function (entries) {
            renderLeaderboard(entries);
            populateDepartments(entries);
        });
    });
}

function renderLeaderboard(entries) {
    var tbody = $("#leaderboardBody");
    tbody.empty();

    if (!entries || entries.length === 0) {
        $("#leaderboardTable").hide();
        $("#noLeaderboard").show();
        return;
    }
    $("#leaderboardTable").show();
    $("#noLeaderboard").hide();

    $.each(entries, function (i, e) {
        var rankDisplay = e.rank;
        if (e.rank === 1) rankDisplay = '<span style="color:gold;font-size:18px;"><i class="fa fa-trophy"></i></span> 1';
        else if (e.rank === 2) rankDisplay = '<span style="color:silver;font-size:16px;"><i class="fa fa-trophy"></i></span> 2';
        else if (e.rank === 3) rankDisplay = '<span style="color:#cd7f32;font-size:14px;"><i class="fa fa-trophy"></i></span> 3';

        tbody.append(
            '<tr' + (e.rank <= 3 ? ' style="font-weight:bold;"' : '') + '>' +
            '<td>' + rankDisplay + '</td>' +
            '<td>' + escapeHtml(e.user_name || '') + '</td>' +
            '<td>' + escapeHtml(e.user_email || '') + '</td>' +
            '<td>' + e.score + '</td>' +
            '<td>' + (e.badge_count || 0) + ' <i class="fa fa-star" style="color:#f0ad4e;"></i></td>' +
            '</tr>'
        );
    });
}

function populateDepartments(entries) {
    var select = $("#filterDepartment");
    var currentVal = select.val();
    var depts = {};
    $.each(entries, function (i, e) {
        if (e.department) depts[e.department] = true;
    });
    // Only populate if not already done
    if (select.find("option").length <= 1) {
        $.each(Object.keys(depts).sort(), function (i, d) {
            select.append('<option value="' + escapeHtml(d) + '">' + escapeHtml(d) + '</option>');
        });
        if (currentVal) select.val(currentVal);
    }
}

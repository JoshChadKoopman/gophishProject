let presentations = [];
let currentCourseTP = null;
let currentCoursePage = 0;
let coursePages = [];
let courseProgressMap = {}; // { presentationId: { status, current_page, total_pages, progress_pct } }
let currentQuiz = null; // quiz loaded for course viewer
let editQuizExisted = false; // whether a quiz existed before editing
let contentLibraryData = []; // cached content library items
let selectedSatRating = 0; // satisfaction rating selected

// ---- Configurable praise messages (loaded from API) ----
let praiseMessages = {
    course_complete: { heading: 'Course Complete!', body: 'Congratulations! You finished <strong>{{.CourseName}}</strong>', button_text: 'Awesome!', icon: '⭐', color_scheme: '#27ae60' },
    quiz_passed:     { heading: 'Quiz Passed!', body: 'Great work! You scored {{.Score}}/{{.Total}} on <strong>{{.CourseName}}</strong>', button_text: 'Well Done!', icon: '🏆', color_scheme: '#f39c12' },
    cert_earned:     { heading: 'Certificate Earned!', body: 'You\'ve earned the <strong>{{.CertName}}</strong> certificate. Keep up the great work!', button_text: 'View Certificate', icon: '🎓', color_scheme: '#2c3e50' },
    tier_complete:   { heading: 'Tier Completed!', body: 'Outstanding! You\'ve completed the <strong>{{.TierName}}</strong> tier.', button_text: 'Continue Learning', icon: '🏅', color_scheme: '#8e44ad' },
};

const loadPraiseMessages = () => {
    if (typeof api !== 'undefined' && api.praiseMessages) {
        api.praiseMessages.get()
            .done(function (msgs) {
                if (msgs && Array.isArray(msgs)) {
                    msgs.forEach(function (m) {
                        if (m.event_type && m.is_active !== false) {
                            praiseMessages[m.event_type] = m;
                        }
                    });
                }
            });
    }
};

const renderPraiseBody = (template, vars) => {
    let result = template || '';
    if (vars) {
        Object.keys(vars).forEach(function (key) {
            result = result.replace(new RegExp('\\{\\{\\.' + key + '\\}\\}', 'g'), vars[key]);
        });
    }
    return result;
};

// ---- Anti-skip protection state ----
let antiSkipPolicy = null;       // current policy for the open course
let pageDwellStart = null;       // timestamp when current page was entered
let pageDwellTimer = null;       // interval that updates the countdown
let pageUnlocked = false;        // whether the Next button is unlocked for current page
let dwellCountdownEl = null;     // reference to the countdown display element

const formatFileSize = (bytes) => {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
};

const getFileIcon = (contentType) => {
    if (contentType && contentType.includes('pdf')) return 'fa-file-pdf-o';
    if (contentType && (contentType.includes('powerpoint') || contentType.includes('presentation'))) return 'fa-file-powerpoint-o';
    if (contentType && contentType.includes('video')) return 'fa-file-video-o';
    return 'fa-file-o';
};

const getThumbClass = (contentType) => {
    if (contentType && contentType.includes('pdf')) return 'type-pdf';
    if (contentType && (contentType.includes('powerpoint') || contentType.includes('presentation'))) return 'type-ppt';
    if (contentType && contentType.includes('video')) return 'type-video';
    return 'type-default';
};

const getTypeLabel = (contentType) => {
    if (contentType && contentType.includes('pdf')) return 'PDF';
    if (contentType && contentType.includes('powerpoint')) return 'PowerPoint';
    if (contentType && contentType.includes('presentation')) return 'Presentation';
    if (contentType && contentType.includes('video')) return 'Video';
    return 'File';
};

// Build a thumbnail URL that includes the api_key for authentication
const thumbUrl = (tpId) => {
    return '/api/training/' + tpId + '/thumbnail?api_key=' + encodeURIComponent(user.api_key);
};

// Extract YouTube embed ID from various URL formats
const extractYouTubeId = (url) => {
    if (!url) return null;
    let match = url.match(/(?:youtube\.com\/(?:watch\?v=|embed\/|v\/)|youtu\.be\/)([\w-]{11})/);
    return match ? match[1] : null;
};

// ---- Page entry helpers (for upload & edit modals) ----
const createPageEntryHtml = (prefix, index, title, body, mediaType, mediaUrl) => {
    title = title || '';
    body = body || '';
    mediaType = mediaType || '';
    mediaUrl = mediaUrl || '';
    return `<div class="page-entry" data-page-index="${index}">
        <div class="page-entry-header" style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
            <span class="page-number-label" style="font-size:12px; font-weight:600; color:#888;">Page ${index + 1}</span>
            <button type="button" class="btn btn-xs btn-danger remove-page-btn" title="Remove this page"><i class="fa fa-times"></i> Remove</button>
        </div>
        <input type="text" class="form-control page-title-input" placeholder="Page title (e.g. Introduction)" value="${escapeHtml(title)}" style="margin-bottom:6px;" />
        <textarea class="form-control page-body-input" rows="3" placeholder="Page content...">${escapeHtml(body)}</textarea>
        <div class="page-media-row" style="margin-top:8px;">
            <label style="font-size:12px; font-weight:600; color:#555;">Page Media (optional):</label>
            <div style="display:flex; gap:8px; align-items:center; flex-wrap:wrap;">
                <select class="form-control page-media-type" style="width:140px;">
                    <option value=""${mediaType === '' ? ' selected' : ''}>None</option>
                    <option value="youtube"${mediaType === 'youtube' ? ' selected' : ''}>YouTube</option>
                    <option value="image"${mediaType === 'image' ? ' selected' : ''}>Image URL</option>
                    <option value="video"${mediaType === 'video' ? ' selected' : ''}>Video URL</option>
                </select>
                <input type="text" class="form-control page-media-url" placeholder="Paste URL here..." value="${escapeHtml(mediaUrl)}" style="flex:1; min-width:200px;${mediaType ? '' : ' display:none;'}" />
            </div>
        </div>
    </div>`;
};

const collectPages = (listSelector) => {
    let pages = [];
    $(listSelector).find('.page-entry').each(function () {
        let title = $(this).find('.page-title-input').val().trim();
        let body = $(this).find('.page-body-input').val().trim();
        let mediaType = $(this).find('.page-media-type').val() || '';
        let mediaUrl = $(this).find('.page-media-url').val().trim();
        if (title || body || (mediaType && mediaUrl)) {
            let page = { title: title, body: body };
            if (mediaType && mediaUrl) {
                page.media_type = mediaType;
                page.media_url = mediaUrl;
            }
            pages.push(page);
        }
    });
    return pages;
};

const renderPagesInList = (listSelector, pages) => {
    let container = $(listSelector);
    container.empty();
    if (pages && pages.length > 0) {
        pages.forEach((p, i) => {
            container.append(createPageEntryHtml('', i, p.title, p.body, p.media_type || '', p.media_url || ''));
        });
    }
};

const reindexPages = (listSelector) => {
    $(listSelector).find('.page-entry').each(function (i) {
        $(this).attr('data-page-index', i);
        $(this).find('.page-number-label').text('Page ' + (i + 1));
    });
};

// ---- Auto-extract slides from uploaded presentation ----
const autoExtractSlides = (mode) => {
    let listSelector, btn;
    if (mode === 'upload') {
        listSelector = '#uploadPagesList';
        btn = $('#autoExtractUpload');
        let fileInput = $('#presentationFile')[0];
        if (!fileInput.files || fileInput.files.length === 0) {
            modalError("Please select a file first to auto-extract slides.");
            return;
        }
        let formData = new FormData();
        formData.append("file", fileInput.files[0]);
        btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Extracting...');
        $.ajax({
            url: "/api/training/extract-slides",
            method: "POST",
            data: formData,
            processData: false,
            contentType: false,
            beforeSend: function (xhr) {
                xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
                xhr.setRequestHeader('X-CSRF-Token', csrf_token);
            }
        })
        .done(function (data) {
            if (data.pages && data.pages.length > 0) {
                data.pages.forEach(function (p) {
                    let idx = $(listSelector + ' .page-entry').length;
                    $(listSelector).append(createPageEntryHtml('', idx, p.title, p.body, p.media_type || '', p.media_url || ''));
                });
                successFlash('Extracted ' + data.pages.length + ' slide(s) from the presentation.');
            } else {
                modalError("No slides could be extracted from this file.");
            }
        })
        .fail(function (data) {
            let msg = "Error extracting slides";
            if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
            modalError(msg);
        })
        .always(function () {
            btn.prop("disabled", false).html('<i class="fa fa-magic"></i> Auto-Extract Slides');
        });
    } else if (mode === 'edit') {
        listSelector = '#editPagesList';
        btn = $('#autoExtractEdit');
        let tpId = $('#editId').val();
        if (!tpId) return;
        btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Extracting...');
        $.ajax({
            url: "/api/training/" + tpId + "/extract-slides",
            method: "POST",
            dataType: "json",
            contentType: "application/json",
            beforeSend: function (xhr) {
                xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
                xhr.setRequestHeader('X-CSRF-Token', csrf_token);
            }
        })
        .done(function (data) {
            if (data.pages && data.pages.length > 0) {
                data.pages.forEach(function (p) {
                    let idx = $(listSelector + ' .page-entry').length;
                    $(listSelector).append(createPageEntryHtml('', idx, p.title, p.body, p.media_type || '', p.media_url || ''));
                });
                successFlash('Extracted ' + data.pages.length + ' slide(s) from the presentation.');
            } else {
                modalError("No slides could be extracted from this file.");
            }
        })
        .fail(function (data) {
            let msg = "Error extracting slides";
            if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
            $("#editModal\\.flashes").empty().append(
                '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> ' + msg + '</div>'
            );
        })
        .always(function () {
            btn.prop("disabled", false).html('<i class="fa fa-magic"></i> Auto-Extract Slides');
        });
    }
};

// ---- Dismiss helpers ----
const dismissUpload = () => {
    $("#presentationName").val("");
    $("#presentationDescription").val("");
    $("#presentationFile").val("");
    $("#presentationThumbnail").val("");
    $("#presentationYouTube").val("");
    $("#uploadPagesList").empty();
    $("#thumbPreview").hide();
    $("#thumbPreviewImg").attr("src", "");
    $("#modal\\.flashes").empty();
};

const dismissEdit = () => {
    $("#editId").val("");
    $("#editName").val("");
    $("#editDescription").val("");
    $("#editYouTube").val("");
    $("#editPagesList").empty();
    $("#editModal\\.flashes").empty();
};

// ---- Progress helpers ----
const getProgressForTP = (tpId) => {
    return courseProgressMap[tpId] || { status: 'no_progress', current_page: 0, total_pages: 0, progress_pct: 0 };
};

const getStatusLabel = (status) => {
    if (status === 'complete') return 'Completed';
    if (status === 'in_progress') return 'In Progress';
    return 'Not Started';
};

const getStatusBadgeHtml = (status) => {
    if (status === 'complete') {
        return '<span class="label" style="font-size:11px; padding:4px 12px; background:#27ae60; color:#fff; border-radius:4px;"><i class="fa fa-check-circle"></i> Completed</span>';
    }
    if (status === 'in_progress') {
        return '<span class="label" style="font-size:11px; padding:4px 12px; background:#2980b9; color:#fff; border-radius:4px;"><i class="fa fa-spinner"></i> In Progress</span>';
    }
    return '<span class="label" style="font-size:11px; padding:4px 12px; background:#999; color:#fff; border-radius:4px;"><i class="fa fa-clock-o"></i> Not Started</span>';
};

const saveProgressToServer = (tpId, currentPage, totalPages, status) => {
    $.ajax({
        url: "/api/training/" + tpId + "/progress",
        method: "PUT",
        data: JSON.stringify({
            current_page: currentPage,
            total_pages: totalPages,
            status: status
        }),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .done(function (data) {
        let pct = totalPages > 0 ? Math.round((currentPage / totalPages) * 100) : 0;
        if (status === 'complete') pct = 100;
        courseProgressMap[tpId] = {
            status: data.status || status,
            current_page: data.current_page || currentPage,
            total_pages: data.total_pages || totalPages,
            progress_pct: pct
        };
    });
};

const loadAllProgress = (callback) => {
    $.ajax({
        url: "/api/training/my-courses",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    })
    .done(function (data) {
        courseProgressMap = {};
        if (data && Array.isArray(data) && data.length > 0) {
            data.forEach(function (item) {
                let pId = item.presentation ? item.presentation.id : null;
                if (pId) {
                    let pct = item.progress_pct || 0;
                    let status = 'no_progress';
                    let currentPage = 0;
                    let totalPages = 0;
                    if (item.progress) {
                        status = item.progress.status || 'no_progress';
                        currentPage = item.progress.current_page || 0;
                        totalPages = item.progress.total_pages || 0;
                    }
                    if (status === 'complete') pct = 100;
                    courseProgressMap[pId] = {
                        status: status,
                        current_page: currentPage,
                        total_pages: totalPages,
                        progress_pct: pct
                    };
                }
            });
        }
        if (callback) callback();
    })
    .fail(function () {
        if (callback) callback();
    });
};

// ---- Confetti + Gold Star ----
const showCompletionCelebration = (courseName) => {
    // Load configurable praise message for course completion
    let praise = praiseMessages.course_complete || {};
    let praiseIcon = praise.icon || '⭐';
    let praiseHeading = praise.heading || 'Course Complete!';
    let praiseBody = renderPraiseBody(praise.body || 'Congratulations! You finished <strong>{{.CourseName}}</strong>', { CourseName: escapeHtml(courseName) });
    let praiseButton = praise.button_text || 'Awesome!';
    let praiseColor = praise.color_scheme || '#27ae60';

    let overlay = $(`
        <div id="completionOverlay" style="
            position:fixed; top:0; left:0; width:100%; height:100%;
            z-index:100000; display:flex; align-items:center; justify-content:center;
            background:rgba(0,0,0,0.6);
        ">
            <div id="completionCard" style="
                background:#fff; border-radius:16px; padding:50px 60px; text-align:center;
                box-shadow:0 20px 60px rgba(0,0,0,0.3);
                max-width:480px; width:90%;
                animation: completionPop 0.5s ease;
            ">
                <div id="goldStar" style="font-size:80px; margin-bottom:16px;">${praiseIcon}</div>
                <h2 style="margin:0 0 8px 0; font-size:28px; font-weight:700; color:#2c3e50;">${escapeHtml(praiseHeading)}</h2>
                <p style="font-size:16px; color:#666; margin:0 0 24px 0;">${praiseBody}</p>
                <span class="label" style="font-size:14px; padding:8px 24px; background:${praiseColor}; color:#fff; border-radius:20px;">
                    <i class="fa fa-check-circle"></i> Completed
                </span>
                <br/><br/>
                <button id="closeCelebration" class="btn btn-primary btn-lg" style="margin-top:10px;">
                    <i class="fa fa-thumbs-up"></i> ${escapeHtml(praiseButton)}
                </button>
            </div>
        </div>
    `);
    $('body').append(overlay);

    // Inject keyframe animations if not already present
    if ($('#confettiAnimStyles').length === 0) {
        $('head').append(`<style id="confettiAnimStyles">
            @keyframes completionPop {
                0% { transform: scale(0.5); opacity:0; }
                60% { transform: scale(1.05); }
                100% { transform: scale(1); opacity:1; }
            }
            @keyframes confettiFall {
                0% { transform: translateY(-20px) rotate(0deg); opacity:1; }
                100% { transform: translateY(110vh) rotate(720deg); opacity:0; }
            }
            @keyframes starPulse {
                0%,100% { transform: scale(1); }
                50% { transform: scale(1.2); }
            }
            #goldStar { animation: starPulse 1s ease infinite; }
        </style>`);
    }

    // Launch confetti
    launchConfetti();

    // Close handler
    $('#closeCelebration').on('click', function () {
        $('#completionOverlay').fadeOut(300, function () { $(this).remove(); });
    });
    $('#completionOverlay').on('click', function (e) {
        if (e.target === this) {
            $(this).fadeOut(300, function () { $(this).remove(); });
        }
    });
};

const launchConfetti = () => {
    const colors = ['#e74c3c', '#3498db', '#2ecc71', '#f39c12', '#9b59b6', '#e67e22', '#1abc9c', '#e91e63', '#ff9800', '#00bcd4'];
    const container = document.getElementById('completionOverlay');
    if (!container) return;

    for (let i = 0; i < 120; i++) {
        let confetti = document.createElement('div');
        let size = Math.random() * 10 + 6;
        let isCircle = Math.random() > 0.5;
        confetti.style.cssText = `
            position:fixed;
            width:${size}px;
            height:${isCircle ? size : size * 0.5}px;
            background:${colors[Math.floor(Math.random() * colors.length)]};
            border-radius:${isCircle ? '50%' : '2px'};
            top:-20px;
            left:${Math.random() * 100}%;
            z-index:100001;
            pointer-events:none;
            opacity:1;
            animation: confettiFall ${2 + Math.random() * 3}s ease-out forwards;
            animation-delay: ${Math.random() * 0.8}s;
            transform: rotate(${Math.random() * 360}deg);
        `;
        container.appendChild(confetti);
    }
};

// ---- Detail modal ----
const showDetailModal = (tp) => {
    currentCourseTP = tp;
    let progress = getProgressForTP(tp.id);

    $("#detailTitle").text(tp.name);
    $("#detailDescription").text(tp.description || "No description provided.");

    let uploadDate = moment(tp.created_date).format('MMM D, YYYY');
    let typeLabel = getTypeLabel(tp.content_type);
    $("#detailMeta").html(
        '<i class="fa fa-tag"></i> ' + escapeHtml(typeLabel) + ' &nbsp;&middot;&nbsp; ' +
        '<i class="fa fa-calendar"></i> ' + uploadDate + ' &nbsp;&middot;&nbsp; ' +
        '<i class="fa fa-hdd-o"></i> ' + formatFileSize(tp.file_size)
    );

    let container = $("#detailThumbContainer");
    container.empty();
    if (tp.thumbnail_path) {
        container.html('<img src="' + thumbUrl(tp.id) + '" alt="' + escapeHtml(tp.name) + '" />');
    } else {
        let fileIcon = getFileIcon(tp.content_type);
        let thumbClass = getThumbClass(tp.content_type);
        container.html('<div class="detail-icon-placeholder ' + thumbClass + '"><i class="fa ' + fileIcon + '"></i></div>');
    }

    // Update status badge
    let badgeHtml = getStatusBadgeHtml(progress.status);
    $("#detailStatusBadge").replaceWith(
        $(badgeHtml).attr('id', 'detailStatusBadge')
    );

    // Update enrol button text based on status
    if (progress.status === 'complete') {
        $("#detailEnrollBtn").html('<i class="fa fa-refresh"></i> Take Again').removeClass('btn-success').addClass('btn-info');
    } else if (progress.status === 'in_progress') {
        $("#detailEnrollBtn").html('<i class="fa fa-play-circle"></i> Continue').removeClass('btn-info').addClass('btn-success');
    } else {
        $("#detailEnrollBtn").html('<i class="fa fa-play-circle"></i> Enrol Now').removeClass('btn-info').addClass('btn-success');
    }

    $("#detailModal").modal("show");
};

// ---- Course viewer ----
const openCourseViewer = (tp) => {
    currentCourseTP = tp;

    // Parse content pages
    try {
        coursePages = tp.content_pages ? JSON.parse(tp.content_pages) : [];
    } catch (e) {
        coursePages = [];
    }
    if (!Array.isArray(coursePages)) coursePages = [];

    // If there are no content pages, show a default page with the description
    if (coursePages.length === 0) {
        coursePages = [{
            title: tp.name,
            body: tp.description || "No additional content has been added to this training yet.\n\nWatch the video above to complete this module."
        }];
    }

    // Restore progress if resuming
    let progress = getProgressForTP(tp.id);
    if (progress.status === 'in_progress' && progress.current_page > 0 && progress.current_page < coursePages.length) {
        currentCoursePage = progress.current_page;
    } else {
        currentCoursePage = 0;
    }

    // Set title
    $("#courseViewerLabel").text(tp.name);

    // Setup video
    let videoId = extractYouTubeId(tp.youtube_url);
    if (videoId) {
        $("#courseVideoIframe").attr("src", "https://www.youtube.com/embed/" + videoId + "?rel=0");
        $("#courseVideoSection").show();
    } else {
        $("#courseVideoIframe").attr("src", "");
        $("#courseVideoSection").hide();
    }

    // Save start progress if not started or taking again
    if (progress.status !== 'in_progress') {
        saveProgressToServer(tp.id, 0, coursePages.length, 'in_progress');
    }

    // Load quiz for this course (async, non-blocking)
    loadQuizForViewer(tp.id, function () {});
    $("#courseQuizSection").hide();

    // Load anti-skip policy (async, then render page with protection)
    loadAntiSkipPolicy(tp.id, function () {
        renderCoursePage();
    });

    $("#detailModal").modal("hide");
    $("#courseViewerModal").modal("show");
};

const renderCoursePage = () => {
    if (coursePages.length === 0) return;

    let page = coursePages[currentCoursePage];
    let html = '';

    // ---- Per-page media ----
    if (page.media_type && page.media_url) {
        if (page.media_type === 'youtube') {
            let vid = extractYouTubeId(page.media_url);
            if (vid) {
                html += '<div class="page-media-embed" style="margin-bottom:20px; background:#000; border-radius:6px; overflow:hidden;">';
                html += '<div style="position:relative; padding-bottom:56.25%; height:0; overflow:hidden;">';
                html += '<iframe class="page-video-frame" src="https://www.youtube.com/embed/' + vid + '?rel=0" style="position:absolute; top:0; left:0; width:100%; height:100%; border:none;" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>';
                html += '</div></div>';
            }
        } else if (page.media_type === 'image') {
            html += '<div class="page-media-embed" style="margin-bottom:20px; text-align:center;">';
            html += '<img src="' + escapeHtml(page.media_url) + '" style="max-width:100%; max-height:500px; border-radius:6px; border:1px solid #ddd;" alt="Page media" />';
            html += '</div>';
        } else if (page.media_type === 'video') {
            html += '<div class="page-media-embed" style="margin-bottom:20px; background:#000; border-radius:6px; overflow:hidden;">';
            html += '<video controls style="width:100%; max-height:500px;" src="' + escapeHtml(page.media_url) + '">Your browser does not support the video tag.</video>';
            html += '</div>';
        }
    }

    if (page.title) {
        html += '<h2>' + escapeHtml(page.title) + '</h2>';
    }
    if (page.body) {
        html += '<div class="page-body">' + escapeHtml(page.body) + '</div>';
    }
    $("#coursePageContent").html(html);

    // Progress bar calculation
    let progressPct = 0;
    if (coursePages.length <= 1) {
        progressPct = 0;
    } else {
        progressPct = Math.round((currentCoursePage / (coursePages.length - 1)) * 100);
    }
    let progressColor = progressPct >= 100 ? '#27ae60' : (progressPct > 0 ? '#3498db' : '#ccc');
    $("#courseProgressBar").css({ width: progressPct + '%', background: progressColor });
    $("#courseProgressText").text(progressPct + '% complete  —  Page ' + (currentCoursePage + 1) + ' of ' + coursePages.length);

    // Page indicator
    $("#coursePageIndicator").text('Page ' + (currentCoursePage + 1) + ' of ' + coursePages.length);

    // Previous button
    if (currentCoursePage === 0) {
        $("#coursePrevBtn").prop("disabled", true).css("visibility", "hidden");
    } else {
        $("#coursePrevBtn").prop("disabled", false).css("visibility", "visible");
    }

    // Next button – on last page show "Finish"
    if (currentCoursePage >= coursePages.length - 1) {
        $("#courseNextBtn").html('<i class="fa fa-check"></i> Finish').removeClass("btn-primary").addClass("btn-success");
    } else {
        $("#courseNextBtn").html('Next <i class="fa fa-arrow-right"></i>').removeClass("btn-success").addClass("btn-primary");
    }

    // ---- Anti-skip protection ----
    applyAntiSkipProtection();

    // Save progress as user navigates (in_progress while not finished)
    if (currentCourseTP) {
        saveProgressToServer(currentCourseTP.id, currentCoursePage, coursePages.length, 'in_progress');
    }
};

const finishCourse = () => {
    if (!currentCourseTP) return;

    // If course has a quiz, show quiz section instead of completing
    if (currentQuiz && currentQuiz.questions && currentQuiz.questions.length > 0) {
        $("#courseContentSection").hide();
        $("#courseNavSection").hide();
        $("#courseQuizSection").show();
        renderQuizViewer();
        return;
    }

    completeCourse();
};

const completeCourse = () => {
    if (!currentCourseTP) return;

    // Save completion progress
    saveProgressToServer(currentCourseTP.id, coursePages.length, coursePages.length, 'complete');

    // Close viewer
    $("#courseVideoIframe").attr("src", "");
    $(".page-video-frame").attr("src", "");
    $("#courseViewerModal").modal("hide");

    // Update local progress map immediately
    courseProgressMap[currentCourseTP.id] = {
        status: 'complete',
        current_page: coursePages.length,
        total_pages: coursePages.length,
        progress_pct: 100
    };

    // Update the card status tag
    updateCardStatus(currentCourseTP.id, 'complete');

    // Show celebration, then satisfaction modal after dismiss
    let name = currentCourseTP.name;
    let tpId = currentCourseTP.id;
    setTimeout(() => {
        showCompletionCelebration(name);
        // When celebration is dismissed, show satisfaction rating modal
        $(document).one('click', '#closeCelebration', function () {
            setTimeout(() => { showSatisfactionModal(tpId, name); }, 500);
        });
    }, 400);
};

const updateCardStatus = (tpId, status) => {
    let card = $(`.training-card[data-training-id="${tpId}"]`);
    if (card.length === 0) return;

    // Remove any existing status badge on the card
    card.find('.card-status-badge').remove();

    let badgeHtml = '';
    if (status === 'complete') {
        badgeHtml = '<span class="card-status-badge" style="display:inline-block; font-size:10px; padding:3px 10px; background:#27ae60; color:#fff; border-radius:10px; font-weight:600; margin-top:4px;"><i class="fa fa-check-circle"></i> Completed</span>';
    } else if (status === 'in_progress') {
        badgeHtml = '<span class="card-status-badge" style="display:inline-block; font-size:10px; padding:3px 10px; background:#2980b9; color:#fff; border-radius:10px; font-weight:600; margin-top:4px;"><i class="fa fa-spinner"></i> In Progress</span>';
    }

    if (badgeHtml) {
        card.find('.card-meta').after(badgeHtml);
    }

    // Update progress bar to 100%
    if (status === 'complete') {
        card.find('.card-progress-bar div div').css({ width: '100%', background: '#27ae60' });
    }
};

// =====================================================================
// Anti-Skip Protection — Smart features prevent passive clicking
// =====================================================================

// Load anti-skip policy for a presentation
const loadAntiSkipPolicy = (presId, callback) => {
    $.ajax({
        url: "/api/training/" + presId + "/anti-skip-policy",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    })
    .done(function (data) {
        antiSkipPolicy = data;
        callback();
    })
    .fail(function () {
        // Default fallback
        antiSkipPolicy = { min_dwell_seconds: 10, require_acknowledge: false, require_scroll: false, enforce_sequential: true };
        callback();
    });
};

// Apply anti-skip protection to the current page
const applyAntiSkipProtection = () => {
    // Clear previous timer
    if (pageDwellTimer) {
        clearInterval(pageDwellTimer);
        pageDwellTimer = null;
    }
    pageDwellStart = Date.now();
    pageUnlocked = false;

    // Remove previous anti-skip UI
    $("#antiSkipBar").remove();

    if (!antiSkipPolicy || antiSkipPolicy.min_dwell_seconds <= 0) {
        pageUnlocked = true;
        $("#courseNextBtn").prop("disabled", false).css("opacity", 1);
        return;
    }

    let minDwell = antiSkipPolicy.min_dwell_seconds;
    let needAck = antiSkipPolicy.require_acknowledge;

    // Lock the Next button initially
    $("#courseNextBtn").prop("disabled", true).css("opacity", 0.5);

    // Build anti-skip bar
    let barHtml = '<div id="antiSkipBar" style="background:#fff8e1; border:1px solid #ffe082; border-radius:6px; padding:10px 16px; margin-top:12px; display:flex; align-items:center; gap:12px; flex-wrap:wrap;">';
    barHtml += '<i class="fa fa-hourglass-half" style="color:#f57c00; font-size:18px;" id="antiSkipIcon"></i>';
    barHtml += '<span id="antiSkipCountdown" style="font-weight:600; color:#e65100; min-width:180px;">Please read this page... <span id="dwellSecondsLeft">' + minDwell + '</span>s</span>';
    barHtml += '<div style="flex:1; background:#eee; border-radius:4px; height:6px; min-width:100px;"><div id="antiSkipProgressFill" style="width:0%; height:100%; background:#ff9800; border-radius:4px; transition:width 1s linear;"></div></div>';
    if (needAck) {
        barHtml += '<label id="antiSkipAckLabel" style="margin:0; font-size:13px; color:#555; display:none; cursor:pointer;"><input type="checkbox" id="antiSkipAckCheck" style="margin-right:6px;" /> I have read and understood this page</label>';
    }
    barHtml += '</div>';
    $("#coursePageContent").after(barHtml);

    // Start countdown timer
    let elapsed = 0;
    pageDwellTimer = setInterval(function () {
        elapsed++;
        let remaining = Math.max(0, minDwell - elapsed);
        let pct = Math.min(100, Math.round((elapsed / minDwell) * 100));
        $("#dwellSecondsLeft").text(remaining);
        $("#antiSkipProgressFill").css("width", pct + "%");

        if (remaining <= 0) {
            clearInterval(pageDwellTimer);
            pageDwellTimer = null;
            $("#antiSkipIcon").removeClass("fa-hourglass-half").addClass("fa-check-circle").css("color", "#43a047");
            $("#antiSkipProgressFill").css("background", "#43a047");

            if (needAck) {
                $("#antiSkipCountdown").html('<span style="color:#43a047;">Timer complete!</span> Please confirm below.');
                $("#antiSkipAckLabel").show();
            } else {
                $("#antiSkipCountdown").html('<span style="color:#43a047;"><i class="fa fa-check"></i> You may proceed</span>');
                unlockNextButton();
            }
        }
    }, 1000);

    // Acknowledge checkbox handler
    if (needAck) {
        $(document).off("change", "#antiSkipAckCheck").on("change", "#antiSkipAckCheck", function () {
            if ($(this).is(":checked")) {
                unlockNextButton();
                $("#antiSkipCountdown").html('<span style="color:#43a047;"><i class="fa fa-check"></i> Confirmed — you may proceed</span>');
            } else {
                pageUnlocked = false;
                $("#courseNextBtn").prop("disabled", true).css("opacity", 0.5);
            }
        });
    }
};

const unlockNextButton = () => {
    pageUnlocked = true;
    $("#courseNextBtn").prop("disabled", false).css("opacity", 1);
};

// Calculate dwell time since page was entered
const getCurrentDwell = () => {
    if (!pageDwellStart) return 0;
    return Math.floor((Date.now() - pageDwellStart) / 1000);
};

// Send engagement data for a page to the server
const sendPageEngagement = (pageIndex, callback) => {
    if (!currentCourseTP) { callback(); return; }
    let dwellSec = getCurrentDwell();
    let ackChecked = $("#antiSkipAckCheck").is(":checked");

    $.ajax({
        url: "/api/training/" + currentCourseTP.id + "/engage",
        method: "PUT",
        data: JSON.stringify({
            page_index: pageIndex,
            dwell_seconds: dwellSec,
            scroll_depth_pct: getScrollDepth(),
            acknowledged: ackChecked
        }),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .always(function () {
        callback();
    });
};

// Estimate scroll depth of the page content area
const getScrollDepth = () => {
    let content = document.getElementById('coursePageContent');
    if (!content) return 100;
    let parent = content.closest('.modal-body') || content.parentElement;
    if (!parent) return 100;
    let scrollTop = parent.scrollTop || 0;
    let scrollHeight = parent.scrollHeight || 1;
    let clientHeight = parent.clientHeight || 1;
    if (scrollHeight <= clientHeight) return 100; // content fits without scrolling
    let depth = Math.round(((scrollTop + clientHeight) / scrollHeight) * 100);
    return Math.min(100, Math.max(0, depth));
};

// Server-side validation before advancing to the next page
const validateAndAdvancePage = () => {
    if (!currentCourseTP) return;
    $.ajax({
        url: "/api/training/" + currentCourseTP.id + "/validate-advance",
        method: "POST",
        data: JSON.stringify({
            current_page: currentCoursePage,
            next_page: currentCoursePage + 1,
            total_pages: coursePages.length
        }),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .done(function (result) {
        if (result.allowed) {
            currentCoursePage++;
            renderCoursePage();
            $("#courseContentSection")[0].scrollIntoView({ behavior: 'smooth' });
        } else {
            showAntiSkipWarning(result.reason || "Please complete this page before continuing.");
        }
    })
    .fail(function () {
        // On network failure, allow advance (graceful degradation)
        currentCoursePage++;
        renderCoursePage();
        $("#courseContentSection")[0].scrollIntoView({ behavior: 'smooth' });
    });
};

// Server-side validation before finishing a course
const validateAndFinishCourse = () => {
    if (!currentCourseTP) return;
    $.ajax({
        url: "/api/training/" + currentCourseTP.id + "/validate-complete",
        method: "POST",
        data: JSON.stringify({ total_pages: coursePages.length }),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .done(function (result) {
        if (result.allowed) {
            finishCourse();
        } else {
            let msg = result.reason || "You must complete all pages before finishing.";
            if (result.missing_pages && result.missing_pages.length > 0) {
                let pageNums = result.missing_pages.map(p => "Page " + (p + 1)).join(", ");
                msg += "\n\nIncomplete pages: " + pageNums;
            }
            showAntiSkipWarning(msg);
        }
    })
    .fail(function () {
        // On network failure, allow finish (graceful degradation)
        finishCourse();
    });
};

// Show a warning toast for anti-skip violations
const showAntiSkipWarning = (message) => {
    // Remove any existing warning
    $(".anti-skip-warning").remove();
    let html = '<div class="anti-skip-warning" style="position:fixed; top:20px; right:20px; z-index:99999; max-width:400px; padding:14px 20px; background:#fff3e0; border:2px solid #ff9800; border-radius:8px; box-shadow:0 4px 12px rgba(0,0,0,0.15); animation:fadeInRight 0.3s;">';
    html += '<div style="display:flex; align-items:flex-start; gap:10px;">';
    html += '<i class="fa fa-shield" style="color:#e65100; font-size:22px; margin-top:2px;"></i>';
    html += '<div>';
    html += '<strong style="color:#e65100;">Anti-Skip Protection</strong>';
    html += '<p style="margin:4px 0 0; color:#bf360c; font-size:13px; white-space:pre-line;">' + escapeHtml(message) + '</p>';
    html += '</div>';
    html += '<button onclick="$(this).closest(\'.anti-skip-warning\').fadeOut(200, function(){$(this).remove();})" style="background:none; border:none; color:#999; font-size:18px; cursor:pointer; padding:0; margin-left:8px;">&times;</button>';
    html += '</div></div>';
    $("body").append(html);

    // Auto-dismiss after 8 seconds
    setTimeout(function () {
        $(".anti-skip-warning").fadeOut(300, function () { $(this).remove(); });
    }, 8000);
};

// =====================================================================
// End of Anti-Skip Protection
// =====================================================================

// ---- Load presentations ----
const load = () => {
    $("#trainingList").hide();
    $("#emptyMessage").hide();
    $("#loading").show();

    loadAllProgress(() => {
        $.ajax({
            url: "/api/training/",
            method: "GET",
            dataType: "json",
            beforeSend: function (xhr) {
                xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            }
        })
        .done((tps) => {
            presentations = tps;
            $("#loading").hide();
            if (!presentations || presentations.length === 0) {
                $("#emptyMessage").show();
                $("#trainingList").hide();
                return;
            }
            $("#emptyMessage").hide();
            $("#trainingList").show();

            let grid = $("#trainingGrid");
            grid.empty();

            $.each(presentations, (i, tp) => {
                let fileIcon = getFileIcon(tp.content_type);
                let thumbClass = getThumbClass(tp.content_type);
                let typeLabel = getTypeLabel(tp.content_type);
                let uploadDate = moment(tp.created_date).format('MMM D, YYYY');
                let progress = getProgressForTP(tp.id);

                // Build thumbnail area
                let thumbContent;
                if (tp.thumbnail_path) {
                    thumbContent = '<img src="' + thumbUrl(tp.id) + '" alt="' + escapeHtml(tp.name) + '" />';
                } else {
                    thumbContent = '<i class="fa ' + fileIcon + '"></i>';
                }

                // Status badge on card
                let statusBadge = '';
                if (progress.status === 'complete') {
                    statusBadge = '<span class="card-status-badge" style="display:inline-block; font-size:10px; padding:3px 10px; background:#27ae60; color:#fff; border-radius:10px; font-weight:600; margin-top:4px;"><i class="fa fa-check-circle"></i> Completed</span>';
                } else if (progress.status === 'in_progress') {
                    statusBadge = '<span class="card-status-badge" style="display:inline-block; font-size:10px; padding:3px 10px; background:#2980b9; color:#fff; border-radius:10px; font-weight:600; margin-top:4px;"><i class="fa fa-spinner"></i> In Progress</span>';
                }

                // Progress bar on card
                let cardProgressPct = progress.progress_pct || 0;
                let cardProgressColor = progress.status === 'complete' ? '#27ae60' : (progress.status === 'in_progress' ? '#3498db' : '#ddd');

                let actions = `
                    <a class="btn btn-primary btn-sm" href="/api/training/${tp.id}/download?api_key=${encodeURIComponent(user.api_key)}" target="_blank" title="View / Download" onclick="event.stopPropagation();">
                        <i class="fa fa-download"></i>
                    </a>`;
                if (typeof permissions !== 'undefined' && permissions.manage_training) {
                    actions += `
                    <button class="btn btn-info btn-sm assign_button" data-training-id="${tp.id}" title="Assign Course" onclick="event.stopPropagation();">
                        <i class="fa fa-users"></i>
                    </button>`;
                }
                if (typeof permissions !== 'undefined' && permissions.modify_system) {
                    actions += `
                    <button class="btn btn-primary btn-sm edit_button" data-training-id="${tp.id}" title="Edit" onclick="event.stopPropagation();">
                        <i class="fa fa-pencil"></i>
                    </button>
                    <button class="btn btn-danger btn-sm delete_button" data-training-id="${tp.id}" title="Delete" onclick="event.stopPropagation();">
                        <i class="fa fa-trash-o"></i>
                    </button>`;
                }

                let card = `
                <div class="col-lg-3 col-md-4 col-sm-6">
                    <div class="training-card" data-training-id="${tp.id}">
                        <div class="training-card-thumb ${tp.thumbnail_path ? '' : thumbClass}">
                            ${thumbContent}
                        </div>
                        <div class="training-card-caption">
                            <h4 title="${escapeHtml(tp.name)}">${escapeHtml(tp.name)}</h4>
                            <p class="card-desc" title="${escapeHtml(tp.description || '')}">${escapeHtml(tp.description || typeLabel)}</p>
                            <div class="card-meta">
                                <i class="fa fa-calendar"></i> ${uploadDate} &nbsp;&middot;&nbsp;
                                <i class="fa fa-hdd-o"></i> ${formatFileSize(tp.file_size)}
                            </div>
                            ${statusBadge}
                            <div class="card-progress-bar" style="margin-top:6px;">
                                <div style="background:#eee; border-radius:6px; height:5px; overflow:hidden;">
                                    <div style="width:${cardProgressPct}%; height:100%; border-radius:6px; background:${cardProgressColor}; transition:width 0.4s ease;"></div>
                                </div>
                            </div>
                            <div class="card-actions" style="margin-top:8px;">${actions}</div>
                        </div>
                    </div>
                </div>`;
                grid.append(card);
            });
        })
        .fail(() => {
            $("#loading").hide();
            errorFlash("Error fetching training presentations");
        });
    });
};

// ---- Upload ----
const uploadPresentation = () => {
    let name = $("#presentationName").val();
    let description = $("#presentationDescription").val();
    let fileInput = $("#presentationFile")[0];
    let thumbInput = $("#presentationThumbnail")[0];
    let youtubeUrl = $("#presentationYouTube").val().trim();
    let pages = collectPages("#uploadPagesList");

    if (!name) {
        modalError("Please enter a presentation name");
        return;
    }
    if (!fileInput.files || fileInput.files.length === 0) {
        modalError("Please select a file to upload");
        return;
    }

    let formData = new FormData();
    formData.append("name", name);
    formData.append("description", description);
    formData.append("file", fileInput.files[0]);
    formData.append("youtube_url", youtubeUrl);
    formData.append("content_pages", JSON.stringify(pages));

    if (thumbInput.files && thumbInput.files.length > 0) {
        formData.append("thumbnail", thumbInput.files[0]);
    }

    $("#uploadSubmit").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Uploading...');

    $.ajax({
        url: "/api/training/",
        method: "POST",
        data: formData,
        processData: false,
        contentType: false,
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .done((data) => {
        dismissUpload();
        load();
        $("#uploadModal").modal("hide");
        successFlash('Training presentation "' + escapeHtml(data.name) + '" has been uploaded successfully!');
    })
    .fail((data) => {
        let msg = "Error uploading presentation";
        if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
        modalError(msg);
    })
    .always(() => {
        $("#uploadSubmit").prop("disabled", false).html('<i class="fa fa-upload"></i> Upload');
    });
};

// ---- Edit ----
const editPresentation = (id) => {
    let tp = presentations.find(p => p.id === id);
    if (!tp) return;

    $("#editId").val(tp.id);
    $("#editName").val(tp.name);
    $("#editDescription").val(tp.description || '');
    $("#editYouTube").val(tp.youtube_url || '');

    // Populate content pages
    let pages = [];
    try {
        pages = tp.content_pages ? JSON.parse(tp.content_pages) : [];
    } catch (e) {
        pages = [];
    }
    renderPagesInList("#editPagesList", pages);

    // Load quiz builder
    loadQuizForEdit(tp.id);

    $("#editModal").modal("show");
};

const saveEdit = () => {
    let id = parseInt($("#editId").val());
    let pages = collectPages("#editPagesList");
    let data = {
        name: $("#editName").val(),
        description: $("#editDescription").val(),
        youtube_url: $("#editYouTube").val().trim(),
        content_pages: JSON.stringify(pages)
    };

    if (!data.name) {
        $("#editModal\\.flashes").empty().append(
            '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> Please enter a name</div>'
        );
        return;
    }

    $("#editSubmit").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Saving...');

    $.ajax({
        url: "/api/training/" + id,
        method: "PUT",
        data: JSON.stringify(data),
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    })
    .done(() => {
        // Save quiz after presentation update succeeds
        saveQuizForPresentation(id, function () {
            dismissEdit();
            load();
            $("#editModal").modal("hide");
            successFlash("Training presentation updated successfully!");
            $("#editSubmit").prop("disabled", false).html('<i class="fa fa-save"></i> Save Changes');
        });
    })
    .fail((data) => {
        let msg = "Error updating presentation";
        if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
        $("#editModal\\.flashes").empty().append(
            '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> ' + msg + '</div>'
        );
        $("#editSubmit").prop("disabled", false).html('<i class="fa fa-save"></i> Save Changes');
    });
};

// ---- Delete ----
const deletePresentation = (id) => {
    let tp = presentations.find(p => p.id === id);
    let name = tp ? tp.name : "this presentation";

    Swal.fire({
        title: "Are you sure?",
        text: 'This will delete "' + name + '". This action cannot be undone.',
        type: "warning",
        animation: false,
        showCancelButton: true,
        confirmButtonText: "Delete",
        confirmButtonColor: "#d9534f",
        reverseButtons: true,
        allowOutsideClick: false,
        showLoaderOnConfirm: true,
        preConfirm: function () {
            return new Promise(function (resolve, reject) {
                query("/training/" + id, "DELETE", null, true)
                    .done(function (msg) {
                        resolve()
                    })
                    .fail(function (data) {
                        reject(data.responseJSON ? data.responseJSON.message : "Error deleting presentation")
                    })
            });
        }
    }).then(function (result) {
        if (result.value) {
            Swal.fire(
                'Deleted!',
                'Training presentation "' + escapeHtml(name) + '" has been deleted.',
                'success'
            );
            load();
        }
    });
};

// ---- Quiz builder helpers (edit modal) ----
const createQuizQuestionHtml = (index, questionText, options, correctOption) => {
    questionText = questionText || '';
    options = options || ['', '', '', ''];
    correctOption = (typeof correctOption === 'number') ? correctOption : 0;
    let optionsHtml = '';
    for (let i = 0; i < 4; i++) {
        let val = (options[i] !== undefined) ? options[i] : '';
        let checked = (i === correctOption) ? ' checked' : '';
        optionsHtml += `
            <div style="display:flex; align-items:center; gap:6px; margin-bottom:4px;">
                <input type="radio" name="quiz_correct_${index}" value="${i}"${checked} />
                <input type="text" class="form-control quiz-option-input" placeholder="Option ${i + 1}" value="${escapeHtml(val)}" style="flex:1;" />
            </div>`;
    }
    return `<div class="quiz-question-entry" data-q-index="${index}" style="background:#f9f9f9; border:1px solid #e0e0e0; border-radius:6px; padding:12px; margin-bottom:10px; position:relative;">
        <div style="display:flex; justify-content:space-between; align-items:center; margin-bottom:8px;">
            <span class="q-number-label" style="font-size:12px; font-weight:600; color:#888;">Question ${index + 1}</span>
            <button type="button" class="btn btn-xs btn-danger remove-quiz-q-btn" title="Remove"><i class="fa fa-times"></i></button>
        </div>
        <input type="text" class="form-control quiz-question-text" placeholder="Question text..." value="${escapeHtml(questionText)}" style="margin-bottom:8px;" />
        <label style="font-size:11px; color:#888; margin-bottom:4px;">Options (select the correct one):</label>
        ${optionsHtml}
    </div>`;
};

const collectQuizQuestions = () => {
    let questions = [];
    $("#editQuizQuestions .quiz-question-entry").each(function () {
        let text = $(this).find('.quiz-question-text').val().trim();
        let opts = [];
        $(this).find('.quiz-option-input').each(function () {
            opts.push($(this).val().trim());
        });
        let correct = parseInt($(this).find('input[type=radio]:checked').val()) || 0;
        if (text) {
            questions.push({
                question_text: text,
                options: JSON.stringify(opts),
                correct_option: correct,
                sort_order: questions.length
            });
        }
    });
    return questions;
};

const loadQuizForEdit = (presentationId) => {
    editQuizExisted = false;
    $("#editQuizEnabled").prop("checked", false);
    $("#editQuizSection").hide();
    $("#editQuizQuestions").empty();
    $("#editQuizPassPct").val(70);

    api.quiz.get(presentationId)
        .done(function (quiz) {
            if (quiz && quiz.id) {
                editQuizExisted = true;
                $("#editQuizEnabled").prop("checked", true);
                $("#editQuizSection").show();
                $("#editQuizPassPct").val(quiz.pass_percentage || 70);
                if (quiz.questions && quiz.questions.length > 0) {
                    quiz.questions.forEach(function (q, i) {
                        let opts = [];
                        try { opts = JSON.parse(q.options); } catch (e) { opts = []; }
                        $("#editQuizQuestions").append(createQuizQuestionHtml(i, q.question_text, opts, q.correct_option));
                    });
                }
            }
        })
        .fail(function () {
            // No quiz — that's fine
        });
};

const saveQuizForPresentation = (presentationId, callback) => {
    let enabled = $("#editQuizEnabled").is(":checked");
    if (!enabled) {
        if (editQuizExisted) {
            api.quiz.delete(presentationId)
                .done(function () { if (callback) callback(); })
                .fail(function () { if (callback) callback(); });
        } else {
            if (callback) callback();
        }
        return;
    }
    let questions = collectQuizQuestions();
    if (questions.length === 0) {
        if (callback) callback();
        return;
    }
    let data = {
        pass_percentage: parseInt($("#editQuizPassPct").val()) || 70,
        questions: questions
    };
    api.quiz.post(presentationId, data)
        .done(function () { if (callback) callback(); })
        .fail(function () { if (callback) callback(); });
};

// ---- Quiz viewer (course viewer modal) ----
const loadQuizForViewer = (presentationId, callback) => {
    currentQuiz = null;
    api.quiz.get(presentationId)
        .done(function (quiz) {
            if (quiz && quiz.id && quiz.questions && quiz.questions.length > 0) {
                currentQuiz = quiz;
            }
            if (callback) callback();
        })
        .fail(function () {
            if (callback) callback();
        });
};

const renderQuizViewer = () => {
    if (!currentQuiz || !currentQuiz.questions) return;
    let container = $("#quizQuestionsList");
    container.empty();
    currentQuiz.questions.forEach(function (q, i) {
        let opts = [];
        try { opts = JSON.parse(q.options); } catch (e) { opts = []; }
        let optsHtml = '';
        opts.forEach(function (opt, oi) {
            optsHtml += `
                <div style="margin-bottom:6px;">
                    <label style="font-weight:normal; cursor:pointer;">
                        <input type="radio" name="viewer_q_${i}" value="${oi}" style="margin-right:8px;" />
                        ${escapeHtml(opt)}
                    </label>
                </div>`;
        });
        container.append(`
            <div class="viewer-question" data-q-index="${i}" style="background:#f9f9f9; border:1px solid #e0e0e0; border-radius:6px; padding:16px; margin-bottom:12px;">
                <p style="font-weight:600; color:#2c3e50; margin:0 0 10px 0;">${i + 1}. ${escapeHtml(q.question_text)}</p>
                ${optsHtml}
            </div>`);
    });
    $("#quizResultSection").hide().empty();
    $("#submitQuizBtn").show().prop("disabled", false).html('<i class="fa fa-check"></i> Submit Answers');
};

const submitQuiz = () => {
    if (!currentQuiz || !currentCourseTP) return;
    let answers = [];
    let allAnswered = true;
    currentQuiz.questions.forEach(function (q, i) {
        let selected = $('input[name="viewer_q_' + i + '"]:checked').val();
        if (selected === undefined) {
            allAnswered = false;
            answers.push(-1);
        } else {
            answers.push(parseInt(selected));
        }
    });
    if (!allAnswered) {
        $("#quizResultSection").show().html(
            '<div class="alert alert-warning"><i class="fa fa-exclamation-triangle"></i> Please answer all questions before submitting.</div>'
        );
        return;
    }
    $("#submitQuizBtn").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Grading...');

    api.quizAttempt.post(currentCourseTP.id, { answers: answers })
        .done(function (data) {
            let resultHtml = '';
            if (data.passed) {
                let qPraise = praiseMessages.quiz_passed || {};
                let qIcon = qPraise.icon || '🏆';
                let qColor = qPraise.color_scheme || '#f39c12';
                let qHeading = qPraise.heading || 'Congratulations! You passed!';
                let qBody = renderPraiseBody(qPraise.body || 'Score: {{.Score}}/{{.Total}}', {
                    Score: data.score,
                    Total: data.total,
                    CourseName: currentCourseTP ? escapeHtml(currentCourseTP.name) : 'this course'
                });
                let qButton = qPraise.button_text || 'Complete Course';
                resultHtml = `
                    <div class="alert alert-success" style="font-size:16px;">
                        <span style="font-size:24px;">${qIcon}</span><br/>
                        <strong>${escapeHtml(qHeading)}</strong><br/>
                        ${qBody}<br/>
                        <small>Score: ${data.score} / ${data.total} (${Math.round(data.score * 100 / data.total)}%)</small>
                    </div>
                    <button id="quizFinishBtn" class="btn btn-success btn-lg">
                        <i class="fa fa-check-circle"></i> ${escapeHtml(qButton)}
                    </button>`;
                $("#submitQuizBtn").hide();
            } else {
                resultHtml = `
                    <div class="alert alert-danger" style="font-size:16px;">
                        <i class="fa fa-times-circle" style="font-size:24px;"></i><br/>
                        <strong>Not quite. Try again!</strong><br/>
                        Score: ${data.score} / ${data.total} (${Math.round(data.score * 100 / data.total)}%)<br/>
                        <small>You need ${currentQuiz.pass_percentage}% to pass.</small>
                    </div>
                    <button id="quizRetryBtn" class="btn btn-primary">
                        <i class="fa fa-refresh"></i> Retry Quiz
                    </button>`;
                $("#submitQuizBtn").hide();
            }
            $("#quizResultSection").show().html(resultHtml);
        })
        .fail(function (data) {
            let msg = "Error submitting quiz";
            if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
            $("#quizResultSection").show().html(
                '<div class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> ' + msg + '</div>'
            );
            $("#submitQuizBtn").prop("disabled", false).html('<i class="fa fa-check"></i> Submit Answers');
        });
};

// ---- Assignment helpers ----
const openAssignModal = (tp) => {
    $("#assignPresentationId").val(tp.id);
    $("#assignCourseName").text(tp.name);
    $("#assignModal\\.flashes").empty();
    $("#assignDueDate").val("");

    // Load users
    let userSelect = $("#assignUserId");
    userSelect.find("option:not(:first)").remove();
    $.ajax({
        url: "/api/users/",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (users) {
        if (users && users.length > 0) {
            users.forEach(function (u) {
                userSelect.append('<option value="' + u.id + '">' + escapeHtml(u.username) + '</option>');
            });
        }
    });

    // Load groups
    let groupSelect = $("#assignGroupId");
    groupSelect.find("option:not(:first)").remove();
    api.groups.summary()
        .done(function (summaries) {
            if (summaries.groups && summaries.groups.length > 0) {
                summaries.groups.forEach(function (g) {
                    groupSelect.append('<option value="' + g.id + '">' + escapeHtml(g.name) + ' (' + g.num_targets + ' targets)</option>');
                });
            }
        });

    $("#assignModal").modal("show");
};

const submitAssignment = () => {
    let presentationId = parseInt($("#assignPresentationId").val());
    let activeTab = $("#assignModal .tab-pane.active").attr("id");
    let dueDateStr = $("#assignDueDate").val().trim();
    let dueDate = '';
    if (dueDateStr) {
        dueDate = moment(dueDateStr, "MMMM Do YYYY, h:mm a").utc().format();
    }

    $("#assignSubmitBtn").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Assigning...');

    if (activeTab === 'assignUserTab') {
        let userId = parseInt($("#assignUserId").val());
        if (!userId) {
            $("#assignModal\\.flashes").empty().append(
                '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> Please select a user.</div>'
            );
            $("#assignSubmitBtn").prop("disabled", false).html('<i class="fa fa-paper-plane"></i> Assign');
            return;
        }
        api.assignments.post({ user_id: userId, presentation_id: presentationId, due_date: dueDate || undefined })
            .done(function () {
                $("#assignModal").modal("hide");
                successFlash("Course assigned successfully!");
            })
            .fail(function (data) {
                let msg = "Error assigning course";
                if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
                $("#assignModal\\.flashes").empty().append(
                    '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> ' + msg + '</div>'
                );
            })
            .always(function () {
                $("#assignSubmitBtn").prop("disabled", false).html('<i class="fa fa-paper-plane"></i> Assign');
            });
    } else {
        let groupId = parseInt($("#assignGroupId").val());
        if (!groupId) {
            $("#assignModal\\.flashes").empty().append(
                '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> Please select a group.</div>'
            );
            $("#assignSubmitBtn").prop("disabled", false).html('<i class="fa fa-paper-plane"></i> Assign');
            return;
        }
        api.assignments.assignGroup({ group_id: groupId, presentation_id: presentationId, due_date: dueDate || undefined })
            .done(function (result) {
                $("#assignModal").modal("hide");
                successFlash("Assigned to " + result.assigned + " user(s). Skipped: " + result.skipped_no_account + " no account, " + result.skipped_already_assigned + " already assigned.");
            })
            .fail(function (data) {
                let msg = "Error assigning course to group";
                if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
                $("#assignModal\\.flashes").empty().append(
                    '<div style="text-align:center" class="alert alert-danger"><i class="fa fa-exclamation-circle"></i> ' + msg + '</div>'
                );
            })
            .always(function () {
                $("#assignSubmitBtn").prop("disabled", false).html('<i class="fa fa-paper-plane"></i> Assign');
            });
    }
};

// =====================================================================
// Assignment Management Dashboard — Automated overdue checks, reminders,
// bulk operations, priority filters, and real-time status tracking
// =====================================================================

let assignmentDashboardData = [];

const loadAssignmentDashboard = () => {
    $.ajax({
        url: "/api/training/my-assignments",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (data) {
        assignmentDashboardData = data || [];
        renderAssignmentDashboard();
    });
};

const renderAssignmentDashboard = () => {
    let container = $("#myAssignmentsList");
    if (!container.length) return;
    container.empty();

    if (assignmentDashboardData.length === 0) {
        container.html('<div class="text-center text-muted" style="padding:30px;"><i class="fa fa-check-circle" style="font-size:40px; color:#27ae60;"></i><br/><br/>No assignments! You\'re all caught up.</div>');
        return;
    }

    let priorityIcons = { low: 'fa-arrow-down', normal: 'fa-minus', high: 'fa-arrow-up', critical: 'fa-exclamation-triangle' };
    let priorityColors = { low: '#95a5a6', normal: '#3498db', high: '#e67e22', critical: '#e74c3c' };

    assignmentDashboardData.forEach(function (a) {
        let statusBadge = getAssignmentStatusBadge(a.status, a.is_overdue);
        let priorityIcon = priorityIcons[a.priority] || 'fa-minus';
        let priorityColor = priorityColors[a.priority] || '#3498db';
        let dueDateStr = a.due_date && a.due_date !== '0001-01-01T00:00:00Z' ?
            moment(a.due_date).format('MMM D, YYYY') : 'No due date';
        let daysRemainingHtml = '';
        if (a.days_remaining > 0 && a.status !== 'completed') {
            daysRemainingHtml = '<span style="font-size:11px; color:#7f8c8d;">' + a.days_remaining + ' days left</span>';
        } else if (a.is_overdue) {
            daysRemainingHtml = '<span style="font-size:11px; color:#e74c3c; font-weight:600;">OVERDUE</span>';
        }

        let html = `<div class="assignment-item" data-assignment-id="${a.id}" style="
            display:flex; align-items:center; gap:12px; padding:12px 16px;
            border-bottom:1px solid #eee; cursor:pointer; transition:background 0.15s;
        " onmouseover="this.style.background='#f8f9fa'" onmouseout="this.style.background='transparent'">
            <div style="flex-shrink:0;">
                <i class="fa ${priorityIcon}" style="color:${priorityColor}; font-size:16px;" title="${a.priority} priority"></i>
            </div>
            <div style="flex:1; min-width:0;">
                <div style="font-weight:600; font-size:14px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">
                    ${escapeHtml(a.course_name || 'Course #' + a.presentation_id)}
                </div>
                <div style="font-size:12px; color:#7f8c8d; margin-top:2px;">
                    <i class="fa fa-calendar-o"></i> ${dueDateStr} ${daysRemainingHtml}
                </div>
            </div>
            <div style="flex-shrink:0;">${statusBadge}</div>
            <div style="flex-shrink:0;">
                <button class="btn btn-xs btn-primary open-assigned-course" data-presentation-id="${a.presentation_id}" title="Open Course">
                    <i class="fa fa-play-circle"></i>
                </button>
            </div>
        </div>`;
        container.append(html);
    });
};

const getAssignmentStatusBadge = (status, isOverdue) => {
    if (isOverdue || status === 'overdue') {
        return '<span class="label" style="font-size:10px; padding:3px 8px; background:#e74c3c; color:#fff; border-radius:3px;">Overdue</span>';
    }
    if (status === 'completed') {
        return '<span class="label" style="font-size:10px; padding:3px 8px; background:#27ae60; color:#fff; border-radius:3px;"><i class="fa fa-check"></i> Done</span>';
    }
    if (status === 'in_progress') {
        return '<span class="label" style="font-size:10px; padding:3px 8px; background:#2980b9; color:#fff; border-radius:3px;">In Progress</span>';
    }
    if (status === 'cancelled') {
        return '<span class="label" style="font-size:10px; padding:3px 8px; background:#95a5a6; color:#fff; border-radius:3px;">Cancelled</span>';
    }
    return '<span class="label" style="font-size:10px; padding:3px 8px; background:#f39c12; color:#fff; border-radius:3px;">Pending</span>';
};

// Admin: load assignment summary for dashboard
const loadAssignmentSummary = (callback) => {
    $.ajax({
        url: "/api/training/assignments/summary",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (data) {
        renderAssignmentSummary(data);
        if (callback) callback(data);
    }).fail(function () {
        if (callback) callback(null);
    });
};

const renderAssignmentSummary = (summary) => {
    let container = $("#assignmentSummary");
    if (!container.length || !summary) return;

    container.html(`
        <div style="display:flex; gap:12px; flex-wrap:wrap;">
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#2c3e50;">${summary.total_assignments}</div>
                <div style="font-size:11px; color:#7f8c8d;">Total</div>
            </div>
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#f39c12;">${summary.pending}</div>
                <div style="font-size:11px; color:#7f8c8d;">Pending</div>
            </div>
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#2980b9;">${summary.in_progress}</div>
                <div style="font-size:11px; color:#7f8c8d;">In Progress</div>
            </div>
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#27ae60;">${summary.completed}</div>
                <div style="font-size:11px; color:#7f8c8d;">Completed</div>
            </div>
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#e74c3c;">${summary.overdue}</div>
                <div style="font-size:11px; color:#7f8c8d;">Overdue</div>
            </div>
            <div class="stat-card" style="flex:1; min-width:80px; text-align:center; padding:10px; background:#fff; border-radius:8px; border:1px solid #eee;">
                <div style="font-size:24px; font-weight:700; color:#e74c3c;">${summary.critical_priority}</div>
                <div style="font-size:11px; color:#7f8c8d;">Critical</div>
            </div>
        </div>
    `);
};

// Trigger automated overdue marking
const triggerMarkOverdue = () => {
    $.ajax({
        url: "/api/training/assignments/mark-overdue",
        method: "POST",
        dataType: "json",
        contentType: "application/json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
            xhr.setRequestHeader('X-CSRF-Token', csrf_token);
        }
    }).done(function (data) {
        if (data.marked_overdue > 0) {
            successFlash(data.marked_overdue + ' assignment(s) marked as overdue.');
            loadAssignmentDashboard();
        }
    });
};

// =====================================================================
// Completion Certificates — View, verify, download, manage
// =====================================================================

let myCertificates = [];
let certTemplates = [];

const loadMyCertificates = (callback) => {
    $.ajax({
        url: "/api/training/my-certificates",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (data) {
        myCertificates = data || [];
        renderCertificateWall();
        if (callback) callback();
    });
};

const loadCertificateTemplates = (callback) => {
    $.ajax({
        url: "/api/training/certificates/templates",
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (data) {
        certTemplates = data || [];
        if (callback) callback();
    });
};

const renderCertificateWall = () => {
    let container = $("#myCertificatesGrid");
    if (!container.length) return;
    container.empty();

    if (myCertificates.length === 0) {
        container.html('<div class="text-center text-muted" style="padding:30px;"><i class="fa fa-certificate" style="font-size:40px; color:#ccc;"></i><br/><br/>No certificates earned yet.<br/>Complete courses to earn certificates!</div>');
        return;
    }

    myCertificates.forEach(function (cert) {
        let tmpl = cert.template || {};
        let colorScheme = tmpl.color_scheme || '#2c3e50';
        let validBadge = cert.is_valid ?
            '<span style="font-size:10px; color:#27ae60;"><i class="fa fa-check-circle"></i> Valid</span>' :
            '<span style="font-size:10px; color:#e74c3c;"><i class="fa fa-times-circle"></i> ' + (cert.is_revoked ? 'Revoked' : 'Expired') + '</span>';
        let expiryInfo = '';
        if (cert.expires_date && cert.expires_date !== '0001-01-01T00:00:00Z') {
            expiryInfo = '<div style="font-size:10px; color:#7f8c8d;">Expires: ' + moment(cert.expires_date).format('MMM D, YYYY') + '</div>';
        }

        let html = `<div class="cert-card" data-cert-id="${cert.id}" style="
            background:#fff; border-radius:10px; border:1px solid #e0e0e0; overflow:hidden;
            box-shadow:0 2px 8px rgba(0,0,0,0.06); cursor:pointer; transition:transform 0.2s, box-shadow 0.2s;
            width:260px; display:inline-block; margin:8px; vertical-align:top;
        " onmouseover="this.style.transform='translateY(-2px)'; this.style.boxShadow='0 6px 20px rgba(0,0,0,0.12)'"
           onmouseout="this.style.transform='none'; this.style.boxShadow='0 2px 8px rgba(0,0,0,0.06)'">
            <div style="background:${colorScheme}; padding:16px; text-align:center; color:#fff;">
                <i class="fa fa-certificate" style="font-size:32px; opacity:0.9;"></i>
                <div style="font-size:14px; font-weight:700; margin-top:8px;">${escapeHtml(tmpl.name || 'Completion Certificate')}</div>
            </div>
            <div style="padding:12px;">
                <div style="font-weight:600; font-size:13px; margin-bottom:4px;">${escapeHtml(cert.course_name || 'Course')}</div>
                <div style="font-size:11px; color:#7f8c8d;">Issued: ${moment(cert.issued_date).format('MMM D, YYYY')}</div>
                ${expiryInfo}
                <div style="margin-top:6px; display:flex; justify-content:space-between; align-items:center;">
                    ${validBadge}
                    <span style="font-size:10px; font-family:monospace; color:#95a5a6;">${escapeHtml(cert.formatted_code || '')}</span>
                </div>
            </div>
        </div>`;
        container.append(html);
    });
};

const showCertificateDetail = (certId) => {
    let cert = myCertificates.find(c => c.id === certId);
    if (!cert) return;

    let tmpl = cert.template || {};
    let colorScheme = tmpl.color_scheme || '#2c3e50';

    let validityHtml = cert.is_valid ?
        '<div style="color:#27ae60; font-size:16px; font-weight:600;"><i class="fa fa-check-circle"></i> Certificate is VALID</div>' :
        '<div style="color:#e74c3c; font-size:16px; font-weight:600;"><i class="fa fa-times-circle"></i> Certificate is ' + (cert.is_revoked ? 'REVOKED' : 'EXPIRED') + '</div>';

    let expiryHtml = '';
    if (cert.expires_date && cert.expires_date !== '0001-01-01T00:00:00Z') {
        expiryHtml = '<p><strong>Expires:</strong> ' + moment(cert.expires_date).format('MMMM D, YYYY') + '</p>';
    }

    let content = `
        <div style="text-align:center; background:${colorScheme}; padding:30px; color:#fff; border-radius:10px 10px 0 0;">
            <i class="fa fa-certificate" style="font-size:60px; opacity:0.9;"></i>
            <h3 style="margin:12px 0 4px 0;">${escapeHtml(tmpl.name || 'Completion Certificate')}</h3>
            <p style="opacity:0.8; margin:0;">${escapeHtml(tmpl.description || '')}</p>
        </div>
        <div style="padding:24px;">
            <h4 style="margin-top:0;">${escapeHtml(cert.course_name || 'Course')}</h4>
            <p><strong>Awarded to:</strong> ${escapeHtml(cert.user_name || '')}</p>
            <p><strong>Issued:</strong> ${moment(cert.issued_date).format('MMMM D, YYYY')}</p>
            ${expiryHtml}
            <p><strong>Verification Code:</strong> <code style="font-size:14px; background:#f5f5f5; padding:4px 8px; border-radius:4px;">${escapeHtml(cert.formatted_code || cert.verification_code)}</code></p>
            <hr/>
            ${validityHtml}
            <p style="font-size:12px; color:#7f8c8d; margin-top:8px;">
                Category: <span class="label" style="background:${colorScheme}; color:#fff; font-size:10px; padding:2px 8px; border-radius:3px;">${escapeHtml(tmpl.category || 'general')}</span>
            </p>
        </div>
    `;

    $("#certDetailContent").html(content);
    $("#certDetailModal").modal("show");
};

// Verify a certificate by code (public search)
const verifyCertificate = (code) => {
    if (!code || code.trim().length === 0) {
        $("#certVerifyResult").html('<div class="alert alert-warning">Please enter a verification code.</div>');
        return;
    }
    $("#certVerifyBtn").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Verifying...');
    $.ajax({
        url: "/api/training/certificates/verify/" + encodeURIComponent(code.trim()),
        method: "GET",
        dataType: "json",
        beforeSend: function (xhr) {
            xhr.setRequestHeader('Authorization', 'Bearer ' + user.api_key);
        }
    }).done(function (data) {
        if (data.valid) {
            let tmpl = data.template || {};
            let colorScheme = tmpl ? (tmpl.color_scheme || '#27ae60') : '#27ae60';
            $("#certVerifyResult").html(`
                <div style="background:${colorScheme}; color:#fff; padding:20px; border-radius:8px; text-align:center; margin-top:12px;">
                    <i class="fa fa-check-circle" style="font-size:32px;"></i>
                    <h4 style="margin:8px 0;">Certificate Verified!</h4>
                    <p><strong>${escapeHtml(data.user_name)}</strong></p>
                    <p>${escapeHtml(data.course_name)}</p>
                    <p style="opacity:0.8; font-size:12px;">Issued: ${moment(data.issued_date).format('MMM D, YYYY')}</p>
                    ${data.template ? '<p style="opacity:0.8; font-size:12px;">' + escapeHtml(data.template.name) + '</p>' : ''}
                </div>
            `);
        } else {
            $("#certVerifyResult").html('<div class="alert alert-danger"><i class="fa fa-times-circle"></i> Certificate not found or invalid.</div>');
        }
    }).fail(function () {
        $("#certVerifyResult").html('<div class="alert alert-danger"><i class="fa fa-times-circle"></i> Certificate not found.</div>');
    }).always(function () {
        $("#certVerifyBtn").prop("disabled", false).html('<i class="fa fa-search"></i> Verify');
    });
};

// ---- Content Library ----
const difficultyLabels = { 1: 'Bronze', 2: 'Silver', 3: 'Gold', 4: 'Platinum' };
const difficultyClasses = { 1: 'diff-bronze', 2: 'diff-silver', 3: 'diff-gold', 4: 'diff-platinum' };
const categoryLabels = {
    phishing: 'Phishing', passwords: 'Passwords', social_engineering: 'Social Engineering',
    data_protection: 'Data Protection', malware: 'Malware', physical_security: 'Physical Security',
    mobile_security: 'Mobile Security', remote_work: 'Remote Work', compliance: 'Compliance',
    incident_response: 'Incident Response', cloud_security: 'Cloud Security', ai_security: 'AI Security'
};

const openContentLibrary = () => {
    $("#contentLibraryModal").modal("show");
    loadContentLibrary();
};

const loadContentLibrary = () => {
    let category = $("#libFilterCategory").val();
    let difficulty = $("#libFilterDifficulty").val();

    $("#libGrid").hide();
    $("#libEmpty").hide();
    $("#libLoading").show();

    api.contentLibrary.browse(category, difficulty)
        .done(function (items) {
            contentLibraryData = items || [];
            populateLibraryCategories();
            renderLibraryGrid();
        })
        .fail(function () {
            contentLibraryData = [];
            renderLibraryGrid();
        });
};

const populateLibraryCategories = () => {
    let select = $("#libFilterCategory");
    if (select.find("option").length > 1) return; // Already populated
    api.contentLibrary.categories()
        .done(function (cats) {
            if (cats && cats.length > 0) {
                cats.forEach(function (c) {
                    select.append('<option value="' + escapeHtml(c.slug) + '">' + escapeHtml(c.label || c.slug) + ' (' + c.count + ')</option>');
                });
            }
        });
};

const renderLibraryGrid = () => {
    let searchTerm = ($("#libSearchInput").val() || '').toLowerCase();
    let items = contentLibraryData;

    // Client-side search filter
    if (searchTerm) {
        items = items.filter(function (item) {
            return item.title.toLowerCase().includes(searchTerm) ||
                   (item.description && item.description.toLowerCase().includes(searchTerm)) ||
                   (item.tags && item.tags.some(t => t.toLowerCase().includes(searchTerm)));
        });
    }

    $("#libLoading").hide();
    let grid = $("#libGrid");
    grid.empty();

    if (items.length === 0) {
        $("#libEmpty").show();
        grid.hide();
        $("#libResultCount").text("0 items");
        return;
    }

    $("#libEmpty").hide();
    grid.show();
    $("#libResultCount").text(items.length + " item" + (items.length !== 1 ? "s" : ""));

    items.forEach(function (item) {
        let diffLabel = difficultyLabels[item.difficulty_level] || 'Unknown';
        let diffClass = difficultyClasses[item.difficulty_level] || 'diff-bronze';
        let catLabel = categoryLabels[item.category] || item.category;
        let quizBadge = item.has_quiz ? '<i class="fa fa-question-circle" style="color:#e67e22;" title="Has Quiz"></i> ' + item.question_count + 'Q' : '<span style="color:#ccc;">No quiz</span>';
        let tagHtml = '';
        if (item.tags && item.tags.length > 0) {
            tagHtml = item.tags.slice(0, 3).map(t => '<span style="display:inline-block; font-size:10px; padding:1px 6px; background:#eee; border-radius:8px; margin-right:3px;">' + escapeHtml(t) + '</span>').join('');
        }

        grid.append(
            '<div class="col-lg-3 col-md-4 col-sm-6">' +
            '<div class="lib-card" data-slug="' + escapeHtml(item.slug) + '">' +
            '<div class="lib-card-header">' +
            '<span class="lib-diff-badge ' + diffClass + '">' + diffLabel + '</span>' +
            '<span class="lib-category-badge">' + escapeHtml(catLabel) + '</span>' +
            '<h4 title="' + escapeHtml(item.title) + '">' + escapeHtml(item.title) + '</h4>' +
            '<p>' + escapeHtml(item.description || '') + '</p>' +
            '</div>' +
            '<div class="lib-card-footer">' +
            '<span><i class="fa fa-clock-o"></i> ' + item.estimated_minutes + ' min</span>' +
            '<span><i class="fa fa-files-o"></i> ' + item.page_count + ' pages</span>' +
            '<span>' + quizBadge + '</span>' +
            '</div>' +
            (tagHtml ? '<div style="padding:6px 16px 10px;">' + tagHtml + '</div>' : '') +
            '</div>' +
            '</div>'
        );
    });
};

const showLibraryDetail = (slug) => {
    api.contentLibrary.detail(slug)
        .done(function (item) {
            if (!item || !item.slug) return;

            $("#libDetailTitle").text(item.title);
            $("#libDetailDesc").text(item.description || "");
            $("#libDetailCategory").text(categoryLabels[item.category] || item.category);
            let diffLabel = difficultyLabels[item.difficulty_level] || 'Unknown';
            let diffClass = difficultyClasses[item.difficulty_level] || 'diff-bronze';
            $("#libDetailDifficulty").html('<span class="' + diffClass + '" style="padding:2px 10px; border-radius:10px; font-size:12px;">' + diffLabel + '</span>');
            $("#libDetailDuration").text(item.estimated_minutes + ' minutes');
            $("#libDetailPageCount").text(item.pages ? item.pages.length : 0);

            if (item.quiz && item.quiz.questions) {
                $("#libDetailQuiz").html('<span style="color:#27ae60;"><i class="fa fa-check-circle"></i> ' + item.quiz.questions.length + ' questions, ' + item.quiz.pass_percentage + '% to pass</span>');
            } else {
                $("#libDetailQuiz").html('<span style="color:#999;">No quiz</span>');
            }

            // Tags
            let tagHtml = '';
            if (item.tags && item.tags.length > 0) {
                tagHtml = '<strong style="font-size:11px; color:#888;">Tags:</strong> ';
                tagHtml += item.tags.map(t => '<span class="label label-default" style="font-size:11px;">' + escapeHtml(t) + '</span> ').join('');
            }
            $("#libDetailTags").html(tagHtml);

            // Compliance
            let compHtml = '';
            if (item.compliance_mapped && item.compliance_mapped.length > 0) {
                compHtml = '<strong style="font-size:11px; color:#888;">Compliance:</strong> ';
                compHtml += item.compliance_mapped.map(c => '<span class="label label-primary" style="font-size:11px;">' + escapeHtml(c) + '</span> ').join('');
            }
            $("#libDetailCompliance").html(compHtml);

            // Nanolearning tip
            if (item.nanolearning_tip) {
                $("#libDetailNanoText").text(item.nanolearning_tip);
                $("#libDetailNanoTip").show();
            } else {
                $("#libDetailNanoTip").hide();
            }

            // Content preview
            let pagesHtml = '';
            if (item.pages && item.pages.length > 0) {
                item.pages.forEach(function (p, i) {
                    pagesHtml += '<div style="margin-bottom:12px; padding:10px 14px; background:#f9f9f9; border-radius:6px; border-left:3px solid #3498db;">';
                    pagesHtml += '<strong style="font-size:12px; color:#2c3e50;">Page ' + (i + 1) + (p.title ? ': ' + escapeHtml(p.title) : '') + '</strong>';
                    if (p.body) {
                        let preview = p.body.length > 120 ? p.body.substring(0, 120) + '...' : p.body;
                        pagesHtml += '<p style="font-size:12px; color:#666; margin:4px 0 0;">' + escapeHtml(preview) + '</p>';
                    }
                    if (p.tip_box) {
                        pagesHtml += '<div style="margin-top:4px; font-size:11px; background:#e8f5e9; padding:4px 8px; border-radius:4px; color:#2e7d32;"><i class="fa fa-lightbulb-o"></i> ' + escapeHtml(p.tip_box) + '</div>';
                    }
                    pagesHtml += '</div>';
                });
            }
            $("#libDetailPagesContent").html(pagesHtml || '<p style="color:#999;">No pages available.</p>');

            // Seed button
            $("#libSeedSingleBtn").attr("data-slug", item.slug);

            $("#libDetailModal").modal("show");
        })
        .fail(function () {
            errorFlash("Failed to load content details.");
        });
};

const seedSingleContent = (slug) => {
    if (!slug) return;
    let btn = $("#libSeedSingleBtn");
    btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Adding...');
    api.contentLibrary.seedSingle(slug)
        .done(function (data) {
            successFlash("Content added to your organization! Refresh to see it in training.");
            btn.html('<i class="fa fa-check-circle"></i> Added!').addClass("btn-default").removeClass("btn-success");
            load(); // Refresh training list
        })
        .fail(function (data) {
            let msg = "Failed to add content";
            if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
            errorFlash(msg);
            btn.prop("disabled", false).html('<i class="fa fa-plus-circle"></i> Add to My Organization');
        });
};

const seedAllContent = () => {
    let btn = $("#seedAllBtn");
    if (!confirm("This will import all built-in training content into your organization. Continue?")) return;
    btn.prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Seeding...');
    api.contentLibrary.seedAll()
        .done(function (data) {
            successFlash("All built-in content has been seeded! Created: " +
                (data.presentations_created || 0) + " presentations, " +
                (data.sessions_created || 0) + " sessions, " +
                (data.quizzes_created || 0) + " quizzes.");
            btn.prop("disabled", false).html('<i class="fa fa-download"></i> Seed All Built-in Content');
            load();
        })
        .fail(function (data) {
            let msg = "Failed to seed content";
            if (data.responseJSON && data.responseJSON.message) msg = data.responseJSON.message;
            errorFlash(msg);
            btn.prop("disabled", false).html('<i class="fa fa-download"></i> Seed All Built-in Content');
        });
};

// ---- Satisfaction Rating ----
const ratingLabels = ['', 'Poor 😞', 'Fair 😐', 'Good 🙂', 'Very Good 😊', 'Excellent 🌟'];

const showSatisfactionModal = (tpId, name) => {
    selectedSatRating = 0;
    $(".sat-star").css("color", "#ddd").removeClass("active");
    $("#satFeedback").val("");
    $("#satSubmitBtn").prop("disabled", true);
    $("#satRatingLabel").text("");
    $("#satisfactionModal").data("tpId", tpId).modal("show");
};

const highlightStars = (value) => {
    $(".sat-star").each(function () {
        let v = parseInt($(this).data("value"));
        if (v <= value) {
            $(this).css("color", "#f39c12").addClass("active");
        } else {
            $(this).css("color", "#ddd").removeClass("active");
        }
    });
};

const submitSatisfactionRating = () => {
    let tpId = $("#satisfactionModal").data("tpId");
    if (!tpId || selectedSatRating < 1) return;

    let feedback = $("#satFeedback").val().trim();
    $("#satSubmitBtn").prop("disabled", true).html('<i class="fa fa-spinner fa-spin"></i> Submitting...');

    api.trainingSatisfaction.rate(tpId, selectedSatRating, feedback)
        .done(function () {
            $("#satisfactionModal").modal("hide");
            successFlashFade("Thank you for your feedback! ⭐", 3);
        })
        .fail(function () {
            $("#satisfactionModal").modal("hide");
            // Silent fail — don't bother user
        });
};

// ---- Document ready ----
$(document).ready(function () {
    load();
    loadPraiseMessages();

    // Thumbnail preview on file selection
    $("#presentationThumbnail").on("change", function () {
        if (this.files && this.files[0]) {
            let reader = new FileReader();
            reader.onload = function (e) {
                $("#thumbPreviewImg").attr("src", e.target.result);
                $("#thumbPreview").show();
            };
            reader.readAsDataURL(this.files[0]);
        } else {
            $("#thumbPreview").hide();
            $("#thumbPreviewImg").attr("src", "");
        }
    });

    // Show/hide page media URL input when type changes (delegated)
    $(document).on("change", ".page-media-type", function () {
        let urlInput = $(this).closest('.page-media-row').find('.page-media-url');
        if ($(this).val()) {
            urlInput.show().attr("placeholder",
                $(this).val() === 'youtube' ? "YouTube URL..." :
                $(this).val() === 'image' ? "Image URL (https://...)..." :
                "Video URL (https://...)..."
            );
        } else {
            urlInput.hide().val('');
        }
    });

    // Auto-extract slides
    $("#autoExtractUpload").on("click", function () {
        autoExtractSlides('upload');
    });
    $("#autoExtractEdit").on("click", function () {
        autoExtractSlides('edit');
    });

    // Add content page – Upload modal
    $("#addUploadPage").on("click", function () {
        let idx = $("#uploadPagesList .page-entry").length;
        $("#uploadPagesList").append(createPageEntryHtml('upload', idx, '', ''));
    });

    // Add content page – Edit modal
    $("#addEditPage").on("click", function () {
        let idx = $("#editPagesList .page-entry").length;
        $("#editPagesList").append(createPageEntryHtml('edit', idx, '', ''));
    });

    // ---- REMOVE content page (delegated – works for BOTH upload and edit modals) ----
    $(document).on("click", ".remove-page-btn", function (e) {
        e.preventDefault();
        e.stopPropagation();
        let entry = $(this).closest(".page-entry");
        let listContainer = entry.parent();
        entry.slideUp(200, function () {
            $(this).remove();
            // Re-index remaining pages
            reindexPages('#' + listContainer.attr('id'));
        });
    });

    // Upload button click
    $("#uploadSubmit").on("click", function () {
        uploadPresentation();
    });

    // Edit submit click
    $("#editSubmit").on("click", function () {
        saveEdit();
    });

    // Reset upload form when modal closes
    $("#uploadModal").on("hidden.bs.modal", function () {
        dismissUpload();
    });

    // Reset edit form when modal closes
    $("#editModal").on("hidden.bs.modal", function () {
        dismissEdit();
    });

    // Card click – open detail modal (delegated)
    $("#trainingGrid").on("click", ".training-card", function () {
        let id = parseInt($(this).attr("data-training-id"));
        let tp = presentations.find(p => p.id === id);
        if (tp) {
            showDetailModal(tp);
        }
    });

    // Enrol Now button – open course viewer
    $("#detailEnrollBtn").on("click", function () {
        if (currentCourseTP) {
            openCourseViewer(currentCourseTP);
        }
    });

    // Course viewer navigation – Next
    $("#courseNextBtn").on("click", function () {
        if ($(this).prop("disabled")) return;
        // Record engagement for current page before advancing
        sendPageEngagement(currentCoursePage, function () {
            $(".page-video-frame").attr("src", "");
            if (currentCoursePage >= coursePages.length - 1) {
                // Finish – validate completion with server first
                validateAndFinishCourse();
            } else {
                // Validate advance with server
                validateAndAdvancePage();
            }
        });
    });

    // Course viewer navigation – Previous
    $("#coursePrevBtn").on("click", function () {
        // Record engagement for current page before going back
        sendPageEngagement(currentCoursePage, function () {
            $(".page-video-frame").attr("src", "");
            if (currentCoursePage > 0) {
                currentCoursePage--;
                renderCoursePage();
            }
        });
    });

    // Stop video and clear anti-skip timer when course viewer closes
    $("#courseViewerModal").on("hidden.bs.modal", function () {
        $("#courseVideoIframe").attr("src", "");
        $(".page-video-frame").attr("src", "");
        // Clean up anti-skip state
        if (pageDwellTimer) { clearInterval(pageDwellTimer); pageDwellTimer = null; }
        antiSkipPolicy = null;
    });

    // Edit button click (delegated)
    $("#trainingGrid").on("click", ".edit_button", function (e) {
        e.stopPropagation();
        let id = parseInt($(this).attr("data-training-id"));
        editPresentation(id);
    });

    // Delete button click (delegated)
    $("#trainingGrid").on("click", ".delete_button", function (e) {
        e.stopPropagation();
        let id = parseInt($(this).attr("data-training-id"));
        deletePresentation(id);
    });

    // Assign button click (delegated)
    $("#trainingGrid").on("click", ".assign_button", function (e) {
        e.stopPropagation();
        let id = parseInt($(this).attr("data-training-id"));
        let tp = presentations.find(p => p.id === id);
        if (tp) openAssignModal(tp);
    });

    // Assignment submit
    $("#assignSubmitBtn").on("click", function () {
        submitAssignment();
    });

    // Assignment due date picker
    if ($.fn.datetimepicker) {
        $("#assignDueDate").datetimepicker({
            widgetPositioning: { vertical: "bottom" },
            showTodayButton: true,
            useCurrent: false,
            format: "MMMM Do YYYY, h:mm a"
        });
    }

    // Quiz builder – toggle quiz section
    $("#editQuizEnabled").on("change", function () {
        if ($(this).is(":checked")) {
            $("#editQuizSection").slideDown(200);
            // Add a default question if none exist
            if ($("#editQuizQuestions .quiz-question-entry").length === 0) {
                $("#editQuizQuestions").append(createQuizQuestionHtml(0));
            }
        } else {
            $("#editQuizSection").slideUp(200);
        }
    });

    // Quiz builder – add question
    $("#addQuizQuestion").on("click", function () {
        let idx = $("#editQuizQuestions .quiz-question-entry").length;
        $("#editQuizQuestions").append(createQuizQuestionHtml(idx));
    });

    // Quiz builder – remove question (delegated)
    $(document).on("click", ".remove-quiz-q-btn", function (e) {
        e.preventDefault();
        e.stopPropagation();
        let entry = $(this).closest(".quiz-question-entry");
        entry.slideUp(200, function () {
            $(this).remove();
            // Re-index remaining questions
            $("#editQuizQuestions .quiz-question-entry").each(function (i) {
                $(this).attr('data-q-index', i);
                $(this).find('.q-number-label').text('Question ' + (i + 1));
                // Update radio name attributes
                $(this).find('input[type=radio]').attr('name', 'quiz_correct_' + i);
            });
        });
    });

    // Quiz viewer – submit answers
    $("#submitQuizBtn").on("click", function () {
        submitQuiz();
    });

    // Quiz viewer – retry (delegated)
    $(document).on("click", "#quizRetryBtn", function () {
        renderQuizViewer();
    });

    // Quiz viewer – finish after passing (delegated)
    $(document).on("click", "#quizFinishBtn", function () {
        completeCourse();
    });

    // Reset quiz section when course viewer closes
    $("#courseViewerModal").on("hidden.bs.modal", function () {
        $("#courseQuizSection").hide();
        $("#courseContentSection").show();
        $("#courseNavSection").show();
        currentQuiz = null;
    });

    // ---- Content Library event handlers ----
    $("#contentLibraryBtn").on("click", function () {
        openContentLibrary();
    });

    // Seed all built-in content
    $("#seedAllBtn").on("click", function () {
        seedAllContent();
    });

    // Library filter changes
    $("#libFilterCategory, #libFilterDifficulty").on("change", function () {
        loadContentLibrary();
    });

    // Library search (debounced)
    let libSearchTimer = null;
    $("#libSearchInput").on("input", function () {
        clearTimeout(libSearchTimer);
        libSearchTimer = setTimeout(function () {
            renderLibraryGrid();
        }, 300);
    });

    // Library card click – show detail
    $(document).on("click", ".lib-card", function () {
        let slug = $(this).data("slug");
        if (slug) showLibraryDetail(slug);
    });

    // Seed single content from detail modal
    $("#libSeedSingleBtn").on("click", function () {
        let slug = $(this).attr("data-slug");
        seedSingleContent(slug);
    });

    // ---- Satisfaction Rating event handlers ----
    // Star hover
    $(document).on("mouseenter", ".sat-star", function () {
        let val = parseInt($(this).data("value"));
        highlightStars(val);
        $("#satRatingLabel").text(ratingLabels[val] || '');
    });

    // Star hover leave – restore selected
    $("#satisfactionStars").on("mouseleave", function () {
        highlightStars(selectedSatRating);
        $("#satRatingLabel").text(selectedSatRating > 0 ? ratingLabels[selectedSatRating] : '');
    });

    // Star click – select rating
    $(document).on("click", ".sat-star", function () {
        selectedSatRating = parseInt($(this).data("value"));
        highlightStars(selectedSatRating);
        $("#satRatingLabel").text(ratingLabels[selectedSatRating] || '');
        $("#satSubmitBtn").prop("disabled", false);
    });

    // Submit rating
    $("#satSubmitBtn").on("click", function () {
        submitSatisfactionRating();
    });

    // Skip rating
    $("#satSkipBtn").on("click", function (e) {
        e.preventDefault();
        $("#satisfactionModal").modal("hide");
    });

    // Open course from URL parameter (e.g. ?open=123 from academy)
    let urlParams = new URLSearchParams(window.location.search);
    let openParam = urlParams.get('open');
    if (openParam) {
        // Wait for load to finish, then open the course
        let openInterval = setInterval(function () {
            if (presentations.length > 0 || !$("#loading").is(":visible")) {
                clearInterval(openInterval);
                let tpId = parseInt(openParam);
                let tp = presentations.find(p => p.id === tpId);
                if (tp) {
                    openCourseViewer(tp);
                }
            }
        }, 200);
    }

    // ---- Assignment Dashboard event handlers ----
    loadAssignmentDashboard();

    // Open assigned course
    $(document).on("click", ".open-assigned-course", function (e) {
        e.stopPropagation();
        let presId = parseInt($(this).data("presentation-id"));
        let tp = presentations.find(p => p.id === presId);
        if (tp) {
            openCourseViewer(tp);
        }
    });

    // Mark overdue button
    $(document).on("click", "#markOverdueBtn", function () {
        triggerMarkOverdue();
    });

    // ---- Certificate Wall event handlers ----
    loadMyCertificates();
    loadCertificateTemplates();

    // Certificate card click — show detail
    $(document).on("click", ".cert-card", function () {
        let certId = parseInt($(this).data("cert-id"));
        showCertificateDetail(certId);
    });

    // Certificate verify
    $(document).on("click", "#certVerifyBtn", function () {
        let code = $("#certVerifyInput").val();
        verifyCertificate(code);
    });

    // Certificate verify on enter
    $(document).on("keypress", "#certVerifyInput", function (e) {
        if (e.which === 13) {
            let code = $(this).val();
            verifyCertificate(code);
        }
    });
});

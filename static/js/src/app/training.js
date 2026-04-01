let presentations = [];
let currentCourseTP = null;
let currentCoursePage = 0;
let coursePages = [];
let courseProgressMap = {}; // { presentationId: { status, current_page, total_pages, progress_pct } }
let currentQuiz = null; // quiz loaded for course viewer
let editQuizExisted = false; // whether a quiz existed before editing

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
                <div id="goldStar" style="font-size:80px; margin-bottom:16px;">⭐</div>
                <h2 style="margin:0 0 8px 0; font-size:28px; font-weight:700; color:#2c3e50;">Course Complete!</h2>
                <p style="font-size:16px; color:#666; margin:0 0 24px 0;">Congratulations! You finished <strong>${escapeHtml(courseName)}</strong></p>
                <span class="label" style="font-size:14px; padding:8px 24px; background:#27ae60; color:#fff; border-radius:20px;">
                    <i class="fa fa-check-circle"></i> Completed
                </span>
                <br/><br/>
                <button id="closeCelebration" class="btn btn-primary btn-lg" style="margin-top:10px;">
                    <i class="fa fa-thumbs-up"></i> Awesome!
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

    renderCoursePage();
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

    // Show celebration after a short delay
    let name = currentCourseTP.name;
    setTimeout(() => {
        showCompletionCelebration(name);
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
                resultHtml = `
                    <div class="alert alert-success" style="font-size:16px;">
                        <i class="fa fa-trophy" style="font-size:24px; color:#f39c12;"></i><br/>
                        <strong>Congratulations! You passed!</strong><br/>
                        Score: ${data.score} / ${data.total} (${Math.round(data.score * 100 / data.total)}%)
                    </div>
                    <button id="quizFinishBtn" class="btn btn-success btn-lg">
                        <i class="fa fa-check-circle"></i> Complete Course
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

// ---- Document ready ----
$(document).ready(function () {
    load();

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
        $(".page-video-frame").attr("src", "");
        if (currentCoursePage >= coursePages.length - 1) {
            // Finish – complete the course!
            finishCourse();
        } else {
            currentCoursePage++;
            renderCoursePage();
            $("#courseContentSection")[0].scrollIntoView({ behavior: 'smooth' });
        }
    });

    // Course viewer navigation – Previous
    $("#coursePrevBtn").on("click", function () {
        $(".page-video-frame").attr("src", "");
        if (currentCoursePage > 0) {
            currentCoursePage--;
            renderCoursePage();
        }
    });

    // Stop video when course viewer closes
    $("#courseViewerModal").on("hidden.bs.modal", function () {
        $("#courseVideoIframe").attr("src", "");
        $(".page-video-frame").attr("src", "");
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
});

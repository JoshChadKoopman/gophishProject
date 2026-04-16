/**
 * Nivoxis Security Assistant — Gmail Add-on (Google Apps Script)
 *
 * Features:
 *   1. Shows a sidebar with real-time threat scores on every email
 *   2. Allows one-click report-to-Nivoxis
 *   3. Displays a banner on suspicious emails before the user interacts
 *
 * Configuration: Set NIVOXIS_API_BASE and PLUGIN_API_KEY in Script Properties.
 */

// ── Configuration ──
function getConfig() {
  var props = PropertiesService.getScriptProperties();
  return {
    apiBase: props.getProperty('NIVOXIS_API_BASE') || 'https://your-nivoxis-instance.com/api',
    apiKey: props.getProperty('PLUGIN_API_KEY') || ''
  };
}

// ── Main Trigger: fires when user opens an email ──
function onEmailOpen(e) {
  var message = getCurrentMessage(e);
  if (!message) {
    return buildErrorCard('Could not read email.');
  }

  var config = getConfig();
  var analysis = analyzeEmail(config, message);

  return buildAnalysisCard(analysis, message);
}

// ── Get current email message ──
function getCurrentMessage(e) {
  try {
    var messageId = e.gmail.messageId;
    var message = GmailApp.getMessageById(messageId);
    return message;
  } catch (err) {
    Logger.log('Error getting message: ' + err);
    return null;
  }
}

// ── Send email to Nivoxis for AI analysis ──
function analyzeEmail(config, message) {
  var payload = {
    message_id: message.getId(),
    subject: message.getSubject(),
    sender_email: extractEmail(message.getFrom()),
    sender_name: message.getFrom(),
    body: message.getPlainBody().substring(0, 5000),
    headers: message.getHeader('Received') || '',
    provider: 'gmail',
    user_email: Session.getActiveUser().getEmail()
  };

  try {
    var response = UrlFetchApp.fetch(config.apiBase + '/inbox/addin/analyze', {
      method: 'post',
      contentType: 'application/json',
      headers: {
        'Authorization': 'Bearer ' + config.apiKey
      },
      payload: JSON.stringify(payload),
      muteHttpExceptions: true
    });

    if (response.getResponseCode() === 200) {
      return JSON.parse(response.getContentText());
    } else {
      Logger.log('API error: ' + response.getResponseCode() + ' ' + response.getContentText());
      return null;
    }
  } catch (err) {
    Logger.log('Analysis request failed: ' + err);
    return null;
  }
}

// ── Build the analysis results card ──
function buildAnalysisCard(analysis, message) {
  var card = CardService.newCardBuilder();
  card.setHeader(
    CardService.newCardHeader()
      .setTitle('Nivoxis Security')
      .setSubtitle('Email Threat Analysis')
      .setImageUrl('https://your-nivoxis-instance.com/static/images/nivoxis-icon.png')
      .setImageStyle(CardService.ImageStyle.CIRCLE)
  );

  if (!analysis) {
    var section = CardService.newCardSection();
    section.addWidget(
      CardService.newTextParagraph()
        .setText('⚠️ <b>Analysis Unavailable</b><br>Could not reach Nivoxis server.')
    );
    card.addSection(section);
    return card.build();
  }

  // ── Threat Level Banner ──
  var bannerSection = CardService.newCardSection();
  var threatEmoji = { 'safe': '✅', 'suspicious': '⚠️', 'likely_phishing': '🚨', 'confirmed_phishing': '🚨' };
  var emoji = threatEmoji[analysis.threat_level] || 'ℹ️';

  bannerSection.addWidget(
    CardService.newDecoratedText()
      .setText(emoji + ' <b>' + (analysis.threat_level || 'unknown').replace(/_/g, ' ').toUpperCase() + '</b>')
      .setBottomLabel('Confidence: ' + Math.round((analysis.confidence_score || 0) * 100) + '%')
  );

  if (analysis.summary) {
    bannerSection.addWidget(
      CardService.newTextParagraph().setText(analysis.summary)
    );
  }

  if (analysis.was_simulation) {
    bannerSection.addWidget(
      CardService.newDecoratedText()
        .setText('📋 <b>This was a phishing simulation</b>')
    );
  }

  card.addSection(bannerSection);

  // ── Indicators ──
  if (analysis.indicators && analysis.indicators.length > 0) {
    var indSection = CardService.newCardSection().setHeader('Threat Indicators');
    analysis.indicators.forEach(function(ind) {
      var sevEmoji = { 'critical': '🔴', 'high': '🟠', 'medium': '🟡', 'low': '🟢', 'info': 'ℹ️' };
      indSection.addWidget(
        CardService.newDecoratedText()
          .setText((sevEmoji[ind.severity] || 'ℹ️') + ' <b>' + (ind.name || ind.type || '') + '</b>')
          .setBottomLabel(ind.description || '')
      );
    });
    card.addSection(indSection);
  }

  // ── Recommendation & Learning Tip ──
  var adviceSection = CardService.newCardSection().setHeader('Recommendation');
  if (analysis.recommendation) {
    adviceSection.addWidget(CardService.newTextParagraph().setText(analysis.recommendation));
  }
  if (analysis.learning_tip) {
    adviceSection.addWidget(
      CardService.newTextParagraph().setText('💡 ' + analysis.learning_tip)
    );
  }
  card.addSection(adviceSection);

  // ── Action Buttons ──
  var actionSection = CardService.newCardSection();
  actionSection.addWidget(
    CardService.newTextButton()
      .setText('🚨 Report as Phishing')
      .setTextButtonStyle(CardService.TextButtonStyle.FILLED)
      .setBackgroundColor('#E53935')
      .setOnClickAction(
        CardService.newAction()
          .setFunctionName('reportPhishing')
          .setParameters({ messageId: message.getId(), subject: message.getSubject() })
      )
  );
  actionSection.addWidget(
    CardService.newTextButton()
      .setText('🔄 Re-Analyze')
      .setOnClickAction(CardService.newAction().setFunctionName('onEmailOpen'))
  );
  card.addSection(actionSection);

  return card.build();
}

// ── Report email as phishing ──
function reportPhishing(e) {
  var config = getConfig();
  var messageId = e.parameters.messageId;
  var subject = e.parameters.subject;

  try {
    var response = UrlFetchApp.fetch(config.apiBase + '/report?rid=' + encodeURIComponent(messageId), {
      method: 'post',
      contentType: 'application/json',
      headers: {
        'Authorization': 'Bearer ' + config.apiKey,
        'Accept': 'application/json'
      },
      payload: JSON.stringify({
        reporter_email: Session.getActiveUser().getEmail(),
        subject: subject,
        provider: 'gmail'
      }),
      muteHttpExceptions: true
    });

    var data = {};
    try { data = JSON.parse(response.getContentText()); } catch(ex) {}

    // Build confirmation card with feedback
    var card = CardService.newCardBuilder();
    card.setHeader(
      CardService.newCardHeader()
        .setTitle('Nivoxis Security')
        .setSubtitle('Report Submitted')
    );

    var section = CardService.newCardSection();
    section.addWidget(
      CardService.newTextParagraph()
        .setText('✅ <b>Email Reported Successfully</b><br><br>' +
          (data.summary || 'Thank you for reporting this email. Your security team has been notified.'))
    );

    if (data.learning_tip) {
      section.addWidget(
        CardService.newTextParagraph().setText('💡 ' + data.learning_tip)
      );
    }

    card.addSection(section);
    return CardService.newActionResponseBuilder()
      .setNavigation(CardService.newNavigation().updateCard(card.build()))
      .build();

  } catch (err) {
    Logger.log('Report failed: ' + err);
    return buildErrorCard('Failed to report email. Please try again.');
  }
}

// ── Error card builder ──
function buildErrorCard(message) {
  var card = CardService.newCardBuilder();
  card.setHeader(
    CardService.newCardHeader()
      .setTitle('Nivoxis Security')
      .setSubtitle('Error')
  );
  var section = CardService.newCardSection();
  section.addWidget(CardService.newTextParagraph().setText('⚠️ ' + message));
  card.addSection(section);
  return card.build();
}

// ── Helpers ──
function extractEmail(fromField) {
  var match = fromField.match(/<(.+?)>/);
  return match ? match[1] : fromField;
}

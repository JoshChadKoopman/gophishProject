package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// FeedbackPage contains the fields for an educational feedback page shown
// to recipients after they click a simulated phishing link.
type FeedbackPage struct {
	Id                   int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId               int64     `json:"-" gorm:"column:user_id"`
	OrgId                int64     `json:"-" gorm:"column:org_id"`
	Name                 string    `json:"name"`
	Language             string    `json:"language" gorm:"column:language"`
	HTML                 string    `json:"html" gorm:"column:html"`
	RedirectURL          string    `json:"redirect_url" gorm:"column:redirect_url"`
	RedirectDelaySeconds int       `json:"redirect_delay_seconds" gorm:"column:redirect_delay_seconds"`
	ModifiedDate         time.Time `json:"modified_date"`
}

// ErrFeedbackPageNameNotSpecified is returned when no name is provided.
var ErrFeedbackPageNameNotSpecified = errors.New("Feedback page name not specified")

// ErrFeedbackPageContentNotSpecified is returned when HTML is empty.
var ErrFeedbackPageContentNotSpecified = errors.New("Feedback page content not specified")

// Validate ensures required fields are present.
func (fp *FeedbackPage) Validate() error {
	if fp.Name == "" {
		return ErrFeedbackPageNameNotSpecified
	}
	if fp.HTML == "" {
		return ErrFeedbackPageContentNotSpecified
	}
	if err := ValidateTemplate(fp.HTML); err != nil {
		return err
	}
	if fp.RedirectURL != "" {
		if err := ValidateTemplate(fp.RedirectURL); err != nil {
			return err
		}
	}
	if fp.RedirectDelaySeconds < 0 {
		fp.RedirectDelaySeconds = 0
	}
	if fp.Language == "" {
		fp.Language = "en"
	}
	return nil
}

// GetFeedbackPages returns all feedback pages for the given scope.
func GetFeedbackPages(scope OrgScope) ([]FeedbackPage, error) {
	fps := []FeedbackPage{}
	err := scopeQuery(db.Table("feedback_pages"), scope).Find(&fps).Error
	if err != nil {
		log.Error(err)
		return fps, err
	}
	return fps, nil
}

// GetFeedbackPage returns a single feedback page by ID and scope.
func GetFeedbackPage(id int64, scope OrgScope) (FeedbackPage, error) {
	fp := FeedbackPage{}
	err := scopeQuery(db.Where("id=?", id), scope).Find(&fp).Error
	if err != nil {
		log.Error(err)
	}
	return fp, err
}

// GetFeedbackPageByName returns a feedback page by name and scope.
func GetFeedbackPageByName(name string, scope OrgScope) (FeedbackPage, error) {
	fp := FeedbackPage{}
	err := scopeQuery(db.Where("name=?", name), scope).Find(&fp).Error
	if err != nil {
		log.Error(err)
	}
	return fp, err
}

// PostFeedbackPage creates a new feedback page.
func PostFeedbackPage(fp *FeedbackPage) error {
	if err := fp.Validate(); err != nil {
		return err
	}
	fp.ModifiedDate = time.Now().UTC()
	return db.Save(fp).Error
}

// PutFeedbackPage updates an existing feedback page.
func PutFeedbackPage(fp *FeedbackPage) error {
	if err := fp.Validate(); err != nil {
		return err
	}
	fp.ModifiedDate = time.Now().UTC()
	return db.Where("id=?", fp.Id).Save(fp).Error
}

// DeleteFeedbackPage removes a feedback page by ID and scope.
func DeleteFeedbackPage(id int64, scope OrgScope) error {
	return scopeQuery(db.Where("id=?", id), scope).Delete(FeedbackPage{}).Error
}

// DefaultFeedbackHTML returns the default Nivoxis-branded educational
// feedback page HTML. It uses template variables so it can be
// personalized per recipient.
func DefaultFeedbackHTML(lang string) string {
	switch lang {
	case "nl":
		return defaultFeedbackNL
	case "fr":
		return defaultFeedbackFR
	case "de":
		return defaultFeedbackDE
	case "es":
		return defaultFeedbackES
	default:
		return defaultFeedbackEN
	}
}

const defaultFeedbackEN = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Security Awareness - Simulated Phishing</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; border-radius: 12px; box-shadow: 0 4px 24px rgba(0,0,0,0.1); max-width: 600px; width: 100%; overflow: hidden; }
.header { background: linear-gradient(135deg, #e74c3c, #c0392b); color: #fff; padding: 32px; text-align: center; }
.header svg { width: 64px; height: 64px; margin-bottom: 16px; }
.header h1 { font-size: 24px; margin-bottom: 8px; }
.header p { opacity: 0.9; font-size: 15px; }
.body { padding: 32px; }
.body h2 { color: #2c3e50; margin-bottom: 16px; font-size: 18px; }
.tip { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid #eee; }
.tip:last-child { border-bottom: none; }
.tip-icon { background: #fef3f2; color: #e74c3c; width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-weight: bold; }
.tip-text { color: #555; font-size: 14px; line-height: 1.5; }
.tip-text strong { color: #2c3e50; }
.footer { padding: 24px 32px; background: #f8f9fa; text-align: center; }
.footer p { color: #888; font-size: 13px; }
.btn { display: inline-block; background: #3498db; color: #fff; padding: 10px 24px; border-radius: 6px; text-decoration: none; font-size: 14px; margin-top: 12px; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <h1>This Was a Simulated Phishing Email</h1>
    <p>Don't worry — this was a training exercise by your organization.</p>
  </div>
  <div class="body">
    <h2>What Should You Watch For?</h2>
    <div class="tip">
      <div class="tip-icon">1</div>
      <div class="tip-text"><strong>Check the sender address carefully.</strong> Phishing emails often use lookalike domains (e.g., supp0rt@company.com instead of support@company.com).</div>
    </div>
    <div class="tip">
      <div class="tip-icon">2</div>
      <div class="tip-text"><strong>Hover before you click.</strong> Always hover over links to see the actual URL before clicking. If it looks suspicious, don't click.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">3</div>
      <div class="tip-text"><strong>Beware of urgency.</strong> Phrases like "Act now!" or "Your account will be suspended" are common pressure tactics used by attackers.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">4</div>
      <div class="tip-text"><strong>When in doubt, report it.</strong> Use the report button in your email client or contact your IT security team directly.</div>
    </div>
  </div>
  <div class="footer">
    <p>This simulation is part of your organization's security awareness program.</p>
    <p>Powered by Nivoxis</p>
  </div>
</div>
</body>
</html>`

const defaultFeedbackNL = `<!DOCTYPE html>
<html lang="nl">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Beveiligingsbewustzijn - Gesimuleerde Phishing</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; border-radius: 12px; box-shadow: 0 4px 24px rgba(0,0,0,0.1); max-width: 600px; width: 100%; overflow: hidden; }
.header { background: linear-gradient(135deg, #e74c3c, #c0392b); color: #fff; padding: 32px; text-align: center; }
.header svg { width: 64px; height: 64px; margin-bottom: 16px; }
.header h1 { font-size: 24px; margin-bottom: 8px; }
.header p { opacity: 0.9; font-size: 15px; }
.body { padding: 32px; }
.body h2 { color: #2c3e50; margin-bottom: 16px; font-size: 18px; }
.tip { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid #eee; }
.tip:last-child { border-bottom: none; }
.tip-icon { background: #fef3f2; color: #e74c3c; width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-weight: bold; }
.tip-text { color: #555; font-size: 14px; line-height: 1.5; }
.tip-text strong { color: #2c3e50; }
.footer { padding: 24px 32px; background: #f8f9fa; text-align: center; }
.footer p { color: #888; font-size: 13px; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <h1>Dit Was een Gesimuleerde Phishing E-mail</h1>
    <p>Geen zorgen — dit was een trainingsoefening van uw organisatie.</p>
  </div>
  <div class="body">
    <h2>Waar Moet U Op Letten?</h2>
    <div class="tip">
      <div class="tip-icon">1</div>
      <div class="tip-text"><strong>Controleer het afzenderadres zorgvuldig.</strong> Phishing-e-mails gebruiken vaak gelijkaardige domeinen (bijv. supp0rt@bedrijf.com in plaats van support@bedrijf.com).</div>
    </div>
    <div class="tip">
      <div class="tip-icon">2</div>
      <div class="tip-text"><strong>Beweeg voor u klikt.</strong> Beweeg altijd over links om de werkelijke URL te zien voordat u klikt. Als het er verdacht uitziet, klik dan niet.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">3</div>
      <div class="tip-text"><strong>Pas op voor urgentie.</strong> Zinnen als "Handel nu!" of "Uw account wordt opgeschort" zijn veelgebruikte druktactieken van aanvallers.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">4</div>
      <div class="tip-text"><strong>Bij twijfel, meld het.</strong> Gebruik de meldknop in uw e-mailclient of neem rechtstreeks contact op met uw IT-beveiligingsteam.</div>
    </div>
  </div>
  <div class="footer">
    <p>Deze simulatie maakt deel uit van het beveiligingsbewustzijnsprogramma van uw organisatie.</p>
    <p>Powered by Nivoxis</p>
  </div>
</div>
</body>
</html>`

const defaultFeedbackFR = `<!DOCTYPE html>
<html lang="fr">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sensibilisation - E-mail de Phishing Simulé</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; border-radius: 12px; box-shadow: 0 4px 24px rgba(0,0,0,0.1); max-width: 600px; width: 100%; overflow: hidden; }
.header { background: linear-gradient(135deg, #e74c3c, #c0392b); color: #fff; padding: 32px; text-align: center; }
.header svg { width: 64px; height: 64px; margin-bottom: 16px; }
.header h1 { font-size: 24px; margin-bottom: 8px; }
.header p { opacity: 0.9; font-size: 15px; }
.body { padding: 32px; }
.body h2 { color: #2c3e50; margin-bottom: 16px; font-size: 18px; }
.tip { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid #eee; }
.tip:last-child { border-bottom: none; }
.tip-icon { background: #fef3f2; color: #e74c3c; width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-weight: bold; }
.tip-text { color: #555; font-size: 14px; line-height: 1.5; }
.tip-text strong { color: #2c3e50; }
.footer { padding: 24px 32px; background: #f8f9fa; text-align: center; }
.footer p { color: #888; font-size: 13px; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <h1>Ceci Était un E-mail de Phishing Simulé</h1>
    <p>Pas d'inquiétude — il s'agit d'un exercice de formation de votre organisation.</p>
  </div>
  <div class="body">
    <h2>À Quoi Devez-Vous Faire Attention ?</h2>
    <div class="tip">
      <div class="tip-icon">1</div>
      <div class="tip-text"><strong>Vérifiez attentivement l'adresse de l'expéditeur.</strong> Les e-mails de phishing utilisent souvent des domaines similaires (ex : supp0rt@entreprise.com au lieu de support@entreprise.com).</div>
    </div>
    <div class="tip">
      <div class="tip-icon">2</div>
      <div class="tip-text"><strong>Survolez avant de cliquer.</strong> Passez toujours la souris sur les liens pour voir l'URL réelle avant de cliquer. Si elle semble suspecte, ne cliquez pas.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">3</div>
      <div class="tip-text"><strong>Méfiez-vous de l'urgence.</strong> Des phrases comme « Agissez maintenant ! » ou « Votre compte sera suspendu » sont des tactiques de pression courantes utilisées par les attaquants.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">4</div>
      <div class="tip-text"><strong>En cas de doute, signalez-le.</strong> Utilisez le bouton de signalement dans votre client de messagerie ou contactez directement votre équipe de sécurité informatique.</div>
    </div>
  </div>
  <div class="footer">
    <p>Cette simulation fait partie du programme de sensibilisation à la sécurité de votre organisation.</p>
    <p>Powered by Nivoxis</p>
  </div>
</div>
</body>
</html>`

const defaultFeedbackDE = `<!DOCTYPE html>
<html lang="de">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Sicherheitsbewusstsein - Simulierte Phishing-E-Mail</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; border-radius: 12px; box-shadow: 0 4px 24px rgba(0,0,0,0.1); max-width: 600px; width: 100%; overflow: hidden; }
.header { background: linear-gradient(135deg, #e74c3c, #c0392b); color: #fff; padding: 32px; text-align: center; }
.header svg { width: 64px; height: 64px; margin-bottom: 16px; }
.header h1 { font-size: 24px; margin-bottom: 8px; }
.header p { opacity: 0.9; font-size: 15px; }
.body { padding: 32px; }
.body h2 { color: #2c3e50; margin-bottom: 16px; font-size: 18px; }
.tip { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid #eee; }
.tip:last-child { border-bottom: none; }
.tip-icon { background: #fef3f2; color: #e74c3c; width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-weight: bold; }
.tip-text { color: #555; font-size: 14px; line-height: 1.5; }
.tip-text strong { color: #2c3e50; }
.footer { padding: 24px 32px; background: #f8f9fa; text-align: center; }
.footer p { color: #888; font-size: 13px; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <h1>Dies War eine Simulierte Phishing-E-Mail</h1>
    <p>Keine Sorge — dies war eine Trainingsübung Ihrer Organisation.</p>
  </div>
  <div class="body">
    <h2>Worauf Sollten Sie Achten?</h2>
    <div class="tip">
      <div class="tip-icon">1</div>
      <div class="tip-text"><strong>Überprüfen Sie die Absenderadresse sorgfältig.</strong> Phishing-E-Mails verwenden oft ähnlich aussehende Domains (z.B. supp0rt@firma.com statt support@firma.com).</div>
    </div>
    <div class="tip">
      <div class="tip-icon">2</div>
      <div class="tip-text"><strong>Fahren Sie mit der Maus über Links, bevor Sie klicken.</strong> Überprüfen Sie immer die tatsächliche URL, bevor Sie klicken. Wenn sie verdächtig aussieht, klicken Sie nicht.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">3</div>
      <div class="tip-text"><strong>Vorsicht bei Dringlichkeit.</strong> Formulierungen wie „Handeln Sie jetzt!" oder „Ihr Konto wird gesperrt" sind häufige Druckmittel von Angreifern.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">4</div>
      <div class="tip-text"><strong>Im Zweifelsfall melden.</strong> Nutzen Sie die Melde-Schaltfläche in Ihrem E-Mail-Client oder kontaktieren Sie direkt Ihr IT-Sicherheitsteam.</div>
    </div>
  </div>
  <div class="footer">
    <p>Diese Simulation ist Teil des Sicherheitsbewusstseinsprogramms Ihrer Organisation.</p>
    <p>Powered by Nivoxis</p>
  </div>
</div>
</body>
</html>`

const defaultFeedbackES = `<!DOCTYPE html>
<html lang="es">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Concienciación - Correo de Phishing Simulado</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; background: #f0f2f5; display: flex; align-items: center; justify-content: center; min-height: 100vh; padding: 20px; }
.card { background: #fff; border-radius: 12px; box-shadow: 0 4px 24px rgba(0,0,0,0.1); max-width: 600px; width: 100%; overflow: hidden; }
.header { background: linear-gradient(135deg, #e74c3c, #c0392b); color: #fff; padding: 32px; text-align: center; }
.header svg { width: 64px; height: 64px; margin-bottom: 16px; }
.header h1 { font-size: 24px; margin-bottom: 8px; }
.header p { opacity: 0.9; font-size: 15px; }
.body { padding: 32px; }
.body h2 { color: #2c3e50; margin-bottom: 16px; font-size: 18px; }
.tip { display: flex; align-items: flex-start; gap: 12px; padding: 12px 0; border-bottom: 1px solid #eee; }
.tip:last-child { border-bottom: none; }
.tip-icon { background: #fef3f2; color: #e74c3c; width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; flex-shrink: 0; font-weight: bold; }
.tip-text { color: #555; font-size: 14px; line-height: 1.5; }
.tip-text strong { color: #2c3e50; }
.footer { padding: 24px 32px; background: #f8f9fa; text-align: center; }
.footer p { color: #888; font-size: 13px; }
</style>
</head>
<body>
<div class="card">
  <div class="header">
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>
    <h1>Este Fue un Correo de Phishing Simulado</h1>
    <p>No se preocupe — fue un ejercicio de formación de su organización.</p>
  </div>
  <div class="body">
    <h2>¿Qué Debe Observar?</h2>
    <div class="tip">
      <div class="tip-icon">1</div>
      <div class="tip-text"><strong>Revise cuidadosamente la dirección del remitente.</strong> Los correos de phishing suelen usar dominios similares (ej: soport3@empresa.com en lugar de soporte@empresa.com).</div>
    </div>
    <div class="tip">
      <div class="tip-icon">2</div>
      <div class="tip-text"><strong>Pase el ratón antes de hacer clic.</strong> Siempre verifique la URL real antes de hacer clic. Si parece sospechosa, no haga clic.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">3</div>
      <div class="tip-text"><strong>Cuidado con la urgencia.</strong> Frases como "¡Actúe ahora!" o "Su cuenta será suspendida" son tácticas de presión comunes utilizadas por los atacantes.</div>
    </div>
    <div class="tip">
      <div class="tip-icon">4</div>
      <div class="tip-text"><strong>En caso de duda, repórtelo.</strong> Use el botón de reporte en su cliente de correo o contacte directamente a su equipo de seguridad informática.</div>
    </div>
  </div>
  <div class="footer">
    <p>Esta simulación forma parte del programa de concienciación de seguridad de su organización.</p>
    <p>Powered by Nivoxis</p>
  </div>
</div>
</body>
</html>`

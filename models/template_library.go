package models

// LibraryTemplate represents a pre-built phishing template available for import.
type LibraryTemplate struct {
	Slug            string `json:"slug"`
	Name            string `json:"name"`
	Category        string `json:"category"`
	DifficultyLevel int    `json:"difficulty_level"`
	Description     string `json:"description"`
	Subject         string `json:"subject"`
	Text            string `json:"text"`
	HTML            string `json:"html"`
	EnvelopeSender  string `json:"envelope_sender"`
	Language        string `json:"language"`
	TargetRole      string `json:"target_role"`
}

// TemplateLibrary is the built-in collection of ready-to-use phishing templates.
var TemplateLibrary = []LibraryTemplate{
	// ─── CREDENTIAL HARVESTING (Easy → Hard) ──────────────────────
	{
		Slug:            "password-expiry",
		Name:            "Password Expiry Notice",
		Category:        "Credential Harvesting",
		DifficultyLevel: 1,
		Description:     "IT department warns the user their password is about to expire.",
		Subject:         "Action Required: Your Password Expires in 24 Hours",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

Your {{.OrgName}} account password will expire in 24 hours. To avoid losing access to your email, files, and applications you must reset your password now.

Reset your password: {{.URL}}

If you do not reset your password before the deadline, your account will be locked and you will need to contact the IT helpdesk.

Thank you,
{{.OrgName}} IT Support`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>Your <strong>{{.OrgName}}</strong> account password will expire in <strong>24 hours</strong>. To avoid losing access to your email, files, and applications you must reset your password immediately.</p>
<p style="text-align:center;margin:30px 0">
<a href="{{.URL}}" style="background:{{.OrgColor}};color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold">Reset Password Now</a>
</p>
<p style="color:#888;font-size:12px">If you do not reset your password before the deadline, your account will be locked.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
{{.OrgName}} IT Support &bull; This is an automated message
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "office365-login",
		Name:            "Microsoft 365 Sign-In Alert",
		Category:        "Credential Harvesting",
		DifficultyLevel: 2,
		Description:     "Spoofed Microsoft 365 unusual sign-in activity alert.",
		Subject:         "Unusual sign-in activity on your account",
		Language:        "en",
		TargetRole:      "all",
		Text: `Microsoft account
Unusual sign-in activity

We detected something unusual about a recent sign-in to your Microsoft account.

Sign-in details:
  Country/region: Russia
  IP address: 185.220.101.xx
  Date: {{.Date}}
  Platform: Linux

If this was you, you can safely ignore this message.
If this wasn't you, please secure your account: {{.URL}}

Thanks,
The Microsoft account team`,
		HTML: `<div style="font-family:Segoe UI,Arial,sans-serif;max-width:600px;margin:0 auto;background:#fff">
<div style="padding:20px 30px;border-bottom:1px solid #e0e0e0">
<img src="https://img-prod-cms-rt-microsoft-com.akamaized.net/cms/api/am/imageFileData/RE1Mu3b?ver=5c31" alt="Microsoft" style="height:24px" onerror="this.style.display='none'">
</div>
<div style="padding:30px">
<h2 style="font-weight:400;font-size:20px">Unusual sign-in activity</h2>
<p>We detected something unusual about a recent sign-in to your Microsoft account <strong>{{.Email}}</strong>.</p>
<table style="width:100%;border-collapse:collapse;margin:20px 0">
<tr><td style="padding:8px 0;color:#666">Country/region:</td><td style="padding:8px 0"><strong>Russia</strong></td></tr>
<tr><td style="padding:8px 0;color:#666">IP address:</td><td style="padding:8px 0">185.220.101.xx</td></tr>
<tr><td style="padding:8px 0;color:#666">Platform:</td><td style="padding:8px 0">Linux</td></tr>
</table>
<p>If this wasn't you, please secure your account immediately.</p>
<p><a href="{{.URL}}" style="background:#0078d4;color:#fff;padding:10px 24px;text-decoration:none;border-radius:2px;display:inline-block;font-size:14px">Review recent activity</a></p>
<p style="font-size:12px;color:#666;margin-top:30px">Thanks,<br>The Microsoft account team</p>
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "google-security-alert",
		Name:            "Google Security Alert",
		Category:        "Credential Harvesting",
		DifficultyLevel: 2,
		Description:     "Spoofed Google critical security alert for new device sign-in.",
		Subject:         "Critical security alert for {{.Email}}",
		Language:        "en",
		TargetRole:      "all",
		Text: `Google

Someone just used your password to try to sign in to your account.

{{.Email}}

Details:
  New device sign-in
  Windows, Chrome
  Location: Unknown

Google stopped this sign-in attempt. Review your account activity now: {{.URL}}

You received this email to let you know about important changes to your Google Account and services.`,
		HTML: `<div style="font-family:Google Sans,Roboto,Arial,sans-serif;max-width:500px;margin:0 auto;background:#fff;border:1px solid #dadce0;border-radius:8px;overflow:hidden">
<div style="padding:30px 30px 0;text-align:center">
<svg viewBox="0 0 74 24" width="74" style="display:inline-block"><path fill="#4285F4" d="M9.24 8.19v2.46h5.88c-.18 1.38-.64 2.39-1.34 3.1-.86.86-2.2 1.8-4.54 1.8-3.62 0-6.45-2.92-6.45-6.54s2.83-6.54 6.45-6.54c1.95 0 3.38.77 4.43 1.76L15.4 2.5C13.94 1.08 11.98 0 9.24 0 4.28 0 .11 4.04.11 9s4.17 9 9.13 9c2.68 0 4.7-.88 6.28-2.52 1.62-1.62 2.13-3.91 2.13-5.75 0-.57-.04-1.1-.13-1.54H9.24z"></path></svg>
</div>
<div style="padding:20px 30px 30px;text-align:center">
<div style="width:48px;height:48px;background:#ea4335;border-radius:50%;margin:0 auto 15px;line-height:48px;color:#fff;font-size:24px">!</div>
<h2 style="font-size:18px;font-weight:400;margin:0 0 10px">Someone has your password</h2>
<p style="color:#5f6368;font-size:14px">Someone just used your password to try to sign in to your account <strong>{{.Email}}</strong>.</p>
<p style="color:#5f6368;font-size:14px">Google blocked the sign-in attempt. You should change your password now.</p>
<p style="margin:25px 0"><a href="{{.URL}}" style="background:#1a73e8;color:#fff;padding:10px 24px;border-radius:4px;text-decoration:none;font-size:14px">Check activity</a></p>
</div>
</div>{{.Tracker}}`,
	},
	// ─── BUSINESS EMAIL COMPROMISE ────────────────────────────────
	{
		Slug:            "ceo-wire-transfer",
		Name:            "CEO Wire Transfer Request",
		Category:        "Business Email Compromise",
		DifficultyLevel: 3,
		Description:     "Spoofed CEO urgently requesting a confidential wire transfer.",
		Subject:         "Urgent - Confidential",
		Language:        "en",
		TargetRole:      "finance",
		Text: `{{.FirstName}},

I need you to process a wire transfer today. This is time-sensitive and confidential — please do not discuss with anyone else until it's completed.

Amount: €47,500.00
Beneficiary: Meridian Partners Ltd
Purpose: Consulting retainer (Q2)

I'm heading into a board meeting and won't be reachable by phone for the next few hours. Please confirm once the transfer is initiated.

Click here for the payment details: {{.URL}}

Thanks,
CEO`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;padding:20px">
<p>{{.FirstName}},</p>
<p>I need you to process a wire transfer today. This is time-sensitive and <strong>confidential</strong> — please do not discuss with anyone else until it's completed.</p>
<table style="margin:20px 0;border-collapse:collapse">
<tr><td style="padding:5px 15px 5px 0;color:#666">Amount:</td><td style="padding:5px 0"><strong>€47,500.00</strong></td></tr>
<tr><td style="padding:5px 15px 5px 0;color:#666">Beneficiary:</td><td style="padding:5px 0">Meridian Partners Ltd</td></tr>
<tr><td style="padding:5px 15px 5px 0;color:#666">Purpose:</td><td style="padding:5px 0">Consulting retainer (Q2)</td></tr>
</table>
<p>I'm heading into a board meeting and won't be reachable by phone for the next few hours. Please confirm once the transfer is initiated.</p>
<p><a href="{{.URL}}" style="color:#0066cc">Click here for the payment details</a></p>
<p>Thanks,<br><strong>CEO</strong></p>
<p style="font-size:11px;color:#999">Sent from my iPhone</p>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "vendor-invoice",
		Name:            "Vendor Invoice Payment",
		Category:        "Business Email Compromise",
		DifficultyLevel: 2,
		Description:     "Fake vendor sends an overdue invoice requesting immediate payment.",
		Subject:         "Invoice #INV-2024-8847 — Payment Overdue",
		Language:        "en",
		TargetRole:      "finance",
		Text: `Dear {{.FirstName}},

Please find attached the overdue invoice #INV-2024-8847 for services rendered.

The payment of €12,340.00 was due on 2024-03-15. Please process this payment at your earliest convenience to avoid any late fees.

View and download your invoice: {{.URL}}

If you have any questions regarding this invoice, please don't hesitate to contact our accounts receivable team.

Best regards,
Sarah Mitchell
Accounts Receivable
Apex Business Solutions`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#1a1a2e;padding:20px;text-align:center">
<h2 style="color:#fff;margin:0;font-size:18px">APEX BUSINESS SOLUTIONS</h2>
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>Please find below the details for overdue invoice <strong>#INV-2024-8847</strong>.</p>
<div style="background:#fff3cd;border:1px solid #ffc107;padding:12px;border-radius:4px;margin:20px 0">
<strong>⚠ Payment Overdue</strong> — This invoice was due on March 15, 2024.
</div>
<table style="width:100%;border-collapse:collapse;margin:20px 0">
<tr style="border-bottom:1px solid #eee"><td style="padding:10px 0;color:#666">Invoice Number</td><td style="padding:10px 0;text-align:right">#INV-2024-8847</td></tr>
<tr style="border-bottom:1px solid #eee"><td style="padding:10px 0;color:#666">Amount Due</td><td style="padding:10px 0;text-align:right;font-weight:bold;color:#dc3545">€12,340.00</td></tr>
<tr><td style="padding:10px 0;color:#666">Service</td><td style="padding:10px 0;text-align:right">IT Consulting — Q1 2024</td></tr>
</table>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:#1a1a2e;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">View Invoice &amp; Pay</a>
</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
Apex Business Solutions &bull; 42 Commerce Street, Amsterdam
</div>
</div>{{.Tracker}}`,
	},
	// ─── DELIVERY / PACKAGE NOTIFICATIONS ─────────────────────────
	{
		Slug:            "package-delivery",
		Name:            "Package Delivery Failed",
		Category:        "Delivery Notification",
		DifficultyLevel: 1,
		Description:     "Fake courier notification about a failed package delivery attempt.",
		Subject:         "Delivery Attempt Failed — Action Required",
		Language:        "en",
		TargetRole:      "all",
		Text: `Hello {{.FirstName}},

We attempted to deliver your package today but were unable to complete the delivery.

Tracking Number: NL8847291035
Delivery Attempt: April 2, 2024 at 14:32

To reschedule your delivery, please visit: {{.URL}}

If we do not hear from you within 48 hours, the package will be returned to sender.

PostNL Customer Service`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#ff6200;padding:15px 20px;text-align:center">
<h2 style="color:#fff;margin:0;font-size:20px">PostNL</h2>
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<h3 style="color:#333;margin-top:0">Delivery Attempt Failed</h3>
<p>Hello {{.FirstName}},</p>
<p>We attempted to deliver your package today but were unable to complete the delivery.</p>
<div style="background:#f8f9fa;border-radius:6px;padding:15px;margin:20px 0">
<table style="width:100%">
<tr><td style="padding:4px 0;color:#666">Tracking Number:</td><td><strong>NL8847291035</strong></td></tr>
<tr><td style="padding:4px 0;color:#666">Status:</td><td style="color:#dc3545"><strong>Delivery Failed</strong></td></tr>
<tr><td style="padding:4px 0;color:#666">Attempt Date:</td><td>April 2, 2024 at 14:32</td></tr>
</table>
</div>
<p style="text-align:center">
<a href="{{.URL}}" style="background:#ff6200;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;display:inline-block">Reschedule Delivery</a>
</p>
<p style="font-size:12px;color:#888">If we do not hear from you within 48 hours, the package will be returned to sender.</p>
</div>
</div>{{.Tracker}}`,
	},
	// ─── IT / HELPDESK ────────────────────────────────────────────
	{
		Slug:            "mfa-enrollment",
		Name:            "Mandatory MFA Enrollment",
		Category:        "IT Helpdesk",
		DifficultyLevel: 2,
		Description:     "IT security team requires mandatory MFA enrollment by deadline.",
		Subject:         "Action Required: Mandatory MFA Enrollment by Friday",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

As part of our ongoing security improvements, all {{.OrgName}} employees are now required to enroll in Multi-Factor Authentication (MFA) by this Friday.

Failure to enroll will result in your account being temporarily suspended.

Enroll now: {{.URL}}

This process takes approximately 2 minutes. You will need your mobile phone to complete the setup.

If you have already enrolled, please disregard this message.

Best regards,
{{.OrgName}} Information Security Team`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<div style="background:#fff3cd;border-left:4px solid #ffc107;padding:12px 15px;margin-bottom:20px">
<strong>⚠ Deadline: This Friday</strong>
</div>
<p>Dear {{.FirstName}},</p>
<p>As part of our ongoing security improvements, all <strong>{{.OrgName}}</strong> employees are now required to enroll in <strong>Multi-Factor Authentication (MFA)</strong>.</p>
<p>Failure to enroll will result in your account being <strong>temporarily suspended</strong>.</p>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:{{.OrgColor}};color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold">Enroll in MFA Now</a>
</p>
<p style="font-size:13px;color:#666">This process takes approximately 2 minutes. You will need your mobile phone.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
{{.OrgName}} Information Security Team
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "shared-document",
		Name:            "Shared Document Notification",
		Category:        "IT Helpdesk",
		DifficultyLevel: 1,
		Description:     "Colleague shared a document via OneDrive/SharePoint requiring sign-in.",
		Subject:         "{{.From}} shared a document with you",
		Language:        "en",
		TargetRole:      "all",
		Text: `{{.From}} shared a file with you.

"Q2 Budget Review - Final.xlsx"

Open the document: {{.URL}}

This link will work for anyone at {{.OrgName}}.

Microsoft OneDrive`,
		HTML: `<div style="font-family:Segoe UI,Arial,sans-serif;max-width:520px;margin:0 auto;background:#fff;border:1px solid #e0e0e0">
<div style="padding:20px 25px;border-bottom:1px solid #e0e0e0">
<span style="font-size:16px;color:#0078d4">OneDrive</span>
</div>
<div style="padding:25px">
<div style="display:flex;align-items:center;margin-bottom:20px">
<div style="width:40px;height:40px;border-radius:50%;background:#0078d4;color:#fff;text-align:center;line-height:40px;font-size:18px;margin-right:12px">{{.FirstName | printf "%.1s"}}</div>
<div>
<strong>{{.From}}</strong> shared a file with you
</div>
</div>
<div style="background:#f3f2f1;border-radius:4px;padding:15px;margin:15px 0">
<span style="color:#0078d4;font-size:14px">📄 Q2 Budget Review - Final.xlsx</span>
</div>
<p style="text-align:center;margin:20px 0">
<a href="{{.URL}}" style="background:#0078d4;color:#fff;padding:8px 20px;border-radius:2px;text-decoration:none;font-size:14px">Open</a>
</p>
<p style="font-size:12px;color:#888">This link will work for anyone at {{.OrgName}}.</p>
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "software-update",
		Name:            "Mandatory Software Update",
		Category:        "IT Helpdesk",
		DifficultyLevel: 2,
		Description:     "IT department requires immediate software update for security patch.",
		Subject:         "Critical Security Update Required — Install by End of Day",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

A critical security vulnerability has been identified in our VPN client software. All employees must install the security patch by end of business today.

Affected software: GlobalProtect VPN Client v5.2.x
Severity: Critical
Patch: GP-2024-0412

Download and install the update: {{.URL}}

This update is mandatory and failure to install may result in loss of VPN access.

Thank you for your prompt attention,
{{.OrgName}} IT Security`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<div style="background:#f8d7da;border-left:4px solid #dc3545;padding:12px 15px;margin-bottom:20px">
<strong>🔴 Critical Security Patch — Install Today</strong>
</div>
<p>Dear {{.FirstName}},</p>
<p>A critical security vulnerability has been identified in our VPN client software. All employees must install the security patch by <strong>end of business today</strong>.</p>
<table style="width:100%;margin:20px 0;border-collapse:collapse">
<tr><td style="padding:6px 0;color:#666">Affected Software:</td><td>GlobalProtect VPN Client v5.2.x</td></tr>
<tr><td style="padding:6px 0;color:#666">Severity:</td><td style="color:#dc3545"><strong>Critical</strong></td></tr>
<tr><td style="padding:6px 0;color:#666">Patch ID:</td><td>GP-2024-0412</td></tr>
</table>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:#dc3545;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold">Download Security Patch</a>
</p>
<p style="font-size:12px;color:#888">Failure to install may result in loss of VPN access.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
{{.OrgName}} IT Security
</div>
</div>{{.Tracker}}`,
	},
	// ─── HR / PAYROLL ─────────────────────────────────────────────
	{
		Slug:            "payroll-update",
		Name:            "Payroll Information Update",
		Category:        "HR / Payroll",
		DifficultyLevel: 2,
		Description:     "HR requests employees to verify payroll information for upcoming cycle.",
		Subject:         "Action Required: Verify Your Payroll Details",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

As part of our annual payroll verification process, we need all employees to confirm their banking and tax information before the next pay cycle.

Please log in to the HR portal and verify your details by end of week: {{.URL}}

If your information is not verified in time, your salary payment may be delayed.

Best regards,
Human Resources
{{.OrgName}}`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<h3 style="margin-top:0">Annual Payroll Verification</h3>
<p>Dear {{.FirstName}},</p>
<p>As part of our annual payroll verification process, we need all employees to confirm their banking and tax information before the next pay cycle.</p>
<p>Please log in to the HR portal and verify your details <strong>by end of this week</strong>.</p>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:{{.OrgColor}};color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold">Verify Payroll Details</a>
</p>
<div style="background:#fff3cd;padding:10px 15px;border-radius:4px;font-size:13px">
⚠ If your information is not verified in time, your salary payment may be delayed.
</div>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
Human Resources &bull; {{.OrgName}}
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "bonus-notification",
		Name:            "Annual Bonus Notification",
		Category:        "HR / Payroll",
		DifficultyLevel: 3,
		Description:     "HR notifies employee about their annual performance bonus — high click rate.",
		Subject:         "Your Annual Performance Bonus Has Been Approved",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

We are pleased to inform you that your annual performance bonus has been approved for the current fiscal year.

To view the details of your bonus, including the amount and payment schedule, please log in to the compensation portal: {{.URL}}

This information is strictly confidential. Please do not share or forward this email.

Congratulations on your hard work.

Best regards,
Compensation & Benefits
{{.OrgName}} Human Resources`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<h3 style="margin-top:0;color:#28a745">🎉 Performance Bonus Approved</h3>
<p>Dear {{.FirstName}},</p>
<p>We are pleased to inform you that your <strong>annual performance bonus</strong> has been approved for the current fiscal year.</p>
<p>To view the details of your bonus, including the amount and payment schedule, please log in to the compensation portal.</p>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:#28a745;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px;font-weight:bold">View Bonus Details</a>
</p>
<p style="font-size:12px;color:#dc3545"><strong>Confidential:</strong> Please do not share or forward this email.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
Compensation &amp; Benefits &bull; {{.OrgName}} Human Resources
</div>
</div>{{.Tracker}}`,
	},
	// ─── SOCIAL ENGINEERING ───────────────────────────────────────
	{
		Slug:            "linkedin-connection",
		Name:            "LinkedIn Connection Request",
		Category:        "Social Engineering",
		DifficultyLevel: 1,
		Description:     "Fake LinkedIn connection request from a relevant industry contact.",
		Subject:         "You have a new connection request",
		Language:        "en",
		TargetRole:      "all",
		Text: `LinkedIn

Hi {{.FirstName}},

David Müller, Senior VP at CyberTech Solutions, would like to connect with you on LinkedIn.

"Hi {{.FirstName}}, I came across your profile and I'm impressed with your work at {{.OrgName}}. I'd love to connect and discuss potential collaboration opportunities."

Accept invitation: {{.URL}}

View David's profile: {{.URL}}

You are receiving LinkedIn notification emails.
Unsubscribe: {{.URL}}`,
		HTML: `<div style="font-family:-apple-system,system-ui,BlinkMacSystemFont,Segoe UI,Roboto,sans-serif;max-width:520px;margin:0 auto;background:#fff;border:1px solid #e0e0e0">
<div style="background:#0a66c2;padding:12px 20px">
<span style="color:#fff;font-size:22px;font-weight:bold">in</span>
</div>
<div style="padding:25px">
<div style="text-align:center;margin-bottom:20px">
<div style="width:72px;height:72px;border-radius:50%;background:#ddd;margin:0 auto 10px;line-height:72px;font-size:28px;color:#666">DM</div>
<h3 style="margin:0;font-size:16px">David Müller</h3>
<p style="color:#666;font-size:13px;margin:4px 0">Senior VP at CyberTech Solutions</p>
</div>
<div style="background:#f3f6f8;border-radius:8px;padding:15px;margin:15px 0;font-style:italic;font-size:14px;color:#333">
"Hi {{.FirstName}}, I came across your profile and I'm impressed with your work at {{.OrgName}}. I'd love to connect and discuss potential collaboration opportunities."
</div>
<p style="text-align:center;margin:20px 0">
<a href="{{.URL}}" style="background:#0a66c2;color:#fff;padding:8px 24px;border-radius:20px;text-decoration:none;font-size:14px;font-weight:bold">Accept</a>
<a href="{{.URL}}" style="color:#0a66c2;padding:8px 24px;border-radius:20px;text-decoration:none;font-size:14px;border:1px solid #0a66c2;margin-left:8px">Ignore</a>
</p>
</div>
<div style="padding:12px 20px;border-top:1px solid #e0e0e0;font-size:11px;color:#888;text-align:center">
This email was sent to {{.Email}}. <a href="{{.URL}}" style="color:#0a66c2">Unsubscribe</a>
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "voicemail-notification",
		Name:            "Voicemail Notification",
		Category:        "Social Engineering",
		DifficultyLevel: 2,
		Description:     "Fake Microsoft Teams voicemail notification with audio playback link.",
		Subject:         "You have a new voicemail from +31 20 555 0142",
		Language:        "en",
		TargetRole:      "all",
		Text: `Microsoft Teams

You received a voicemail

Caller: +31 20 555 0142
Duration: 0:47
Date: Today

Play voicemail: {{.URL}}

This message was sent to {{.Email}} by Microsoft Teams.`,
		HTML: `<div style="font-family:Segoe UI,Arial,sans-serif;max-width:520px;margin:0 auto;background:#fff;border:1px solid #e0e0e0">
<div style="padding:20px 25px;border-bottom:1px solid #e0e0e0">
<span style="font-size:16px;font-weight:600;color:#464775">Microsoft Teams</span>
</div>
<div style="padding:25px">
<h3 style="font-weight:400;font-size:18px;margin-top:0">You have a new voicemail</h3>
<div style="background:#f5f5f5;border-radius:8px;padding:15px;margin:20px 0">
<table style="width:100%">
<tr><td style="padding:4px 0;color:#666;width:80px">Caller:</td><td><strong>+31 20 555 0142</strong></td></tr>
<tr><td style="padding:4px 0;color:#666">Duration:</td><td>0:47</td></tr>
<tr><td style="padding:4px 0;color:#666">Date:</td><td>Today</td></tr>
</table>
</div>
<p style="text-align:center;margin:25px 0">
<a href="{{.URL}}" style="background:#464775;color:#fff;padding:10px 24px;border-radius:4px;text-decoration:none;font-size:14px">▶ Play Voicemail</a>
</p>
</div>
<div style="padding:12px 20px;border-top:1px solid #e0e0e0;font-size:11px;color:#888">
This message was sent to {{.Email}} by Microsoft Teams.
</div>
</div>{{.Tracker}}`,
	},
	// ─── QR CODE PHISHING ─────────────────────────────────────────
	{
		Slug:            "wifi-qr-poster",
		Name:            "Office Wi-Fi QR Code",
		Category:        "QR Code Phishing",
		DifficultyLevel: 3,
		Description:     "Internal memo with QR code for 'new office Wi-Fi' that leads to credential harvesting.",
		Subject:         "New Office Wi-Fi Network — Scan to Connect",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

We have upgraded the office Wi-Fi network for improved speed and security. To connect to the new network, please scan the QR code in the email or visit the link below.

Connect to new Wi-Fi: {{.URL}}

You will need to authenticate with your {{.OrgName}} credentials to complete the setup.

Thank you,
{{.OrgName}} Facilities & IT`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:{{.OrgColor}};padding:20px;text-align:center">
{{if .OrgLogo}}<img src="{{.OrgLogo}}" alt="{{.OrgName}}" style="max-height:50px">{{else}}<h2 style="color:#fff;margin:0">{{.OrgName}}</h2>{{end}}
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0;text-align:center">
<h3 style="margin-top:0">📶 New Office Wi-Fi Network</h3>
<p>We have upgraded our office Wi-Fi for improved speed and security. Scan the QR code below with your phone to connect.</p>
<div style="margin:25px auto">{{.QRCode}}</div>
<p style="font-size:13px;color:#666">Or <a href="{{.URL}}" style="color:{{.OrgColor}}">click here</a> to connect from your laptop.</p>
<p style="font-size:12px;color:#888;margin-top:20px">You will need to authenticate with your {{.OrgName}} credentials.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">
{{.OrgName}} Facilities &amp; IT
</div>
</div>{{.Tracker}}`,
	},
	{
		Slug:            "parking-qr",
		Name:            "Parking Validation QR Code",
		Category:        "QR Code Phishing",
		DifficultyLevel: 3,
		Description:     "Building management asks employees to register for digital parking via QR code.",
		Subject:         "Register for Digital Parking — Scan QR Code",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

Starting next month, the office parking garage will switch to a digital validation system. Please register your vehicle by scanning the QR code in the email below.

Register your vehicle: {{.URL}}

You will need your employee ID and license plate number.

Best regards,
Building Management`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#2c3e50;padding:20px;text-align:center">
<h2 style="color:#fff;margin:0">🅿 Parking Registration</h2>
</div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0;text-align:center">
<p style="text-align:left">Dear {{.FirstName}},</p>
<p style="text-align:left">Starting next month, the office parking garage will switch to a <strong>digital validation system</strong>. Please register your vehicle by scanning the QR code below.</p>
<div style="margin:25px auto">{{.QRCode}}</div>
<p style="font-size:13px;color:#666">Or <a href="{{.URL}}">click here</a> to register online.</p>
<p style="font-size:12px;color:#888">You will need your employee ID and license plate number.</p>
</div>
<div style="padding:15px;text-align:center;font-size:11px;color:#aaa">Building Management</div>
</div>{{.Tracker}}`,
	},
	// ─── SMS / SMISHING TEMPLATES ─────────────────────────────────
	{
		Slug:            "sms-package-delivery",
		Name:            "[SMS] Package Delivery",
		Category:        "SMS Phishing",
		DifficultyLevel: 1,
		Description:     "SMS notification about a package requiring address confirmation.",
		Subject:         "",
		Language:        "en",
		TargetRole:      "all",
		Text:            `PostNL: We were unable to deliver your package. Please confirm your address to reschedule: {{.URL}}`,
		HTML:            "",
	},
	{
		Slug:            "sms-bank-alert",
		Name:            "[SMS] Bank Security Alert",
		Category:        "SMS Phishing",
		DifficultyLevel: 2,
		Description:     "SMS alert about suspicious bank transaction requiring verification.",
		Subject:         "",
		Language:        "en",
		TargetRole:      "all",
		Text:            `ING Alert: Suspicious transaction of €847.00 detected on your account. If this wasn't you, verify now: {{.URL}}`,
		HTML:            "",
	},
	{
		Slug:            "sms-it-mfa",
		Name:            "[SMS] IT MFA Reset",
		Category:        "SMS Phishing",
		DifficultyLevel: 2,
		Description:     "SMS from IT department about MFA token expiration.",
		Subject:         "",
		Language:        "en",
		TargetRole:      "all",
		Text:            `{{.OrgName}} IT: Your MFA token has expired. Re-enroll immediately to maintain access: {{.URL}}`,
		HTML:            "",
	},
}

// GetTemplateLibrary returns all pre-built templates, optionally filtered by
// category and difficulty. Use GetTemplateLibraryFiltered for additional filters.
func GetTemplateLibrary(category string, difficulty int) []LibraryTemplate {
	return GetTemplateLibraryFiltered(category, difficulty, "")
}

// GetTemplateLibraryFiltered returns templates matching all non-empty filters.
func GetTemplateLibraryFiltered(category string, difficulty int, language string) []LibraryTemplate {
	if category == "" && difficulty == 0 && language == "" {
		return TemplateLibrary
	}
	var filtered []LibraryTemplate
	for _, t := range TemplateLibrary {
		if category != "" && t.Category != category {
			continue
		}
		if difficulty > 0 && t.DifficultyLevel != difficulty {
			continue
		}
		if language != "" && t.Language != language {
			continue
		}
		filtered = append(filtered, t)
	}
	return filtered
}

// GetTemplateLibraryStats returns a summary of the library contents.
type TemplateLibraryStats struct {
	TotalTemplates int            `json:"total_templates"`
	Categories     int            `json:"categories"`
	Languages      int            `json:"languages"`
	ByCategory     map[string]int `json:"by_category"`
	ByDifficulty   map[int]int    `json:"by_difficulty"`
	ByLanguage     map[string]int `json:"by_language"`
}

func GetTemplateLibraryStats() TemplateLibraryStats {
	stats := TemplateLibraryStats{
		TotalTemplates: len(TemplateLibrary),
		ByCategory:     make(map[string]int),
		ByDifficulty:   make(map[int]int),
		ByLanguage:     make(map[string]int),
	}
	cats := map[string]bool{}
	langs := map[string]bool{}
	for _, t := range TemplateLibrary {
		stats.ByCategory[t.Category]++
		stats.ByDifficulty[t.DifficultyLevel]++
		stats.ByLanguage[t.Language]++
		cats[t.Category] = true
		langs[t.Language] = true
	}
	stats.Categories = len(cats)
	stats.Languages = len(langs)
	return stats
}

// GetLibraryTemplate returns a single library template by slug.
func GetLibraryTemplate(slug string) (LibraryTemplate, bool) {
	for _, t := range TemplateLibrary {
		if t.Slug == slug {
			return t, true
		}
	}
	return LibraryTemplate{}, false
}

// GetTemplateLibraryCategories returns the distinct categories in the library.
func GetTemplateLibraryCategories() []string {
	seen := map[string]bool{}
	var cats []string
	for _, t := range TemplateLibrary {
		if !seen[t.Category] {
			seen[t.Category] = true
			cats = append(cats, t.Category)
		}
	}
	return cats
}

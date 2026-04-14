package models

// Compliance framework name constants used in content library entries.
const (
	compCyberEssentials = "Cyber Essentials"
)

// Common quiz distractor options.
const (
	quizDistractorEncryption = "A type of encryption"
)

// builtInContentLibrary is the Nivoxis built-in microlearning content library.
// Each entry is a complete self-contained training session with pages and a quiz.
// Organized by the 4 academy tiers: Bronze (1), Silver (2), Gold (3), Platinum (4).
var builtInContentLibrary = []BuiltInTrainingContent{

	// ═══════════════════════════════════════════════════════════════════════
	// BRONZE TIER — Fundamentals (Difficulty 1)
	// ═══════════════════════════════════════════════════════════════════════

	{
		Slug:             "phishing-101",
		Title:            "Phishing 101 — Recognizing Phishing Emails",
		Category:         ContentCategoryPhishing,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Learn what phishing is, how attackers craft deceptive emails, and the red flags that reveal a fake message.",
		EstimatedMinutes: 8,
		Tags:             []string{"phishing", "email", "beginner"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST", compCyberEssentials},
		NanolearningTip:  "Always hover over links before clicking. If the URL doesn't match the sender's domain, it's likely phishing.",
		Pages: []TrainingPage{
			{Title: "What Is Phishing?", Body: "Phishing is a cyberattack where criminals send deceptive messages — usually emails — pretending to be someone you trust (your bank, your employer, a delivery service). Their goal is to trick you into clicking a malicious link, downloading malware, or sharing sensitive information like passwords or credit card numbers.\n\n**Key Fact:** Over 90% of data breaches start with a phishing email.", TipBox: "Phishing isn't limited to email — it also happens via SMS (smishing), phone calls (vishing), and social media."},
			{Title: "Anatomy of a Phishing Email", Body: "A typical phishing email contains several common elements:\n\n1. **Spoofed sender** — The 'From' address looks legitimate but doesn't match the real domain\n2. **Urgency or fear** — \"Your account will be locked in 24 hours!\"\n3. **Generic greeting** — \"Dear Customer\" instead of your name\n4. **Suspicious link** — The displayed text says one thing, but the actual URL goes somewhere else\n5. **Unexpected attachment** — An invoice, shipping notice, or document you didn't expect", TipBox: "Hover over any link before clicking it. The real URL appears in the bottom-left corner of your browser."},
			{Title: "Common Phishing Scenarios", Body: "Attackers use scenarios designed to create urgency:\n\n- **Password reset requests** — \"Your password has expired\"\n- **Package delivery notifications** — \"Your parcel couldn't be delivered\"\n- **Invoice or payment alerts** — \"Payment of €2,499 is overdue\"\n- **IT department messages** — \"Update your email settings now\"\n- **Prize notifications** — \"You've won a gift card!\"\n\nAll of these try to make you act quickly without thinking."},
			{Title: "5 Red Flags to Spot a Phish", Body: "Before clicking anything, check for these warning signs:\n\n🚩 **Mismatched URLs** — Hover to reveal the real destination\n🚩 **Spelling & grammar errors** — Professional companies proofread their emails\n🚩 **Sense of urgency** — \"Act NOW or lose access!\"\n🚩 **Request for sensitive data** — No legitimate company asks for passwords via email\n🚩 **Unfamiliar sender** — Check the actual email address, not just the display name", TipBox: "When in doubt, contact the supposed sender directly using a phone number or URL you already know — never use the contact info from the suspicious email."},
			{Title: "What To Do If You Suspect Phishing", Body: "If you receive a suspicious email:\n\n1. **Don't click** any links or open attachments\n2. **Don't reply** to the sender\n3. **Report it** using the Report Button in your email client\n4. **Delete it** from your inbox\n5. **If you already clicked**, immediately change your password and notify your IT team\n\nReporting suspicious emails helps protect your entire organization.", TipBox: "Your IT team would rather receive 100 false reports than miss 1 real phishing attack."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is the primary goal of a phishing email?", Options: []string{"To sell you a product", "To trick you into revealing sensitive information", "To test your email security settings", "To update your software"}, CorrectOption: 1},
				{QuestionText: "Which of the following is a red flag in a suspicious email?", Options: []string{"The email addresses you by your full name", "The email contains a company logo", "The sender's email address doesn't match the company domain", "The email was sent during business hours"}, CorrectOption: 2},
				{QuestionText: "What should you do if you receive a suspected phishing email?", Options: []string{"Reply to ask if it's legitimate", "Forward it to all colleagues as a warning", "Report it using the Report Button and delete it", "Click the link to check where it goes"}, CorrectOption: 2},
				{QuestionText: "Which technique do phishing emails commonly use?", Options: []string{"Creating a sense of urgency", "Providing detailed technical support", "Asking you to visit the office", "Requesting a phone call"}, CorrectOption: 0},
			},
		},
	},

	{
		Slug:             "password-security-basics",
		Title:            "Password Security — Creating Strong Passwords",
		Category:         ContentCategoryPasswords,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Understand why password security matters, how to create strong passwords, and why you should use a password manager.",
		EstimatedMinutes: 7,
		Tags:             []string{"passwords", "authentication", "beginner"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST", "HIPAA", "PCI-DSS"},
		NanolearningTip:  "Use a password manager to generate and store unique passwords for every account. Never reuse passwords.",
		Pages: []TrainingPage{
			{Title: "Why Passwords Matter", Body: "Your passwords are the keys to your digital life. A weak or reused password is like leaving your front door unlocked.\n\n**Alarming statistics:**\n- 81% of data breaches involve weak or stolen passwords\n- The average person reuses the same password across 5+ accounts\n- The most common password is still \"123456\"\n\nWhen one account is breached, attackers try the same credentials on your other accounts — this is called **credential stuffing**."},
			{Title: "What Makes a Strong Password?", Body: "A strong password is:\n\n✅ **Long** — At least 14 characters (16+ is better)\n✅ **Unique** — Never reused across accounts\n✅ **Complex** — Mix of uppercase, lowercase, numbers, and symbols\n✅ **Unpredictable** — No dictionary words, birthdays, or pet names\n\n**Best approach: Passphrases**\nUse a random sequence of words: `correct-horse-battery-staple` is far stronger than `P@ssw0rd!`", TipBox: "A 14-character random passphrase takes billions of years to crack by brute force. A common 8-character password takes minutes."},
			{Title: "Password Managers", Body: "A password manager is a secure vault that:\n\n- **Generates** unique random passwords for every account\n- **Stores** them encrypted behind one master password\n- **Auto-fills** login forms so you never need to type passwords\n\nPopular options: Bitwarden, 1Password, KeePass, LastPass\n\nYou only need to remember ONE strong master password. The password manager handles the rest."},
			{Title: "Multi-Factor Authentication (MFA)", Body: "Even the strongest password isn't enough on its own. **Multi-Factor Authentication** adds a second layer:\n\n1. **Something you know** — Your password\n2. **Something you have** — Your phone (authenticator app or SMS code)\n3. **Something you are** — Fingerprint or face recognition\n\n**Always enable MFA** on email, banking, cloud storage, and work accounts. It blocks 99.9% of automated attacks.", TipBox: "Use an authenticator app (Google Authenticator, Microsoft Authenticator) instead of SMS — SIM-swapping attacks can intercept text messages."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is the minimum recommended password length?", Options: []string{"6 characters", "8 characters", "14 characters", "4 characters"}, CorrectOption: 2},
				{QuestionText: "What is credential stuffing?", Options: []string{"Creating new passwords", "Using stolen passwords from one breach on other accounts", "Filling out forms automatically", quizDistractorEncryption}, CorrectOption: 1},
				{QuestionText: "What is the best way to manage multiple passwords?", Options: []string{"Write them on a sticky note", "Use the same password everywhere", "Use a password manager", "Save them in an Excel file"}, CorrectOption: 2},
				{QuestionText: "Which MFA method is most secure?", Options: []string{"SMS text messages", "Security questions", "An authenticator app", "Email verification"}, CorrectOption: 2},
			},
		},
	},

	{
		Slug:             "social-engineering-basics",
		Title:            "Social Engineering — The Human Factor",
		Category:         ContentCategorySocialEng,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Discover how attackers exploit human psychology instead of technology, and learn to recognize manipulation tactics.",
		EstimatedMinutes: 7,
		Tags:             []string{"social engineering", "manipulation", "beginner"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST"},
		NanolearningTip:  "If someone creates pressure to act fast or asks you to bypass normal procedures, stop and verify through an independent channel.",
		Pages: []TrainingPage{
			{Title: "What Is Social Engineering?", Body: "Social engineering is the art of manipulating people into giving up confidential information or performing actions that compromise security.\n\nUnlike hacking into a computer, social engineers **hack humans** by exploiting:\n- **Trust** — Impersonating someone you know\n- **Fear** — Threatening consequences\n- **Curiosity** — \"Look at this interesting file\"\n- **Helpfulness** — \"I'm from IT, I need your password to fix an issue\""},
			{Title: "Common Social Engineering Techniques", Body: "**Pretexting** — Creating a fabricated scenario (\"I'm the new IT admin\")\n\n**Baiting** — Leaving an infected USB drive in a parking lot\n\n**Tailgating** — Following an employee through a secure door\n\n**Quid pro quo** — \"I'll fix your computer if you give me remote access\"\n\n**Vishing** — Phone calls from fake tech support, banks, or government agencies\n\nAttackers research you on LinkedIn, social media, and company websites to make their approach more convincing."},
			{Title: "Real-World Examples", Body: "🔴 **CEO Fraud**: An employee receives an urgent email from the \"CEO\" requesting an immediate wire transfer. The email looks legitimate but came from a lookalike domain.\n\n🔴 **IT Helpdesk Scam**: A caller claims to be from the helpdesk and asks for your password to \"resolve a ticket.\"\n\n🔴 **Delivery Scam**: A courier asks the receptionist to hold a USB drive for a colleague — the drive contains malware.\n\nIn each case, the attacker relied on **human trust and urgency**, not technical exploits."},
			{Title: "How to Protect Yourself", Body: "🛡️ **Verify requests** through an independent channel — call back using a known number\n🛡️ **Be skeptical of urgency** — Legitimate requests can wait for proper verification\n🛡️ **Never share credentials** — IT will never ask for your password\n🛡️ **Challenge strangers** in secure areas — \"Can I see your badge?\"\n🛡️ **Think before you click** — External USB drives, QR codes, or unexpected attachments could be bait", TipBox: "The golden rule: If something feels off, trust your instincts and verify independently."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What do social engineers primarily exploit?", Options: []string{"Software vulnerabilities", "Hardware weaknesses", "Human psychology and trust", "Network protocols"}, CorrectOption: 2},
				{QuestionText: "What is 'pretexting'?", Options: []string{"Testing software before release", "Creating a false scenario to gain trust", "Backing up data", "Encrypting communications"}, CorrectOption: 1},
				{QuestionText: "Someone calls claiming to be from IT and asks for your password. You should:", Options: []string{"Give it — they need it to help you", "Hang up and call the IT helpdesk directly", "Email your password instead", "Ask them to call back later"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "data-protection-basics",
		Title:            "Data Protection — Keeping Information Safe",
		Category:         ContentCategoryDataProtection,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Learn the fundamentals of data classification, safe handling, and GDPR essentials for everyday work.",
		EstimatedMinutes: 8,
		Tags:             []string{"data protection", "GDPR", "privacy", "beginner"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "GDPR", "HIPAA", "DORA"},
		NanolearningTip:  "Before sending an email with sensitive data, double-check the recipient list. One wrong address can cause a data breach.",
		Pages: []TrainingPage{
			{Title: "Why Data Protection Matters", Body: "Every day you handle data that, if exposed, could harm individuals or your organization:\n\n- **Personal data** — Names, email addresses, phone numbers, health records\n- **Financial data** — Credit card numbers, bank accounts, invoices\n- **Business secrets** — Strategy documents, contracts, pricing\n- **Credentials** — Passwords, API keys, access tokens\n\nA data breach can result in fines up to **€20 million or 4% of annual revenue** under GDPR."},
			{Title: "Data Classification", Body: "Not all data needs the same protection. Most organizations use:\n\n🔴 **Confidential** — Trade secrets, HR records, financial data\n🟡 **Internal** — Strategy docs, internal memos, project plans\n🟢 **Public** — Marketing materials, press releases, public website content\n\n**Rule of thumb**: If you're unsure, treat it as confidential.", TipBox: "Always check your organization's data classification policy before sharing information externally."},
			{Title: "Safe Data Handling", Body: "Follow these practices every day:\n\n✅ **Lock your screen** when leaving your desk (Windows: Win+L, Mac: Ctrl+Cmd+Q)\n✅ **Encrypt sensitive files** before sending by email\n✅ **Use secure sharing** — Company SharePoint/OneDrive, not personal Dropbox\n✅ **Double-check recipients** before sending emails with sensitive data\n✅ **Shred physical documents** containing personal or financial information\n✅ **Report incidents** immediately — even accidental data exposure"},
			{Title: "GDPR in 60 Seconds", Body: "The General Data Protection Regulation (GDPR) gives individuals control over their personal data:\n\n- **Lawful basis** — You need a valid reason to process personal data\n- **Purpose limitation** — Use data only for the stated purpose\n- **Data minimization** — Collect only what you need\n- **Right to erasure** — Individuals can request deletion of their data\n- **Breach notification** — Organizations must report breaches within 72 hours\n\nEvery employee plays a role in GDPR compliance.", TipBox: "When in doubt about data handling, ask your Data Protection Officer (DPO) before proceeding."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "Under GDPR, how quickly must a data breach be reported to the supervisory authority?", Options: []string{"24 hours", "72 hours", "1 week", "1 month"}, CorrectOption: 1},
				{QuestionText: "Which of the following is considered personal data under GDPR?", Options: []string{"Company revenue figures", "An employee's email address", "A product specification document", "A press release"}, CorrectOption: 1},
				{QuestionText: "What should you do before sending an email with sensitive data?", Options: []string{"CC your manager", "Double-check the recipient list", "Use your personal email", "Compress the file"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "malware-ransomware-basics",
		Title:            "Malware & Ransomware — Understanding the Threats",
		Category:         ContentCategoryMalware,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Learn what malware and ransomware are, how they spread, and what you can do to prevent infection.",
		EstimatedMinutes: 7,
		Tags:             []string{"malware", "ransomware", "virus", "beginner"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST", compCyberEssentials},
		NanolearningTip:  "Never enable macros in unexpected email attachments. Macros are one of the most common ways ransomware enters organizations.",
		Pages: []TrainingPage{
			{Title: "What Is Malware?", Body: "Malware (malicious software) is any program designed to harm your computer or steal your data. Types include:\n\n🦠 **Virus** — Attaches to files and spreads when you share them\n🐛 **Worm** — Spreads automatically across networks\n🐴 **Trojan** — Disguises itself as legitimate software\n📷 **Spyware** — Secretly monitors your activity\n🔐 **Ransomware** — Encrypts your files and demands payment\n⌨️ **Keylogger** — Records your keystrokes to steal passwords"},
			{Title: "How Malware Spreads", Body: "The most common infection vectors:\n\n📧 **Email attachments** — Word docs with macros, ZIP files, PDFs\n🔗 **Malicious links** — Websites that download malware automatically\n💾 **USB drives** — Infected drives left in public places\n📱 **Fake apps** — Malicious apps on unofficial app stores\n🔄 **Software vulnerabilities** — Unpatched systems are easy targets\n\n**Ransomware specifically** often enters through phishing emails or Remote Desktop Protocol (RDP) brute-force attacks."},
			{Title: "Prevention Best Practices", Body: "Protect yourself and your organization:\n\n✅ **Keep software updated** — Enable automatic updates\n✅ **Don't enable macros** in documents from unknown sources\n✅ **Back up your data** regularly (follow the 3-2-1 rule: 3 copies, 2 media types, 1 offsite)\n✅ **Use antivirus** and keep it updated\n✅ **Don't plug in unknown USB devices**\n✅ **Download software** only from official sources", TipBox: "If you see a pop-up saying 'Enable Content' or 'Enable Macros' in a document you received by email — DON'T click it."},
			{Title: "What to Do If You're Infected", Body: "If you suspect a malware infection:\n\n1. **Disconnect** from the network immediately (pull the Ethernet cable, disable Wi-Fi)\n2. **Don't turn off** the computer — evidence may be needed\n3. **Contact IT** immediately\n4. **Don't pay ransom** — there's no guarantee you'll get your files back\n5. **Document** what happened — when, what you clicked, what you saw\n\nSpeed matters. The faster you disconnect, the less damage the malware can do."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What should you do if a document asks you to 'Enable Macros'?", Options: []string{"Enable them — it's normal", "Close the document and report it to IT", "Save the file and open it later", "Forward it to a colleague to check"}, CorrectOption: 1},
				{QuestionText: "What is ransomware?", Options: []string{"Software that speeds up your computer", "Malware that encrypts files and demands payment", "A type of antivirus", "A backup solution"}, CorrectOption: 1},
				{QuestionText: "What is the first thing to do if you suspect a malware infection?", Options: []string{"Restart the computer", "Run a full antivirus scan", "Disconnect from the network", "Delete all your files"}, CorrectOption: 2},
			},
		},
	},

	{
		Slug:             "physical-security-basics",
		Title:            "Physical Security — Protect Your Workspace",
		Category:         ContentCategoryPhysicalSec,
		DifficultyLevel:  ContentDiffBronze,
		Description:      "Learn how physical security practices protect digital assets — from clean desks to tailgating prevention.",
		EstimatedMinutes: 5,
		Tags:             []string{"physical security", "clean desk", "beginner"},
		ComplianceMapped: []string{"ISO27001", "NIST", compCyberEssentials},
		NanolearningTip:  "Always lock your screen when leaving your desk, even for a coffee break. It only takes seconds for someone to access your data.",
		Pages: []TrainingPage{
			{Title: "Why Physical Security Matters", Body: "Cybersecurity isn't just digital. Physical access can lead to:\n\n- **Data theft** — A visitor photographs your screen\n- **Credential theft** — Someone reads the password on your sticky note\n- **Device theft** — A laptop left unattended in a café\n- **Unauthorized access** — Tailgating through a secure door\n\nThe strongest password is useless if someone can physically access your computer."},
			{Title: "Clean Desk & Clear Screen", Body: "Practice the **Clean Desk Policy**:\n\n🗄️ Lock away sensitive documents when you leave\n🖥️ Lock your screen: **Win+L** (Windows) or **Ctrl+Cmd+Q** (Mac)\n📝 Don't write passwords on sticky notes\n🗑️ Shred confidential papers — don't just throw them in the bin\n💻 Take your laptop or lock it in a drawer\n\nAt the end of the day, your desk should be clear of all sensitive information.", TipBox: "Set your screen to auto-lock after 2 minutes of inactivity."},
			{Title: "Visitor & Access Control", Body: "Help maintain physical security:\n\n🚪 **Never hold doors open** for strangers in secure areas\n🪪 **Challenge unknown visitors** — ask to see their badge\n📋 **Sign-in visitors** at reception\n📱 **Report lost/stolen** access badges immediately\n🚫 **No piggybacking** — Each person scans their own badge\n\nIf someone asks you to let them in because they \"forgot their badge,\" politely direct them to reception."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is tailgating?", Options: []string{"Following too closely while driving", "Following an authorized person through a secure door", "Leaving work early", "Sending follow-up emails"}, CorrectOption: 1},
				{QuestionText: "What should you do when leaving your desk?", Options: []string{"Leave your screen on for convenience", "Lock your screen and secure documents", "Ask a colleague to watch your computer", "Nothing if you'll be back soon"}, CorrectOption: 1},
			},
		},
	},

	// ═══════════════════════════════════════════════════════════════════════
	// SILVER TIER — Intermediate (Difficulty 2)
	// ═══════════════════════════════════════════════════════════════════════

	{
		Slug:             "spear-phishing-bec",
		Title:            "Spear Phishing & Business Email Compromise",
		Category:         ContentCategoryPhishing,
		DifficultyLevel:  ContentDiffSilver,
		Description:      "Understand targeted phishing attacks, CEO fraud, and invoice manipulation — the costliest forms of email crime.",
		EstimatedMinutes: 9,
		Tags:             []string{"spear phishing", "BEC", "CEO fraud", "intermediate"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST", "DORA"},
		NanolearningTip:  "Any request to change payment details, transfer money, or share sensitive data should be verified through a phone call to a known number.",
		Pages: []TrainingPage{
			{Title: "From Phishing to Spear Phishing", Body: "While regular phishing casts a wide net, **spear phishing** is a targeted attack against a specific person or organization.\n\nAttackers research you using:\n- LinkedIn profiles (job title, colleagues)\n- Company website (org chart, recent news)\n- Social media (interests, travel plans)\n- Previous data breaches (your email/password)\n\nThe result: An email so personalized it's nearly impossible to distinguish from a real one."},
			{Title: "Business Email Compromise (BEC)", Body: "BEC is the most financially devastating form of cybercrime.\n\n**Types of BEC:**\n\n💰 **CEO Fraud** — \"I need you to wire €50,000 to this vendor today. Don't tell anyone — it's confidential.\"\n\n🧾 **Invoice Fraud** — A supplier's email is spoofed to change payment bank details\n\n📧 **Account Compromise** — An actual employee's email is hijacked and used to send requests\n\n👤 **Attorney Impersonation** — \"Your legal team requests immediate document transfer\"\n\n**FBI reported $2.9 billion in BEC losses in 2023 alone.**"},
			{Title: "Red Flags for BEC", Body: "Watch for these BEC-specific indicators:\n\n🚩 Requests that **bypass normal approval processes**\n🚩 **Emphasis on secrecy** — \"Don't discuss this with anyone\"\n🚩 **Change in payment details** — \"We've switched banks, use this new account\"\n🚩 **Unusual timing** — Sent late Friday or before a holiday\n🚩 **Slightly different email domain** — company.com vs. c0mpany.com or company.co\n🚩 **Personal tone from executives** you don't normally hear from directly"},
			{Title: "How to Defend Against BEC", Body: "**For financial requests:**\n✅ Always verify by phone using a known number (not from the email)\n✅ Require dual approval for transfers above a threshold\n✅ Confirm payment detail changes through a separate channel\n\n**For all email:**\n✅ Check the full email address, not just the display name\n✅ Look for subtle domain spoofing (rn → m, l → 1)\n✅ Be especially cautious with urgency + secrecy requests\n✅ Report suspected BEC immediately — speed matters", TipBox: "If a CEO truly needs an urgent transfer, they can verify by phone. Any legitimate executive will understand the security precaution."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What makes spear phishing different from regular phishing?", Options: []string{"It uses phone calls", "It's targeted at specific individuals using research", "It only affects mobile devices", "It's less dangerous"}, CorrectOption: 1},
				{QuestionText: "A supplier emails saying they've changed their bank account. You should:", Options: []string{"Update the payment details immediately", "Verify by calling the supplier on a known phone number", "Reply to the email asking for confirmation", "Wait a week before updating"}, CorrectOption: 1},
				{QuestionText: "Which is a common characteristic of BEC?", Options: []string{"Mass distribution to thousands of people", "Emphasis on secrecy and urgency", "Always contains malware attachments", "Comes from unknown email addresses"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "remote-work-security",
		Title:            "Secure Remote Working Practices",
		Category:         ContentCategoryRemoteWork,
		DifficultyLevel:  ContentDiffSilver,
		Description:      "Master the security essentials for working from home, co-working spaces, and while traveling.",
		EstimatedMinutes: 8,
		Tags:             []string{"remote work", "VPN", "WiFi", "intermediate"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST"},
		NanolearningTip:  "Always use your company VPN on public Wi-Fi. Without it, anyone on the same network could intercept your traffic.",
		Pages: []TrainingPage{
			{Title: "The Remote Work Threat Landscape", Body: "Working outside the office exposes you to risks that corporate networks normally handle:\n\n- **Unsecured Wi-Fi** — Coffee shops, hotels, airports\n- **Shoulder surfing** — People watching your screen\n- **Device theft** — Laptops left unattended\n- **Home network weaknesses** — Default router passwords, unpatched firmware\n- **Shared devices** — Family members using work computers"},
			{Title: "Securing Your Connection", Body: "**Always use the company VPN** when working remotely. It encrypts all traffic between your device and the corporate network.\n\n🔒 **Do:**\n- Connect to VPN before opening any work applications\n- Use your phone's mobile hotspot instead of public Wi-Fi\n- Verify you're connected to the right Wi-Fi network\n\n🚫 **Don't:**\n- Use free public Wi-Fi for work without VPN\n- Connect to networks named \"Free Airport WiFi\" (could be fake)\n- Disable VPN for speed — security always comes first", TipBox: "An attacker can set up a fake Wi-Fi hotspot in under 60 seconds using a €30 device."},
			{Title: "Securing Your Home Office", Body: "Your home office needs basic security too:\n\n🏠 **Router security:**\n- Change the default admin password\n- Enable WPA3 (or WPA2) encryption\n- Update firmware regularly\n- Use a guest network for IoT devices\n\n💻 **Device security:**\n- Enable full-disk encryption\n- Set auto-lock to 2 minutes\n- Keep all software updated\n- Don't let family members use work devices"},
			{Title: "Working in Public Spaces", Body: "When working in co-working spaces, cafés, or while traveling:\n\n👀 **Use a privacy screen** — Prevents shoulder surfing\n🔇 **Use headphones** for calls — Don't discuss sensitive info aloud\n💻 **Never leave devices unattended** — Take them with you, always\n🖨️ **Avoid public printers** — Documents stay in the queue\n📱 **Disable Bluetooth & AirDrop** when not in use\n🔌 **Beware of public USB charging** — Use your own charger (\"juice jacking\" can steal data)"},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What should you always use when working on public Wi-Fi?", Options: []string{"A web browser", "The company VPN", "Social media", "A USB drive"}, CorrectOption: 1},
				{QuestionText: "Why is 'juice jacking' a risk?", Options: []string{"It drains your battery", "Public USB ports can steal data from your device", "It makes your phone too hot", "It slows down charging"}, CorrectOption: 1},
				{QuestionText: "What should you do with IoT devices on your home network?", Options: []string{"Connect them to your work network", "Put them on a separate guest network", "Turn them off during work hours", "Nothing — they're safe"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "mobile-device-security",
		Title:            "Mobile Device Security — Protecting Your Phone & Tablet",
		Category:         ContentCategoryMobileSec,
		DifficultyLevel:  ContentDiffSilver,
		Description:      "Secure your mobile devices against the most common threats including malicious apps, SMS phishing, and data theft.",
		EstimatedMinutes: 7,
		Tags:             []string{"mobile", "smartphone", "smishing", "intermediate"},
		ComplianceMapped: []string{"NIS2", "ISO27001", compCyberEssentials},
		NanolearningTip:  "Only install apps from official stores (App Store/Google Play) and review permissions before granting access to your camera, contacts, or location.",
		Pages: []TrainingPage{
			{Title: "Why Mobile Security Matters", Body: "Your phone contains more sensitive data than most computers:\n\n📧 Work email and calendar\n💬 Chat messages with confidential info\n🏦 Banking apps\n📸 Photos that could contain sensitive documents\n📍 Your real-time location\n🔑 Authenticator apps for MFA\n\nLosing your phone or getting it infected is like handing your digital life to an attacker."},
			{Title: "Securing Your Mobile Device", Body: "Essential mobile security practices:\n\n🔒 **Strong screen lock** — 6-digit PIN minimum, biometrics preferred\n📲 **Auto-updates** — Enable automatic OS and app updates\n🗑️ **Uninstall unused apps** — Every app is a potential attack surface\n☁️ **Enable remote wipe** — Find My iPhone / Find My Device\n🔐 **Encrypt your device** — Modern phones are encrypted by default when you set a PIN\n📡 **Turn off Bluetooth/NFC** when not in use"},
			{Title: "App Security & Smishing", Body: "**App safety:**\n- Only download from official stores (App Store, Google Play)\n- Read reviews and check the developer name\n- Review permissions: Does a flashlight app need access to your contacts?\n\n**Smishing (SMS phishing):**\nScam texts are exploding. Common examples:\n- \"Your package couldn't be delivered\" + link\n- \"Unusual sign-in detected\" + link\n- \"You've won a prize\" + link\n\n**Rule:** Never click links in unexpected text messages.", TipBox: "If you receive a suspicious SMS about a delivery, go directly to the courier's website — don't use the link in the message."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is smishing?", Options: []string{quizDistractorEncryption, "Phishing via SMS text messages", "A mobile operating system", "A password technique"}, CorrectOption: 1},
				{QuestionText: "Where should you download mobile apps from?", Options: []string{"Any website", "Official app stores only", "Email links", "Social media ads"}, CorrectOption: 1},
				{QuestionText: "What should you enable to protect a lost phone?", Options: []string{"Airplane mode", "Remote wipe capability", "Bluetooth", "Screen rotation"}, CorrectOption: 1},
			},
		},
	},

	// ═══════════════════════════════════════════════════════════════════════
	// GOLD TIER — Advanced (Difficulty 3)
	// ═══════════════════════════════════════════════════════════════════════

	{
		Slug:             "advanced-phishing-techniques",
		Title:            "Advanced Phishing — QR Codes, Deepfakes & AI-Generated Attacks",
		Category:         ContentCategoryPhishing,
		DifficultyLevel:  ContentDiffGold,
		Description:      "Explore sophisticated phishing techniques including QR code attacks (quishing), AI-generated emails, and deepfake voice calls.",
		EstimatedMinutes: 10,
		Tags:             []string{"phishing", "QR", "deepfake", "AI", "advanced"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST"},
		NanolearningTip:  "Never scan a QR code from an untrusted source. Attackers can place malicious QR stickers over legitimate ones.",
		Pages: []TrainingPage{
			{Title: "The Evolution of Phishing", Body: "Phishing has evolved far beyond obvious scam emails. Today's attacks use:\n\n🤖 **AI-generated content** — ChatGPT-quality emails with perfect grammar in any language\n📱 **QR code attacks (Quishing)** — Malicious QR codes on posters, emails, and even parking meters\n🎭 **Deepfake voice** — AI clones of your CEO's voice requesting urgent transfers\n🎬 **Deepfake video** — Fake video calls impersonating colleagues\n📲 **Multi-channel attacks** — Combining email + phone call for credibility"},
			{Title: "QR Code Attacks (Quishing)", Body: "QR codes are everywhere — and attackers exploit our trust in them:\n\n**Attack scenarios:**\n- Fake QR codes stuck over legitimate ones (parking meters, restaurant menus)\n- QR codes in phishing emails to bypass email security filters\n- Fake Wi-Fi network QR codes in public spaces\n- QR codes on fake invoices or delivery notices\n\n**Protection:**\n✅ Use your phone's built-in QR scanner — it previews the URL\n✅ Check the URL before opening — does it match the expected domain?\n✅ Never scan QR codes from suspicious emails\n✅ In public: check if a sticker has been placed over the original QR code"},
			{Title: "AI-Generated Phishing & Deepfakes", Body: "**AI-powered phishing:**\n- Perfect grammar and natural tone in any language\n- Personalized using publicly available information\n- No more typos — the #1 red flag is disappearing\n- Can generate thousands of unique emails to avoid spam filters\n\n**Deepfake attacks:**\n- Voice clones created from just 3 seconds of audio (from YouTube, podcasts, etc.)\n- Video deepfakes on Zoom/Teams calls\n- In 2024, a Hong Kong company lost $25 million to a deepfake video call\n\n**New red flags to watch for:**\n🚩 Unusual requests, even if the voice sounds right\n🚩 Resistance to callback verification\n🚩 Slight delays or artifacts in video calls", TipBox: "Establish a verbal passphrase with your team for high-value requests. Something a deepfake wouldn't know."},
			{Title: "Defending Against Advanced Attacks", Body: "**Technical defenses:**\n- Email authentication (SPF, DKIM, DMARC) blocks spoofed domains\n- AI-based email security tools detect anomalous patterns\n- Browser extensions that flag suspicious URLs\n\n**Human defenses:**\n- Always verify financial requests by phone\n- Use pre-agreed code words for sensitive operations\n- Be extra cautious during mergers, leadership changes, or crises\n- Report anything unusual — even if you're not 100% sure\n\n**Remember:** The attacker only needs to succeed once. You need to be vigilant every time."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is 'quishing'?", Options: []string{quizDistractorEncryption, "Phishing using QR codes", "A quiz-based learning method", "Quiet phishing"}, CorrectOption: 1},
				{QuestionText: "How can attackers create a deepfake voice clone?", Options: []string{"They need 1 hour of recorded audio", "They need physical access to your phone", "From as little as 3 seconds of audio", "It's not possible yet"}, CorrectOption: 2},
				{QuestionText: "What's the best defense against a deepfake phone call requesting a fund transfer?", Options: []string{"Trust the voice — it sounds authentic", "Call back on a known number to verify", "Send a confirmation email", "Complete the transfer quickly"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "incident-response-employees",
		Title:            "Incident Response — What to Do When Things Go Wrong",
		Category:         ContentCategoryIncident,
		DifficultyLevel:  ContentDiffGold,
		Description:      "Learn the correct steps to take when you suspect a security incident — speed and accuracy save organizations.",
		EstimatedMinutes: 8,
		Tags:             []string{"incident response", "breach", "reporting", "advanced"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST", "DORA", "HIPAA"},
		NanolearningTip:  "If you think you clicked a phishing link, immediately disconnect from the network and call your IT security team. Minutes matter.",
		Pages: []TrainingPage{
			{Title: "What Is a Security Incident?", Body: "A security incident is any event that threatens the confidentiality, integrity, or availability of information. Examples:\n\n🔓 **Unauthorized access** — Someone logs into your account\n💻 **Malware infection** — Ransomware encrypts your files\n📧 **Data breach** — Sensitive data sent to the wrong person\n🎣 **Successful phishing** — You clicked a link and entered credentials\n📱 **Lost/stolen device** — Laptop, phone, or USB drive\n🔌 **Unusual behavior** — Your computer is unusually slow or shows pop-ups"},
			{Title: "The First 5 Minutes", Body: "What you do in the first minutes matters enormously:\n\n1️⃣ **DON'T PANIC** — Stay calm and think clearly\n2️⃣ **DISCONNECT** — Unplug Ethernet, disable Wi-Fi\n3️⃣ **DON'T TURN OFF** the computer — Preserve evidence\n4️⃣ **DOCUMENT** — Write down: What happened? When? What did you click/see?\n5️⃣ **REPORT** — Contact your IT/Security team immediately\n\n**Speed is critical.** A ransomware attack can encrypt an entire network in under 4 hours. Early detection limits damage.", TipBox: "Save your IT security team's contact number in your phone. In a crisis, you shouldn't have to search for it."},
			{Title: "What to Report and How", Body: "When reporting an incident, provide:\n\n📝 **What happened** — \"I clicked a link in a phishing email\"\n⏰ **When** — Date and time as precisely as possible\n💻 **What device** — Your computer name, phone model\n📧 **Evidence** — The email, screenshot, URL you clicked\n🔑 **Affected accounts** — Which systems might be compromised\n\n**How to report:**\n- Use your organization's incident reporting tool\n- Call the IT Security hotline\n- If email is compromised, use the phone\n\n**Never:** Try to fix it yourself, hide it, or delete evidence."},
			{Title: "After the Incident", Body: "After reporting:\n\n✅ **Change passwords** for all potentially affected accounts\n✅ **Enable MFA** if not already active\n✅ **Monitor** your accounts for unusual activity\n✅ **Cooperate** fully with the investigation team\n✅ **Learn** from the experience — it happens to the best of us\n\n**There is no shame in reporting.** Organizations that cultivate a blame-free reporting culture detect incidents 50% faster.\n\nEvery incident reported is a lesson that makes the organization stronger."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "After clicking a suspected phishing link, what should you do first?", Options: []string{"Change your password immediately", "Disconnect from the network", "Turn off your computer", "Delete the email"}, CorrectOption: 1},
				{QuestionText: "Should you turn off your computer during a suspected malware infection?", Options: []string{"Yes — to stop the malware", "No — preserve evidence for investigation", "Yes — then turn it back on", "Only if it's slow"}, CorrectOption: 1},
				{QuestionText: "Why is a blame-free reporting culture important?", Options: []string{"To avoid paperwork", "It encourages faster incident reporting", "It's required by law", "To reduce IT workload"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "cloud-saas-security",
		Title:            "Cloud & SaaS Security — Working Safely in the Cloud",
		Category:         ContentCategoryCloudSec,
		DifficultyLevel:  ContentDiffGold,
		Description:      "Understand the security implications of cloud services, SaaS applications, and how to use them safely.",
		EstimatedMinutes: 8,
		Tags:             []string{"cloud", "SaaS", "sharing", "advanced"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "SOC2"},
		NanolearningTip:  "Before sharing a document link externally, check the sharing settings. 'Anyone with the link' means the entire internet can access it.",
		Pages: []TrainingPage{
			{Title: "The Shared Responsibility Model", Body: "When using cloud services, security is shared:\n\n☁️ **Cloud provider** is responsible for: Infrastructure, physical security, patching\n👤 **You** are responsible for: Access control, data classification, sharing settings, user behavior\n\n**Common misconception:** \"It's in the cloud, so it's automatically secure.\"\n\n**Reality:** Most cloud breaches are caused by customer misconfigurations, not provider failures."},
			{Title: "Secure File Sharing", Body: "Cloud sharing is powerful but dangerous if misconfigured:\n\n🚫 **Don't:**\n- Share with \"Anyone with the link\" for sensitive documents\n- Use personal cloud storage for work files\n- Leave sharing links active after the project ends\n\n✅ **Do:**\n- Share with specific people/groups\n- Set expiration dates on shared links\n- Use \"View only\" instead of \"Edit\" when possible\n- Regularly audit who has access to your shared folders\n- Use the company-approved cloud platform, not personal Dropbox/Google Drive"},
			{Title: "SaaS Application Security", Body: "Every SaaS tool is an attack surface:\n\n🔐 **Account hygiene:**\n- Unique strong password for every SaaS app\n- Enable MFA on all SaaS accounts\n- Review and revoke OAuth app permissions regularly\n\n⚠️ **Shadow IT risks:**\n- Signing up for free tools with your work email creates data exposure\n- Chat apps, design tools, and AI tools may store your data\n- Always check with IT before adopting new SaaS tools\n\n🧹 **Offboarding:**\n- When employees leave, their SaaS access must be revoked across ALL apps", TipBox: "Check your Google/Microsoft account's 'Connected Apps' section and revoke any you don't recognize or use."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "In the cloud shared responsibility model, who is responsible for access control?", Options: []string{"The cloud provider", "The customer (you)", "Both equally", "Neither — it's automatic"}, CorrectOption: 1},
				{QuestionText: "What is 'Shadow IT'?", Options: []string{"A backup system", "Unauthorized use of SaaS tools not approved by IT", "Dark mode in applications", "A type of malware"}, CorrectOption: 1},
			},
		},
	},

	// ═══════════════════════════════════════════════════════════════════════
	// PLATINUM TIER — Expert (Difficulty 4)
	// ═══════════════════════════════════════════════════════════════════════

	{
		Slug:             "ai-threats-deepfakes",
		Title:            "AI-Powered Cyber Threats & Deepfakes",
		Category:         ContentCategoryAISec,
		DifficultyLevel:  ContentDiffPlatinum,
		Description:      "Explore how AI is weaponized by attackers — from automated phishing at scale to deepfake impersonation and AI-assisted code exploitation.",
		EstimatedMinutes: 10,
		Tags:             []string{"AI", "deepfake", "LLM", "expert"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "NIST"},
		NanolearningTip:  "AI-generated content has no spelling mistakes. The absence of errors is no longer proof an email is legitimate.",
		Pages: []TrainingPage{
			{Title: "AI as an Attack Multiplier", Body: "AI doesn't create new attack types — it makes existing attacks faster, cheaper, and more convincing:\n\n🤖 **AI-generated phishing** — Perfect emails in 25+ languages, personalized at scale\n🎭 **Deepfake voice/video** — Real-time impersonation on calls\n🔓 **Password cracking** — AI-accelerated brute force and pattern prediction\n🕵️ **Reconnaissance** — AI scrapes and correlates public data for targeting\n💻 **Code generation** — AI writes custom malware and exploit scripts\n\n**The asymmetry:** Attackers use AI to attack millions. Defenders must protect every single person."},
			{Title: "Deepfake Impersonation in Practice", Body: "**Real cases:**\n\n💰 **$25M deepfake video call** (2024) — A finance worker was tricked by a deepfake video call with multiple \"colleagues\" who were all AI-generated.\n\n🎤 **CEO voice clone** — Attackers cloned a CEO's voice from a conference talk and called the CFO requesting an emergency transfer.\n\n📹 **Fake job interviews** — AI-generated candidates on video calls to infiltrate companies.\n\n**Detection tips:**\n- Request a callback on a verified number\n- Use code words for sensitive requests\n- Watch for slight audio/video lag or artifacts\n- If something feels wrong, trust your instincts"},
			{Title: "Safe Use of AI Tools at Work", Body: "AI tools like ChatGPT, Copilot, and Claude are powerful but come with risks:\n\n🚫 **Never input into public AI tools:**\n- Source code or API keys\n- Customer data or personal information\n- Financial data or strategy documents\n- Meeting notes with sensitive content\n\n✅ **Safe practices:**\n- Use only company-approved AI tools\n- Treat AI outputs as unverified — fact-check everything\n- Understand that anything you input may be stored or used for training\n- Follow your organization's AI usage policy", TipBox: "If your company doesn't have an AI usage policy yet, ask your IT team to create one. It's now essential."},
			{Title: "Building Organizational Resilience", Body: "**Prepare your organization for AI-era threats:**\n\n🛡️ **Process controls:** Multi-person approval for high-value actions\n📞 **Verification protocols:** Callback procedures for financial requests\n🗣️ **Code words:** Pre-agreed verbal passphrases for sensitive communications\n📊 **AI-powered defense:** Use AI security tools to detect AI attacks\n🎓 **Continuous training:** The threat landscape changes monthly — training must keep up\n\n**The human firewall is more important than ever.** Technology alone cannot stop AI-powered social engineering. Educated, vigilant employees are your best defense."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "What should you NEVER input into a public AI tool?", Options: []string{"A recipe you want to try", "Customer data or source code", "A general knowledge question", "A creative writing prompt"}, CorrectOption: 1},
				{QuestionText: "In the $25M deepfake case, how did the attackers succeed?", Options: []string{"They hacked the bank system", "They used deepfake video of multiple colleagues on a call", "They sent a phishing email", "They guessed the password"}, CorrectOption: 1},
				{QuestionText: "What is the best defense against deepfake impersonation?", Options: []string{"Better video call software", "Verify through an independent channel with a code word", "Block all video calls", "Install deepfake detection AI"}, CorrectOption: 1},
			},
		},
	},

	{
		Slug:             "compliance-nis2-dora",
		Title:            "Compliance Essentials — NIS2, DORA & ISO 27001",
		Category:         ContentCategoryCompliance,
		DifficultyLevel:  ContentDiffPlatinum,
		Description:      "Understand your role in meeting regulatory requirements including NIS2, DORA, and ISO 27001.",
		EstimatedMinutes: 10,
		Tags:             []string{"compliance", "NIS2", "DORA", "ISO27001", "expert"},
		ComplianceMapped: []string{"NIS2", "ISO27001", "DORA", "SOC2", "HIPAA"},
		NanolearningTip:  "Compliance is everyone's responsibility. A single employee's mistake can make an entire organization non-compliant.",
		Pages: []TrainingPage{
			{Title: "Why Compliance Matters to You", Body: "Cybersecurity regulations aren't just for IT — they affect every employee:\n\n⚖️ **NIS2** — EU directive requiring essential & important entities to manage cyber risk\n🏦 **DORA** — Digital operational resilience for financial institutions\n📋 **ISO 27001** — International standard for information security management\n🏥 **HIPAA** — Protection of health information (US)\n💳 **PCI-DSS** — Security for payment card data\n\n**Your role:** Follow security policies, report incidents, complete training, handle data correctly. Every compliance framework depends on trained, aware employees."},
			{Title: "NIS2 — What It Means for You", Body: "The NIS2 Directive (effective October 2024) requires:\n\n✅ **Risk management** — Organizations must assess and mitigate cyber risks\n✅ **Incident reporting** — Significant incidents must be reported within 24 hours\n✅ **Supply chain security** — Your vendors must also be secure\n✅ **Training** — ALL employees must receive cybersecurity awareness training\n✅ **Management accountability** — C-level executives are personally liable\n\n**As an employee, you contribute by:**\n- Completing your security awareness training\n- Following the incident reporting procedure\n- Applying security policies in your daily work", TipBox: "Under NIS2, fines can reach €10 million or 2% of global annual turnover — whichever is higher."},
			{Title: "DORA — Digital Operational Resilience", Body: "DORA applies to financial institutions and their ICT providers:\n\n🏦 **Key requirements:**\n- ICT risk management framework\n- Incident reporting within 4 hours for major incidents\n- Resilience testing including threat-led penetration tests\n- Third-party risk management for ICT providers\n- Information sharing with other financial entities\n\n**Your role in DORA compliance:**\n- Protect customer financial data\n- Report incidents immediately\n- Follow ICT policies strictly\n- Participate in resilience exercises"},
			{Title: "Building a Security Culture", Body: "Compliance isn't a checkbox — it's a culture:\n\n🏗️ **Security champions** — Volunteer to be your team's security advocate\n📣 **Speak up** — Report concerns without fear of blame\n📚 **Stay current** — Threats evolve; your knowledge must too\n🤝 **Lead by example** — Lock screens, use MFA, verify requests\n📊 **Measure progress** — Track your Behavioral Risk Score (BRS)\n\n**Remember:** The strongest security technology fails without trained, vigilant people. You are the human firewall."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 70,
			Questions: []BuiltInQuestion{
				{QuestionText: "Under NIS2, how quickly must significant incidents be reported?", Options: []string{"72 hours", "24 hours", "1 week", "1 month"}, CorrectOption: 1},
				{QuestionText: "What does DORA primarily apply to?", Options: []string{"Healthcare organizations", "Financial institutions and their ICT providers", "Retail companies", "Government agencies only"}, CorrectOption: 1},
				{QuestionText: "Who is responsible for cybersecurity compliance in an organization?", Options: []string{"Only the IT department", "Only the CISO", "Everyone — all employees play a role", "Only management"}, CorrectOption: 2},
			},
		},
	},
}

package ai

import "fmt"

// DifficultyLevel represents how hard a phishing email is to detect.
// Level 1 = obvious red flags; Level 4 = sophisticated, hard to spot.
const (
	DifficultyEasy       = 1 // Obvious red flags: typos, suspicious sender, urgency
	DifficultyMedium     = 2 // Moderate: plausible premise but some tells
	DifficultyHard       = 3 // Professional: good grammar, realistic scenario
	DifficultySophisticated = 4 // Expert: highly targeted, near-indistinguishable from real
)

// systemPrompts maps difficulty levels to the system prompt used for template
// generation. Each prompt instructs the LLM to produce a phishing simulation
// email at the appropriate difficulty level.
var systemPrompts = map[int]string{
	DifficultyEasy: `You are a cybersecurity training assistant that generates phishing simulation emails for employee awareness training.

Generate a DIFFICULTY LEVEL 1 (Easy) phishing email. This email should be relatively obvious to spot as phishing. Include these telltale signs:
- Minor spelling or grammar mistakes
- A sense of urgency or threatening language ("Your account will be suspended!")
- A slightly suspicious sender name or domain
- A generic greeting ("Dear Customer" instead of using the recipient's name)
- A call to action that asks the user to click a link or provide credentials

The email should still look like a real phishing attempt — not a parody. The goal is to train employees to recognize basic phishing patterns.`,

	DifficultyMedium: `You are a cybersecurity training assistant that generates phishing simulation emails for employee awareness training.

Generate a DIFFICULTY LEVEL 2 (Medium) phishing email. This email should be moderately convincing with a few subtle tells:
- Correct grammar and professional tone
- A plausible business scenario (password reset, invoice, shared document)
- Sender name looks legitimate but the domain might be slightly off
- Uses some personalization (recipient's first name if provided)
- The call to action is reasonable but the link destination would be suspicious on close inspection

The email should require some attention to detect. Most trained employees should catch it, but it shouldn't be immediately obvious.`,

	DifficultyHard: `You are a cybersecurity training assistant that generates phishing simulation emails for employee awareness training.

Generate a DIFFICULTY LEVEL 3 (Hard) phishing email. This email should be highly convincing:
- Perfect grammar and professional formatting
- Mimics a well-known service or internal company communication
- Uses the recipient's name, department, and job title if provided
- The scenario is timely and contextually relevant (e.g., annual review, benefits enrollment, IT policy update)
- The call to action is subtle and well-integrated into the email flow
- Only a careful reader checking headers or hovering over links would detect it

This level is for employees who have demonstrated good phishing awareness and need a greater challenge.`,

	DifficultySophisticated: `You are a cybersecurity training assistant that generates phishing simulation emails for employee awareness training.

Generate a DIFFICULTY LEVEL 4 (Sophisticated/Expert) phishing email. This email should be extremely hard to distinguish from a legitimate email:
- Flawless language matching the impersonated organization's communication style
- Highly targeted: uses recipient's full name, job title, department, and recent plausible context
- Impersonates a known internal colleague, executive, or trusted vendor
- The scenario is highly specific and time-sensitive but not overtly urgent
- The email includes realistic formatting, signatures, and disclaimers
- The call to action feels completely natural (review a document, confirm attendance, approve a request)

This is for testing the most security-aware employees. Only someone who verifies out-of-band (calling the sender, checking headers) would catch it.`,
}

// GetSystemPrompt returns the system prompt for the given difficulty level.
// If the level is out of range it defaults to DifficultyMedium.
func GetSystemPrompt(level int) string {
	if p, ok := systemPrompts[level]; ok {
		return p
	}
	return systemPrompts[DifficultyMedium]
}

// BuildUserPrompt constructs the user-facing prompt from the generation request
// parameters. It includes any recipient context that was provided.
func BuildUserPrompt(req GenerateRequest) string {
	prompt := fmt.Sprintf("Generate a phishing simulation email with the following parameters:\n\n")

	if req.Prompt != "" {
		prompt += fmt.Sprintf("Scenario/Theme: %s\n", req.Prompt)
	}
	if req.Language != "" && req.Language != "en" {
		prompt += fmt.Sprintf("Language: Write the email in %s\n", req.Language)
	}
	if req.TargetRole != "" {
		prompt += fmt.Sprintf("Target job role: %s\n", req.TargetRole)
	}
	if req.TargetDepartment != "" {
		prompt += fmt.Sprintf("Target department: %s\n", req.TargetDepartment)
	}
	if req.TargetIndustry != "" {
		prompt += fmt.Sprintf("Target industry: %s\n", req.TargetIndustry)
	}
	if req.SenderName != "" {
		prompt += fmt.Sprintf("Impersonated sender name: %s\n", req.SenderName)
	}
	if req.CompanyName != "" {
		prompt += fmt.Sprintf("Target company name: %s\n", req.CompanyName)
	}

	prompt += `
Return your response as a JSON object with exactly these fields:
{
  "subject": "The email subject line",
  "html": "The full HTML email body (use simple inline-styled HTML suitable for email clients)",
  "text": "A plain-text version of the email body"
}

IMPORTANT:
- Include {{.FirstName}}, {{.LastName}}, and {{.URL}} as GoPhish template variables where appropriate
- {{.URL}} should be used for any links the recipient is supposed to click
- Do not include any explanation outside the JSON object
- The HTML should be clean, professional email HTML with inline styles
- Do NOT wrap the JSON in markdown code fences`

	return prompt
}

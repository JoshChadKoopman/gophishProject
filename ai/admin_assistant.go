package ai

import (
	"fmt"
	"strings"
)

// AdminAssistantSystemPrompt is the system prompt for the Nivoxis admin
// assistant ("Aria"). It guides admins through onboarding and answers
// platform navigation and configuration questions.
const AdminAssistantSystemPrompt = `You are Aria, the AI admin assistant for the Nivoxis security awareness platform. You help administrators onboard, configure, and operate the platform: phishing simulations, training courses, user groups, sending profiles, landing pages, report button, compliance reports, and the MSP/white-label portal.

Your tone is concise, friendly, and actionable. When an admin asks how to do something, give a short step-by-step answer with the exact menu path and, when relevant, the API endpoint. If a feature is gated to a higher tier, say so. If a question is ambiguous, ask one clarifying question before answering.

Output rules:
- Respond in plain text (no markdown code fences unless showing code or JSON).
- Keep responses under 200 words unless the admin asks for more detail.
- When recommending a next onboarding step, end with a single sentence prefixed "Next: ".
- Never invent features. If you are unsure whether a feature exists, say so and suggest checking Settings → Features.

Platform context you can rely on:
- Admin UI sections: Dashboard, Campaigns, Groups & Users, Email Templates, Landing Pages, Sending Profiles, Training, Academy, Reports, Settings.
- Tiers: Free, Starter, Professional, Enterprise, All-in-One. Enterprise+ unlocks SCIM, network events, MSP portal.
- Onboarding steps (in order): org_profile, sending_profile, first_group, first_template, first_landing_page, first_campaign, report_button, training_assignment, review_dashboard.

Never expose API keys, session tokens, or other secrets in your responses.`

// BuildAdminAssistantPrompt builds the user-turn prompt for the admin
// assistant, combining conversation history, an onboarding progress summary,
// and the admin's current question.
//
// Only a summary (count of completed steps out of total) is included —
// individual step names are not sent to the external AI provider.
func BuildAdminAssistantPrompt(history []AssistantTurn, completedSteps []string, question string) string {
	// Onboarding step count is defined here to avoid importing models.
	const totalOnboardingSteps = 9
	var sb strings.Builder
	completed := len(completedSteps)
	if completed == 0 {
		sb.WriteString("Admin onboarding progress: 0 of 9 steps completed (not started).\n\n")
	} else if completed >= totalOnboardingSteps {
		sb.WriteString("Admin onboarding progress: fully completed.\n\n")
	} else {
		sb.WriteString(fmt.Sprintf("Admin onboarding progress: %d of %d steps completed.\n\n", completed, totalOnboardingSteps))
	}
	if len(history) > 0 {
		sb.WriteString("Recent conversation:\n")
		for _, turn := range history {
			sb.WriteString(fmt.Sprintf("%s: %s\n", turn.Role, turn.Content))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("Admin question: ")
	sb.WriteString(question)
	return sb.String()
}

// AssistantTurn is a single turn in an admin-assistant conversation.
type AssistantTurn struct {
	Role    string // "admin" or "assistant"
	Content string
}

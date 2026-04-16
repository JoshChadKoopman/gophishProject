package models

import (
	"testing"
)

func TestOnboardingStepsOrder(t *testing.T) {
	if len(OnboardingSteps) != 9 {
		t.Errorf("Expected 9 onboarding steps, got %d", len(OnboardingSteps))
	}
	if OnboardingSteps[0] != "org_profile" {
		t.Errorf("Expected first step to be org_profile, got %s", OnboardingSteps[0])
	}
	if OnboardingSteps[len(OnboardingSteps)-1] != "review_dashboard" {
		t.Errorf("Expected last step to be review_dashboard, got %s", OnboardingSteps[len(OnboardingSteps)-1])
	}
}

func TestAssistantConversationTableName(t *testing.T) {
	c := AssistantConversation{}
	if c.TableName() != "assistant_conversations" {
		t.Errorf("Expected assistant_conversations, got %s", c.TableName())
	}
}

func TestAssistantMessageTableName(t *testing.T) {
	m := AssistantMessage{}
	if m.TableName() != "assistant_messages" {
		t.Errorf("Expected assistant_messages, got %s", m.TableName())
	}
}

func TestAdminOnboardingProgressTableName(t *testing.T) {
	p := AdminOnboardingProgress{}
	if p.TableName() != "admin_onboarding_progress" {
		t.Errorf("Expected admin_onboarding_progress, got %s", p.TableName())
	}
}

func TestAssistantRoleConstants(t *testing.T) {
	if AssistantRoleAdmin != "admin" {
		t.Errorf("Expected admin, got %s", AssistantRoleAdmin)
	}
	if AssistantRoleAssistant != "assistant" {
		t.Errorf("Expected assistant, got %s", AssistantRoleAssistant)
	}
}

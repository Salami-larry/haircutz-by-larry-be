package jobs

import (
	"testing"
	"time"
)

func TestReminderConstants(t *testing.T) {
	if ReminderLeadWindow != 15*time.Minute {
		t.Fatalf("ReminderLeadWindow = %v", ReminderLeadWindow)
	}
	if ReminderMinLead != 15*time.Minute {
		t.Fatalf("ReminderMinLead = %v", ReminderMinLead)
	}
	if PostTimeNagInterval != 30*time.Minute {
		t.Fatalf("PostTimeNagInterval = %v", PostTimeNagInterval)
	}
}

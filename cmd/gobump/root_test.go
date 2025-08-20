package cmd

import (
	"strings"
	"testing"
)

func TestRootCmdWorkFlag(t *testing.T) {
	// Test that the work flag is properly registered
	cmd := RootCmd()

	// Check if the work flag exists
	workFlag := cmd.Flags().Lookup("work")
	if workFlag == nil {
		t.Error("work flag not found")
		return
	}

	// Verify flag properties
	if workFlag.Value.Type() != "bool" {
		t.Errorf("work flag type: got %s, want bool", workFlag.Value.Type())
	}

	if workFlag.DefValue != "false" {
		t.Errorf("work flag default: got %s, want false", workFlag.DefValue)
	}

	if !strings.Contains(workFlag.Usage, "go work vendor") {
		t.Errorf("work flag usage doesn't mention 'go work vendor': %s", workFlag.Usage)
	}
}

func TestRootCLIFlagsStructure(t *testing.T) {
	// Verify the rootCLIFlags struct has the work field
	flags := rootCLIFlags{
		work: true,
	}

	if !flags.work {
		t.Error("work field not properly set in rootCLIFlags")
	}
}

package fornaxmirror

import "testing"

func TestDecodeConfigAcceptsSourceList(t *testing.T) {
	config, err := DecodeConfig([]byte(`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write"},{"workspace_id":"2","key_prefix":"loop.workflow.","access":"read_only"}]}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(config.Sources) != 2 || config.Sources[0].Access != AccessReadWrite || config.Sources[1].KeyPrefix != "loop.workflow." {
		t.Fatalf("config = %#v", config)
	}
	source, err := config.ReadWriteSource()
	if err != nil {
		t.Fatal(err)
	}
	if source.WorkspaceID != "1" || source.KeyPrefix != "fanloop.workflow." {
		t.Fatalf("read_write source = %#v", source)
	}
	for _, input := range []string{
		`{"workspace_id":"1","workflows":{"linear-runtime":"fanloop.workflow.linear_runtime"}}`,
		`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"read_write","ak":"secret"}]}`,
		`{"sources":[{"workspace_id":"1","key_prefix":"fanloop.workflow.","access":"write"}]}`,
		`{"sources":[]}`,
	} {
		if _, err = DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("invalid config passed: %s", input)
		}
	}
}

func TestSelectClaimsPersonalSourceWins(t *testing.T) {
	selected, err := SelectClaims([]PackageClaim{
		{WorkflowID: "general", Key: "fanloop.workflow.general", Version: "1.0.0", Access: AccessReadWrite, SourceIndex: 0},
		{WorkflowID: "pstack-native", Key: "loop.workflow.pstack_native", Version: "1.0.2", Access: AccessReadOnly, SourceIndex: 1},
		{WorkflowID: "pstack-native", Key: "fanloop.workflow.pstack_native", Version: "1.0.0", Access: AccessReadWrite, SourceIndex: 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected) != 2 {
		t.Fatalf("selected = %#v", selected)
	}
	if selected[0].WorkflowID != "general" || selected[0].Key != "fanloop.workflow.general" || selected[0].Version != "1.0.0" {
		t.Fatalf("first = %#v", selected[0])
	}
	if selected[1].WorkflowID != "pstack-native" || selected[1].Key != "fanloop.workflow.pstack_native" || selected[1].Version != "1.0.0" {
		t.Fatalf("second = %#v", selected[1])
	}
}

func TestSelectClaimsRejectsTiesAndSameSourceDuplicates(t *testing.T) {
	_, err := SelectClaims([]PackageClaim{
		{WorkflowID: "general", Key: "loop.workflow.general", Version: "1.0.0", Access: AccessReadOnly, SourceIndex: 0},
		{WorkflowID: "general", Key: "loop.workflow.general_copy", Version: "1.0.1", Access: AccessReadOnly, SourceIndex: 1},
	})
	if err == nil || err.Error() != "workflow id general is claimed by 2 sources" {
		t.Fatalf("tie error = %v", err)
	}
	_, err = SelectClaims([]PackageClaim{
		{WorkflowID: "general", Key: "loop.workflow.general", Version: "1.0.0", Access: AccessReadOnly, SourceIndex: 0},
		{WorkflowID: "general", Key: "loop.workflow.general_old", Version: "0.9.0", Access: AccessReadOnly, SourceIndex: 0},
	})
	if err == nil || err.Error() != "workflow id general appears twice in one source" {
		t.Fatalf("duplicate error = %v", err)
	}
}

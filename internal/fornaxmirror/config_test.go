package fornaxmirror

import "testing"

func TestDecodeConfigAcceptsOnlyWorkspaceAndExactKeys(t *testing.T) {
	config, err := DecodeConfig([]byte(`{"workspace_id":"1","workflows":{"linear-runtime":"fanloop.workflow.linear_runtime"}}`))
	if err != nil {
		t.Fatal(err)
	}
	if config.WorkspaceID != "1" || config.Workflows["linear-runtime"] != "fanloop.workflow.linear_runtime" {
		t.Fatalf("config = %#v", config)
	}
	for _, input := range []string{
		`{"workspace_id":"1","workflows":[{"id":"linear-runtime","skill_key":"fanloop.workflow.linear-runtime","skill_id":"1"}]}`,
		`{"workspace_id":"1","workflows":{"linear-runtime":"fanloop.workflow.linear-runtime"}}`,
		`{"workspace_id":"1","workflows":{"linear-runtime":"fanloop.workflow.linear_runtime"},"ak":"secret"}`,
	} {
		if _, err = DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("invalid config passed: %s", input)
		}
	}
}

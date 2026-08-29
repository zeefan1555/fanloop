package flowidl

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	"github.com/zeefan1555/fanloop/internal/idl/commonidl"
)

type fakeService struct{}

func (fakeService) Init(context.Context, string, *FlowInitRequest, bool) (*FlowInitResponse, error) {
	return nil, nil
}
func (fakeService) Status(context.Context, string, *FlowStatusRequest) (*FlowStatusResponse, error) {
	return nil, nil
}
func (fakeService) Progress(context.Context, string, *FlowProgressRequest, bool) (*FlowProgressResponse, error) {
	return nil, nil
}
func (fakeService) Result(context.Context, string, *FlowResultRequest, bool) (*FlowResultResponse, error) {
	return nil, nil
}

func TestGeneratedContractCompiles(t *testing.T) {
	var _ FlowService = fakeService{}
	encoded, err := json.Marshal(ResultEffect_advanced)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `"advanced"` {
		t.Fatalf("enum JSON = %s", encoded)
	}
}

func TestJsonValueUnionIsDetectable(t *testing.T) {
	raw := "https://example.com/mr/42"
	value := &commonidl.JsonValue{StringValue: &raw}
	if value.CountSetFieldsJsonValue() != 1 {
		t.Fatal("expected one union field")
	}
}

func TestStaticValidationRejectsZeroEnum(t *testing.T) {
	request := NewFlowProgressRequest()
	request.StepId = "step"
	request.Summary = "working"
	request.Evidence = []*Evidence{}
	if err := request.IsValid(); err == nil {
		t.Fatal("expected unspecified progress status rejection")
	}
}

func TestResultRequiresConditionResultsAndAllowsEmptyEvidence(t *testing.T) {
	request := NewFlowResultRequest()
	request.StepId = "current"
	request.Summary = "result ready"
	request.Evidence = []*Evidence{}
	if err := request.IsValid(); err == nil {
		t.Fatal("expected empty Condition results rejection")
	}
	request.ConditionResults = []*ConditionResult{{
		ConditionId: "completed",
		Output: &OutputValue{
			Type:  OutputType_enum_value,
			Value: &commonidl.JsonValue{StringValue: ptr("completed")},
		},
	}}
	request.Route = &RouteSelection{NextStepId: ptr("next")}
	if err := request.IsValid(); err != nil {
		t.Fatalf("valid Result rejected: %v", err)
	}
}

func TestJsonValueNaturalJSONRoundTrip(t *testing.T) {
	input := `{"approved":true,"message_id":"om_xxx","attempt":2,"reviewers":["ou_a","ou_b"]}`
	var value commonidl.JsonValue
	if err := json.Unmarshal([]byte(input), &value); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(&value)
	if err != nil {
		t.Fatal(err)
	}
	var got, want any
	if err := json.Unmarshal(encoded, &got); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(input), &want); err != nil {
		t.Fatal(err)
	}
	if !deepJSONEqual(got, want) {
		t.Fatalf("round trip = %s", encoded)
	}
}

func TestJsonValueNaturalJSONRejectsInvalidValues(t *testing.T) {
	for _, input := range []string{`{"x":}`, `1 2`} {
		t.Run(input, func(t *testing.T) {
			var value commonidl.JsonValue
			if err := json.Unmarshal([]byte(input), &value); err == nil {
				t.Fatalf("expected %s to fail", input)
			}
		})
	}

	value := commonidl.JsonValue{StringValue: ptr("text"), BoolValue: ptr(true)}
	if _, err := json.Marshal(&value); err == nil {
		t.Fatal("expected multi-field union to fail")
	}
}

func deepJSONEqual(left, right any) bool {
	leftNumber, leftIsNumber := left.(float64)
	rightNumber, rightIsNumber := right.(float64)
	if leftIsNumber || rightIsNumber {
		return leftIsNumber && rightIsNumber && math.Abs(leftNumber-rightNumber) == 0
	}
	leftObject, leftIsObject := left.(map[string]any)
	rightObject, rightIsObject := right.(map[string]any)
	if leftIsObject || rightIsObject {
		if !leftIsObject || !rightIsObject || len(leftObject) != len(rightObject) {
			return false
		}
		for key, leftValue := range leftObject {
			if !deepJSONEqual(leftValue, rightObject[key]) {
				return false
			}
		}
		return true
	}
	leftList, leftIsList := left.([]any)
	rightList, rightIsList := right.([]any)
	if leftIsList || rightIsList {
		if !leftIsList || !rightIsList || len(leftList) != len(rightList) {
			return false
		}
		for index := range leftList {
			if !deepJSONEqual(leftList[index], rightList[index]) {
				return false
			}
		}
		return true
	}
	return left == right
}

func ptr[T any](value T) *T { return &value }

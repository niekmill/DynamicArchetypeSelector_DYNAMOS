package main

import (
	"testing"

	"github.com/Jorrit05/DYNAMOS/pkg/lib"
	pb "github.com/Jorrit05/DYNAMOS/pkg/proto"
)


func validationResponseFor(allowedPerProvider map[string][]string, options map[string]bool) *pb.ValidationResponse {
	archetypes := map[string]*pb.UserAllowedArchetypes{}
	dataProviders := map[string]*pb.DataProvider{}
	for provider, archs := range allowedPerProvider {
		archetypes[provider] = &pb.UserAllowedArchetypes{Archetypes: archs}
		dataProviders[provider] = &pb.DataProvider{Archetypes: archs}
	}
	return &pb.ValidationResponse{
		Type:               "validationResponse",
		RequestType:        "sqlDataRequest",
		RequestApproved:    true,
		Options:            options,
		User:               &pb.User{Id: "test-user", UserName: "test@example.com"},
		ValidArchetypes:    &pb.UserArchetypes{Archetypes: archetypes},
		ValidDataproviders: dataProviders,
	}
}

func agentDetailsFor(providers ...string) map[string]lib.AgentDetails {
	out := map[string]lib.AgentDetails{}
	for _, p := range providers {
		out[p] = lib.AgentDetails{
			Name:       p,
			RoutingKey: p + "-in",
			Dns:        p + "." + p + ".svc.cluster.local",
		}
	}
	return out
}

func TestPickArchetypeBasedOnTopsis_PicksLowestCostWhenBothAllowed(t *testing.T) {
	vr := validationResponseFor(map[string][]string{
		"UVA": {"computeToData", "dataThroughTtp"},
		"VU":  {"computeToData", "dataThroughTtp"},
	}, map[string]bool{})

	result, _, err := pickArchetypeBasedOnTopsis(vr, agentDetailsFor("UVA", "VU"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "dataThroughTtp" {
		t.Fatalf("expected dataThroughTtp (cheaper at 2 providers), got %q", result)
	}
}


func TestPickArchetypeBasedOnTopsis_RestrictsToIntersection(t *testing.T) {
	vr := validationResponseFor(map[string][]string{
		"UVA": {"computeToData", "dataThroughTtp"},
		"VU":  {"computeToData"},
	}, map[string]bool{})

	result, _, err := pickArchetypeBasedOnTopsis(vr, agentDetailsFor("UVA", "VU"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "computeToData" {
		t.Fatalf("expected computeToData (only one in intersection), got %q", result)
	}
}


func TestPickArchetypeBasedOnTopsis_ErrorsWhenIntersectionEmpty(t *testing.T) {
	vr := validationResponseFor(map[string][]string{
		"UVA": {"computeToData"},
		"VU":  {"dataThroughTtp"},
	}, map[string]bool{})

	_, _, err := pickArchetypeBasedOnTopsis(vr, agentDetailsFor("UVA", "VU"))
	if err == nil {
		t.Fatalf("expected error when intersection is empty, got nil")
	}
}

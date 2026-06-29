package main

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/Jorrit05/DYNAMOS/pkg/api"
	"github.com/Jorrit05/DYNAMOS/pkg/etcd"
	"github.com/Jorrit05/DYNAMOS/pkg/lib"
	pb "github.com/Jorrit05/DYNAMOS/pkg/proto"
	"github.com/Jorrit05/DYNAMOS/pkg/solver"
	"go.opencensus.io/trace"
)

func solverEnabled() bool {
	v := strings.ToLower(os.Getenv("SOLVER_ENABLED"))
	return v != "false" && v != "0" && v != "off"
}

func executionArchetypeFor(name string) string {
	for _, a := range solver.DefaultArchetypes() {
		if a.Name != name {
			continue
		}
		family, _ := a.Attributes["family"].(string)
		if family != "" && family != name {
			return family
		}
		return name
	}
	return name
}

type UnauthorizedProviderError struct {
	ProviderName string
}

func (e *UnauthorizedProviderError) Error() string {
	return fmt.Sprintf("third party '%s' is not online", e.ProviderName)
}

func startCompositionRequest(ctx context.Context, validationResponse *pb.ValidationResponse, authorizedProviders map[string]lib.AgentDetails, compositionRequest *pb.CompositionRequest) (map[string]string, context.Context, *pb.TopsisDecision, error) {
	logger.Debug("Entering startCompositionRequest")

	ctx, span := trace.StartSpan(ctx, "startCompositionRequest")
	defer span.End()

	archetype, decision, err := chooseArchetype(validationResponse, authorizedProviders)
	if err != nil {
		return nil, ctx, nil, err
	}
	executionArchetype := executionArchetypeFor(archetype)
	if executionArchetype != archetype {
		logger.Sugar().Infof("Chosen archetype: %s (executing as base family %s)", archetype, executionArchetype)
	} else {
		logger.Sugar().Infof("Chosen archetype: %s", archetype)
	}

	var archetypeConfig api.Archetype
	_, err = etcd.GetAndUnmarshalJSON(etcdClient, fmt.Sprintf("/archetypes/%s", executionArchetype), &archetypeConfig)
	if err != nil {
		return nil, ctx, nil, err
	}

	compositionRequest.User = validationResponse.User
	compositionRequest.DataProviders = []string{}
	compositionRequest.ArchetypeId = executionArchetype
	compositionRequest.RequestType = validationResponse.RequestType
	compositionRequest.JobName = lib.GenerateJobName(validationResponse.User.UserName, 8)

	// Save the ActiveJob to etcd
	// var activeJob = &pb.ActiveJob{
	// 	JobId:               compositionRequest.JobName,
	// 	Type:                validationResponse.RequestType,
	// 	User:                validationResponse.User,
	// 	Archetype:           archetype,
	// 	AuthorizedProviders: make(map[string]string),
	// }
	// for name, agent := range authorizedProviders {
	// 	activeJob.AuthorizedProviders[name] = agent.Dns
	// }

	// etcd.SaveStructToEtcd(etcdClient, fmt.Sprintf("/activeJobs/%s/%s", validationResponse.User.Id, compositionRequest.JobName), activeJob)

	// Use to return the proper endpoints to the user
	userTargets := make(map[string]string)

	if archetypeConfig.ComputeProvider != "other" {
		// Compute to data
		compositionRequest.Role = "all"
		for key := range authorizedProviders {
			compositionRequest.DestinationQueue = authorizedProviders[key].RoutingKey
			c.SendCompositionRequest(ctx, compositionRequest)
			userTargets[key] = authorizedProviders[key].Dns
		}
	} else {
		// TODO: Here I am assuming that the initial archetype choice is correct
		// and that this is the only possible archetype.
		// I should probably build in that if there is no TTP online or available
		// Or Universities have different TTPs. That these scenarios are handled as well.
		ttp, err := chooseThirdParty(validationResponse)
		if err != nil {
			return nil, ctx, nil, err
		}

		// Send to each validData provider the role data provider
		// Send to the thirdParty the role Compute provider
		compositionRequest.Role = "dataProvider"
		tmpDataProvider := []string{}
		for key := range authorizedProviders {
			tmpDataProvider = append(tmpDataProvider, key)
			compositionRequest.DestinationQueue = authorizedProviders[key].RoutingKey
			c.SendCompositionRequest(ctx, compositionRequest)
		}

		compositionRequest.DataProviders = tmpDataProvider
		compositionRequest.Role = "computeProvider"
		compositionRequest.DestinationQueue = ttp.RoutingKey
		userTargets[ttp.Name] = ttp.Dns

		c.SendCompositionRequest(ctx, compositionRequest)
	}
	return userTargets, ctx, decision, nil
}


func pickArchetypeBasedOnWeight() (*api.Archetype, error) {
	logger.Sugar().Info("Start pickArchetypeBasedOnWeight")

	target := &api.Archetype{}

	archeTypes, err := etcd.GetPrefixListEtcd(etcdClient, "/archetypes", target)
	if err != nil {
		return nil, err
	}

	if len(archeTypes) == 0 {
		return nil, fmt.Errorf("no archetypes available")
	}

	archetypeNames := make([]string, len(archeTypes))
	for i, archetype := range archeTypes {
		archetypeNames[i] = archetype.Name 
	}
	logger.Sugar().Infof("Possible archetypes: %v", archetypeNames)

	lightest := archeTypes[0]

	for _, archeType := range archeTypes {
		if archeType.Weight < lightest.Weight {
			lightest = archeType
		}
	}

	return lightest, nil
}

func getArchetypeBasedOnOptions(validationResponse *pb.ValidationResponse, authorizedDataProviders map[string]lib.AgentDetails) string {
	logger.Sugar().Debugf("Start getArchetypeBasedOnOptions, options: %v", validationResponse.Options)

	// This ranges over the options. And selects an archetype based on the options.
	for option, value := range validationResponse.Options {
		switch option {
		case "aggregate":
			// If aggregate is enabled, it will select the 'dataThroughTtp' archetype, if this is allowed on all the authorizedDataProviders
			if value {
				allowed := true
				for provider, _ := range authorizedDataProviders {
					if !slices.Contains(validationResponse.ValidArchetypes.Archetypes[provider].Archetypes, "dataThroughTtp") {
						logger.Sugar().Debugf("allowed false, slice: %v", validationResponse.ValidArchetypes.Archetypes[provider].Archetypes)

						allowed = false
					}
				}

				if allowed {
					return "dataThroughTtp"
				}
			}
		}
	}
	return ""
}

func pickArchetypeBasedOnTopsis(validationResponse *pb.ValidationResponse, authorizedDataProviders map[string]lib.AgentDetails) (string, *pb.TopsisDecision, error) {
	logger.Sugar().Debug("starting pickArchetypeBasedOnTopsis")

	counts := map[string]int{}
	for provider := range authorizedDataProviders {
		valid, ok := validationResponse.ValidArchetypes.Archetypes[provider]
		if !ok {
			return "", nil, fmt.Errorf("no valid archetypes for provider %s", provider)
		}
		for _, name := range valid.Archetypes {
			counts[name]++
		}
	}
	intersection := map[string]bool{}
	for name, c := range counts {
		if c == len(authorizedDataProviders) {
			intersection[name] = true
		}
	}
	if len(intersection) == 0 {
		return "", nil, fmt.Errorf("no archetype is allowed by every authorized provider")
	}

	candidates := []solver.Archetype{}
	for _, a := range solver.DefaultArchetypes() {
		if intersection[a.Name] {
			candidates = append(candidates, a)
		}
	}
	if len(candidates) == 0 {
		return "", nil, fmt.Errorf("none of the intersection archetypes are known to the solver")
	}

	expectedRows := int(validationResponse.ExpectedRows)
	hasHint := expectedRows > 0

	extraHops := 0
	if v, ok := validationResponse.Options["graph"]; ok && v {
		extraHops++
	}
	if v, ok := validationResponse.Options["aggregate"]; ok && v {
		extraHops++
	}

	requestedProviders := len(validationResponse.ValidDataproviders) + len(validationResponse.InvalidDataproviders)
	logger.Sugar().Infof(
		"Request context: user=%q requestType=%q expectedRows=%d hasHint=%v hops=%d providers(authorized=%d, requested=%d, invalid=%v) options=%v",
		validationResponse.User.GetUserName(),
		validationResponse.RequestType,
		expectedRows, hasHint, extraHops,
		len(authorizedDataProviders), requestedProviders,
		validationResponse.InvalidDataproviders,
		validationResponse.Options,
	)

	ctx := solver.RequestContext{
		SQLLimit:            expectedRows,
		HasSQLLimit:         hasHint,
		OptionalServiceHops: extraHops,
		DataProviderCount: requestedProviders,
		IsFirstRequest:    validationResponse.IsFirstRequest,
	}

	defaults := solver.DefaultCriteria()
	var activeSet map[string]bool
	if len(validationResponse.ActiveCriteria) > 0 {
		activeSet = map[string]bool{}
		for _, name := range validationResponse.ActiveCriteria {
			activeSet[name] = true
		}
	}
	criteria := make([]solver.Criterion, 0, len(defaults))
	for _, c := range defaults {
		if activeSet != nil && !activeSet[c.Name] {
			continue
		}
		if w, ok := validationResponse.Weights[c.Name]; ok && w > 0 {
			c.Weight = w
		}
		criteria = append(criteria, c)
	}
	if len(criteria) == 0 {
		return "", nil, fmt.Errorf("no active criteria after filtering — check the 'criteria' field")
	}

	constraints := make([]solver.Constraint, 0, len(validationResponse.Constraints))
	for _, pc := range validationResponse.Constraints {
		constraints = append(constraints, solver.Constraint{
			Criterion: pc.Criterion,
			Operator:  solver.Operator(pc.Operator),
			Threshold: pc.Threshold,
		})
	}

	logger.Sugar().Infof("========== TOPSIS evaluation ==========")
	logger.Sugar().Infof("TOPSIS candidates (%d): %s", len(candidates), candidateNames(candidates))
	for _, c := range criteria {
		logger.Sugar().Infof("TOPSIS criterion: name=%-25s weight=%.3f bounds=[%g,%g] minimize=%v",
			c.Name, c.Weight, c.Lower, c.Upper, c.Minimize)
	}
	if len(constraints) == 0 {
		logger.Sugar().Infof("TOPSIS constraints: (none)")
	} else {
		for _, k := range constraints {
			logger.Sugar().Infof("TOPSIS constraint: %s %s %g", k.Criterion, k.Operator, k.Threshold)
		}
	}
	logger.Sugar().Infof("TOPSIS raw decision matrix:")
	for _, a := range candidates {
		parts := make([]string, 0, len(criteria))
		for _, c := range criteria {
			parts = append(parts, fmt.Sprintf("%s=%.4g", c.Name, c.Score(a, ctx)))
		}
		logger.Sugar().Infof("  %-25s %s", a.Name, joinCriterionScores(parts))
	}

	// 7. Solve.
	result, err := solver.Topsis(candidates, criteria, constraints, ctx)
	if err != nil {
		return "", nil, fmt.Errorf("topsis: %w", err)
	}

	// 8. Post-solve trace.
	if len(result.Dropped) == 0 {
		logger.Sugar().Infof("TOPSIS dropped: (none)")
	} else {
		for name, reason := range result.Dropped {
			logger.Sugar().Infof("TOPSIS dropped: %s -> %s", name, reason)
		}
	}
	logger.Sugar().Infof("TOPSIS normalised matrix (survivors):")
	for _, a := range candidates {
		row, ok := result.Normalised[a.Name]
		if !ok {
			continue
		}
		parts := make([]string, 0, len(row))
		for i, c := range criteria {
			parts = append(parts, fmt.Sprintf("%s=%.4f", c.Name, row[i]))
		}
		logger.Sugar().Infof("  %-25s %s", a.Name, joinCriterionScores(parts))
	}
	if len(result.Ranking) == 0 {
		return "", nil, fmt.Errorf("topsis returned no ranking")
	}
	logger.Sugar().Infof("TOPSIS ranking:")
	for i, r := range result.Ranking {
		logger.Sugar().Infof("  %d. %-25s score=%.4f", i+1, r.Name, r.Score)
	}

	winner := result.Ranking[0].Name
	logger.Sugar().Infof("EXITING SOLVER, winner=%s", winner)
	logger.Sugar().Infof("========================================")

	scores := make(map[string]float64, len(result.Ranking))
	for _, r := range result.Ranking {
		scores[r.Name] = r.Score
	}
	return winner, &pb.TopsisDecision{
		ChosenArchetype: winner,
		Scores:          scores,
		Source:          "topsis",
	}, nil
}

func candidateNames(arr []solver.Archetype) string {
	names := make([]string, 0, len(arr))
	for _, a := range arr {
		names = append(names, a.Name)
	}
	return strings.Join(names, ", ")
}

func joinCriterionScores(parts []string) string {
	return strings.Join(parts, "  ")
}

func chooseArchetype(validationResponse *pb.ValidationResponse, authorizedDataProviders map[string]lib.AgentDetails) (string, *pb.TopsisDecision, error) {
	logger.Sugar().Debug("starting chooseArchetype")
	logger.Sugar().Debugf("length options: %v", len(validationResponse.Options))

	for k := range validationResponse.ValidDataproviders {
		logger.Sugar().Debugf("validDataprovider: %s", k)
	}

	if solverEnabled() {
		if archetype, decision, err := pickArchetypeBasedOnTopsis(validationResponse, authorizedDataProviders); err == nil {
			return archetype, decision, nil
		} else {
			logger.Sugar().Warnf("TOPSIS picker failed, falling back to options/weight: %v", err)
		}
	} else {
		logger.Sugar().Infof("SOLVER_ENABLED=false, skipping TOPSIS")
	}

	if validationResponse.Options != nil && len(validationResponse.Options) > 0 {
		archetype := getArchetypeBasedOnOptions(validationResponse, authorizedDataProviders)
		if archetype != "" {
			return archetype, &pb.TopsisDecision{ChosenArchetype: archetype, Source: "options-rules"}, nil
		}
	}

	archeType, err := pickArchetypeBasedOnWeight()
	if err != nil {
		return "", nil, err
	}
	allowed := true
	for provider := range authorizedDataProviders {
		if !slices.Contains(validationResponse.ValidArchetypes.Archetypes[provider].Archetypes, archeType.Name) {
			allowed = false
		}
	}
	if allowed {
		return archeType.Name, &pb.TopsisDecision{ChosenArchetype: archeType.Name, Source: "weight-based"}, nil
	}

	for provider := range authorizedDataProviders {
		someArchetype := validationResponse.ValidArchetypes.Archetypes[provider].Archetypes[0]
		if someArchetype != "" {
			return someArchetype, &pb.TopsisDecision{ChosenArchetype: someArchetype, Source: "intersection-fallback"}, nil
		}
	}

	return "", nil, fmt.Errorf("unexpected error: could not retrieve an archetype from the intersection")
}

func chooseThirdParty(validationResponse *pb.ValidationResponse) (lib.AgentDetails, error) {

	intersectionMap := make(map[string]int)
	totalProviders := len(validationResponse.ValidDataproviders)

	// Iterate over all valid dataproviders
	for _, dataProvider := range validationResponse.ValidDataproviders {
		// For each compute provider in a valid dataprovider
		for _, computeProvider := range dataProvider.ComputeProviders {
			intersectionMap[computeProvider]++
		}
	}

	// Extract the intersection from the map
	var intersection []string
	for provider, count := range intersectionMap {
		if count == totalProviders {
			intersection = append(intersection, provider)
		}
	}

	// If the intersection is empty, return an error
	if len(intersection) == 0 {
		return lib.AgentDetails{}, fmt.Errorf("no common compute providers found")
	}

	// If the intersection is not empty, return the first item
	var agentData lib.AgentDetails

	json, err := etcd.GetAndUnmarshalJSON(etcdClient, fmt.Sprintf("/agents/online/%s", intersection[0]), &agentData)
	if err != nil {
		return lib.AgentDetails{}, err
	} else if json == nil {
		return lib.AgentDetails{}, &UnauthorizedProviderError{ProviderName: intersection[0]}
	}

	return agentData, nil
}

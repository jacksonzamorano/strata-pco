package main

import (
	d "github.com/jacksonzamorano/strata-pco/definitions"
	"github.com/jacksonzamorano/strata/component"
)

func listServiceTypes(
	input struct{},
	ctx *component.ComponentContainer,
) (*d.ListServiceTypesOutput, error) {
	serviceTypes, err := fetchServiceTypes(ctx)
	if err != nil {
		ctx.Logger.Log("Error when listing service types: %s", err.Error())
		return nil, err
	}
	return &d.ListServiceTypesOutput{Items: serviceTypes}, nil
}

func listPlans(
	input d.ListPlansInput,
	ctx *component.ComponentContainer,
) (*d.ListPlansOutput, error) {
	plans, err := fetchPlans(ctx, input)
	if err != nil {
		ctx.Logger.Log("Error when listing plans: %s", err.Error())
		return nil, err
	}
	return &d.ListPlansOutput{Items: plans}, nil
}

func getPlanDetails(
	input d.GetPlanDetailsInput,
	ctx *component.ComponentContainer,
) (*d.GetPlanDetailsOutput, error) {
	plan, teams, err := fetchPlanDetails(ctx, input)
	if err != nil {
		ctx.Logger.Log("Error when getting plan details: %s", err.Error())
		return nil, err
	}
	return &d.GetPlanDetailsOutput{Plan: plan, Teams: teams}, nil
}

func clearAuthorization(
	input struct{},
	ctx *component.ComponentContainer,
) (*string, error) {
	clearStoredAuthorization(ctx)
	message := "Planning Center authorization cleared."
	return &message, nil
}

func main() {
	component.CreateComponent(
		d.Manifest,
		component.Mount(d.ListServiceTypes, listServiceTypes),
		component.Mount(d.ListPlans, listPlans),
		component.Mount(d.GetPlanDetails, getPlanDetails),
		component.Mount(d.ClearAuthorization, clearAuthorization),
	).Start()
}

package main

import (
	d "github.com/jacksonzamorano/strata-pco/definitions"
	"github.com/jacksonzamorano/strata/component"
)

func listServiceTypes(
	input *component.ComponentInput[struct{}, d.ListServiceTypesOutput],
	ctx *component.ComponentContainer,
) *component.ComponentReturn[d.ListServiceTypesOutput] {
	serviceTypes, err := fetchServiceTypes(ctx)
	if err != nil {
		ctx.Logger.Log("Error when listing service types: %s", err.Error())
		return input.Error(err.Error())
	}
	return input.Return(d.ListServiceTypesOutput{Items: serviceTypes})
}

func listPlans(
	input *component.ComponentInput[d.ListPlansInput, d.ListPlansOutput],
	ctx *component.ComponentContainer,
) *component.ComponentReturn[d.ListPlansOutput] {
	plans, err := fetchPlans(ctx, input.Body)
	if err != nil {
		ctx.Logger.Log("Error when listing plans: %s", err.Error())
		return input.Error(err.Error())
	}
	return input.Return(d.ListPlansOutput{Items: plans})
}

func getPlanDetails(
	input *component.ComponentInput[d.GetPlanDetailsInput, d.GetPlanDetailsOutput],
	ctx *component.ComponentContainer,
) *component.ComponentReturn[d.GetPlanDetailsOutput] {
	plan, teams, err := fetchPlanDetails(ctx, input.Body)
	if err != nil {
		ctx.Logger.Log("Error when getting plan details: %s", err.Error())
		return input.Error(err.Error())
	}
	return input.Return(d.GetPlanDetailsOutput{Plan: plan, Teams: teams})
}

func clearAuthorization(
	input *component.ComponentInput[struct{}, string],
	ctx *component.ComponentContainer,
) *component.ComponentReturn[string] {
	clearStoredAuthorization(ctx)
	return input.Return("Planning Center authorization cleared.")
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

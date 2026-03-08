package definitions

import (
	"time"

	"github.com/jacksonzamorano/strata/component"
)

var Manifest = component.ComponentManifest{
	Name:    "pco",
	Version: "1.0.2",
}

var ListServiceTypes = component.Define[struct{}, ListServiceTypesOutput](Manifest, "list-service-types")
var ListPlans = component.Define[ListPlansInput, ListPlansOutput](Manifest, "list-plans")
var GetPlanDetails = component.Define[GetPlanDetailsInput, GetPlanDetailsOutput](Manifest, "get-plan-details")
var ClearAuthorization = component.Define[struct{}, string](Manifest, "clear-authorization")

type ListServiceTypesOutput struct {
	Items []ServiceTypeSummary `json:"items"`
}

type ListPlansInput struct {
	ServiceTypeID string    `json:"serviceTypeId"`
	From          time.Time `json:"from"`
	To            time.Time `json:"to"`
}

type ListPlansOutput struct {
	Items []PlanSummary `json:"items"`
}

type GetPlanDetailsInput struct {
	ServiceTypeID string `json:"serviceTypeId"`
	PlanID        string `json:"planId"`
}

type GetPlanDetailsOutput struct {
	Plan  PlanSummary `json:"plan"`
	Teams []PlanTeam  `json:"teams"`
}

type ServiceTypeSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlanSummary struct {
	ID             string     `json:"id"`
	Title          string     `json:"title"`
	SeriesTitle    string     `json:"seriesTitle"`
	SortDate       time.Time  `json:"sortDate"`
	FirstServiceAt *time.Time `json:"firstServiceAt,omitempty"`
	LastTimeAt     *time.Time `json:"lastTimeAt,omitempty"`
}

type PlanTeam struct {
	ID          string           `json:"id"`
	Name        string           `json:"name"`
	Assignments []TeamAssignment `json:"assignments"`
}

type TeamAssignment struct {
	PersonID string `json:"personId"`
	Name     string `json:"name"`
	Position string `json:"position"`
	Status   string `json:"status"`
}

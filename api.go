package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	d "github.com/jacksonzamorano/strata-pco/definitions"
	"github.com/jacksonzamorano/strata/component"
)

const (
	pcoBaseURL          = "https://api.planningcenteronline.com/services/v2"
	pcoApplicationIDKey = "pco_application_id"
	pcoSecretKey        = "pco_secret"
	defaultHTTPTimeout  = 15 * time.Second
)

var httpClient = &http.Client{Timeout: defaultHTTPTimeout}

type authorization struct {
	applicationID string
	secret        string
}

type pcoCollectionResponse[T any] struct {
	Data     []T                   `json:"data"`
	Included []pcoIncludedResource `json:"included"`
	Links    pcoLinks              `json:"links"`
}

type pcoDocumentResponse[T any] struct {
	Data     T                     `json:"data"`
	Included []pcoIncludedResource `json:"included"`
}

type pcoLinks struct {
	Next string `json:"next"`
}

type pcoIncludedResource struct {
	ID         string                     `json:"id"`
	Type       string                     `json:"type"`
	Attributes map[string]json.RawMessage `json:"attributes"`
}

type pcoErrorResponse struct {
	Errors []struct {
		Status string `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"errors"`
}

type pcoServiceTypeResource struct {
	ID         string                   `json:"id"`
	Attributes pcoServiceTypeAttributes `json:"attributes"`
}

type pcoServiceTypeAttributes struct {
	Name string `json:"name"`
}

type pcoPlanResource struct {
	ID            string               `json:"id"`
	Attributes    pcoPlanAttributes    `json:"attributes"`
	Relationships pcoPlanRelationships `json:"relationships"`
}

type pcoPlanAttributes struct {
	Title       string  `json:"title"`
	SeriesTitle string  `json:"series_title"`
	SortDate    string  `json:"sort_date"`
	LastTimeAt  *string `json:"last_time_at"`
}

type pcoPlanRelationships struct {
	Series pcoRelationship `json:"series"`
}

type pcoTeamMemberResource struct {
	ID            string                     `json:"id"`
	Attributes    pcoTeamMemberAttributes    `json:"attributes"`
	Relationships pcoTeamMemberRelationships `json:"relationships"`
}

type pcoTeamMemberAttributes struct {
	Status           string `json:"status"`
	TeamPositionName string `json:"team_position_name"`
}

type pcoTeamMemberRelationships struct {
	Person pcoRelationship `json:"person"`
	Team   pcoRelationship `json:"team"`
}

type pcoRelationship struct {
	Data *pcoRelationshipData `json:"data"`
}

type pcoRelationshipData struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

func fetchServiceTypes(ctx *component.ComponentContainer) ([]d.ServiceTypeSummary, error) {
	query := url.Values{}
	query.Set("per_page", "100")

	payload, err := pcoRequestAllPages[pcoServiceTypeResource](ctx, http.MethodGet, "/service_types", query)
	if err != nil {
		return nil, err
	}

	items := make([]d.ServiceTypeSummary, 0, len(payload.Data))
	for _, item := range payload.Data {
		items = append(items, d.ServiceTypeSummary{
			ID:   item.ID,
			Name: item.Attributes.Name,
		})
	}
	return items, nil
}

func fetchPlans(ctx *component.ComponentContainer, input d.ListPlansInput) ([]d.PlanSummary, error) {
	if len(strings.TrimSpace(input.ServiceTypeID)) == 0 {
		return nil, errors.New("service type ID is required")
	}
	if !input.From.IsZero() && !input.To.IsZero() && input.To.Before(input.From) {
		return nil, errors.New("end range must be on or after start range")
	}

	query := url.Values{}
	query.Set("per_page", "100")
	query.Set("order", "sort_date")

	filters := []string{}
	if !input.From.IsZero() {
		filters = append(filters, "after")
		query.Set("after", input.From.Format(time.RFC3339))
	}
	if !input.To.IsZero() {
		filters = append(filters, "before")
		query.Set("before", input.To.Format(time.RFC3339))
	}
	if len(filters) > 0 {
		query.Set("filter", strings.Join(filters, ","))
	}

	endpoint := fmt.Sprintf("/service_types/%s/plans", url.PathEscape(input.ServiceTypeID))
	payload, err := pcoRequestAllPages[pcoPlanResource](ctx, http.MethodGet, endpoint, query)
	if err != nil {
		return nil, err
	}

	items := make([]d.PlanSummary, 0, len(payload.Data))
	for _, item := range payload.Data {
		items = append(items, normalizePlan(item, payload.Included))
	}
	return filterPlansByRange(items, input.From, input.To), nil
}

func fetchPlanDetails(ctx *component.ComponentContainer, input d.GetPlanDetailsInput) (d.PlanSummary, []d.PlanTeam, error) {
	var zero d.PlanSummary

	if len(strings.TrimSpace(input.ServiceTypeID)) == 0 {
		return zero, nil, errors.New("service type ID is required")
	}
	if len(strings.TrimSpace(input.PlanID)) == 0 {
		return zero, nil, errors.New("plan ID is required")
	}

	planEndpoint := fmt.Sprintf("/service_types/%s/plans/%s", url.PathEscape(input.ServiceTypeID), url.PathEscape(input.PlanID))
	planPayload, err := pcoRequest[pcoDocumentResponse[pcoPlanResource]](ctx, http.MethodGet, planEndpoint, nil)
	if err != nil {
		return zero, nil, err
	}

	membersEndpoint := fmt.Sprintf("/service_types/%s/plans/%s/team_members", url.PathEscape(input.ServiceTypeID), url.PathEscape(input.PlanID))
	memberQuery := url.Values{}
	memberQuery.Set("per_page", "100")
	memberQuery.Set("include", "person,team")
	memberPayload, err := pcoRequestAllPages[pcoTeamMemberResource](ctx, http.MethodGet, membersEndpoint, memberQuery)
	if err != nil {
		return zero, nil, err
	}

	return normalizePlan(planPayload.Data, planPayload.Included), normalizePlanTeams(memberPayload.Data, memberPayload.Included), nil
}

func pcoRequest[T any](ctx *component.ComponentContainer, method, endpoint string, query url.Values) (T, error) {
	requestURL := pcoBaseURL + endpoint
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}
	return pcoRequestURL[T](ctx, method, requestURL)
}

func pcoRequestURL[T any](ctx *component.ComponentContainer, method, requestURL string) (T, error) {
	var output T

	auth, err := getAuthorization(ctx)
	if err != nil {
		return output, err
	}

	req, err := http.NewRequest(method, requestURL, nil)
	if err != nil {
		return output, err
	}
	req.SetBasicAuth(auth.applicationID, auth.secret)
	req.Header.Set("Accept", "application/json")

	res, err := httpClient.Do(req)
	if err != nil {
		return output, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return output, err
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return output, formatPCOError(res.StatusCode, body)
	}
	if err := json.Unmarshal(body, &output); err != nil {
		return output, fmt.Errorf("failed to decode Planning Center response: %w", err)
	}
	return output, nil
}

func pcoRequestAllPages[T any](ctx *component.ComponentContainer, method, endpoint string, query url.Values) (pcoCollectionResponse[T], error) {
	requestURL := pcoBaseURL + endpoint
	if len(query) > 0 {
		requestURL += "?" + query.Encode()
	}

	var merged pcoCollectionResponse[T]
	seenIncluded := map[string]struct{}{}

	for len(requestURL) > 0 {
		page, err := pcoRequestURL[pcoCollectionResponse[T]](ctx, method, requestURL)
		if err != nil {
			return merged, err
		}
		merged.Data = append(merged.Data, page.Data...)
		for _, item := range page.Included {
			key := item.Type + ":" + item.ID
			if _, ok := seenIncluded[key]; ok {
				continue
			}
			seenIncluded[key] = struct{}{}
			merged.Included = append(merged.Included, item)
		}
		requestURL = page.Links.Next
	}

	return merged, nil
}

func getAuthorization(ctx *component.ComponentContainer) (authorization, error) {
	appID := strings.TrimSpace(ctx.Storage.GetString(pcoApplicationIDKey))
	secret := strings.TrimSpace(ctx.Storage.GetString(pcoSecretKey))

	if len(appID) > 0 && len(secret) > 0 {
		return authorization{applicationID: appID, secret: secret}, nil
	}

	missingAppID := len(appID) == 0
	missingSecret := len(secret) == 0

	if missingAppID {
		value, ok := ctx.RequestSecret("Planning Center application ID")
		if !ok || len(strings.TrimSpace(value)) == 0 {
			return authorization{}, errors.New("planning center application ID prompt was cancelled")
		}
		appID = strings.TrimSpace(value)
	}
	if missingSecret {
		value, ok := ctx.RequestSecret("Planning Center secret")
		if !ok || len(strings.TrimSpace(value)) == 0 {
			return authorization{}, errors.New("planning center secret prompt was cancelled")
		}
		secret = strings.TrimSpace(value)
	}

	if missingAppID {
		_ = ctx.Storage.SetString(pcoApplicationIDKey, appID)
	}
	if missingSecret {
		_ = ctx.Storage.SetString(pcoSecretKey, secret)
	}

	return authorization{applicationID: appID, secret: secret}, nil
}

func clearStoredAuthorization(ctx *component.ComponentContainer) {
	_ = ctx.Storage.SetString(pcoApplicationIDKey, "")
	_ = ctx.Storage.SetString(pcoSecretKey, "")
}

func formatPCOError(statusCode int, body []byte) error {
	var payload pcoErrorResponse
	if err := json.Unmarshal(body, &payload); err == nil && len(payload.Errors) > 0 {
		parts := make([]string, 0, len(payload.Errors))
		for _, item := range payload.Errors {
			msg := strings.TrimSpace(strings.Join([]string{item.Title, item.Detail}, ": "))
			msg = strings.Trim(msg, ": ")
			if len(msg) > 0 {
				parts = append(parts, msg)
			}
		}
		if len(parts) > 0 {
			return fmt.Errorf("planning center request failed (%d): %s", statusCode, strings.Join(parts, "; "))
		}
	}

	trimmed := strings.TrimSpace(string(body))
	if len(trimmed) == 0 {
		return fmt.Errorf("planning center request failed with status %d", statusCode)
	}
	return fmt.Errorf("planning center request failed (%d): %s", statusCode, trimmed)
}

func normalizePlan(resource pcoPlanResource, included []pcoIncludedResource) d.PlanSummary {
	var lastTimeAt *time.Time
	if resource.Attributes.LastTimeAt != nil {
		if parsed, ok := parsePCOTime(*resource.Attributes.LastTimeAt); ok {
			lastTimeAt = &parsed
		}
	}

	plan := d.PlanSummary{
		ID:          resource.ID,
		Title:       resource.Attributes.Title,
		SeriesTitle: resource.Attributes.SeriesTitle,
		SortDate:    mustParsePCOTime(resource.Attributes.SortDate),
		LastTimeAt:  lastTimeAt,
	}
	if len(plan.SeriesTitle) == 0 {
		plan.SeriesTitle = seriesTitleForPlan(resource.Relationships.Series, included)
	}
	return plan
}

func normalizePlanTeams(resources []pcoTeamMemberResource, included []pcoIncludedResource) []d.PlanTeam {
	peopleByID := map[string]string{}
	teamsByID := map[string]string{}
	for _, item := range included {
		switch item.Type {
		case "Person":
			peopleByID[item.ID] = readIncludedString(item, "name", "full_name")
		case "Team":
			teamsByID[item.ID] = readIncludedString(item, "name")
		}
	}

	orderedTeamIDs := []string{}
	teams := map[string]*d.PlanTeam{}
	for _, member := range resources {
		teamID := member.ID
		if member.Relationships.Team.Data != nil && len(member.Relationships.Team.Data.ID) > 0 {
			teamID = member.Relationships.Team.Data.ID
		}
		teamName := teamsByID[teamID]
		if len(teamName) == 0 {
			teamName = member.Attributes.TeamPositionName
		}
		if len(teamName) == 0 {
			teamName = "Unassigned"
		}

		team, ok := teams[teamID]
		if !ok {
			team = &d.PlanTeam{ID: teamID, Name: teamName}
			teams[teamID] = team
			orderedTeamIDs = append(orderedTeamIDs, teamID)
		}

		assignment := d.TeamAssignment{
			PersonID: personIDForMember(member),
			Name:     personNameForMember(member, peopleByID),
			Position: member.Attributes.TeamPositionName,
			Status:   normalizeAssignmentStatus(member.Attributes.Status),
		}
		team.Assignments = append(team.Assignments, assignment)
	}

	result := make([]d.PlanTeam, 0, len(orderedTeamIDs))
	for _, teamID := range orderedTeamIDs {
		team := teams[teamID]
		sort.Slice(team.Assignments, func(i, j int) bool {
			return strings.ToLower(team.Assignments[i].Name) < strings.ToLower(team.Assignments[j].Name)
		})
		result = append(result, *team)
	}
	return result
}

func normalizeAssignmentStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "p", "pending":
		return "pending"
	case "u", "unconfirmed":
		return "unconfirmed"
	case "c", "confirmed":
		return "confirmed"
	case "d", "declined":
		return "declined"
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func filterPlansByRange(items []d.PlanSummary, from, to time.Time) []d.PlanSummary {
	if from.IsZero() && to.IsZero() {
		return items
	}

	filtered := make([]d.PlanSummary, 0, len(items))
	for _, item := range items {
		if item.SortDate.IsZero() {
			continue
		}
		if !from.IsZero() && item.SortDate.Before(from) {
			continue
		}
		if !to.IsZero() && item.SortDate.After(to) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func personIDForMember(member pcoTeamMemberResource) string {
	if member.Relationships.Person.Data != nil {
		return member.Relationships.Person.Data.ID
	}
	return ""
}

func personNameForMember(member pcoTeamMemberResource, peopleByID map[string]string) string {
	personID := personIDForMember(member)
	if len(personID) == 0 {
		return ""
	}
	return peopleByID[personID]
}

func seriesTitleForPlan(series pcoRelationship, included []pcoIncludedResource) string {
	if series.Data == nil {
		return ""
	}
	for _, item := range included {
		if item.Type != series.Data.Type || item.ID != series.Data.ID {
			continue
		}
		return readIncludedString(item, "title", "name")
	}
	return ""
}

func readIncludedString(resource pcoIncludedResource, keys ...string) string {
	for _, key := range keys {
		raw, ok := resource.Attributes[key]
		if !ok {
			continue
		}
		var value string
		if err := json.Unmarshal(raw, &value); err == nil {
			return value
		}
	}
	return ""
}

func mustParsePCOTime(value string) time.Time {
	parsed, ok := parsePCOTime(value)
	if !ok {
		return time.Time{}
	}
	return parsed
}

func parsePCOTime(value string) (time.Time, bool) {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) == 0 {
		return time.Time{}, false
	}

	layouts := []string{time.RFC3339, "2006-01-02"}
	for _, layout := range layouts {
		parsed, err := time.Parse(layout, trimmed)
		if err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

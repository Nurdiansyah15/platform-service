package api_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"platform-service/internal/testhelper"
)

func benefitsPath(solutionID string) string {
	return "/api/v1/solutions/" + solutionID + "/benefits"
}

type benefitData struct {
	ID                  string  `json:"id"`
	SolutionID          string  `json:"solution_id"`
	Code                string  `json:"code"`
	ValueType           string  `json:"value_type"`
	AggregationStrategy string  `json:"aggregation_strategy"`
	Unit                *string `json:"unit"`
	Metering            string  `json:"metering"`
	UsageReset          string  `json:"usage_reset"`
}

func benefitBody(code string) map[string]any {
	return map[string]any{"code": code, "name": "Max Employees", "value_type": "integer", "aggregation_strategy": "sum"}
}

// ── Create ───────────────────────────────────────────────────────────────────

func TestBenefit_Create_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.POST(benefitsPath(uuid.NewString()), benefitBody("max_employees")).Code)
}

func TestBenefit_Create_Forbidden_NonAdmin(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	assert.Equal(t, 403, testSrv.POST(benefitsPath(solID), benefitBody("max_employees"), userHeader()).Code)
}

func TestBenefit_Create_Forbidden_LicenseManaged(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	w := testOnPremSrv.POST(benefitsPath(solID), benefitBody("max_employees"), adminHeader())
	require.Equal(t, 403, w.Code, w.Body.String())
	assert.Equal(t, "LICENSE_MANAGED", testhelper.DecodeError(t, w).ErrorCode)
}

func TestBenefit_Create_InvalidPayload(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	tests := map[string]map[string]any{
		"tanpa value_type":           {"code": "a", "name": "A", "aggregation_strategy": "sum"},
		"value_type tak dikenal":     {"code": "a", "name": "A", "value_type": "float", "aggregation_strategy": "sum"},
		"code kebab-case":            {"code": "max-employees", "name": "A", "value_type": "integer", "aggregation_strategy": "sum"},
		"boolean dengan sum":         {"code": "a", "name": "A", "value_type": "boolean", "aggregation_strategy": "sum"},
		"string di-meter":            {"code": "a", "name": "A", "value_type": "string", "aggregation_strategy": "override", "metering": "counter"},
		"gauge dengan reset bulanan": {"code": "a", "name": "A", "value_type": "integer", "aggregation_strategy": "max", "metering": "gauge", "usage_reset": "monthly"},
		"unit terlalu panjang":       {"code": "a", "name": "A", "value_type": "integer", "aggregation_strategy": "sum", "unit": strings.Repeat("u", 51)},
	}
	for name, body := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, 422, testSrv.POST(benefitsPath(solID), body, adminHeader()).Code)
		})
	}
}

func TestBenefit_Create_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.POST(benefitsPath(uuid.NewString()), benefitBody("max_employees"), adminHeader()).Code)
}

func TestBenefit_Create_Duplicate(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	mustCreateBenefit(t, solID, "max_employees", "integer", "sum", nil)
	assert.Equal(t, 409, testSrv.POST(benefitsPath(solID), benefitBody("max_employees"), adminHeader()).Code)

	// Code sama di solution lain boleh.
	otherSol := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	mustCreateBenefit(t, otherSol, "max_employees", "integer", "sum", nil)
}

func TestBenefit_Create_Success_MeteredCounter(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	w := testSrv.POST(benefitsPath(solID), map[string]any{
		"code": "api_calls", "name": "API Calls", "value_type": "integer", "aggregation_strategy": "sum",
		"unit": "call", "metering": "counter", "usage_reset": "monthly",
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())

	var got benefitData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, solID, got.SolutionID)
	assert.Equal(t, "api_calls", got.Code)
	assert.Equal(t, "counter", got.Metering)
	assert.Equal(t, "monthly", got.UsageReset)
	require.NotNil(t, got.Unit)
	assert.Equal(t, "call", *got.Unit)
}

func TestBenefit_Create_Success_Defaults(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	w := testSrv.POST(benefitsPath(solID), map[string]any{
		"code": "export_pdf", "name": "Export PDF", "value_type": "boolean", "aggregation_strategy": "any_true",
	}, adminHeader())
	require.Equal(t, 201, w.Code, w.Body.String())
	var got benefitData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "none", got.Metering)
	assert.Equal(t, "never", got.UsageReset)
	assert.Nil(t, got.Unit)
}

// ── List ─────────────────────────────────────────────────────────────────────

func TestBenefit_List_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET(benefitsPath(uuid.NewString())).Code)
}

func TestBenefit_List_SolutionNotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET(benefitsPath(uuid.NewString()), userHeader()).Code)
}

func TestBenefit_List_InactiveSolutionHiddenFromNonAdmin(t *testing.T) {
	solID := testhelper.MustSeedSolution(t, testSrv.DB, testhelper.UniqueCode("off"), "inactive")
	assert.Equal(t, 404, testSrv.GET(benefitsPath(solID), userHeader()).Code)
	assert.Equal(t, 200, testSrv.GET(benefitsPath(solID), adminHeader()).Code)
}

func TestBenefit_List_Success_Filter(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	mustCreateBenefit(t, solID, "max_employees", "integer", "sum", nil)
	mustCreateBenefit(t, solID, "api_calls", "integer", "sum", map[string]any{"metering": "counter"})
	mustCreateBenefit(t, solID, "export_pdf", "boolean", "any_true", nil)

	w := testSrv.GET(benefitsPath(solID)+"?value_type=integer", userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var resp struct {
		Data []benefitData `json:"data"`
		Meta struct {
			Total int64 `json:"total"`
		} `json:"meta"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	codes := []string{}
	for _, b := range resp.Data {
		codes = append(codes, b.Code)
	}
	assert.Equal(t, []string{"api_calls", "max_employees"}, codes)
	assert.Equal(t, int64(2), resp.Meta.Total)

	w = testSrv.GET(benefitsPath(solID)+"?metering=counter", userHeader())
	require.Equal(t, 200, w.Code)
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.Len(t, resp.Data, 1)
	assert.Equal(t, "api_calls", resp.Data[0].Code)
}

func TestBenefit_List_ServiceToken(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	mustCreateBenefit(t, solID, "max_employees", "integer", "sum", nil)
	c := mustCreateClient(t, solID, "entitlements:read")
	assert.Equal(t, 200, testSrv.GET(benefitsPath(solID), testhelper.BearerHeader(mustServiceToken(t, c, ""))).Code)
}

// ── Get ──────────────────────────────────────────────────────────────────────

func TestBenefit_Get_Unauthenticated(t *testing.T) {
	assert.Equal(t, 401, testSrv.GET("/api/v1/benefits/"+uuid.NewString()).Code)
}

func TestBenefit_Get_InvalidID(t *testing.T) {
	assert.Equal(t, 422, testSrv.GET("/api/v1/benefits/abc", userHeader()).Code)
}

func TestBenefit_Get_NotFound(t *testing.T) {
	assert.Equal(t, 404, testSrv.GET("/api/v1/benefits/"+uuid.NewString(), userHeader()).Code)
}

func TestBenefit_Get_InactiveSolutionHidden(t *testing.T) {
	code := testhelper.UniqueCode("bn")
	solID := mustCreateSolution(t, code)
	id := mustCreateBenefit(t, solID, "max_employees", "integer", "sum", nil)
	_, err := testSrv.DB.ExecContext(context.Background(), `UPDATE solutions SET status = 'inactive' WHERE id = $1`, solID)
	require.NoError(t, err)

	w := testSrv.GET("/api/v1/benefits/"+id, userHeader())
	require.Equal(t, 404, w.Code)
	assert.Equal(t, "Benefit not found.", errorMessage(t, testhelper.DecodeError(t, w)))
}

func TestBenefit_Get_Success(t *testing.T) {
	solID := mustCreateSolution(t, testhelper.UniqueCode("bn"))
	id := mustCreateBenefit(t, solID, "storage_gb", "decimal", "max", map[string]any{"metering": "gauge", "unit": "GB"})

	w := testSrv.GET("/api/v1/benefits/"+id, userHeader())
	require.Equal(t, 200, w.Code, w.Body.String())
	var got benefitData
	testhelper.DecodeBody(t, w, &got)
	assert.Equal(t, "storage_gb", got.Code)
	assert.Equal(t, "decimal", got.ValueType)
	assert.Equal(t, "max", got.AggregationStrategy)
	assert.Equal(t, "gauge", got.Metering)
	assert.Equal(t, "never", got.UsageReset)
}

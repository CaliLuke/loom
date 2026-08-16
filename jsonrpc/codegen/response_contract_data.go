package codegen

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	httpcodegen "github.com/CaliLuke/loom/http/codegen"
	"github.com/CaliLuke/loom/jsonrpc/codegen/internal/transportir"
)

type (
	responseContractServiceData struct {
		Service   *httpcodegen.ServiceData
		Endpoints []*responseContractEndpointData
	}

	responseContractEndpointData struct {
		MethodName string
		CasesInit  string
		Cases      []*responseContractCaseData
		Warnings   []string
	}

	responseContractCaseData struct {
		ID            string
		Kind          transportir.ResponseContractCaseKind
		ResultType    string
		HasResult     bool
		ErrorCode     int
		ErrorName     string
		ErrorDataType string
		Stream        *responseContractStreamData
	}

	responseContractStreamData struct {
		Transport string
		Terminal  string
	}
)

func buildResponseContractServiceData(service *expr.HTTPServiceExpr, data *httpcodegen.ServicesData) *responseContractServiceData {
	serviceData := data.Get(service.Name())
	result := &responseContractServiceData{Service: serviceData}
	for _, endpoint := range service.HTTPEndpoints {
		analysis := transportir.AnalyzeResponseContractCases(endpoint)
		methodData := serviceData.Endpoint(endpoint.Name()).Method
		endpointData := &responseContractEndpointData{
			MethodName: methodData.Name,
			CasesInit:  methodData.VarName + "ResponseContractCases",
		}
		if !analysis.Supported() {
			for _, limitation := range analysis.Limitations {
				endpointData.Warnings = append(endpointData.Warnings, fmt.Sprintf(
					"JSON-RPC response contract omitted for %s.%s: %s: %s",
					service.Name(),
					endpoint.Name(),
					limitation.Code,
					limitation.Detail,
				))
			}
			result.Endpoints = append(result.Endpoints, endpointData)
			continue
		}
		for _, contractCase := range analysis.Cases {
			caseData := &responseContractCaseData{
				ID:            contractCase.ID,
				Kind:          contractCase.Kind,
				ResultType:    contractCase.ResultType,
				HasResult:     contractCase.HasResult,
				ErrorCode:     contractCase.ErrorCode,
				ErrorName:     contractCase.ErrorName,
				ErrorDataType: contractCase.ErrorDataType,
			}
			if contractCase.Stream != nil {
				caseData.Stream = &responseContractStreamData{
					Transport: contractCase.Stream.Transport,
					Terminal:  contractCase.Stream.Terminal,
				}
			}
			endpointData.Cases = append(endpointData.Cases, caseData)
		}
		result.Endpoints = append(result.Endpoints, endpointData)
	}
	return result
}

func (data *responseContractServiceData) hasCases() bool {
	for _, endpoint := range data.Endpoints {
		if len(endpoint.Cases) > 0 {
			return true
		}
	}
	return false
}

func (data *responseContractServiceData) warnings() []string {
	var warnings []string
	for _, endpoint := range data.Endpoints {
		warnings = append(warnings, endpoint.Warnings...)
	}
	return warnings
}

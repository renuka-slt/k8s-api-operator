/*
 *  Copyright (c) 2020, WSO2 Inc. (http://www.wso2.org) All Rights Reserved.
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *  http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */
package swaggerOperator

import (
	"encoding/json"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-openapi/spec"
	logger "github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/loggers"
	"github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/oasparser/models/apiDefinition"
	"github.com/wso2/k8s-api-operator/api-operator/pkg/envoy/pkg/oasparser/utills"
)

/**
 * Generate mgw swagger instance.
 *
 * @param location   Swagger file location
 * @return []apiDefinition.MgwSwagger  Mgw swagger instances as a array
 * @return error  Error
 */
func GenerateMgwSwagger(swaggerDefgs *map[string]string) ([]apiDefinition.MgwSwagger, error) {
	var mgwSwaggers []apiDefinition.MgwSwagger

	for _, f := range *swaggerDefgs {
		mgwSwagger := GetMgwSwagger([]byte(f))
		mgwSwaggers = append(mgwSwaggers, mgwSwagger)

	}
	return mgwSwaggers, nil
}

/**
 * Get mgw swagger instance.
 *
 * @param apiContent   Api content as a byte array
 * @return apiDefinition.MgwSwagger  Mgw swagger instance
 */
func GetMgwSwagger(apiContent []byte) apiDefinition.MgwSwagger {
	var mgwSwagger apiDefinition.MgwSwagger

	apiJsn, err := utills.ToJSON(apiContent)
	if err != nil {
		//log.Fatal("Error converting api file to json:", err)
	}

	swaggerVerison := utills.FindSwaggerVersion(apiJsn)

	if swaggerVerison == "2" {
		//map json to struct
		var ApiData2 spec.Swagger
		err = json.Unmarshal(apiJsn, &ApiData2)
		if err != nil {
			//log.Fatal("Error openAPI unmarsheliing: %v\n", err)
			logger.LoggerOasparser.Error("Error openAPI unmarsheliing", err)
		} else {
			mgwSwagger.SetInfoSwagger(ApiData2)
		}

	} else if swaggerVerison == "3" {
		//map json to struct
		var ApiData3 *openapi3.Swagger

		err = json.Unmarshal(apiJsn, &ApiData3)

		if err != nil {
			//log.Fatal("Error openAPI unmarsheliing: %v\n", err)
			logger.LoggerOasparser.Error("Error openAPI unmarsheliing", err)
		} else {
			mgwSwagger.SetInfoOpenApi(*ApiData3)
		}
	}

	mgwSwagger.SetXWso2Extenstions()
	return mgwSwagger
}

/**
 * Check availability of endpoint.
 *
 * @param endpoints  Api endpoints array
 * @return bool Availability as a bool value
 */
func IsEndpointsAvailable(endpoints []apiDefinition.Endpoint) bool {
	if len(endpoints) > 0 {
		return true
	} else {
		return false
	}
}

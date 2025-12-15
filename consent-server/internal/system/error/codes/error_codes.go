/*
 * Copyright (c) 2025, WSO2 LLC. (https://www.wso2.com).
 *
 * WSO2 LLC. licenses this file to you under the Apache License,
 * Version 2.0 (the "License"); you may not use this file except
 * in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 */

package codes

// Error codes for the Consent Management Service
const (
	// General errors
	InternalServerError = "CSE-5000"
	DatabaseError       = "CSE-5001"
	InvalidRequest      = "CSE-4000"
	ValidationError     = "CSE-4001"
	ResourceNotFound    = "CSE-4004"
	ConflictError       = "CSE-4009"

	// Consent-specific errors
	ConsentNotFound         = "CSE-4040"
	ConsentCreationFailed   = "CSE-5010"
	ConsentUpdateFailed     = "CSE-5011"
	ConsentValidationFailed = "CSE-4041"
	ConsentRevokeFailed     = "CSE-5012"
	ConsentExpireFailed     = "CSE-5013"
	ConsentAttributeInvalid = "CSE-4042"
	ConsentStatusInvalid    = "CSE-4043"

	// Purpose-specific errors
	PurposeNotFound         = "CSE-4050"
	PurposeCreationFailed   = "CSE-5020"
	PurposeUpdateFailed     = "CSE-5021"
	PurposeDeleteFailed     = "CSE-5022"
	PurposeValidationFailed = "CSE-4051"
	PurposeInUse            = "CSE-4052"

	// Auth Resource-specific errors
	AuthResourceNotFound         = "CSE-4060"
	AuthResourceCreationFailed   = "CSE-5030"
	AuthResourceUpdateFailed     = "CSE-5031"
	AuthResourceDeleteFailed     = "CSE-5032"
	AuthResourceValidationFailed = "CSE-4061"
)

# API Documentation

## Endpoint Summary

- **Name:** {{name}}
- **HTTP Method:** {{method}}
- **Path:** {{path}}
- **Source:** {{sourceFile}}:{{sourceLine}}
- **Confidence:** {{confidence}}

## Description

{{descriptionOrNotDeterminedFromCode}}

## Request

### Path Parameters

{{pathParametersOrNotDeterminedFromCode}}

### Query Parameters

{{queryParametersOrNotDeterminedFromCode}}

### Headers

{{headersOrNotDeterminedFromCode}}

### Body

{{requestBodyOrNotDeterminedFromCode}}

### Validation

{{validationRulesOrNotDeterminedFromCode}}

## Response

### Success Status

{{successStatusOrNotDeterminedFromCode}}

### Body

{{responseBodyOrNotDeterminedFromCode}}

## Error Codes

| Code | Provenance | Meaning | Evidence |
| --- | --- | --- | --- |
| {{code}} | {{DECLARED_OR_REFERRED_OR_INFERRED}} | {{meaningOrNotDeterminedFromCode}} | {{sourceEvidence}} |

## Examples

### Request Example

{{validatedRequestExampleOrNotDeterminedFromCode}}

### Response Example

{{validatedResponseExampleOrNotDeterminedFromCode}}

## Evidence

- **Controller:** {{controllerEvidence}}
- **DTO / Validation:** {{dtoEvidenceOrNotDeterminedFromCode}}
- **Exception / Error mapping:** {{errorEvidenceOrNotDeterminedFromCode}}

> Unknown or unsupported fields must be written exactly as `Not determined from code`.
